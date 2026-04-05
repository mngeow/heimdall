package store

import (
	"context"
	"time"
)

const githubPollingProvider = "github"

// ListActiveRepositories returns repositories that Symphony should manage.
func (s *Store) ListActiveRepositories(ctx context.Context) ([]Repository, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, provider, repo_ref, owner, name, default_branch, branch_prefix, pr_monitor_label, local_mirror_path, is_active
		 FROM repositories WHERE is_active = 1 ORDER BY repo_ref ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repository
	for rows.Next() {
		var repo Repository
		if err := rows.Scan(&repo.ID, &repo.Provider, &repo.RepoRef, &repo.Owner, &repo.Name, &repo.DefaultBranch, &repo.BranchPrefix, &repo.PRMonitorLabel, &repo.LocalMirrorPath, &repo.IsActive); err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}

	return repos, rows.Err()
}

// ListManagedPullRequests returns the open Symphony-managed pull requests for a repository.
func (s *Store) ListManagedPullRequests(ctx context.Context, repositoryID int64) ([]PullRequest, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, repository_id, repo_binding_id, provider, provider_pr_node_id, number, title, base_branch, head_branch, state, url
		 FROM pull_requests
		 WHERE repository_id = ?
		   AND state = 'open'
		   AND repo_binding_id IS NOT NULL
		 ORDER BY number ASC`,
		repositoryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []PullRequest
	for rows.Next() {
		var pr PullRequest
		if err := rows.Scan(&pr.ID, &pr.RepositoryID, &pr.RepoBindingID, &pr.Provider, &pr.ProviderPRNodeID, &pr.Number, &pr.Title, &pr.BaseBranch, &pr.HeadBranch, &pr.State, &pr.URL); err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}

	return prs, rows.Err()
}

// GetGitHubPollCheckpoint returns the last successful GitHub poll time for a repo scope.
func (s *Store) GetGitHubPollCheckpoint(ctx context.Context, repoRef string) (*time.Time, error) {
	cursor, err := s.GetProviderCursor(ctx, githubPollingProvider, repoRef)
	if err != nil || cursor == nil || cursor.CursorValue == "" {
		return nil, err
	}

	checkpoint, err := time.Parse(time.RFC3339Nano, cursor.CursorValue)
	if err != nil {
		return nil, err
	}

	return &checkpoint, nil
}

// SetGitHubPollCheckpoint records the last successful GitHub poll time for a repo scope.
func (s *Store) SetGitHubPollCheckpoint(ctx context.Context, repoRef string, checkpoint time.Time) error {
	return s.SetProviderCursor(ctx, &ProviderCursor{
		Provider:    githubPollingProvider,
		ScopeKey:    repoRef,
		CursorValue: checkpoint.UTC().Format(time.RFC3339Nano),
		CursorKind:  "timestamp",
	})
}
