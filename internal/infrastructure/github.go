package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v70/github"
)

type GithubAdapter interface {
	FetchPRsForRepo(owner, name string, startDate, endDate time.Time) ([]*github.PullRequest, error)
	GetPR(ctx context.Context, owner, repo string, prNumber int) (*github.PullRequest, error)
	ListIssueCommentReactions(ctx context.Context, owner, repo string, issueNumber int64) ([]*github.Reaction, error)
	ListPullRequestCommentReactions(ctx context.Context, owner, repo string, commentID int64) ([]*github.Reaction, error)
	ListPRComments(ctx context.Context, owner, repo string, prNumber int) ([]*github.IssueComment, error)
	ListPRReviewComments(ctx context.Context, owner, repo string, prNumber int) ([]*github.PullRequestComment, error)
}

// CacheableGithubAdapter estende GithubAdapter com funcionalidades de cache
type CacheableGithubAdapter interface {
	GithubAdapter
	ClearCache() error
	Close() error
}

type githubAdapter struct {
	client *github.Client
}

func NewGithubClient(token string) GithubAdapter {
	client := github.NewClient(nil).WithAuthToken(token)

	return &githubAdapter{client: client}
}

func (c githubAdapter) FetchPRsForRepo(owner, name string, startDate, endDate time.Time) ([]*github.PullRequest, error) {
	ctx := context.Background()

	opts := &github.PullRequestListOptions{
		State:     "closed",
		Sort:      "created",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var repoPRs []*github.PullRequest
	shouldStop := false

	for !shouldStop {
		prs, resp, err := c.client.PullRequests.List(ctx, owner, name, opts)
		if err != nil {
			return nil, fmt.Errorf("erro ao buscar PRs: %v", err)
		}

		for _, pr := range prs {
			// fmt.Println("PR = ", pr.GetTitle())
			// fmt.Println("PR mergedAt ", pr.MergedAt)
			// fmt.Println("PR updatedAt ", pr.UpdatedAt)
			if pr.MergedAt == nil {
				continue // Pula PRs não mergeados
			}

			mergedAt := pr.MergedAt.Time
			// if mergedAt.Before(startDate) {
			// 	// Se chegamos a PRs anteriores ao período, paramos de buscar mais páginas
			// 	shouldStop = true
			// 	break
			// }

			if mergedAt.After(startDate) && mergedAt.Before(endDate.Add(24*time.Hour)) {
				repoPRs = append(repoPRs, pr)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	fmt.Printf("    ✅ %d PRs encontrados em %s/%s\n", len(repoPRs), owner, name)
	return repoPRs, nil
}

// GetPR busca um PR específico pelo número
func (c githubAdapter) GetPR(ctx context.Context, owner, repo string, prNumber int) (*github.PullRequest, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar PR #%d: %v", prNumber, err)
	}
	return pr, nil
}

func (c githubAdapter) ListIssueCommentReactions(ctx context.Context, owner, repo string, issueNumber int64) ([]*github.Reaction, error) {
	reactions, _, err := c.client.Reactions.ListIssueCommentReactions(ctx, owner, repo, issueNumber, nil)
	return reactions, err
}

func (c githubAdapter) ListPullRequestCommentReactions(ctx context.Context, owner, repo string, commentID int64) ([]*github.Reaction, error) {
	reactions, _, err := c.client.Reactions.ListPullRequestCommentReactions(ctx, owner, repo, commentID, nil)
	return reactions, err
}

func (c githubAdapter) ListPRComments(ctx context.Context, owner, repo string, prNumber int) ([]*github.IssueComment, error) {
	var comments []*github.IssueComment
	reviewOpts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	for {
		comm, resp, err := c.client.Issues.ListComments(ctx, owner, repo, prNumber, reviewOpts)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comm...)
		if resp.NextPage == 0 {
			break
		}
		reviewOpts.Page = resp.NextPage
	}
	return comments, nil
}

func (c githubAdapter) ListPRReviewComments(ctx context.Context, owner, repo string, prNumber int) ([]*github.PullRequestComment, error) {
	var comments []*github.PullRequestComment
	reviewOpts := &github.PullRequestListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	for {
		comm, resp, err := c.client.PullRequests.ListComments(ctx, owner, repo, prNumber, reviewOpts)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comm...)
		if resp.NextPage == 0 {
			break
		}
		reviewOpts.Page = resp.NextPage
	}
	return comments, nil
}
