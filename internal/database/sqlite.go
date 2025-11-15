package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// CommentDatabase interface para operações de banco de dados
type CommentDatabase interface {
	// PRs
	GetPR(repoOwner, repoName string, prNumber int) (*PRData, error)
	SavePR(pr *PRData) error
	MarkPRCommentsChecked(repoOwner, repoName string, prNumber int, commentType string, hasComments bool) error
	MarkPRReviewsChecked(repoOwner, repoName string, prNumber int, hasReviews, hasApprovedReviews bool) error

	// Comentários
	GetComment(repoOwner, repoName string, commentID int64) (*CommentData, error)
	SaveComment(comment *CommentData) error
	GetCommentsByPR(repoOwner, repoName string, prNumber int) ([]*CommentData, error)
	GetCommentsByPRAndType(repoOwner, repoName string, prNumber int, commentType string) ([]*CommentData, error)
	MarkReactionsChecked(commentID int64) error

	// Reviews
	GetReview(repoOwner, repoName string, reviewID int64) (*ReviewData, error)
	SaveReview(review *ReviewData) error
	GetReviewsByPR(repoOwner, repoName string, prNumber int) ([]*ReviewData, error)

	// Reações
	GetReactions(commentID int64) ([]*ReactionData, error)
	GetReactionsByType(commentID int64, reactionType string) ([]*ReactionData, error)
	SaveReaction(reaction *ReactionData) error
	SaveReactions(reactions []*ReactionData) error

	// Utilitários
	ClearDatabase() error
	Close() error
}

type sqliteDatabase struct {
	db *sql.DB
}

// NewSQLiteDatabase cria uma nova instância do banco SQLite
func NewSQLiteDatabase(dbPath string) (CommentDatabase, error) {
	// Cria o diretório se não existir
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("erro ao criar diretório do banco: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir banco SQLite: %v", err)
	}

	// Testa a conexão
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao conectar com banco SQLite: %v", err)
	}

	sqliteDB := &sqliteDatabase{db: db}

	// Cria as tabelas se não existirem
	if err := sqliteDB.createTables(); err != nil {
		return nil, fmt.Errorf("erro ao criar tabelas: %v", err)
	}

	return sqliteDB, nil
}

// createTables cria as tabelas necessárias
func (db *sqliteDatabase) createTables() error {
	// Tabela de comentários
	createCommentsTable := `
	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_owner TEXT NOT NULL,
		repo_name TEXT NOT NULL,
		pr_number INTEGER NOT NULL,
		comment_id INTEGER NOT NULL UNIQUE,
		comment_type TEXT NOT NULL,
		username TEXT NOT NULL,
		body TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		cached_at DATETIME NOT NULL,
		reactions_checked BOOLEAN DEFAULT FALSE,
		UNIQUE(comment_id)
	);`

	// Tabela de PRs
	createPRsTable := `
	CREATE TABLE IF NOT EXISTS prs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_owner TEXT NOT NULL,
		repo_name TEXT NOT NULL,
		pr_number INTEGER NOT NULL,
		title TEXT NOT NULL,
		username TEXT NOT NULL,
		merged_at DATETIME NOT NULL,
		has_comments BOOLEAN DEFAULT FALSE,
		has_issue_comments BOOLEAN DEFAULT FALSE,
		has_review_comments BOOLEAN DEFAULT FALSE,
		has_reviews BOOLEAN DEFAULT FALSE,
		has_approved_reviews BOOLEAN DEFAULT FALSE,
		comments_checked BOOLEAN DEFAULT FALSE,
		issue_comments_checked BOOLEAN DEFAULT FALSE,
		review_comments_checked BOOLEAN DEFAULT FALSE,
		reviews_checked BOOLEAN DEFAULT FALSE,
		cached_at DATETIME NOT NULL,
		UNIQUE(repo_owner, repo_name, pr_number)
	);`

	// Tabela de reações
	createReactionsTable := `
	CREATE TABLE IF NOT EXISTS reactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		comment_id INTEGER NOT NULL,
		reaction_type TEXT NOT NULL DEFAULT 'issue_comment',
		content TEXT NOT NULL,
		username TEXT NOT NULL,
		cached_at DATETIME NOT NULL,
		FOREIGN KEY(comment_id) REFERENCES comments(comment_id),
		UNIQUE(comment_id, reaction_type, content, username)
	);`

	// Tabela de reviews
	createReviewsTable := `
	CREATE TABLE IF NOT EXISTS reviews (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_owner TEXT NOT NULL,
		repo_name TEXT NOT NULL,
		pr_number INTEGER NOT NULL,
		review_id INTEGER NOT NULL UNIQUE,
		username TEXT NOT NULL,
		state TEXT NOT NULL,
		body TEXT,
		submitted_at DATETIME NOT NULL,
		cached_at DATETIME NOT NULL,
		UNIQUE(review_id)
	);`

	// Índices para melhor performance
	createIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_comments_repo_pr ON comments(repo_owner, repo_name, pr_number);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_comment_id ON comments(comment_id);`,
		`CREATE INDEX IF NOT EXISTS idx_reactions_comment_id ON reactions(comment_id);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_cached_at ON comments(cached_at);`,
		`CREATE INDEX IF NOT EXISTS idx_prs_repo ON prs(repo_owner, repo_name);`,
		`CREATE INDEX IF NOT EXISTS idx_prs_repo_pr ON prs(repo_owner, repo_name, pr_number);`,
		`CREATE INDEX IF NOT EXISTS idx_reviews_repo_pr ON reviews(repo_owner, repo_name, pr_number);`,
		`CREATE INDEX IF NOT EXISTS idx_reviews_review_id ON reviews(review_id);`,
	}

	// Executa criação das tabelas
	if _, err := db.db.Exec(createCommentsTable); err != nil {
		return fmt.Errorf("erro ao criar tabela comments: %v", err)
	}

	if _, err := db.db.Exec(createPRsTable); err != nil {
		return fmt.Errorf("erro ao criar tabela prs: %v", err)
	}

	if _, err := db.db.Exec(createReactionsTable); err != nil {
		return fmt.Errorf("erro ao criar tabela reactions: %v", err)
	}

	if _, err := db.db.Exec(createReviewsTable); err != nil {
		return fmt.Errorf("erro ao criar tabela reviews: %v", err)
	}

	// Executa criação dos índices
	for _, indexSQL := range createIndexes {
		if _, err := db.db.Exec(indexSQL); err != nil {
			return fmt.Errorf("erro ao criar índice: %v", err)
		}
	}

	return nil
}

// GetPR busca um PR pelo repositório e número
func (db *sqliteDatabase) GetPR(repoOwner, repoName string, prNumber int) (*PRData, error) {
	query := `
		SELECT id, repo_owner, repo_name, pr_number, title, username, merged_at,
		       has_comments, has_issue_comments, has_review_comments, has_reviews, has_approved_reviews,
		       comments_checked, issue_comments_checked, review_comments_checked, reviews_checked, cached_at
		FROM prs 
		WHERE repo_owner = ? AND repo_name = ? AND pr_number = ?`

	row := db.db.QueryRow(query, repoOwner, repoName, prNumber)

	pr := &PRData{}
	err := row.Scan(
		&pr.ID,
		&pr.RepoOwner,
		&pr.RepoName,
		&pr.PRNumber,
		&pr.Title,
		&pr.Username,
		&pr.MergedAt,
		&pr.HasComments,
		&pr.HasIssueComments,
		&pr.HasReviewComments,
		&pr.HasReviews,
		&pr.HasApprovedReviews,
		&pr.CommentsChecked,
		&pr.IssueCommentsChecked,
		&pr.ReviewCommentsChecked,
		&pr.ReviewsChecked,
		&pr.CachedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // PR não encontrado
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar PR: %v", err)
	}

	return pr, nil
}

// SavePR salva um PR no banco
func (db *sqliteDatabase) SavePR(pr *PRData) error {
	query := `
		INSERT OR REPLACE INTO prs 
		(repo_owner, repo_name, pr_number, title, username, merged_at,
		 has_comments, has_issue_comments, has_review_comments, has_reviews, has_approved_reviews,
		 comments_checked, issue_comments_checked, review_comments_checked, reviews_checked, cached_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.db.Exec(query,
		pr.RepoOwner,
		pr.RepoName,
		pr.PRNumber,
		pr.Title,
		pr.Username,
		pr.MergedAt,
		pr.HasComments,
		pr.HasIssueComments,
		pr.HasReviewComments,
		pr.HasReviews,
		pr.HasApprovedReviews,
		pr.CommentsChecked,
		pr.IssueCommentsChecked,
		pr.ReviewCommentsChecked,
		pr.ReviewsChecked,
		pr.CachedAt,
	)

	if err != nil {
		return fmt.Errorf("erro ao salvar PR: %v", err)
	}

	return nil
}

// MarkPRCommentsChecked marca que os comentários de um PR foram verificados
func (db *sqliteDatabase) MarkPRCommentsChecked(repoOwner, repoName string, prNumber int, commentType string, hasComments bool) error {
	// Verifica se o PR já existe
	existingPR, err := db.GetPR(repoOwner, repoName, prNumber)
	if err != nil {
		return fmt.Errorf("erro ao verificar PR existente: %v", err)
	}

	// Se não existe, cria um registro básico
	if existingPR == nil {
		insertQuery := `
			INSERT INTO prs 
			(repo_owner, repo_name, pr_number, title, username, merged_at, cached_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`

		now := time.Now()
		_, err := db.db.Exec(insertQuery, repoOwner, repoName, prNumber, "", "", now, now)
		if err != nil {
			return fmt.Errorf("erro ao criar registro básico do PR: %v", err)
		}
	}

	// Agora atualiza os campos específicos
	var query string
	var args []interface{}

	switch commentType {
	case "issue":
		query = `UPDATE prs SET issue_comments_checked = TRUE, has_issue_comments = ? WHERE repo_owner = ? AND repo_name = ? AND pr_number = ?`
		args = []interface{}{hasComments, repoOwner, repoName, prNumber}
	case "review":
		query = `UPDATE prs SET review_comments_checked = TRUE, has_review_comments = ? WHERE repo_owner = ? AND repo_name = ? AND pr_number = ?`
		args = []interface{}{hasComments, repoOwner, repoName, prNumber}
	case "reviews":
		query = `UPDATE prs SET reviews_checked = TRUE, has_reviews = ? WHERE repo_owner = ? AND repo_name = ? AND pr_number = ?`
		args = []interface{}{hasComments, repoOwner, repoName, prNumber}
	default:
		query = `UPDATE prs SET comments_checked = TRUE, has_comments = ? WHERE repo_owner = ? AND repo_name = ? AND pr_number = ?`
		args = []interface{}{hasComments, repoOwner, repoName, prNumber}
	}

	_, err = db.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("erro ao marcar comentários do PR como verificados: %v", err)
	}

	// Atualiza has_comments se ambos os tipos foram verificados
	updateGeneralQuery := `
		UPDATE prs SET has_comments = (has_issue_comments OR has_review_comments)
		WHERE repo_owner = ? AND repo_name = ? AND pr_number = ? 
		AND issue_comments_checked = TRUE AND review_comments_checked = TRUE`

	_, _ = db.db.Exec(updateGeneralQuery, repoOwner, repoName, prNumber)

	return nil
}

// GetComment busca um comentário pelo ID
func (db *sqliteDatabase) GetComment(repoOwner, repoName string, commentID int64) (*CommentData, error) {
	query := `
		SELECT id, repo_owner, repo_name, pr_number, comment_id, comment_type, 
		       username, body, created_at, updated_at, cached_at, reactions_checked
		FROM comments 
		WHERE comment_id = ?`

	row := db.db.QueryRow(query, commentID)

	comment := &CommentData{}
	err := row.Scan(
		&comment.ID,
		&comment.RepoOwner,
		&comment.RepoName,
		&comment.PRNumber,
		&comment.CommentID,
		&comment.CommentType,
		&comment.Username,
		&comment.Body,
		&comment.CreatedAt,
		&comment.UpdatedAt,
		&comment.CachedAt,
		&comment.ReactionsChecked,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Comentário não encontrado
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar comentário: %v", err)
	}

	return comment, nil
}

// SaveComment salva um comentário no banco
func (db *sqliteDatabase) SaveComment(comment *CommentData) error {
	query := `
		INSERT OR REPLACE INTO comments 
		(repo_owner, repo_name, pr_number, comment_id, comment_type, username, body, created_at, updated_at, cached_at, reactions_checked)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.db.Exec(query,
		comment.RepoOwner,
		comment.RepoName,
		comment.PRNumber,
		comment.CommentID,
		comment.CommentType,
		comment.Username,
		comment.Body,
		comment.CreatedAt,
		comment.UpdatedAt,
		comment.CachedAt,
		comment.ReactionsChecked,
	)

	if err != nil {
		return fmt.Errorf("erro ao salvar comentário: %v", err)
	}

	return nil
}

// GetCommentsByPR busca todos os comentários de um PR
func (db *sqliteDatabase) GetCommentsByPR(repoOwner, repoName string, prNumber int) ([]*CommentData, error) {
	query := `
		SELECT id, repo_owner, repo_name, pr_number, comment_id, comment_type, 
		       username, body, created_at, updated_at, cached_at, reactions_checked
		FROM comments 
		WHERE repo_owner = ? AND repo_name = ? AND pr_number = ?
		ORDER BY created_at`

	rows, err := db.db.Query(query, repoOwner, repoName, prNumber)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar comentários do PR: %v", err)
	}
	defer rows.Close()

	var comments []*CommentData
	for rows.Next() {
		comment := &CommentData{}
		err := rows.Scan(
			&comment.ID,
			&comment.RepoOwner,
			&comment.RepoName,
			&comment.PRNumber,
			&comment.CommentID,
			&comment.CommentType,
			&comment.Username,
			&comment.Body,
			&comment.CreatedAt,
			&comment.UpdatedAt,
			&comment.CachedAt,
			&comment.ReactionsChecked,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao escanear comentário: %v", err)
		}
		comments = append(comments, comment)
	}

	return comments, nil
}

// GetCommentsByPRAndType busca comentários de um PR filtrados por tipo
func (db *sqliteDatabase) GetCommentsByPRAndType(repoOwner, repoName string, prNumber int, commentType string) ([]*CommentData, error) {
	query := `
		SELECT id, repo_owner, repo_name, pr_number, comment_id, comment_type, 
		       username, body, created_at, updated_at, cached_at, reactions_checked
		FROM comments 
		WHERE repo_owner = ? AND repo_name = ? AND pr_number = ? AND comment_type = ?
		ORDER BY created_at`

	rows, err := db.db.Query(query, repoOwner, repoName, prNumber, commentType)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar comentários do PR por tipo: %v", err)
	}
	defer rows.Close()

	var comments []*CommentData
	for rows.Next() {
		comment := &CommentData{}
		err := rows.Scan(
			&comment.ID,
			&comment.RepoOwner,
			&comment.RepoName,
			&comment.PRNumber,
			&comment.CommentID,
			&comment.CommentType,
			&comment.Username,
			&comment.Body,
			&comment.CreatedAt,
			&comment.UpdatedAt,
			&comment.CachedAt,
			&comment.ReactionsChecked,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao fazer scan do comentário: %v", err)
		}
		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro durante iteração dos comentários: %v", err)
	}

	return comments, nil
}

// MarkReactionsChecked marca que as reações de um comentário foram verificadas
func (db *sqliteDatabase) MarkReactionsChecked(commentID int64) error {
	query := `UPDATE comments SET reactions_checked = TRUE WHERE comment_id = ?`

	_, err := db.db.Exec(query, commentID)
	if err != nil {
		return fmt.Errorf("erro ao marcar reações como verificadas: %v", err)
	}

	return nil
}

// GetReactions busca todas as reações de um comentário
func (db *sqliteDatabase) GetReactions(commentID int64) ([]*ReactionData, error) {
	query := `
		SELECT id, comment_id, reaction_type, content, username, cached_at
		FROM reactions 
		WHERE comment_id = ?`

	rows, err := db.db.Query(query, commentID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar reações: %v", err)
	}
	defer rows.Close()

	var reactions []*ReactionData
	for rows.Next() {
		reaction := &ReactionData{}
		err := rows.Scan(
			&reaction.ID,
			&reaction.CommentID,
			&reaction.ReactionType,
			&reaction.Content,
			&reaction.Username,
			&reaction.CachedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao escanear reação: %v", err)
		}
		reactions = append(reactions, reaction)
	}

	return reactions, nil
}

// GetReactionsByType busca reações de um comentário por tipo específico
func (db *sqliteDatabase) GetReactionsByType(commentID int64, reactionType string) ([]*ReactionData, error) {
	query := `
		SELECT id, comment_id, reaction_type, content, username, cached_at
		FROM reactions 
		WHERE comment_id = ? AND reaction_type = ?`

	rows, err := db.db.Query(query, commentID, reactionType)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar reações por tipo: %v", err)
	}
	defer rows.Close()

	var reactions []*ReactionData
	for rows.Next() {
		reaction := &ReactionData{}
		err := rows.Scan(
			&reaction.ID,
			&reaction.CommentID,
			&reaction.ReactionType,
			&reaction.Content,
			&reaction.Username,
			&reaction.CachedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao escanear reação por tipo: %v", err)
		}
		reactions = append(reactions, reaction)
	}

	return reactions, nil
}

// SaveReaction salva uma reação no banco
func (db *sqliteDatabase) SaveReaction(reaction *ReactionData) error {
	query := `
		INSERT OR REPLACE INTO reactions 
		(comment_id, reaction_type, content, username, cached_at)
		VALUES (?, ?, ?, ?, ?)`

	_, err := db.db.Exec(query,
		reaction.CommentID,
		reaction.ReactionType,
		reaction.Content,
		reaction.Username,
		reaction.CachedAt,
	)

	if err != nil {
		return fmt.Errorf("erro ao salvar reação: %v", err)
	}

	return nil
}

// SaveReactions salva múltiplas reações no banco
func (db *sqliteDatabase) SaveReactions(reactions []*ReactionData) error {
	if len(reactions) == 0 {
		return nil
	}

	tx, err := db.db.Begin()
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação: %v", err)
	}
	defer func() {
		_ = tx.Rollback() // Ignore rollback errors
	}()

	query := `
		INSERT OR REPLACE INTO reactions 
		(comment_id, reaction_type, content, username, cached_at)
		VALUES (?, ?, ?, ?, ?)`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("erro ao preparar statement: %v", err)
	}
	defer stmt.Close()

	for _, reaction := range reactions {
		_, err := stmt.Exec(
			reaction.CommentID,
			reaction.ReactionType,
			reaction.Content,
			reaction.Username,
			reaction.CachedAt,
		)
		if err != nil {
			return fmt.Errorf("erro ao salvar reação: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("erro ao confirmar transação: %v", err)
	}

	return nil
}

// MarkPRReviewsChecked marca que os reviews de um PR foram verificados
func (db *sqliteDatabase) MarkPRReviewsChecked(repoOwner, repoName string, prNumber int, hasReviews, hasApprovedReviews bool) error {
	// Verifica se o PR já existe
	existingPR, err := db.GetPR(repoOwner, repoName, prNumber)
	if err != nil {
		return fmt.Errorf("erro ao verificar PR existente: %v", err)
	}

	// Se não existe, cria um registro básico
	if existingPR == nil {
		insertQuery := `
			INSERT INTO prs 
			(repo_owner, repo_name, pr_number, title, username, merged_at, cached_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`

		now := time.Now()
		_, err := db.db.Exec(insertQuery, repoOwner, repoName, prNumber, "", "", now, now)
		if err != nil {
			return fmt.Errorf("erro ao criar registro básico do PR: %v", err)
		}
	}

	// Atualiza os campos de reviews
	query := `UPDATE prs SET reviews_checked = TRUE, has_reviews = ?, has_approved_reviews = ? WHERE repo_owner = ? AND repo_name = ? AND pr_number = ?`
	args := []interface{}{hasReviews, hasApprovedReviews, repoOwner, repoName, prNumber}

	_, err = db.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("erro ao marcar reviews do PR como verificados: %v", err)
	}

	return nil
}

// GetReview busca um review pelo ID
func (db *sqliteDatabase) GetReview(repoOwner, repoName string, reviewID int64) (*ReviewData, error) {
	query := `
		SELECT id, repo_owner, repo_name, pr_number, review_id, username, state, body, submitted_at, cached_at
		FROM reviews 
		WHERE review_id = ?`

	row := db.db.QueryRow(query, reviewID)

	review := &ReviewData{}
	err := row.Scan(
		&review.ID,
		&review.RepoOwner,
		&review.RepoName,
		&review.PRNumber,
		&review.ReviewID,
		&review.Username,
		&review.State,
		&review.Body,
		&review.SubmittedAt,
		&review.CachedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Review não encontrado
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar review: %v", err)
	}

	return review, nil
}

// SaveReview salva um review no banco
func (db *sqliteDatabase) SaveReview(review *ReviewData) error {
	query := `
		INSERT OR REPLACE INTO reviews 
		(repo_owner, repo_name, pr_number, review_id, username, state, body, submitted_at, cached_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.db.Exec(query,
		review.RepoOwner,
		review.RepoName,
		review.PRNumber,
		review.ReviewID,
		review.Username,
		review.State,
		review.Body,
		review.SubmittedAt,
		review.CachedAt,
	)

	if err != nil {
		return fmt.Errorf("erro ao salvar review: %v", err)
	}

	return nil
}

// GetReviewsByPR busca todos os reviews de um PR
func (db *sqliteDatabase) GetReviewsByPR(repoOwner, repoName string, prNumber int) ([]*ReviewData, error) {
	query := `
		SELECT id, repo_owner, repo_name, pr_number, review_id, username, state, body, submitted_at, cached_at
		FROM reviews 
		WHERE repo_owner = ? AND repo_name = ? AND pr_number = ?
		ORDER BY submitted_at ASC`

	rows, err := db.db.Query(query, repoOwner, repoName, prNumber)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar reviews: %v", err)
	}
	defer rows.Close()

	var reviews []*ReviewData
	for rows.Next() {
		review := &ReviewData{}
		err := rows.Scan(
			&review.ID,
			&review.RepoOwner,
			&review.RepoName,
			&review.PRNumber,
			&review.ReviewID,
			&review.Username,
			&review.State,
			&review.Body,
			&review.SubmittedAt,
			&review.CachedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear review: %v", err)
		}
		reviews = append(reviews, review)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar reviews: %v", err)
	}

	return reviews, nil
}

// ClearDatabase limpa todos os dados do banco
func (db *sqliteDatabase) ClearDatabase() error {
	// Remove todas as reações primeiro (por causa da foreign key)
	if _, err := db.db.Exec("DELETE FROM reactions"); err != nil {
		return fmt.Errorf("erro ao limpar tabela reactions: %v", err)
	}

	// Remove todos os comentários
	if _, err := db.db.Exec("DELETE FROM comments"); err != nil {
		return fmt.Errorf("erro ao limpar tabela comments: %v", err)
	}

	// Remove todos os reviews
	if _, err := db.db.Exec("DELETE FROM reviews"); err != nil {
		return fmt.Errorf("erro ao limpar tabela reviews: %v", err)
	}

	// Remove todos os PRs
	if _, err := db.db.Exec("DELETE FROM prs"); err != nil {
		return fmt.Errorf("erro ao limpar tabela prs: %v", err)
	}

	// Reset dos auto-increment
	if _, err := db.db.Exec("DELETE FROM sqlite_sequence WHERE name IN ('comments', 'reactions', 'reviews', 'prs')"); err != nil {
		// Não é um erro fatal se a tabela sqlite_sequence não existir
		fmt.Printf("⚠️  Aviso: Não foi possível resetar sequências: %v\n", err)
	}

	return nil
}

// Close fecha a conexão com o banco
func (db *sqliteDatabase) Close() error {
	return db.db.Close()
}

// IsCommentStale verifica se um comentário em cache está desatualizado (mais de 7 dias)
func (db *sqliteDatabase) IsCommentStale(comment *CommentData) bool {
	cacheDuration := 7 * 24 * time.Hour // 7 dias
	return time.Since(comment.CachedAt) > cacheDuration
}
