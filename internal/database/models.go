package database

import (
	"time"

	"github.com/google/go-github/v70/github"
)

// CommentData representa um comentário armazenado no banco
type CommentData struct {
	ID               int64     `json:"id"`
	RepoOwner        string    `json:"repo_owner"`
	RepoName         string    `json:"repo_name"`
	PRNumber         int       `json:"pr_number"`
	CommentID        int64     `json:"comment_id"`
	CommentType      string    `json:"comment_type"` // "issue" ou "review"
	Username         string    `json:"username"`
	Body             string    `json:"body"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	CachedAt         time.Time `json:"cached_at"`
	ReactionsChecked bool      `json:"reactions_checked"` // Se as reações já foram verificadas
}

// ReactionData representa uma reação armazenada no banco
type ReactionData struct {
	ID           int64     `json:"id"`
	CommentID    int64     `json:"comment_id"`
	ReactionType string    `json:"reaction_type"` // "issue_comment" ou "review_comment"
	Content      string    `json:"content"`       // "+1", "-1", "heart", etc.
	Username     string    `json:"username"`
	CreatedAt    time.Time `json:"created_at"` // Data real de criação da reação
	CachedAt     time.Time `json:"cached_at"`
}

// ReviewData representa um review armazenado no banco
type ReviewData struct {
	ID          int64     `json:"id"`
	RepoOwner   string    `json:"repo_owner"`
	RepoName    string    `json:"repo_name"`
	PRNumber    int       `json:"pr_number"`
	ReviewID    int64     `json:"review_id"`
	Username    string    `json:"username"`
	State       string    `json:"state"` // "APPROVED", "REQUEST_CHANGES", "COMMENTED", "DISMISSED"
	Body        string    `json:"body"`
	SubmittedAt time.Time `json:"submitted_at"`
	CachedAt    time.Time `json:"cached_at"`
}

// PRData representa um PR armazenado no banco
type PRData struct {
	ID                    int64     `json:"id"`
	RepoOwner             string    `json:"repo_owner"`
	RepoName              string    `json:"repo_name"`
	PRNumber              int       `json:"pr_number"`
	Title                 string    `json:"title"`
	Username              string    `json:"username"`
	MergedAt              time.Time `json:"merged_at"`
	HasComments           bool      `json:"has_comments"`            // Se tem comentários (issue ou review)
	HasIssueComments      bool      `json:"has_issue_comments"`      // Se tem issue comments
	HasReviewComments     bool      `json:"has_review_comments"`     // Se tem review comments
	HasReviews            bool      `json:"has_reviews"`             // Se tem reviews
	HasApprovedReviews    bool      `json:"has_approved_reviews"`    // Se tem pelo menos um review approved
	CommentsChecked       bool      `json:"comments_checked"`        // Se os comentários já foram verificados
	IssueCommentsChecked  bool      `json:"issue_comments_checked"`  // Se issue comments foram verificados
	ReviewCommentsChecked bool      `json:"review_comments_checked"` // Se review comments foram verificados
	ReviewsChecked        bool      `json:"reviews_checked"`         // Se reviews foram verificados
	Additions             int       `json:"additions"`               // Linhas adicionadas
	Deletions             int       `json:"deletions"`               // Linhas removidas
	ChangedFiles          int       `json:"changed_files"`           // Arquivos modificados
	CachedAt              time.Time `json:"cached_at"`
}

// PRLabelData representa uma tag/label de um PR armazenada no banco
// PRLabelData representa uma tag/label de um PR armazenada no banco
type PRLabelData struct {
	ID          int64  `json:"id"`
	RepoOwner   string `json:"repo_owner"`
	RepoName    string `json:"repo_name"`
	PRNumber    int    `json:"pr_number"`
	LabelName   string `json:"label_name"`
	Color       string `json:"color"`       // Cor da label em hexadecimal
	Description string `json:"description"` // Descrição da label
}

// FromGithubLabel converte um github.Label para PRLabelData
func FromGithubLabel(label *github.Label, repoOwner, repoName string, prNumber int) *PRLabelData {
	return &PRLabelData{
		RepoOwner:   repoOwner,
		RepoName:    repoName,
		PRNumber:    prNumber,
		LabelName:   label.GetName(),
		Color:       label.GetColor(),
		Description: label.GetDescription(),
	}
}

// CommentWithReactions representa um comentário com suas reações
type CommentWithReactions struct {
	Comment   *CommentData    `json:"comment"`
	Reactions []*ReactionData `json:"reactions"`
}

// FromGithubIssueComment converte um github.IssueComment para CommentData
func FromGithubIssueComment(comment *github.IssueComment, repoOwner, repoName string, prNumber int) *CommentData {
	return &CommentData{
		RepoOwner:        repoOwner,
		RepoName:         repoName,
		PRNumber:         prNumber,
		CommentID:        comment.GetID(),
		CommentType:      "issue",
		Username:         comment.User.GetLogin(),
		Body:             comment.GetBody(),
		CreatedAt:        comment.CreatedAt.Time,
		UpdatedAt:        comment.UpdatedAt.Time,
		CachedAt:         time.Now(),
		ReactionsChecked: false, // Inicialmente as reações não foram verificadas
	}
}

// FromGithubReviewComment converte um github.PullRequestComment para CommentData
func FromGithubReviewComment(comment *github.PullRequestComment, repoOwner, repoName string, prNumber int) *CommentData {
	return &CommentData{
		RepoOwner:        repoOwner,
		RepoName:         repoName,
		PRNumber:         prNumber,
		CommentID:        comment.GetID(),
		CommentType:      "review",
		Username:         comment.User.GetLogin(),
		Body:             comment.GetBody(),
		CreatedAt:        comment.CreatedAt.Time,
		UpdatedAt:        comment.UpdatedAt.Time,
		CachedAt:         time.Now(),
		ReactionsChecked: false, // Inicialmente as reações não foram verificadas
	}
}

// FromGithubReaction converte um github.Reaction para ReactionData (para issue comments)
func FromGithubReaction(reaction *github.Reaction, commentID int64) *ReactionData {
	return &ReactionData{
		CommentID:    commentID,
		ReactionType: "issue_comment",
		Content:      reaction.GetContent(),
		Username:     reaction.User.GetLogin(),
		CreatedAt:    reaction.GetCreatedAt().Time,
		CachedAt:     time.Now(),
	}
}

// FromGithubReviewReaction converte um github.Reaction para ReactionData (para review comments)
func FromGithubReviewReaction(reaction *github.Reaction, commentID int64) *ReactionData {
	return &ReactionData{
		CommentID:    commentID,
		ReactionType: "review_comment",
		Content:      reaction.GetContent(),
		Username:     reaction.User.GetLogin(),
		CreatedAt:    reaction.GetCreatedAt().Time,
		CachedAt:     time.Now(),
	}
}

// FromGithubPR converte um github.PullRequest para PRData
func FromGithubPR(pr *github.PullRequest, repoOwner, repoName string) *PRData {
	return &PRData{
		RepoOwner:             repoOwner,
		RepoName:              repoName,
		PRNumber:              pr.GetNumber(),
		Title:                 pr.GetTitle(),
		Username:              pr.User.GetLogin(),
		MergedAt:              pr.MergedAt.Time,
		Additions:             pr.GetAdditions(),
		Deletions:             pr.GetDeletions(),
		ChangedFiles:          pr.GetChangedFiles(),
		HasComments:           false, // Será atualizado após verificação
		HasIssueComments:      false, // Será atualizado após verificação
		HasReviewComments:     false, // Será atualizado após verificação
		HasReviews:            false, // Será atualizado após verificação
		HasApprovedReviews:    false, // Será atualizado após verificação
		CommentsChecked:       false, // Inicialmente não verificado
		IssueCommentsChecked:  false, // Inicialmente não verificado
		ReviewCommentsChecked: false, // Inicialmente não verificado
		ReviewsChecked:        false, // Inicialmente não verificado
		CachedAt:              time.Now(),
	}
}

// FromGithubReview converte um github.PullRequestReview para ReviewData
func FromGithubReview(review *github.PullRequestReview, repoOwner, repoName string, prNumber int) *ReviewData {
	var submittedAt time.Time
	if review.SubmittedAt != nil {
		submittedAt = review.SubmittedAt.Time
	}

	return &ReviewData{
		RepoOwner:   repoOwner,
		RepoName:    repoName,
		PRNumber:    prNumber,
		ReviewID:    review.GetID(),
		Username:    review.User.GetLogin(),
		State:       review.GetState(),
		Body:        review.GetBody(),
		SubmittedAt: submittedAt,
		CachedAt:    time.Now(),
	}
}
