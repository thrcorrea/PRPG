package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v70/github"
	"github.com/thrcorrea/PRPG/internal/database"
)

// CachedGithubAdapter é um wrapper que adiciona cache em banco de dados ao GithubAdapter
type CachedGithubAdapter struct {
	githubClient GithubAdapter
	db           database.CommentDatabase
}

// NewCachedGithubAdapter cria um novo adaptador com cache em banco de dados
func NewCachedGithubAdapter(token string, dbPath string) (CacheableGithubAdapter, error) {
	githubClient := NewGithubClient(token)

	db, err := database.NewSQLiteDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao inicializar banco de dados: %v", err)
	}

	return &CachedGithubAdapter{
		githubClient: githubClient,
		db:           db,
	}, nil
}

// ensurePRExists garante que o PR existe no cache com dados completos
func (c *CachedGithubAdapter) ensurePRExists(ctx context.Context, owner, repo string, prNumber int) error {
	// Verifica se o PR já existe no cache
	existingPR, err := c.db.GetPR(owner, repo, prNumber)
	if err != nil {
		return fmt.Errorf("erro ao verificar PR existente: %v", err)
	}

	// Se o PR existe e tem dados completos (título não vazio), não precisa buscar
	if existingPR != nil && existingPR.Title != "" {
		return nil
	}

	// Busca dados completos do PR da API
	pr, err := c.githubClient.GetPR(ctx, owner, repo, prNumber)
	if err != nil {
		return fmt.Errorf("erro ao buscar dados do PR da API: %v", err)
	}

	// Converte e salva o PR completo
	prData := database.FromGithubPR(pr, owner, repo)
	if err := c.db.SavePR(prData); err != nil {
		return fmt.Errorf("erro ao salvar PR completo: %v", err)
	}

	return nil
}

// FetchPRsForRepo implementa a interface GithubAdapter (sem cache para PRs)
func (c *CachedGithubAdapter) FetchPRsForRepo(owner, name string, startDate, endDate time.Time) ([]*github.PullRequest, error) {
	// Para PRs, não aplicamos cache pois são menos frequentes e mudam menos
	return c.githubClient.FetchPRsForRepo(owner, name, startDate, endDate)
}

// GetPR implementa a interface GithubAdapter
func (c *CachedGithubAdapter) GetPR(ctx context.Context, owner, repo string, prNumber int) (*github.PullRequest, error) {
	return c.githubClient.GetPR(ctx, owner, repo, prNumber)
}

// ListPRComments busca comentários de um PR com cache
func (c *CachedGithubAdapter) ListPRComments(ctx context.Context, owner, repo string, prNumber int) ([]*github.IssueComment, error) {
	// Primeiro verifica se já temos informações sobre este PR
	prData, err := c.db.GetPR(owner, repo, prNumber)
	if err == nil && prData != nil {
		// Se já verificamos que este PR não tem issue comments, retorna lista vazia
		if prData.IssueCommentsChecked && !prData.HasIssueComments {
			fmt.Printf("    📋 Cache HIT: PR #%d em %s/%s confirmado sem issue comments\n", prNumber, owner, repo)
			return []*github.IssueComment{}, nil
		}
	}

	// Busca comentários existentes no cache
	cachedComments, err := c.db.GetCommentsByPRAndType(owner, repo, prNumber, "issue")
	if err != nil {
		fmt.Printf("    ⚠️  Erro ao buscar comentários do cache: %v\n", err)
		// Continua para buscar da API em caso de erro
	}

	// Verifica se temos comentários válidos no cache
	if len(cachedComments) > 0 && !c.areCommentsStale(cachedComments) {
		// fmt.Printf("    📋 Cache HIT: Comentários do PR #%d em %s/%s\n", prNumber, owner, repo)
		return c.convertCachedCommentsToGithub(cachedComments), nil
	}

	// Cache MISS - busca da API
	fmt.Printf("    🌐 Cache MISS: Buscando comentários do PR #%d em %s/%s da API\n", prNumber, owner, repo)

	// Garante que temos dados completos do PR antes de buscar comentários
	if err := c.ensurePRExists(ctx, owner, repo, prNumber); err != nil {
		fmt.Printf("    ⚠️  Erro ao garantir dados do PR: %v\n", err)
		// Continua mesmo com erro, pois os comentários ainda podem ser buscados
	}

	comments, err := c.githubClient.ListPRComments(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, err
	}

	// Salva os comentários no cache
	for _, comment := range comments {
		commentData := database.FromGithubIssueComment(comment, owner, repo, prNumber)
		if err := c.db.SaveComment(commentData); err != nil {
			fmt.Printf("    ⚠️  Erro ao salvar comentário no cache: %v\n", err)
		}
	}

	// Marca o PR como verificado para issue comments
	hasComments := len(comments) > 0
	if err := c.db.MarkPRCommentsChecked(owner, repo, prNumber, "issue", hasComments); err != nil {
		fmt.Printf("    ⚠️  Erro ao marcar PR como verificado: %v\n", err)
	}

	return comments, nil
}

// ListPRReviewComments busca review comments de um PR com cache
func (c *CachedGithubAdapter) ListPRReviewComments(ctx context.Context, owner, repo string, prNumber int) ([]*github.PullRequestComment, error) {
	// Primeiro verifica se já temos informações sobre este PR
	prData, err := c.db.GetPR(owner, repo, prNumber)
	if err == nil && prData != nil {
		// Se já verificamos que este PR não tem review comments, retorna lista vazia
		if prData.ReviewCommentsChecked && !prData.HasReviewComments {
			fmt.Printf("    📋 Cache HIT: PR #%d em %s/%s confirmado sem review comments\n", prNumber, owner, repo)
			return []*github.PullRequestComment{}, nil
		}
	}

	// Busca review comments existentes no cache
	cachedComments, err := c.db.GetCommentsByPRAndType(owner, repo, prNumber, "review")
	if err != nil {
		fmt.Printf("    ⚠️  Erro ao buscar review comments do cache: %v\n", err)
	}

	// Verifica se temos review comments válidos no cache
	if len(cachedComments) > 0 && !c.areCommentsStale(cachedComments) {
		// fmt.Printf("    📋 Cache HIT: Review comments do PR #%d em %s/%s\n", prNumber, owner, repo)
		return c.convertCachedReviewCommentsToGithub(cachedComments), nil
	}

	// Cache MISS - busca da API
	fmt.Printf("    🌐 Cache MISS: Buscando review comments do PR #%d em %s/%s da API\n", prNumber, owner, repo)

	// Garante que temos dados completos do PR antes de buscar review comments
	if err := c.ensurePRExists(ctx, owner, repo, prNumber); err != nil {
		fmt.Printf("    ⚠️  Erro ao garantir dados do PR: %v\n", err)
		// Continua mesmo com erro, pois os review comments ainda podem ser buscados
	}

	reviewComments, err := c.githubClient.ListPRReviewComments(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, err
	}

	// Salva os review comments no cache
	for _, comment := range reviewComments {
		commentData := database.FromGithubReviewComment(comment, owner, repo, prNumber)
		if err := c.db.SaveComment(commentData); err != nil {
			fmt.Printf("    ⚠️  Erro ao salvar review comment no cache: %v\n", err)
		}
	}

	// Marca o PR como verificado para review comments
	hasComments := len(reviewComments) > 0
	if err := c.db.MarkPRCommentsChecked(owner, repo, prNumber, "review", hasComments); err != nil {
		fmt.Printf("    ⚠️  Erro ao marcar PR como verificado: %v\n", err)
	}

	return reviewComments, nil
}

// ListIssueCommentReactions busca reações de um comentário com cache
func (c *CachedGithubAdapter) ListIssueCommentReactions(ctx context.Context, owner, repo string, commentID int64) ([]*github.Reaction, error) {
	// Primeiro, verifica se o comentário existe no cache e se suas reações já foram verificadas
	comment, err := c.db.GetComment(owner, repo, commentID)
	if err != nil {
		fmt.Printf("    ⚠️  Erro ao buscar comentário do cache: %v\n", err)
	}

	// Se o comentário existe e as reações já foram verificadas, e não está stale
	if comment != nil && comment.ReactionsChecked && !c.isCommentStale(comment) {
		// Busca as reações do cache (especificamente issue_comment type)
		cachedReactions, err := c.db.GetReactionsByType(commentID, "issue_comment")
		if err != nil {
			fmt.Printf("    ⚠️  Erro ao buscar reações do cache: %v\n", err)
		} else {
			// fmt.Printf("    📋 Cache HIT: Reações do comentário %d (%d reações)\n", commentID, len(cachedReactions))
			return c.convertCachedReactionsToGithub(cachedReactions), nil
		}
	}

	// Cache MISS ou dados stale - busca da API
	fmt.Printf("    🌐 Cache MISS: Buscando reações do comentário %d da API\n", commentID)

	reactions, err := c.githubClient.ListIssueCommentReactions(ctx, owner, repo, commentID)
	if err != nil {
		return nil, err
	}

	// Salva as reações no cache (pode ser uma lista vazia)
	var reactionData []*database.ReactionData
	for _, reaction := range reactions {
		reactionData = append(reactionData, database.FromGithubReaction(reaction, commentID))
	}

	if err := c.db.SaveReactions(reactionData); err != nil {
		fmt.Printf("    ⚠️  Erro ao salvar reações no cache: %v\n", err)
	}

	// Marca que as reações deste comentário foram verificadas
	if err := c.db.MarkReactionsChecked(commentID); err != nil {
		fmt.Printf("    ⚠️  Erro ao marcar reações como verificadas: %v\n", err)
	}

	fmt.Printf("    ✅ Reações do comentário %d salvas (%d reações encontradas)\n", commentID, len(reactions))
	return reactions, nil
}

// ListPullRequestCommentReactions busca reações de um review comment com cache
func (c *CachedGithubAdapter) ListPullRequestCommentReactions(ctx context.Context, owner, repo string, commentID int64) ([]*github.Reaction, error) {
	// Primeiro, verifica se o comentário existe no cache e se suas reações já foram verificadas
	comment, err := c.db.GetComment(owner, repo, commentID)
	if err != nil {
		fmt.Printf("    ⚠️  Erro ao buscar review comment do cache: %v\n", err)
	}

	// Se o comentário existe e as reações já foram verificadas, e não está stale
	if comment != nil && comment.ReactionsChecked && !c.isCommentStale(comment) {
		// Busca as reações do cache (especificamente review_comment type)
		cachedReactions, err := c.db.GetReactionsByType(commentID, "review_comment")
		if err != nil {
			fmt.Printf("    ⚠️  Erro ao buscar reações de review comment do cache: %v\n", err)
		} else {
			// fmt.Printf("    📋 Cache HIT: Reações do review comment %d (%d reações)\n", commentID, len(cachedReactions))
			return c.convertCachedReactionsToGithub(cachedReactions), nil
		}
	}

	// Cache MISS ou dados stale - busca da API
	fmt.Printf("    🌐 Cache MISS: Buscando reações do review comment %d da API\n", commentID)

	reactions, err := c.githubClient.ListPullRequestCommentReactions(ctx, owner, repo, commentID)
	if err != nil {
		return nil, err
	}

	// Salva as reações no cache (pode ser uma lista vazia)
	var reactionData []*database.ReactionData
	for _, reaction := range reactions {
		reactionData = append(reactionData, database.FromGithubReviewReaction(reaction, commentID))
	}

	if err := c.db.SaveReactions(reactionData); err != nil {
		fmt.Printf("    ⚠️  Erro ao salvar reações de review comment no cache: %v\n", err)
	}

	// Marca que as reações deste comentário foram verificadas
	if err := c.db.MarkReactionsChecked(commentID); err != nil {
		fmt.Printf("    ⚠️  Erro ao marcar reações de review comment como verificadas: %v\n", err)
	}

	fmt.Printf("    ✅ Reações do review comment %d salvas (%d reações encontradas)\n", commentID, len(reactions))
	return reactions, nil
}

// ListPRReviews busca reviews de um PR com cache
func (c *CachedGithubAdapter) ListPRReviews(ctx context.Context, owner, repo string, prNumber int) ([]*github.PullRequestReview, error) {
	// Primeiro verifica se já temos informações sobre este PR
	prData, err := c.db.GetPR(owner, repo, prNumber)
	if err == nil && prData != nil {
		// Se já verificamos que este PR não tem reviews, retorna lista vazia
		if prData.ReviewsChecked && !prData.HasReviews {
			fmt.Printf("    📋 Cache HIT: PR #%d em %s/%s confirmado sem reviews\n", prNumber, owner, repo)
			return []*github.PullRequestReview{}, nil
		}
	}

	// Busca reviews existentes no cache
	cachedReviews, err := c.db.GetReviewsByPR(owner, repo, prNumber)
	if err != nil {
		fmt.Printf("    ⚠️  Erro ao buscar reviews do cache: %v\n", err)
	}

	// Verifica se temos reviews válidos no cache
	if len(cachedReviews) > 0 && !c.areReviewsStale(cachedReviews) {
		// fmt.Printf("    📋 Cache HIT: Reviews do PR #%d em %s/%s\n", prNumber, owner, repo)
		return c.convertCachedReviewsToGithub(cachedReviews), nil
	}

	// Cache MISS - busca da API
	fmt.Printf("    🌐 Cache MISS: Buscando reviews do PR #%d em %s/%s da API\n", prNumber, owner, repo)

	// Garante que temos dados completos do PR antes de buscar reviews
	if err := c.ensurePRExists(ctx, owner, repo, prNumber); err != nil {
		fmt.Printf("    ⚠️  Erro ao garantir dados do PR: %v\n", err)
		// Continua mesmo com erro, pois os reviews ainda podem ser buscados
	}

	reviews, err := c.githubClient.ListPRReviews(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, err
	}

	// Salva os reviews no cache
	hasApprovedReviews := false
	for _, review := range reviews {
		reviewData := database.FromGithubReview(review, owner, repo, prNumber)
		if err := c.db.SaveReview(reviewData); err != nil {
			fmt.Printf("    ⚠️  Erro ao salvar review no cache: %v\n", err)
		}

		// Verifica se tem pelo menos um review aprovado
		if review.GetState() == "APPROVED" {
			hasApprovedReviews = true
		}
	}

	// Marca o PR como verificado para reviews
	hasReviews := len(reviews) > 0
	if err := c.db.MarkPRReviewsChecked(owner, repo, prNumber, hasReviews, hasApprovedReviews); err != nil {
		fmt.Printf("    ⚠️  Erro ao marcar PR como verificado para reviews: %v\n", err)
	}

	fmt.Printf("    ✅ Reviews do PR #%d salvos (%d reviews encontrados, aprovados: %t)\n", prNumber, len(reviews), hasApprovedReviews)
	return reviews, nil
}

// ClearCache limpa todo o cache do banco de dados
func (c *CachedGithubAdapter) ClearCache() error {
	fmt.Println("🗑️  Limpando cache do banco de dados...")
	return c.db.ClearDatabase()
}

// GetDatabase retorna a instância do banco de dados para consultas diretas
func (c *CachedGithubAdapter) GetDatabase() database.CommentDatabase {
	return c.db
}

// Close fecha as conexões
func (c *CachedGithubAdapter) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// isCommentStale verifica se um comentário específico está desatualizado
func (c *CachedGithubAdapter) isCommentStale(comment *database.CommentData) bool {
	cacheDuration := 7 * 24 * time.Hour // 7 dias
	return time.Since(comment.CachedAt) > cacheDuration
}

// areCommentsStale verifica se os comentários em cache estão desatualizados
func (c *CachedGithubAdapter) areCommentsStale(comments []*database.CommentData) bool {
	cacheDuration := 7 * 24 * time.Hour // 7 dias

	for _, comment := range comments {
		if time.Since(comment.CachedAt) > cacheDuration {
			return true
		}
	}
	return false
}

// convertCachedCommentsToGithub converte comentários do cache para formato GitHub
func (c *CachedGithubAdapter) convertCachedCommentsToGithub(cachedComments []*database.CommentData) []*github.IssueComment {
	var comments []*github.IssueComment

	for _, cached := range cachedComments {
		if cached.CommentType == "issue" {
			comment := &github.IssueComment{
				ID:   &cached.CommentID,
				Body: &cached.Body,
				User: &github.User{
					Login: &cached.Username,
				},
				CreatedAt: &github.Timestamp{Time: cached.CreatedAt},
				UpdatedAt: &github.Timestamp{Time: cached.UpdatedAt},
			}
			comments = append(comments, comment)
		}
	}

	return comments
}

// convertCachedReviewCommentsToGithub converte review comments do cache para formato GitHub
func (c *CachedGithubAdapter) convertCachedReviewCommentsToGithub(cachedComments []*database.CommentData) []*github.PullRequestComment {
	var comments []*github.PullRequestComment

	for _, cached := range cachedComments {
		if cached.CommentType == "review" {
			comment := &github.PullRequestComment{
				ID:   &cached.CommentID,
				Body: &cached.Body,
				User: &github.User{
					Login: &cached.Username,
				},
				CreatedAt: &github.Timestamp{Time: cached.CreatedAt},
				UpdatedAt: &github.Timestamp{Time: cached.UpdatedAt},
			}
			comments = append(comments, comment)
		}
	}

	return comments
}

// convertCachedReactionsToGithub converte reações do cache para formato GitHub
func (c *CachedGithubAdapter) convertCachedReactionsToGithub(cachedReactions []*database.ReactionData) []*github.Reaction {
	var reactions []*github.Reaction

	for _, cached := range cachedReactions {
		reaction := &github.Reaction{
			Content: &cached.Content,
			User: &github.User{
				Login: &cached.Username,
			},
		}
		reactions = append(reactions, reaction)
	}

	return reactions
}

// areReviewsStale verifica se os reviews em cache estão desatualizados
func (c *CachedGithubAdapter) areReviewsStale(reviews []*database.ReviewData) bool {
	cacheDuration := 7 * 24 * time.Hour // 7 dias

	for _, review := range reviews {
		if time.Since(review.CachedAt) > cacheDuration {
			return true
		}
	}
	return false
}

// convertCachedReviewsToGithub converte reviews do cache para formato GitHub
func (c *CachedGithubAdapter) convertCachedReviewsToGithub(cachedReviews []*database.ReviewData) []*github.PullRequestReview {
	var reviews []*github.PullRequestReview

	for _, cached := range cachedReviews {
		review := &github.PullRequestReview{
			ID:    &cached.ReviewID,
			State: &cached.State,
			Body:  &cached.Body,
			User: &github.User{
				Login: &cached.Username,
			},
			SubmittedAt: &github.Timestamp{Time: cached.SubmittedAt},
		}
		reviews = append(reviews, review)
	}

	return reviews
}
