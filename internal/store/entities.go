package store

import (
	"context"
	"database/sql"
	"time"
)

// WorkItem represents a normalized work item
type WorkItem struct {
	ID                 int64
	Provider           string
	ProviderWorkItemID string
	WorkItemKey        string
	Title              string
	Description        string
	StateName          string
	LifecycleBucket    string
	Project            string
	Team               string
	Labels             []string
	RepositoryRef      string
	LastSeenUpdatedAt  *time.Time
}

// WorkItemEvent represents a work item transition event
type WorkItemEvent struct {
	ID              int64
	WorkItemID      int64
	Provider        string
	ProviderEventID string
	EventType       string
	EventVersion    string
	IdempotencyKey  string
	OccurredAt      time.Time
	DetectedAt      time.Time
}

// Repository represents a managed repository
type Repository struct {
	ID              int64
	Provider        string
	RepoRef         string
	Owner           string
	Name            string
	DefaultBranch   string
	BranchPrefix    string
	LocalMirrorPath string
	IsActive        bool
}

// RepoBinding represents the binding between a work item and repository
type RepoBinding struct {
	ID            int64
	WorkItemID    int64
	RepositoryID  int64
	BranchName    string
	ChangeName    string
	BindingStatus string
	LastHeadSHA   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// WorkItem operations
func (s *Store) GetWorkItemByKey(ctx context.Context, provider, key string) (*WorkItem, error) {
	var item WorkItem
	err := s.db.QueryRowContext(ctx,
		`SELECT id, provider, provider_work_item_id, work_item_key, title, state_name, lifecycle_bucket, team_key, last_seen_updated_at
		 FROM work_items WHERE provider = ? AND work_item_key = ?`,
		provider, key,
	).Scan(&item.ID, &item.Provider, &item.ProviderWorkItemID, &item.WorkItemKey, &item.Title, &item.StateName, &item.LifecycleBucket, &item.Team, &item.LastSeenUpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (s *Store) SaveWorkItem(ctx context.Context, item *WorkItem) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO work_items (provider, provider_work_item_id, work_item_key, title, state_name, lifecycle_bucket, team_key, last_seen_updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(provider, work_item_key) DO UPDATE SET
		 title = excluded.title,
		 state_name = excluded.state_name,
		 lifecycle_bucket = excluded.lifecycle_bucket,
		 last_seen_updated_at = excluded.last_seen_updated_at`,
		item.Provider, item.ProviderWorkItemID, item.WorkItemKey, item.Title, item.StateName, item.LifecycleBucket, item.Team, time.Now(),
	)

	return err
}

// WorkItemEvent operations
func (s *Store) SaveWorkItemEvent(ctx context.Context, event *WorkItemEvent) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO work_item_events (work_item_id, provider, provider_event_id, event_type, event_version, idempotency_key, occurred_at, detected_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(idempotency_key) DO NOTHING`,
		event.WorkItemID, event.Provider, event.ProviderEventID, event.EventType, event.EventVersion, event.IdempotencyKey, event.OccurredAt, event.DetectedAt,
	)
	return err
}

// Repository operations
func (s *Store) GetRepositoryByRef(ctx context.Context, repoRef string) (*Repository, error) {
	var repo Repository
	err := s.db.QueryRowContext(ctx,
		`SELECT id, provider, repo_ref, owner, name, default_branch, branch_prefix, local_mirror_path, is_active
		 FROM repositories WHERE repo_ref = ?`,
		repoRef,
	).Scan(&repo.ID, &repo.Provider, &repo.RepoRef, &repo.Owner, &repo.Name, &repo.DefaultBranch, &repo.BranchPrefix, &repo.LocalMirrorPath, &repo.IsActive)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (s *Store) SaveRepository(ctx context.Context, repo *Repository) error {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO repositories (provider, repo_ref, owner, name, default_branch, branch_prefix, local_mirror_path, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(repo_ref) DO UPDATE SET
		 default_branch = excluded.default_branch,
		 local_mirror_path = excluded.local_mirror_path,
		 is_active = excluded.is_active`,
		repo.Provider, repo.RepoRef, repo.Owner, repo.Name, repo.DefaultBranch, repo.BranchPrefix, repo.LocalMirrorPath, repo.IsActive,
	)
	if err != nil {
		return err
	}

	if repo.ID == 0 {
		if id, _ := result.LastInsertId(); id > 0 {
			repo.ID = id
		} else {
			existing, err := s.GetRepositoryByRef(ctx, repo.RepoRef)
			if err != nil {
				return err
			}
			if existing != nil {
				repo.ID = existing.ID
			}
		}
	}
	return nil
}

// RepoBinding operations
func (s *Store) GetActiveBinding(ctx context.Context, workItemID, repositoryID int64) (*RepoBinding, error) {
	var binding RepoBinding
	err := s.db.QueryRowContext(ctx,
		`SELECT id, work_item_id, repository_id, branch_name, change_name, binding_status, last_head_sha, created_at, updated_at
		 FROM repo_bindings WHERE work_item_id = ? AND repository_id = ? AND binding_status = 'active'`,
		workItemID, repositoryID,
	).Scan(&binding.ID, &binding.WorkItemID, &binding.RepositoryID, &binding.BranchName, &binding.ChangeName, &binding.BindingStatus, &binding.LastHeadSHA, &binding.CreatedAt, &binding.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

func (s *Store) SaveRepoBinding(ctx context.Context, binding *RepoBinding) error {
	now := time.Now()
	binding.UpdatedAt = now
	if binding.CreatedAt.IsZero() {
		binding.CreatedAt = now
	}

	result, err := s.db.ExecContext(ctx,
		`INSERT INTO repo_bindings (work_item_id, repository_id, branch_name, change_name, binding_status, last_head_sha, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(work_item_id, repository_id) DO UPDATE SET
		 branch_name = excluded.branch_name,
		 change_name = excluded.change_name,
		 binding_status = excluded.binding_status,
		 last_head_sha = excluded.last_head_sha,
		 updated_at = excluded.updated_at`,
		binding.WorkItemID, binding.RepositoryID, binding.BranchName, binding.ChangeName, binding.BindingStatus, binding.LastHeadSHA, binding.CreatedAt, binding.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if binding.ID == 0 {
		id, _ := result.LastInsertId()
		binding.ID = id
	}
	return nil
}
