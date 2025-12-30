package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v70/github"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/thrcorrea/PRPG/internal/database"
	"github.com/thrcorrea/PRPG/internal/infrastructure"
)

// Repository representa um repositório para análise
type Repository struct {
	Owner              string
	Name               string
	ProductionBranches []string // Lista de branches de produção aceitas (ex: [main, master, production])
}

// UserStats representa as estatísticas de um usuário
type UserStats struct {
	Username                   string
	PRsCount                   int
	WeeklyWins                 int
	TotalScore                 int
	RepoStats                  map[string]int // PRs por repositório
	CommentsCount              int            // Total de comentários feitos pelo usuário
	CommentWeeklyWins          int            // Vitórias semanais por comentários
	CommentScore               int            // Pontuação total por comentários
	WeightedCommentScore       float64        // Pontuação ponderada por reações (👍=+2, 👎=-1)
	WeightedCommentWeeklyWins  int            // Vitórias semanais por qualidade de comentários
	WeightedCommentWeeklyScore int            // Pontuação semanal por qualidade de comentários
	TotalAdditions             int            // Total de linhas adicionadas
	TotalDeletions             int            // Total de linhas removidas
	TotalChangedFiles          int            // Total de arquivos modificados
}

// WeeklyData representa os dados de uma semana específica
type WeeklyData struct {
	StartDate             time.Time
	EndDate               time.Time
	UserPRs               map[string]int
	Winner                string
	RepoData              map[string]map[string]int // repo -> user -> PRs
	UserComments          map[string]int            // comentários por usuário na semana
	CommentWinner         string                    // vencedor da semana por comentários
	UserWeightedComments  map[string]float64        // pontuação ponderada por usuário na semana
	WeightedCommentWinner string                    // vencedor da semana por pontuação ponderada
	MergeTimes            []time.Duration           // tempos entre criação e merge dos PRs da semana
	P90MergeTime          time.Duration             // P90 do tempo de merge da semana
	MedianMergeTime       time.Duration             // Mediana do tempo de merge dos últimos 30 dias
	AverageMergeTime      time.Duration             // Média do tempo de merge dos últimos 30 dias
}

// MonthlyData representa os dados de um mês específico
type MonthlyData struct {
	Year                  int
	Month                 int
	StartDate             time.Time
	EndDate               time.Time
	UserPRs               map[string]int
	Winner                string
	RepoData              map[string]map[string]int // repo -> user -> PRs
	UserComments          map[string]int            // comentários por usuário no mês
	CommentWinner         string                    // vencedor do mês por comentários
	UserWeightedComments  map[string]float64        // pontuação ponderada por usuário no mês
	WeightedCommentWinner string                    // vencedor do mês por pontuação ponderada
	MergeTimes            []time.Duration           // tempos entre criação e merge dos PRs do mês
	P90MergeTime          time.Duration             // P90 do tempo de merge do mês
	MedianMergeTime       time.Duration             // Mediana do tempo de merge dos últimos 30 dias
	AverageMergeTime      time.Duration             // Média do tempo de merge dos últimos 30 dias
}

// PRChampion é a estrutura principal da aplicação
type PRChampion struct {
	client       infrastructure.GithubAdapter
	cachedClient infrastructure.CacheableGithubAdapter // Para operações de cache
	repositories []Repository
	startDate    time.Time
	endDate      time.Time
	weeklyData   []WeeklyData
	monthlyData  []MonthlyData
	userStats    map[string]*UserStats
}

// MetricDisplayConfig configura quais métricas exibir e formato
type MetricDisplayConfig struct {
	ShowP90     bool
	ShowMedian  bool
	ShowAverage bool
	TimeFormat  string // "hours", "days", "minutes"
	GroupBy     string // "week", "month"
}

// NewPRChampion cria uma nova instância do PR Champion
func NewPRChampion(token string, repositories []Repository, startDate, endDate time.Time) (*PRChampion, error) {
	// Cria cliente com cache em banco de dados
	cachedClient, err := infrastructure.NewCachedGithubAdapter(token, "./data/comments.db")
	if err != nil {
		return nil, fmt.Errorf("erro ao criar cliente com cache: %v", err)
	}

	return &PRChampion{
		client:       cachedClient,
		cachedClient: cachedClient,
		repositories: repositories,
		startDate:    startDate,
		endDate:      endDate,
		weeklyData:   []WeeklyData{},
		monthlyData:  []MonthlyData{},
		userStats:    make(map[string]*UserStats),
	}, nil
}

// NewPRChampionFromDatabase cria uma instância do PR Champion apenas para acessar banco de dados
func NewPRChampionFromDatabase(startDate, endDate time.Time) (*PRChampion, error) {
	// Cria cliente com cache apenas para acesso ao banco (sem token da API)
	cachedClient, err := infrastructure.NewCachedGithubAdapter("", "./data/comments.db")
	if err != nil {
		return nil, fmt.Errorf("erro ao criar acesso ao banco: %v", err)
	}

	return &PRChampion{
		client:       cachedClient,
		cachedClient: cachedClient,
		repositories: []Repository{}, // Será carregado do banco
		startDate:    startDate,
		endDate:      endDate,
		weeklyData:   []WeeklyData{},
		monthlyData:  []MonthlyData{},
		userStats:    make(map[string]*UserStats),
	}, nil
}

// ClearCache limpa todo o cache do banco de dados
func (pc *PRChampion) ClearCache() error {
	if pc.cachedClient == nil {
		return fmt.Errorf("cliente com cache não está disponível")
	}
	return pc.cachedClient.ClearCache()
}

// LoadDataFromDatabase carrega dados já salvos no banco e processa para gerar relatórios
func (pc *PRChampion) LoadDataFromDatabase() error {
	fmt.Printf("📊 Carregando dados do banco de dados...\n")

	// Acessa o banco de dados através do client
	db := pc.cachedClient.GetDatabase()

	// Busca todos os PRs ou filtra por data se especificado
	var prs []*database.PRData
	var err error

	if !pc.startDate.IsZero() && !pc.endDate.IsZero() {
		fmt.Printf("🔍 Filtrando PRs entre %s e %s\n",
			pc.startDate.Format("02/01/2006"), pc.endDate.Format("02/01/2006"))
		prs, err = db.GetAllPRsInDateRange(pc.startDate, pc.endDate)
	} else {
		fmt.Printf("📋 Carregando todos os PRs salvos\n")
		prs, err = db.GetAllPRs()
	}

	if err != nil {
		return fmt.Errorf("erro ao carregar PRs do banco: %v", err)
	}

	if len(prs) == 0 {
		fmt.Printf("⚠️  Nenhum PR encontrado no banco de dados\n")
		fmt.Printf("💡 Use o comando 'load' primeiro para carregar dados da API do GitHub\n")
		return nil
	}

	fmt.Printf("📊 Encontrados %d PRs no banco de dados\n", len(prs))

	// Converte PRData para github.PullRequest para reutilizar lógica existente
	githubPRs := pc.convertPRDataToGithubPR(prs)

	// Processa dados semanais dos PRs
	pc.processWeeklyData(githubPRs)

	// Também processa dados mensais dos PRs
	pc.processMonthlyDataFromDatabase(prs)

	// Carrega e processa comentários
	err = pc.loadCommentsFromDatabase(prs, db)
	if err != nil {
		fmt.Printf("⚠️  Erro ao carregar comentários: %v\n", err)
	}

	// Calcula estatísticas dos usuários
	pc.calculateUserStats()

	fmt.Printf("✅ Dados carregados com sucesso do banco!\n")
	return nil
}

// convertPRDataToGithubPR converte PRData do banco para github.PullRequest
func (pc *PRChampion) convertPRDataToGithubPR(prs []*database.PRData) []*github.PullRequest {
	var githubPRs []*github.PullRequest

	for _, pr := range prs {
		// Cria um repositório para manter referências
		repo := &github.Repository{
			Owner: &github.User{Login: &pr.RepoOwner},
			Name:  &pr.RepoName,
		}

		// Cria o PR com dados básicos necessários para processamento
		githubPR := &github.PullRequest{
			Number:       &pr.PRNumber,
			Title:        &pr.Title,
			User:         &github.User{Login: &pr.Username},
			CreatedAt:    &github.Timestamp{Time: pr.CreatedAt},
			MergedAt:     &github.Timestamp{Time: pr.MergedAt},
			Additions:    &pr.Additions,
			Deletions:    &pr.Deletions,
			ChangedFiles: &pr.ChangedFiles,
			Base: &github.PullRequestBranch{
				Repo: repo,
			},
		}

		githubPRs = append(githubPRs, githubPR)
	}

	return githubPRs
}

// loadCommentsFromDatabase carrega comentários do banco e processa pontuações
func (pc *PRChampion) loadCommentsFromDatabase(prs []*database.PRData, db database.CommentDatabase) error {
	fmt.Printf("💬 Carregando comentários do banco de dados...\n")

	// Mapas para rastrear comentários por semana
	weeklyComments := make(map[string]map[string]int)             // weekKey -> username -> count
	weeklyWeightedComments := make(map[string]map[string]float64) // weekKey -> username -> weighted score
	weekStarts := make(map[string]time.Time)

	totalComments := 0

	for _, pr := range prs {
		// Verifica se o PR deve ser ignorado para gamificação
		if pc.shouldIgnorePRForGamification(pr.RepoOwner, pr.RepoName, pr.PRNumber) {
			continue // Ignora comentários de PRs marcados para ignorar
		}

		// Busca comentários deste PR
		comments, err := db.GetCommentsByPR(pr.RepoOwner, pr.RepoName, pr.PRNumber)
		if err != nil {
			fmt.Printf("  ⚠️  Erro ao buscar comentários do PR #%d: %v\n", pr.PRNumber, err)
			continue
		}

		for _, comment := range comments {
			// Filtra usuários excluídos (bots, etc.)
			if isExcludedUser(comment.Username) {
				continue
			}

			// Pula comentários do autor do PR
			if comment.Username == pr.Username {
				continue
			}

			// Verifica se o comentário foi feito após o merge (se aplicável)
			if comment.CreatedAt.After(pr.MergedAt) {
				fmt.Printf("    ❗ Comentário pós-merge ignorado: %s\n", comment.Username)
				continue
			}

			// Determina a semana do comentário baseada no merge do PR
			weekStart := getWeekStart(pr.MergedAt)
			weekKey := weekStart.Format("2006-01-02")

			if weeklyComments[weekKey] == nil {
				weeklyComments[weekKey] = make(map[string]int)
				weeklyWeightedComments[weekKey] = make(map[string]float64)
				weekStarts[weekKey] = weekStart
			}

			// Calcula pontuação ponderada baseada nas reações salvas
			commentScore := pc.calculateCommentScoreFromDatabase(comment, db, pr.MergedAt)

			weeklyComments[weekKey][comment.Username]++
			weeklyWeightedComments[weekKey][comment.Username] += commentScore
			totalComments++
		}
	}

	// Processa comentários semanais
	pc.processWeeklyComments(weeklyComments, weeklyWeightedComments, weekStarts)

	fmt.Printf("� Total de comentários processados: %d\n", totalComments)
	return nil
}

// calculateCommentScoreFromDatabase calcula pontuação usando reações do banco
func (pc *PRChampion) calculateCommentScoreFromDatabase(comment *database.CommentData, db database.CommentDatabase, mergedAt time.Time) float64 {
	// Busca reações do comentário no banco
	reactions, err := db.GetReactions(comment.CommentID)
	if err != nil {
		// Se não conseguir buscar reações, usa pontuação base
		return 1.0
	}

	// Converte ReactionData para github.Reaction para reutilizar lógica
	githubReactions := make([]*github.Reaction, 0, len(reactions))
	for _, reaction := range reactions {
		githubReaction := &github.Reaction{
			Content:   &reaction.Content,
			CreatedAt: &github.Timestamp{Time: reaction.CreatedAt},
			User:      &github.User{Login: &reaction.Username},
		}
		githubReactions = append(githubReactions, githubReaction)
	}

	return pc.calculateScoreFromReactions(githubReactions, mergedAt)
}

// FetchMergedPRs busca todos os PRs mergeados no período especificado para todos os repositórios
func (pc *PRChampion) FetchMergedPRs() error {
	fmt.Printf("🔍 Buscando PRs mergeados de %s para %d repositórios...\n",
		pc.startDate.Format("2006-01-02"), len(pc.repositories))

	var allPRs []*github.PullRequest

	for _, repo := range pc.repositories {
		productionBranches := repo.ProductionBranches
		if len(productionBranches) == 0 {
			productionBranches = []string{"main"} // Branch padrão se não especificada
		}

		fmt.Printf("  📁 Analisando %s/%s (branches: %s)...\n", repo.Owner, repo.Name, strings.Join(productionBranches, ", "))

		repoPRs, err := pc.client.FetchPRsForRepo(repo.Owner, repo.Name, pc.startDate, pc.endDate)
		if err != nil {
			fmt.Printf("  ⚠️  Erro ao buscar PRs do repo %s/%s: %v\n", repo.Owner, repo.Name, err)
			continue // Continua com os outros repositórios
		}

		// Filtra PRs mergeados apenas para as branches de produção
		var productionPRs []*github.PullRequest
		for _, pr := range repoPRs {
			if pr.Base != nil && pr.Base.Ref != nil {
				prBaseBranch := *pr.Base.Ref
				isProductionBranch := false

				for _, branch := range productionBranches {
					if prBaseBranch == branch {
						isProductionBranch = true
						break
					}
				}

				if isProductionBranch {
					productionPRs = append(productionPRs, pr)
				} else {
					fmt.Printf("    ❌ PR #%d ignorado (branch: %s, aceitas: %s)\n",
						pr.GetNumber(), prBaseBranch, strings.Join(productionBranches, ", "))
				}
			}
		}

		fmt.Printf("    ✅ %d PRs encontrados para branches de produção [%s] (total: %d)\n",
			len(productionPRs), strings.Join(productionBranches, ", "), len(repoPRs))

		allPRs = append(allPRs, productionPRs...)
	}

	fmt.Printf("📊 Encontrados %d PRs mergeados no período total\n", len(allPRs))

	pc.processWeeklyData(allPRs)

	// Busca reviews para todos os PRs e filtra apenas os PRs com approve
	// approvedPRs, err := pc.fetchReviewsAndFilterApprovedPRs(allPRs)
	// if err != nil {
	// 	fmt.Printf("⚠️  Erro ao buscar reviews: %v\n", err)
	// 	return err
	// }

	// fmt.Printf("📊 PRs com pelo menos um approve: %d de %d total\n", len(approvedPRs), len(allPRs))

	// // Substitui a lista de PRs pelos PRs aprovados
	// allPRs = approvedPRs
	// pc.processWeeklyData(allPRs)

	// Busca comentários para todos os PRs aprovados
	if err := pc.fetchCommentsForPRs(allPRs); err != nil {
		fmt.Printf("⚠️  Erro ao buscar comentários: %v\n", err)
	}
	pc.calculateUserStats()

	return nil
}

// fetchReviewsAndFilterApprovedPRs busca reviews dos PRs e retorna apenas os que têm pelo menos um approve
func (pc *PRChampion) fetchReviewsAndFilterApprovedPRs(prs []*github.PullRequest) ([]*github.PullRequest, error) {
	fmt.Printf("🔍 Buscando reviews dos PRs para filtrar apenas os aprovados...\n")

	ctx := context.Background()
	var approvedPRs []*github.PullRequest

	for _, pr := range prs {
		repoOwner := pr.Base.Repo.Owner.GetLogin()
		repoName := pr.Base.Repo.GetName()
		prNumber := pr.GetNumber()

		// Busca reviews do PR
		reviews, err := pc.client.ListPRReviews(ctx, repoOwner, repoName, prNumber)
		if err != nil {
			fmt.Printf("  ⚠️  Erro ao buscar reviews do PR #%d em %s/%s: %v\n", prNumber, repoOwner, repoName, err)
			continue
		}

		// Verifica se tem pelo menos um review aprovado
		hasApprove := false
		for _, review := range reviews {
			if review.GetState() == "APPROVED" {
				// Verifica se o review foi submetido antes do merge (se o PR foi mergeado)
				if pr.MergedAt != nil && review.SubmittedAt != nil {
					if review.SubmittedAt.Time.After(pr.MergedAt.Time) {
						fmt.Printf("    ❗ Review approve pós-merge ignorado: %s (review: %s, merge: %s)\n",
							review.User.GetLogin(), review.SubmittedAt.Time.Format("02/01/2006 15:04"), pr.MergedAt.Time.Format("02/01/2006 15:04"))
						continue
					}
				}
				hasApprove = true
				break
			}
		}

		// Só inclui o PR se tiver pelo menos um approve válido
		if hasApprove {
			approvedPRs = append(approvedPRs, pr)
		} else {
			fmt.Printf("    ❌ PR #%d em %s/%s ignorado (sem approve válido)\n", prNumber, repoOwner, repoName)
		}
	}

	return approvedPRs, nil
}

// fetchCommentsForPRs busca comentários de todos os PRs
func (pc *PRChampion) fetchCommentsForPRs(prs []*github.PullRequest) error {
	fmt.Printf("💬 Buscando comentários dos PRs...\n")

	ctx := context.Background()
	totalComments := 0

	// Mapas para rastrear comentários por semana
	weeklyComments := make(map[string]map[string]int)             // weekKey -> username -> count
	weeklyWeightedComments := make(map[string]map[string]float64) // weekKey -> username -> weighted score
	weekStarts := make(map[string]time.Time)

	for _, pr := range prs {
		repoOwner := pr.Base.Repo.Owner.GetLogin()
		repoName := pr.Base.Repo.GetName()
		prNumber := pr.GetNumber()

		// Verifica se o PR deve ser ignorado para gamificação
		if pc.shouldIgnorePRForGamification(repoOwner, repoName, prNumber) {
			continue // Ignora comentários de PRs marcados para ignorar
		}

		comments, err := pc.client.ListPRComments(ctx, repoOwner, repoName, prNumber)
		if err != nil {
			fmt.Printf("  ⚠️  Erro ao buscar comentários do PR #%d em %s/%s: %v\n", prNumber, repoOwner, repoName, err)
			break
		}

		for _, comment := range comments {
			commentTime := comment.CreatedAt.Time
			username := comment.User.GetLogin()

			// Filtra usuários excluídos (bots, sonarqube, etc.)
			if isExcludedUser(username) {
				continue
			}

			if username == pr.User.GetLogin() {
				fmt.Println("    ❗ Comentário do autor do PR ignorado:", username)
				continue // Pula comentários feitos pelo autor do PR
			}

			// Verifica se o comentário foi feito após o merge do PR
			if pr.MergedAt != nil && commentTime.After(pr.MergedAt.Time) {
				fmt.Printf("    ❗ Comentário pós-merge ignorado: %s (comentário: %s, merge: %s)\n",
					username, commentTime.Format("02/01/2006 15:04"), pr.MergedAt.Time.Format("02/01/2006 15:04"))
				continue
			}

			// Determina a semana do comentário
			weekStart := getWeekStart(pr.MergedAt.Time)
			weekKey := weekStart.Format("2006-01-02")

			if weeklyComments[weekKey] == nil {
				weeklyComments[weekKey] = make(map[string]int)
				weeklyWeightedComments[weekKey] = make(map[string]float64)
				weekStarts[weekKey] = weekStart
			}

			// Calcula pontuação ponderada baseada nas reações
			commentScore := pc.calculateCommentScore(ctx, repoOwner, repoName, comment.GetID(), pr.MergedAt.Time)

			weeklyComments[weekKey][username]++
			weeklyWeightedComments[weekKey][username] += commentScore
			totalComments++
		}

		reviewComments, err := pc.client.ListPRReviewComments(ctx, repoOwner, repoName, prNumber)
		if err != nil {
			fmt.Printf("  ⚠️  Erro ao buscar review comments do PR #%d em %s/%s: %v\n", prNumber, repoOwner, repoName, err)
			break
		}

		for _, comment := range reviewComments {
			commentTime := comment.CreatedAt.Time
			username := comment.User.GetLogin()

			// Filtra usuários excluídos (bots, sonarqube, etc.)
			if isExcludedUser(username) {
				continue
			}

			if username == pr.User.GetLogin() {
				fmt.Println("    ❗ Comentário do autor do PR ignorado:", username)
				continue // Pula comentários feitos pelo autor do PR
			}

			// Verifica se o review comment foi feito após o merge do PR
			if pr.MergedAt != nil && commentTime.After(pr.MergedAt.Time) {
				fmt.Printf("    ❗ Review comment pós-merge ignorado: %s (comentário: %s, merge: %s)\n",
					username, commentTime.Format("02/01/2006 15:04"), pr.MergedAt.Time.Format("02/01/2006 15:04"))
				continue
			}

			// Determina a semana do comentário
			weekStart := getWeekStart(pr.MergedAt.Time)
			weekKey := weekStart.Format("2006-01-02")

			if weeklyComments[weekKey] == nil {
				weeklyComments[weekKey] = make(map[string]int)
				weeklyWeightedComments[weekKey] = make(map[string]float64)
				weekStarts[weekKey] = weekStart
			}

			// Calcula pontuação ponderada baseada nas reações
			commentScore := pc.calculateReviewCommentScore(ctx, repoOwner, repoName, comment.GetID(), pr.MergedAt.Time)

			weeklyComments[weekKey][username]++
			weeklyWeightedComments[weekKey][username] += commentScore
			totalComments++
		}

	}

	// Adiciona dados de comentários às semanas existentes
	pc.processWeeklyComments(weeklyComments, weeklyWeightedComments, weekStarts)

	fmt.Printf("💬 Total de comentários encontrados no período: %d\n", totalComments)
	return nil
}

// processWeeklyComments processa os comentários por semana e identifica vencedores
func (pc *PRChampion) processWeeklyComments(weeklyComments map[string]map[string]int, weeklyWeightedComments map[string]map[string]float64, weekStarts map[string]time.Time) {
	// Adiciona dados de comentários às semanas existentes ou cria novas semanas
	for weekKey, userComments := range weeklyComments {
		weekStart := weekStarts[weekKey]

		// Encontra o vencedor da semana por comentários (contagem simples)
		var commentWinner string
		maxComments := 0
		for user, count := range userComments {
			if count > maxComments {
				maxComments = count
				commentWinner = user
			}
		}

		// Encontra o vencedor da semana por pontuação ponderada
		var weightedCommentWinner string
		maxWeightedScore := 0.0
		userWeightedComments := weeklyWeightedComments[weekKey]
		for user, score := range userWeightedComments {
			if score > maxWeightedScore {
				maxWeightedScore = score
				weightedCommentWinner = user
			}
		}

		// Procura se já existe uma semana correspondente
		found := false
		for i := range pc.weeklyData {
			if pc.weeklyData[i].StartDate.Equal(weekStart) {
				pc.weeklyData[i].UserComments = userComments
				pc.weeklyData[i].CommentWinner = commentWinner
				pc.weeklyData[i].UserWeightedComments = userWeightedComments
				pc.weeklyData[i].WeightedCommentWinner = weightedCommentWinner
				found = true
				break
			}
		}

		// Se não encontrou, cria uma nova entrada semanal apenas para comentários
		if !found {
			weekEnd := weekStart.Add(6 * 24 * time.Hour)
			pc.weeklyData = append(pc.weeklyData, WeeklyData{
				StartDate:             weekStart,
				EndDate:               weekEnd,
				UserPRs:               make(map[string]int),
				UserComments:          userComments,
				CommentWinner:         commentWinner,
				UserWeightedComments:  userWeightedComments,
				WeightedCommentWinner: weightedCommentWinner,
				MergeTimes:            make([]time.Duration, 0),
				P90MergeTime:          0,
				MedianMergeTime:       0,
				AverageMergeTime:      0,
			})
		}
	}

	// Reordena por data
	sort.Slice(pc.weeklyData, func(i, j int) bool {
		return pc.weeklyData[i].StartDate.Before(pc.weeklyData[j].StartDate)
	})
}

// processWeeklyData processa os PRs por semana
func (pc *PRChampion) processWeeklyData(prs []*github.PullRequest) {
	// Agrupa PRs por semana
	weeklyMap := make(map[string]map[string]int)
	weekStarts := make(map[string]time.Time)
	weeklyMergeTimes := make(map[string][]time.Duration)

	for _, pr := range prs {
		// Verifica se o PR deve ser ignorado para gamificação
		repoOwner := pr.Base.Repo.Owner.GetLogin()
		repoName := pr.Base.Repo.GetName()
		prNumber := pr.GetNumber()

		if pc.shouldIgnorePRForGamification(repoOwner, repoName, prNumber) {
			continue // Ignora este PR para a gamificação
		}

		mergedAt := pr.MergedAt.Time
		weekStart := getWeekStart(mergedAt)
		weekKey := weekStart.Format("2006-01-02")

		if weeklyMap[weekKey] == nil {
			weeklyMap[weekKey] = make(map[string]int)
			weekStarts[weekKey] = weekStart
			weeklyMergeTimes[weekKey] = make([]time.Duration, 0)
		}

		username := pr.User.GetLogin()
		weeklyMap[weekKey][username]++

		// Calcula tempo entre criação e merge
		if pr.CreatedAt != nil && pr.MergedAt != nil {
			mergeTime := pr.MergedAt.Time.Sub(pr.CreatedAt.Time)
			weeklyMergeTimes[weekKey] = append(weeklyMergeTimes[weekKey], mergeTime)
		}

		// Processa estatísticas de código diretamente aqui
		if pc.userStats[username] == nil {
			pc.userStats[username] = &UserStats{
				Username:  username,
				RepoStats: make(map[string]int),
			}
		}

		stats := pc.userStats[username]
		stats.TotalAdditions += pr.GetAdditions()
		stats.TotalDeletions += pr.GetDeletions()
		stats.TotalChangedFiles += pr.GetChangedFiles()
	}

	// Converte para slice de WeeklyData
	for weekKey, userPRs := range weeklyMap {
		weekStart := weekStarts[weekKey]
		weekEnd := weekStart.Add(6 * 24 * time.Hour)

		// Encontra o vencedor da semana
		var winner string
		maxPRs := 0
		for user, count := range userPRs {
			if count > maxPRs {
				maxPRs = count
				winner = user
			}
		}

		// Calcula P90 considerando todos os PRs dos últimos 30 dias até o fim da semana
		thirtyDaysAgo := weekEnd.Add(-30 * 24 * time.Hour)
		var last30DaysMergeTimes []time.Duration

		for _, pr := range prs {
			// Verifica se o PR deve ser ignorado para gamificação
			repoOwner := pr.Base.Repo.Owner.GetLogin()
			repoName := pr.Base.Repo.GetName()
			prNumber := pr.GetNumber()

			if pc.shouldIgnorePRForGamification(repoOwner, repoName, prNumber) {
				continue
			}

			mergedAt := pr.MergedAt.Time

			// Inclui PRs mergeados nos últimos 30 dias até o fim desta semana
			if mergedAt.After(thirtyDaysAgo) && mergedAt.Before(weekEnd.Add(24*time.Hour)) {
				if pr.CreatedAt != nil && pr.MergedAt != nil {
					mergeTime := pr.MergedAt.Time.Sub(pr.CreatedAt.Time)
					last30DaysMergeTimes = append(last30DaysMergeTimes, mergeTime)
				}
			}
		}

		mergeTimes := weeklyMergeTimes[weekKey]                            // PRs apenas desta semana (para outras estatísticas)
		p90MergeTime := calculateP90Duration(last30DaysMergeTimes)         // P90 dos últimos 30 dias
		medianMergeTime := calculateMedianDuration(last30DaysMergeTimes)   // Mediana dos últimos 30 dias
		averageMergeTime := calculateAverageDuration(last30DaysMergeTimes) // Média dos últimos 30 dias

		pc.weeklyData = append(pc.weeklyData, WeeklyData{
			StartDate:        weekStart,
			EndDate:          weekEnd,
			UserPRs:          userPRs,
			Winner:           winner,
			MergeTimes:       mergeTimes,
			P90MergeTime:     p90MergeTime,
			MedianMergeTime:  medianMergeTime,
			AverageMergeTime: averageMergeTime,
		})
	}

	// Ordena por data
	sort.Slice(pc.weeklyData, func(i, j int) bool {
		return pc.weeklyData[i].StartDate.Before(pc.weeklyData[j].StartDate)
	})
}

// processMonthlyDataFromDatabase agrupa dados mensais do banco de dados
func (pc *PRChampion) processMonthlyDataFromDatabase(prs []*database.PRData) {
	monthlyPRs := make(map[string]map[string]int)         // monthKey -> user -> PRs
	monthlyMergeTimes := make(map[string][]time.Duration) // monthKey -> merge times

	// Primeiro, ordena todos os PRs por data de merge
	sort.Slice(prs, func(i, j int) bool {
		return prs[i].MergedAt.Before(prs[j].MergedAt)
	})

	// Agrupa PRs por mês
	for _, pr := range prs {
		if pr.MergedAt.IsZero() {
			continue
		}

		// Primeiro dia do mês
		monthStart := time.Date(pr.MergedAt.Year(), pr.MergedAt.Month(), 1, 0, 0, 0, 0, pr.MergedAt.Location())
		monthKey := monthStart.Format("2006-01")

		if monthlyPRs[monthKey] == nil {
			monthlyPRs[monthKey] = make(map[string]int)
		}
		if monthlyMergeTimes[monthKey] == nil {
			monthlyMergeTimes[monthKey] = make([]time.Duration, 0)
		}

		monthlyPRs[monthKey][pr.Author]++
		monthlyMergeTimes[monthKey] = append(monthlyMergeTimes[monthKey], pr.MergeTime)
	}

	// Criar dados mensais
	for monthKey, userPRs := range monthlyPRs {
		monthStart, _ := time.Parse("2006-01", monthKey)
		monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)

		// Encontrar vencedor do mês
		winner := ""
		maxPRs := 0
		for user, count := range userPRs {
			if count > maxPRs {
				maxPRs = count
				winner = user
			}
		}

		// Calcular métricas dos últimos 30 dias até o final do mês
		last30DaysStart := monthEnd.Add(-30 * 24 * time.Hour)
		last30DaysMergeTimes := make([]time.Duration, 0)

		for _, pr := range prs {
			if !pr.MergedAt.IsZero() && pr.MergedAt.After(last30DaysStart) && !pr.MergedAt.After(monthEnd) {
				last30DaysMergeTimes = append(last30DaysMergeTimes, pr.MergeTime)
			}
		}

		mergeTimes := monthlyMergeTimes[monthKey]                          // PRs apenas deste mês
		p90MergeTime := calculateP90Duration(last30DaysMergeTimes)         // P90 dos últimos 30 dias
		medianMergeTime := calculateMedianDuration(last30DaysMergeTimes)   // Mediana dos últimos 30 dias
		averageMergeTime := calculateAverageDuration(last30DaysMergeTimes) // Média dos últimos 30 dias

		pc.monthlyData = append(pc.monthlyData, MonthlyData{
			Year:             monthStart.Year(),
			Month:            int(monthStart.Month()),
			StartDate:        monthStart,
			EndDate:          monthEnd,
			UserPRs:          userPRs,
			Winner:           winner,
			MergeTimes:       mergeTimes,
			P90MergeTime:     p90MergeTime,
			MedianMergeTime:  medianMergeTime,
			AverageMergeTime: averageMergeTime,
		})
	}

	// Reordena por data
	sort.Slice(pc.monthlyData, func(i, j int) bool {
		return pc.monthlyData[i].StartDate.Before(pc.monthlyData[j].StartDate)
	})
}

// calculateUserStats calcula as estatísticas finais dos usuários
func (pc *PRChampion) calculateUserStats() {
	for _, week := range pc.weeklyData {
		// Processa PRs
		for username, prCount := range week.UserPRs {
			if pc.userStats[username] == nil {
				pc.userStats[username] = &UserStats{
					Username:  username,
					RepoStats: make(map[string]int),
				}
			}

			stats := pc.userStats[username]
			stats.PRsCount += prCount

			if username == week.Winner {
				stats.WeeklyWins++
				stats.TotalScore++
			}
		}

		// Processa comentários
		for username, commentCount := range week.UserComments {
			if pc.userStats[username] == nil {
				pc.userStats[username] = &UserStats{
					Username:  username,
					RepoStats: make(map[string]int),
				}
			}

			stats := pc.userStats[username]
			stats.CommentsCount += commentCount

			if username == week.CommentWinner {
				stats.CommentWeeklyWins++
				stats.CommentScore++
			}
		}

		// Processa pontuação ponderada de comentários
		for username, weightedScore := range week.UserWeightedComments {
			if pc.userStats[username] == nil {
				pc.userStats[username] = &UserStats{
					Username:  username,
					RepoStats: make(map[string]int),
				}
			}

			stats := pc.userStats[username]
			stats.WeightedCommentScore += weightedScore

			// Se for o vencedor da semana por qualidade de comentários, ganha 1 ponto
			if username == week.WeightedCommentWinner {
				stats.WeightedCommentWeeklyWins++
				stats.WeightedCommentWeeklyScore++
			}
		}
	}
}

// GenerateReport gera o relatório final
func (pc *PRChampion) GenerateReport() {
	// Usar configuração padrão para manter compatibilidade
	defaultConfig := MetricDisplayConfig{
		ShowP90:     true,
		ShowMedian:  true,
		ShowAverage: true,
		TimeFormat:  "hours",
		GroupBy:     "week",
	}
	pc.GenerateReportWithConfig(defaultConfig)
}

// GenerateReportWithConfig gera o relatório com configurações personalizadas
func (pc *PRChampion) GenerateReportWithConfig(config MetricDisplayConfig) {
	fmt.Printf("\n🏆 RELATÓRIO PR CHAMPION - %s a %s\n",
		pc.startDate.Format("02/01/2006"), pc.endDate.Format("02/01/2006"))

	// Lista dos repositórios analisados
	fmt.Printf("📁 Repositórios analisados (%d):\n", len(pc.repositories))
	for _, repo := range pc.repositories {
		fmt.Printf("   • %s/%s\n", repo.Owner, repo.Name)
	}
	fmt.Println()

	// Relatório semanal
	fmt.Println("📅 RESUMO SEMANAL:")
	fmt.Println(strings.Repeat("=", 60))

	for _, week := range pc.weeklyData {
		fmt.Printf("Semana: %s - %s\n",
			week.StartDate.Format("02/01"), week.EndDate.Format("02/01/2006"))

		// Campeão por PRs
		if week.Winner != "" {
			fmt.Printf("🥇 Campeão PRs: %s\n", week.Winner)
			// Top 3 da semana por PRs
			weekTop := pc.getTopUsersForWeek(week.UserPRs, 3)
			for i, user := range weekTop {
				medal := []string{"🥇", "🥈", "🥉"}[i]
				fmt.Printf("   %s %s: %d PRs\n", medal, user.Username, user.PRsCount)
			}
		}

		// Campeão por qualidade de comentários (pontuação ponderada)
		if week.WeightedCommentWinner != "" {
			fmt.Printf("⭐ Campeão Qualidade: %s\n", week.WeightedCommentWinner)
			// Top 3 da semana por pontuação ponderada
			weekTopWeighted := pc.getTopUsersForWeekWeighted(week.UserWeightedComments, 3)
			for i, user := range weekTopWeighted {
				medal := []string{"🥇", "🥈", "🥉"}[i]
				fmt.Printf("   %s %s: %.1f pontos\n", medal, user.Username, user.WeightedCommentScore)
			}
		}

		// P90 do tempo de merge
		if week.P90MergeTime > 0 {
			fmt.Printf("⏱️  P90 Tempo Merge (últimos 30 dias): %s\n", formatDuration(week.P90MergeTime))
		}

		fmt.Println()
	}

	// Ranking geral por pontuação
	fmt.Println("🏅 RANKING GERAL POR PONTUAÇÃO:")
	fmt.Println(strings.Repeat("=", 60))

	topUsers := pc.getTopUsersByScore(5)
	for i, user := range topUsers {
		position := i + 1
		medal := ""
		switch position {
		case 1:
			medal = "🥇"
		case 2:
			medal = "🥈"
		case 3:
			medal = "🥉"
		case 4:
			medal = "🏅"
		case 5:
			medal = "🎖️"
		}

		fmt.Printf("%s %d° lugar: %s\n", medal, position, user.Username)
		fmt.Printf("   📊 Pontuação: %d pontos\n", user.TotalScore)
		fmt.Printf("   🏆 Vitórias semanais: %d\n", user.WeeklyWins)
		fmt.Printf("   📋 Total de PRs: %d\n\n", user.PRsCount)
	}

	// Ranking por pontuação semanal de qualidade de comentários
	fmt.Println("🏅 RANKING SEMANAL POR QUALIDADE DOS COMENTÁRIOS:")
	fmt.Println(strings.Repeat("=", 60))

	topWeightedCommentWeeklyUsers := pc.getTopUsersByWeightedCommentWeeklyScore(5)
	if len(topWeightedCommentWeeklyUsers) == 0 {
		fmt.Println("   Nenhuma vitória semanal por qualidade de comentários foi registrada no período analisado.")
	} else {
		for i, user := range topWeightedCommentWeeklyUsers {
			position := i + 1
			medal := ""
			switch position {
			case 1:
				medal = "🥇"
			case 2:
				medal = "🥈"
			case 3:
				medal = "🥉"
			case 4:
				medal = "🏅"
			case 5:
				medal = "🎖️"
			}

			fmt.Printf("%s %d° lugar: %s\n", medal, position, user.Username)
			fmt.Printf("   🏅 Pontuação semanal: %d pontos\n", user.WeightedCommentWeeklyScore)
			fmt.Printf("   🏆 Vitórias semanais (qualidade): %d\n", user.WeightedCommentWeeklyWins)
			fmt.Printf("   ⭐ Pontuação total com reações: %.1f pontos\n\n", user.WeightedCommentScore)
		}
	}

	// Top 5 por número total de PRs
	fmt.Println("📈 TOP 5 POR TOTAL DE PRS:")
	fmt.Println(strings.Repeat("=", 60))

	topByPRs2 := pc.getTopUsersByPRs(5)
	for i, user := range topByPRs2 {
		position := i + 1
		medal := []string{"🥇", "🥈", "🥉", "🏅", "🎖️"}[i]
		fmt.Printf("%s %d° lugar: %s - %d PRs\n", medal, position, user.Username, user.PRsCount)
	}
	fmt.Println()

	// Top 5 por número total de comentários
	fmt.Println("💬 TOP 5 POR QUALIDADE DE COMENTÁRIOS:")
	fmt.Println(strings.Repeat("=", 60))

	topByComments := pc.getTopUsersByWeightedCommentScore(5)
	if len(topByComments) == 0 {
		fmt.Println("   Nenhum comentário encontrado no período analisado.")
	} else {
		for i, user := range topByComments {
			position := i + 1
			medal := []string{"🥇", "🥈", "🥉", "🏅", "🎖️"}[i]
			fmt.Printf("%s %d° lugar: %s - %.2f comentários\n", medal, position, user.Username, user.WeightedCommentScore)
		}
	}
	fmt.Println()

	// Top 5 por linhas de código
	fmt.Println("💻 TOP 5 POR LINHAS DE CÓDIGO:")
	fmt.Println(strings.Repeat("=", 60))

	topByCode := pc.getTopUsersByCodeLines(5)
	if len(topByCode) == 0 {
		fmt.Println("   Nenhuma estatística de código encontrada no período analisado.")
	} else {
		for i, user := range topByCode {
			position := i + 1
			medal := []string{"🥇", "🥈", "🥉", "🏅", "🎖️"}[i]
			totalLines := user.TotalAdditions + user.TotalDeletions
			fmt.Printf("%s %d° lugar: %s\n", medal, position, user.Username)
			fmt.Printf("   📊 Total: %d linhas (+%d/-%d)\n", totalLines, user.TotalAdditions, user.TotalDeletions)
			fmt.Printf("   📁 Arquivos modificados: %d\n", user.TotalChangedFiles)
		}
	}
	fmt.Println()

	// Labels mais utilizadas
	pc.showLabelStatistics()

	// Relatório de velocidade por semana em formato CSV
	pc.showP90WeeklySummary(config)

	// Estatísticas do cache
	fmt.Println("📈 ESTATÍSTICAS DO CACHE:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("💾 Sistema de cache em banco SQLite ativo")
	fmt.Println("📋 Cache de comentários e reações: 7 dias")
	fmt.Println("🗂️  Local do banco: ./data/comments.db")
	fmt.Println("💡 Use --clear-database para limpar todo o cache")
}

// getTopUsersForWeek retorna os top usuários de uma semana específica
func (pc *PRChampion) getTopUsersForWeek(userPRs map[string]int, limit int) []UserStats {
	var users []UserStats
	for username, prCount := range userPRs {
		users = append(users, UserStats{
			Username: username,
			PRsCount: prCount,
		})
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].PRsCount > users[j].PRsCount
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// getTopUsersForWeekWeighted retorna os top usuários por pontuação ponderada de uma semana específica
func (pc *PRChampion) getTopUsersForWeekWeighted(userWeightedComments map[string]float64, limit int) []UserStats {
	var users []UserStats
	for username, weightedScore := range userWeightedComments {
		users = append(users, UserStats{
			Username:             username,
			WeightedCommentScore: weightedScore,
		})
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].WeightedCommentScore > users[j].WeightedCommentScore
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// getTopUsersByScore retorna os top usuários por pontuação
func (pc *PRChampion) getTopUsersByScore(limit int) []*UserStats {
	var users []*UserStats
	for _, stats := range pc.userStats {
		users = append(users, stats)
	}

	sort.Slice(users, func(i, j int) bool {
		if users[i].TotalScore == users[j].TotalScore {
			return users[i].PRsCount > users[j].PRsCount
		}
		return users[i].TotalScore > users[j].TotalScore
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// getTopUsersByPRs retorna os top usuários por número de PRs
func (pc *PRChampion) getTopUsersByPRs(limit int) []*UserStats {
	var users []*UserStats
	for _, stats := range pc.userStats {
		users = append(users, stats)
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].PRsCount > users[j].PRsCount
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// getTopUsersByComments retorna os top usuários por número de comentários
func (pc *PRChampion) getTopUsersByComments(limit int) []*UserStats {
	var users []*UserStats
	for _, stats := range pc.userStats {
		if stats.CommentsCount > 0 { // Apenas usuários que fizeram comentários
			users = append(users, stats)
		}
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].CommentsCount > users[j].CommentsCount
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// getTopUsersByCommentScore retorna os top usuários por pontuação de comentários
func (pc *PRChampion) getTopUsersByCommentScore(limit int) []*UserStats {
	var users []*UserStats
	for _, stats := range pc.userStats {
		if stats.CommentScore > 0 { // Apenas usuários que ganharam pontos por comentários
			users = append(users, stats)
		}
	}

	sort.Slice(users, func(i, j int) bool {
		if users[i].CommentScore == users[j].CommentScore {
			return users[i].CommentsCount > users[j].CommentsCount
		}
		return users[i].CommentScore > users[j].CommentScore
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// getTopUsersByWeightedCommentScore retorna os top usuários por pontuação ponderada de comentários
func (pc *PRChampion) getTopUsersByWeightedCommentScore(limit int) []*UserStats {
	var users []*UserStats
	for _, stats := range pc.userStats {
		if stats.WeightedCommentScore > 0 { // Apenas usuários com pontuação positiva
			users = append(users, stats)
		}
	}

	sort.Slice(users, func(i, j int) bool {
		if users[i].WeightedCommentScore == users[j].WeightedCommentScore {
			return users[i].CommentsCount > users[j].CommentsCount
		}
		return users[i].WeightedCommentScore > users[j].WeightedCommentScore
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// getTopUsersByWeightedCommentWeeklyScore retorna os top usuários por pontuação semanal de qualidade de comentários
func (pc *PRChampion) getTopUsersByWeightedCommentWeeklyScore(limit int) []*UserStats {
	var users []*UserStats
	for _, stats := range pc.userStats {
		if stats.WeightedCommentWeeklyScore > 0 { // Apenas usuários que ganharam pontos semanais por qualidade
			users = append(users, stats)
		}
	}

	sort.Slice(users, func(i, j int) bool {
		if users[i].WeightedCommentWeeklyScore == users[j].WeightedCommentWeeklyScore {
			return users[i].WeightedCommentWeeklyWins > users[j].WeightedCommentWeeklyWins
		}
		return users[i].WeightedCommentWeeklyScore > users[j].WeightedCommentWeeklyScore
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// getTopUsersByCodeLines retorna os top usuários por total de linhas de código
func (pc *PRChampion) getTopUsersByCodeLines(limit int) []*UserStats {
	var users []*UserStats
	for _, stats := range pc.userStats {
		if stats.TotalAdditions > 0 || stats.TotalDeletions > 0 { // Apenas usuários com código
			users = append(users, stats)
		}
	}

	sort.Slice(users, func(i, j int) bool {
		totalI := users[i].TotalAdditions + users[i].TotalDeletions
		totalJ := users[j].TotalAdditions + users[j].TotalDeletions
		if totalI == totalJ {
			return users[i].PRsCount > users[j].PRsCount // Desempate por número de PRs
		}
		return totalI > totalJ
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// showLabelStatistics exibe estatísticas das labels mais utilizadas nos PRs
func (pc *PRChampion) showLabelStatistics() {
	fmt.Println("🏷️  TOP 10 LABELS MAIS UTILIZADAS:")
	fmt.Println(strings.Repeat("=", 60))

	// Para acessar dados do banco, precisamos usar o cached client
	if pc.cachedClient == nil {
		fmt.Println("   Estatísticas de labels não disponíveis (cache não inicializado)")
		fmt.Println()
		return
	}

	db := pc.cachedClient.GetDatabase()
	labelStats, err := pc.getLabelStatistics(db)
	if err != nil {
		fmt.Printf("   Erro ao buscar estatísticas de labels: %v\n", err)
		fmt.Println()
		return
	}

	if len(labelStats) == 0 {
		fmt.Println("   Nenhuma label encontrada nos PRs analisados.")
		fmt.Println()
		return
	}

	// Mostra top 10
	limit := 10
	if len(labelStats) < limit {
		limit = len(labelStats)
	}

	for i := 0; i < limit; i++ {
		label := labelStats[i]
		position := i + 1
		var emoji string
		switch position {
		case 1:
			emoji = "🥇"
		case 2:
			emoji = "🥈"
		case 3:
			emoji = "🥉"
		default:
			emoji = "🏷️"
		}
		fmt.Printf("%s %d° lugar: %s (%d PRs)\n", emoji, position, label.Name, label.Count)
	}
	fmt.Println()
}

// LabelStats representa estatísticas de uma label
type LabelStats struct {
	Name  string
	Count int
	Color string
}

// getLabelStatistics busca estatísticas das labels do banco
func (pc *PRChampion) getLabelStatistics(db database.CommentDatabase) ([]LabelStats, error) {
	// Busca todas as labels dos PRs no período
	var prs []*database.PRData
	var err error

	if !pc.startDate.IsZero() && !pc.endDate.IsZero() {
		prs, err = db.GetAllPRsInDateRange(pc.startDate, pc.endDate)
	} else {
		prs, err = db.GetAllPRs()
	}

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar PRs: %v", err)
	}

	// Conta labels por nome
	labelCounts := make(map[string]int)
	labelColors := make(map[string]string)

	for _, pr := range prs {
		labels, err := db.GetLabelsByPR(pr.RepoOwner, pr.RepoName, pr.PRNumber)
		if err != nil {
			continue // Continua mesmo com erro para não quebrar todo o relatório
		}

		for _, label := range labels {
			labelCounts[label.LabelName]++
			if labelColors[label.LabelName] == "" && label.Color != "" {
				labelColors[label.LabelName] = label.Color
			}
		}
	}

	// Converte para slice e ordena
	var stats []LabelStats
	for name, count := range labelCounts {
		stats = append(stats, LabelStats{
			Name:  name,
			Count: count,
			Color: labelColors[name],
		})
	}

	// Ordena por contagem decrescente
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Count == stats[j].Count {
			return stats[i].Name < stats[j].Name // Desempate alfabético
		}
		return stats[i].Count > stats[j].Count
	})

	return stats, nil
}

// showP90WeeklySummary exibe resumo das métricas por período em formato CSV
func (pc *PRChampion) showP90WeeklySummary(config MetricDisplayConfig) {
	groupLabel := "SEMANAL"
	if config.GroupBy == "month" {
		groupLabel = "MENSAL"
	}

	fmt.Printf("📊 RESUMO DE VELOCIDADE %s (CSV):\n", groupLabel)
	fmt.Println(strings.Repeat("=", 60))

	// Verificar se há dados disponíveis
	hasData := false
	if config.GroupBy == "month" {
		hasData = len(pc.monthlyData) > 0
	} else {
		hasData = len(pc.weeklyData) > 0
	}

	if !hasData {
		fmt.Printf("   Nenhum dado %s disponível.\n", strings.ToLower(groupLabel))
		fmt.Println()
		return
	}

	// Construir header dinâmico
	periodLabel := "Semana"
	if config.GroupBy == "month" {
		periodLabel = "Mês"
	}

	header := periodLabel
	if config.ShowP90 {
		header += fmt.Sprintf(",p90 (%s)", getTimeUnitLabel(config.TimeFormat))
	}
	if config.ShowMedian {
		header += fmt.Sprintf(",Mediana (%s)", getTimeUnitLabel(config.TimeFormat))
	}
	if config.ShowAverage {
		header += fmt.Sprintf(",Média (%s)", getTimeUnitLabel(config.TimeFormat))
	}
	fmt.Println(header)

	if config.GroupBy == "month" {
		// Exibir dados mensais
		for _, month := range pc.monthlyData {
			monthStr := month.StartDate.Format("01/2006")

			// Verificar se há dados para exibir
			hasData := month.P90MergeTime > 0 || month.MedianMergeTime > 0 || month.AverageMergeTime > 0

			row := monthStr
			if config.ShowP90 {
				if hasData {
					row += "," + formatTime(month.P90MergeTime, config.TimeFormat)
				} else {
					row += ",0"
				}
			}
			if config.ShowMedian {
				if hasData {
					row += "," + formatTime(month.MedianMergeTime, config.TimeFormat)
				} else {
					row += ",0"
				}
			}
			if config.ShowAverage {
				if hasData {
					row += "," + formatTime(month.AverageMergeTime, config.TimeFormat)
				} else {
					row += ",0"
				}
			}

			fmt.Println(row)
		}
	} else {
		// Exibir dados semanais
		for _, week := range pc.weeklyData {
			weekEndStr := week.EndDate.Format("02/01/2006")

			// Verificar se há dados para exibir
			hasData := week.P90MergeTime > 0 || week.MedianMergeTime > 0 || week.AverageMergeTime > 0

			row := weekEndStr
			if config.ShowP90 {
				if hasData {
					row += "," + formatTime(week.P90MergeTime, config.TimeFormat)
				} else {
					row += ",0"
				}
			}
			if config.ShowMedian {
				if hasData {
					row += "," + formatTime(week.MedianMergeTime, config.TimeFormat)
				} else {
					row += ",0"
				}
			}
			if config.ShowAverage {
				if hasData {
					row += "," + formatTime(week.AverageMergeTime, config.TimeFormat)
				} else {
					row += ",0"
				}
			}

			fmt.Println(row)
		}
	}
	fmt.Println()
}

// getTimeUnitLabel retorna o rótulo da unidade de tempo
func getTimeUnitLabel(format string) string {
	switch format {
	case "days":
		return "d"
	case "minutes":
		return "min"
	case "hours":
		fallthrough
	default:
		return "h"
	}
} // isExcludedUser verifica se um usuário deve ser excluído da contagem de comentários
func isExcludedUser(username string) bool {
	excludedUsers := []string{
		"grupogcb",
		"sonarqubecloud",
		"copilot",
		"github-actions",
		"dependabot",
		"codecov",
		"sonarcloud",
		"renovate",
		"greenkeeper",
		"snyk-bot",
	}

	// Converte para lowercase para comparação case-insensitive
	usernameLower := strings.ToLower(username)

	for _, excluded := range excludedUsers {
		if usernameLower == excluded || strings.Contains(usernameLower, excluded) {
			return true
		}
	}

	// Verifica se termina com [bot] (padrão do GitHub para bots)
	return strings.HasSuffix(usernameLower, "[bot]")
}

// shouldIgnorePRForGamification verifica se um PR deve ser ignorado para gamificação
func (pc *PRChampion) shouldIgnorePRForGamification(repoOwner, repoName string, prNumber int) bool {
	if pc.cachedClient == nil {
		return false
	}

	db := pc.cachedClient.GetDatabase()
	if db == nil {
		return false
	}

	labels, err := db.GetLabelsByPR(repoOwner, repoName, prNumber)
	if err != nil {
		// Se houver erro ao buscar labels, não ignora o PR
		return false
	}

	// Verifica se existe a label IGNORE_PR_GAMIFICATION
	for _, label := range labels {
		if label.LabelName == "IGNORE_PR_GAMIFICATION" {
			fmt.Printf("    🚫 PR #%d ignorado para gamificação (label: IGNORE_PR_GAMIFICATION)\n", prNumber)
			return true
		}
	}

	return false
}

// calculateCommentScore calcula a pontuação de um comentário baseada em suas reações
func (pc *PRChampion) calculateCommentScore(ctx context.Context, repoOwner, repoName string, commentID int64, mergedAt time.Time) float64 {
	// Busca as reações do comentário
	reactions, err := pc.client.ListIssueCommentReactions(ctx, repoOwner, repoName, commentID)
	if err != nil {
		// Se não conseguir buscar reações, conta como 0 pontos
		return 0
	}

	return pc.calculateScoreFromReactions(reactions, mergedAt)
}

// calculateScoreFromReactions calcula a pontuação baseada em uma lista de reações
func (pc *PRChampion) calculateScoreFromReactions(reactions []*github.Reaction, mergedAt time.Time) float64 {
	score := 1.0 // Pontuação base do comentário

	for _, reaction := range reactions {
		if reaction.GetCreatedAt().Time.After(mergedAt) {
			continue // Ignora reações feitas após o merge do PR
		}
		switch reaction.GetContent() {
		case "+1": // 👍
			score += 2.0 // +2 adicional (total = 3)
		case "-1": // 👎
			score -= 2.0 // -2 para neutralizar o ponto base e ainda penalizar (-1)
		case "heart", "hooray", "rocket": // ❤️ 🎉 🚀
			score += 0.5 // Reações positivas menores
		case "confused", "eyes": // 😕 👀
			score -= 0.5 // Reações neutras/negativas menores
		}
	}

	// Garante que a pontuação mínima seja -1 (para comentários muito mal recebidos)
	if score < -1.0 {
		score = -1.0
	}

	return score
}

// calculateReviewCommentScore calcula a pontuação de um review comment baseada em suas reações
func (pc *PRChampion) calculateReviewCommentScore(ctx context.Context, repoOwner, repoName string, commentID int64, mergedAt time.Time) float64 {
	// Busca as reações do review comment
	reactions, err := pc.client.ListPullRequestCommentReactions(ctx, repoOwner, repoName, commentID)
	if err != nil {
		// Se não conseguir buscar reações, conta como 1 ponto normal
		return 1.0
	}

	return pc.calculateScoreFromReactions(reactions, mergedAt)
}

// calculateP90Duration calcula o percentil 90 de uma lista de durações
func calculateP90Duration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Ordena as durações
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	// Calcula o índice do P90
	index := int(float64(len(durations)) * 0.9)
	if index >= len(durations) {
		index = len(durations) - 1
	}

	return durations[index]
}

// calculateMedianDuration calcula a mediana de uma lista de durações
func calculateMedianDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Ordena as durações
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	n := len(durations)
	if n%2 == 0 {
		// Par: média dos dois elementos do meio
		mid1 := durations[n/2-1]
		mid2 := durations[n/2]
		return time.Duration((int64(mid1) + int64(mid2)) / 2)
	} else {
		// Ímpar: elemento do meio
		return durations[n/2]
	}
}

// calculateAverageDuration calcula a média dos tempos de duração
func calculateAverageDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}

	return total / time.Duration(len(durations))
}

// formatTime formata duração no formato especificado
func formatTime(d time.Duration, format string) string {
	if d == 0 {
		return "0"
	}

	switch format {
	case "days":
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%d", days)
	case "minutes":
		minutes := int(d.Minutes())
		return fmt.Sprintf("%d", minutes)
	case "hours":
		fallthrough
	default:
		hours := int(d.Hours())
		return fmt.Sprintf("%d", hours)
	}
}

// formatDuration formata uma duração de forma legível
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "N/A"
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		if hours > 0 {
			return fmt.Sprintf("%dd %dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	}

	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}

	return fmt.Sprintf("%dm", minutes)
}

// getWeekStart retorna o início da semana (segunda-feira)
func getWeekStart(t time.Time) time.Time {
	weekday := t.Weekday()
	if weekday == 0 {
		weekday = 7 // Domingo = 7
	}
	daysBack := int(weekday) - 1
	return t.Add(-time.Duration(daysBack) * 24 * time.Hour).Truncate(24 * time.Hour)
}

// parseDate converte string de data no formato DD/MM/YYYY para time.Time
func parseDate(dateStr string) (time.Time, error) {
	layouts := []string{
		"02/01/2006",
		"2006-01-02",
		"02-01-2006",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("formato de data inválido: %s (use DD/MM/YYYY ou YYYY-MM-DD)", dateStr)
}

// parseRepositories converte strings de repositórios para slice de Repository
// Formato aceito: owner/repo ou owner/repo:branch ou owner/repo:branch1|branch2|branch3
// Suporta branches com barras: owner/repo:feat/rebrand-main|main
func parseRepositories(repoStrings []string) ([]Repository, error) {
	var repositories []Repository

	for _, repoStr := range repoStrings {
		// Primeiro encontra a posição do ':' para separar repo das branches
		colonIndex := strings.Index(repoStr, ":")
		var repoPath string
		var productionBranches []string

		if colonIndex != -1 {
			repoPath = repoStr[:colonIndex]
			branchesStr := strings.TrimSpace(repoStr[colonIndex+1:])

			if branchesStr != "" {
				branchList := strings.Split(branchesStr, "|")
				for _, branch := range branchList {
					branch = strings.TrimSpace(branch)
					if branch != "" {
						productionBranches = append(productionBranches, branch)
					}
				}
			}
		} else {
			repoPath = repoStr
		}

		if len(productionBranches) == 0 {
			productionBranches = []string{"main"}
		}

		// Divide owner/repo - só considera as primeiras duas partes separadas por '/'
		slashIndex := strings.Index(repoPath, "/")
		if slashIndex == -1 || slashIndex == len(repoPath)-1 {
			return nil, fmt.Errorf("formato de repositório inválido: %s (use owner/repo ou owner/repo:branch1|branch2)", repoStr)
		}

		owner := strings.TrimSpace(repoPath[:slashIndex])
		repo := strings.TrimSpace(repoPath[slashIndex+1:])

		if owner == "" || repo == "" {
			return nil, fmt.Errorf("formato de repositório inválido: %s (owner e repo não podem ser vazios)", repoStr)
		}

		repositories = append(repositories, Repository{
			Owner:              owner,
			Name:               repo,
			ProductionBranches: productionBranches,
		})
	}

	return repositories, nil
}

var rootCmd = &cobra.Command{
	Use:   "pr-champion",
	Short: "PR Champion - Contabiliza PRs mergeados e gera ranking",
	Long: `PR Champion é uma ferramenta CLI que analisa PRs mergeados em repositórios GitHub
e gera relatórios com rankings baseados em pontuação semanal.

Comandos disponíveis:
  • load   - Carrega dados da API do GitHub e salva no banco
  • report - Gera relatório baseado nos dados salvos no banco
  • clear  - Limpa completamente o banco de dados

Use 'pr-champion [command] --help' para mais informações sobre cada comando.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Comando load para carregar dados do GitHub
var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "Carrega dados da API do GitHub e salva no banco",
	Long: `Carrega PRs mergeados e comentários da API do GitHub no período especificado
e salva todos os dados no banco de dados local para posterior análise.

APENAS PRs mergeados para a branch de produção são considerados!

Repositórios podem ser especificados via:
  • Flag --repos: --repos microsoft/vscode:main|master,facebook/react:main
  • Variável de ambiente: GITHUB_REPOS=microsoft/vscode:main|master,facebook/react:main
  • Flags individuais: --owner microsoft --repo vscode (usa branch 'main' por padrão)

Formato das branches de produção:
  • owner/repo (usa 'main' como padrão)
  • owner/repo:branch (especifica branch customizada)
  • owner/repo:branch1|branch2|branch3 (múltiplas branches aceitas - separador |)
  • owner/repo:feat/rebrand-main|main (suporta branches com barras)`,
	Run: func(cmd *cobra.Command, args []string) {
		loadDataFromGithub(cmd)
	},
}

// Comando report para gerar relatório
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Gera relatório baseado nos dados salvos no banco",
	Long: `Gera relatório de ranking baseado nos dados já carregados no banco de dados.

Este comando não faz chamadas à API do GitHub, apenas processa os dados
já salvos localmente para gerar os rankings e estatísticas.`,
	Run: func(cmd *cobra.Command, args []string) {
		generateReportFromDatabase(cmd)
	},
}

// Comando clear para limpar banco
var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Limpa completamente o banco de dados",
	Long: `Remove completamente todas as tabelas do banco de dados.
As tabelas serão recriadas automaticamente na próxima execução do comando 'load'.`,
	Run: func(cmd *cobra.Command, args []string) {
		clearDatabase()
	},
}

func init() {
	// Adiciona subcomandos
	rootCmd.AddCommand(loadCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(clearCmd)

	// Flags do comando load
	loadCmd.Flags().StringP("token", "t", "", "Token de acesso do GitHub (ou use GITHUB_TOKEN env var)")
	loadCmd.Flags().StringP("owner", "o", "", "Owner do repositório (compatibilidade com repo único)")
	loadCmd.Flags().StringP("repo", "r", "", "Nome do repositório (compatibilidade com repo único)")
	loadCmd.Flags().StringSliceP("repos", "R", []string{}, "Lista de repositórios no formato owner/repo (ou use GITHUB_REPOS env var)")
	loadCmd.Flags().StringP("start", "s", "", "Data de início (DD/MM/YYYY ou YYYY-MM-DD)")
	loadCmd.Flags().StringP("end", "e", "", "Data de fim (DD/MM/YYYY ou YYYY-MM-DD) - padrão: hoje")
	loadCmd.Flags().IntP("days", "d", 0, "Número de dias atrás para analisar (alternativa às datas específicas)")

	// Flags do comando report
	reportCmd.Flags().StringP("start", "s", "", "Data de início para filtrar dados (DD/MM/YYYY ou YYYY-MM-DD)")
	reportCmd.Flags().StringP("end", "e", "", "Data de fim para filtrar dados (DD/MM/YYYY ou YYYY-MM-DD)")
	reportCmd.Flags().IntP("days", "d", 0, "Número de dias atrás para filtrar dados (alternativa às datas específicas)")

	// Flags para personalizar métricas de velocidade de PR
	reportCmd.Flags().Bool("show-p90", true, "Exibir métrica P90 no relatório de velocidade")
	reportCmd.Flags().Bool("show-median", true, "Exibir métrica de mediana no relatório de velocidade")
	reportCmd.Flags().Bool("show-average", true, "Exibir métrica de média no relatório de velocidade")
	reportCmd.Flags().StringP("time-format", "f", "hours", "Formato de tempo: hours, days, minutes")
	reportCmd.Flags().StringP("group-by", "g", "week", "Agrupar dados por: week, month")
}

// loadDataFromGithub carrega dados da API do GitHub e salva no banco
func loadDataFromGithub(cmd *cobra.Command) {
	// Carrega variáveis do arquivo .env se existir
	if err := godotenv.Load(); err != nil {
		// Não é um erro fatal se o arquivo .env não existir
		if !os.IsNotExist(err) {
			fmt.Printf("⚠️  Aviso: Erro ao carregar .env: %v\n", err)
		}
	} else {
		fmt.Println("✅ Arquivo .env carregado com sucesso")
	}

	token, _ := cmd.Flags().GetString("token")
	owner, _ := cmd.Flags().GetString("owner")
	repo, _ := cmd.Flags().GetString("repo")
	reposList, _ := cmd.Flags().GetStringSlice("repos")
	startDateStr, _ := cmd.Flags().GetString("start")
	endDateStr, _ := cmd.Flags().GetString("end")
	daysBack, _ := cmd.Flags().GetInt("days")

	// Validação do token
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
		if token == "" {
			log.Fatal("❌ Token do GitHub é obrigatório. Use --token ou defina GITHUB_TOKEN")
		}
	}

	// Construir lista de repositórios
	var repositories []Repository
	var err error

	if len(reposList) > 0 {
		// Usar lista de repositórios da flag --repos
		repositories, err = parseRepositories(reposList)
		if err != nil {
			log.Fatalf("❌ Erro ao parsear repositórios da flag: %v", err)
		}
	} else if owner != "" && repo != "" {
		// Usar repositório único (compatibilidade)
		repositories = []Repository{{Owner: owner, Name: repo, ProductionBranches: []string{"main"}}}
	} else {
		// Tentar ler da variável de ambiente GITHUB_REPOS
		envRepos := os.Getenv("GITHUB_REPOS")
		if envRepos != "" {
			repoStrings := strings.Split(envRepos, ",")
			// Remove espaços em branco
			for i, repo := range repoStrings {
				repoStrings[i] = strings.TrimSpace(repo)
			}
			repositories, err = parseRepositories(repoStrings)
			if err != nil {
				log.Fatalf("❌ Erro ao parsear repositórios da variável GITHUB_REPOS: %v", err)
			}
			fmt.Printf("📋 Usando repositórios da variável GITHUB_REPOS: %s\n", envRepos)
		} else {
			log.Fatal("❌ Especifique repositórios usando:\n" +
				"   • --repos owner1/repo1:main|master,owner2/repo2\n" +
				"   • --owner e --repo (repositório único)\n" +
				"   • Variável GITHUB_REPOS=owner1/repo1:main|master,owner2/repo2")
		}
	}

	var startDate, endDate time.Time

	// Se foi especificado --days, calcula as datas automaticamente
	if daysBack > 0 {
		endDate = time.Now()
		startDate = endDate.Add(-time.Duration(daysBack) * 24 * time.Hour)
	} else {
		// Parse das datas
		if startDateStr == "" {
			startDate = time.Now().Add(-30 * 24 * time.Hour) // 30 dias atrás por padrão
		} else {
			startDate, err = parseDate(startDateStr)
			if err != nil {
				log.Fatalf("❌ Erro na data de início: %v", err)
			}
		}

		if endDateStr == "" {
			endDate = time.Now() // Até hoje por padrão
		} else {
			endDate, err = parseDate(endDateStr)
			if err != nil {
				log.Fatalf("❌ Erro na data de fim: %v", err)
			}
		}
	}

	// Validação das datas
	if endDate.Before(startDate) {
		log.Fatal("❌ Data de fim deve ser posterior à data de início")
	}

	fmt.Printf("🚀 Carregando dados do GitHub (%s até %s)...\n",
		startDate.Format("02/01/2006"), endDate.Format("02/01/2006"))

	prChampion, err := NewPRChampion(token, repositories, startDate, endDate)
	if err != nil {
		log.Fatalf("❌ Erro ao inicializar PR Champion: %v", err)
	}

	// Garante que a conexão seja fechada no final
	defer func() {
		if prChampion.cachedClient != nil {
			prChampion.cachedClient.Close()
		}
	}()

	if err := prChampion.FetchMergedPRs(); err != nil {
		log.Fatalf("❌ Erro ao buscar PRs: %v", err)
	}

	fmt.Println("✅ Dados carregados com sucesso no banco de dados!")
}

// generateReportFromDatabase gera relatório baseado nos dados salvos no banco
func generateReportFromDatabase(cmd *cobra.Command) {
	fmt.Println("📊 Gerando relatório dos dados salvos...")

	startDateStr, _ := cmd.Flags().GetString("start")
	endDateStr, _ := cmd.Flags().GetString("end")
	daysBack, _ := cmd.Flags().GetInt("days")

	// Ler configurações de métricas
	showP90, _ := cmd.Flags().GetBool("show-p90")
	showMedian, _ := cmd.Flags().GetBool("show-median")
	showAverage, _ := cmd.Flags().GetBool("show-average")
	timeFormat, _ := cmd.Flags().GetString("time-format")
	groupBy, _ := cmd.Flags().GetString("group-by")

	// Validar formato de tempo
	if timeFormat != "hours" && timeFormat != "days" && timeFormat != "minutes" {
		fmt.Printf("⚠️  Formato de tempo inválido: %s. Usando 'hours' como padrão.\n", timeFormat)
		timeFormat = "hours"
	}

	// Validar agrupamento
	if groupBy != "week" && groupBy != "month" {
		fmt.Printf("⚠️  Tipo de agrupamento inválido: %s. Usando 'week' como padrão.\n", groupBy)
		groupBy = "week"
	}

	config := MetricDisplayConfig{
		ShowP90:     showP90,
		ShowMedian:  showMedian,
		ShowAverage: showAverage,
		TimeFormat:  timeFormat,
		GroupBy:     groupBy,
	}

	var startDate, endDate time.Time
	var err error

	// Se foi especificado --days, calcula as datas automaticamente
	if daysBack > 0 {
		endDate = time.Now()
		startDate = endDate.Add(-time.Duration(daysBack) * 24 * time.Hour)
	} else {
		// Parse das datas (opcionais para filtrar dados)
		if startDateStr != "" {
			startDate, err = parseDate(startDateStr)
			if err != nil {
				log.Fatalf("❌ Erro na data de início: %v", err)
			}
		}

		if endDateStr != "" {
			endDate, err = parseDate(endDateStr)
			if err != nil {
				log.Fatalf("❌ Erro na data de fim: %v", err)
			}
		}
	}

	// Cria instância mínima apenas para acessar o banco (sem precisar de token)
	prChampion, err := NewPRChampionFromDatabase(startDate, endDate)
	if err != nil {
		log.Fatalf("❌ Erro ao inicializar acesso ao banco: %v", err)
	}

	// Garante que a conexão seja fechada no final
	defer func() {
		if prChampion.cachedClient != nil {
			prChampion.cachedClient.Close()
		}
	}()

	if err := prChampion.LoadDataFromDatabase(); err != nil {
		log.Fatalf("❌ Erro ao carregar dados do banco: %v", err)
	}

	prChampion.GenerateReportWithConfig(config)
	fmt.Println("✅ Relatório gerado com sucesso!")
}

// clearDatabase limpa completamente o banco de dados
func clearDatabase() {
	fmt.Println("🗑️  Limpando banco de dados...")

	// Cria instância mínima apenas para acessar o banco
	prChampion, err := NewPRChampionFromDatabase(time.Time{}, time.Time{})
	if err != nil {
		log.Fatalf("❌ Erro ao inicializar acesso ao banco: %v", err)
	}

	// Garante que a conexão seja fechada no final
	defer func() {
		if prChampion.cachedClient != nil {
			prChampion.cachedClient.Close()
		}
	}()

	if err := prChampion.ClearCache(); err != nil {
		log.Fatalf("❌ Erro ao limpar banco: %v", err)
	}

	fmt.Println("✅ Banco de dados completamente limpo!")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
