package store

import (
	"context"
	"database/sql"
	"time"
)

// PullRequest represents a GitHub pull request
type PullRequest struct {
	ID               int64
	RepositoryID     int64
	RepoBindingID    *int64
	Provider         string
	ProviderPRNodeID string
	Number           int
	Title            string
	BaseBranch       string
	HeadBranch       string
	State            string
	URL              string
}

// CommandRequest represents a PR comment command
type CommandRequest struct {
	ID                  int64
	PullRequestID       int64
	CommentNodeID       string
	CommandName         string
	CommandArgs         string
	RequestedAgent      string
	ActorLogin          string
	AuthorizationStatus string
	DedupeKey           string
	WorkflowRunID       *int64
	Status              string
	ChangeName          string
	Alias               string
	PromptTail          string
	RequestID           string
}

// WorkflowRun represents a workflow execution
type WorkflowRun struct {
	ID               int64
	WorkItemID       int64
	RepositoryID     int64
	TriggerEventID   *int64
	RunType          string
	Status           string
	StatusReason     string
	ChangeName       string
	BranchName       string
	WorktreePath     string
	RequestedByType  string
	RequestedByLogin string
	AttemptCount     int
}

// WorkflowStep represents a step within a workflow run
type WorkflowStep struct {
	ID            int64
	WorkflowRunID int64
	StepName      string
	StepOrder     int
	Status        string
	Executor      string
	CommandLine   string
	ToolVersion   string
	AttemptCount  int
}

// PendingPermissionRequest represents a blocked opencode permission request.
type PendingPermissionRequest struct {
	ID               int64
	RequestID        string
	SessionID        string
	CommandRequestID int64
	PullRequestID    int64
	RepositoryID     int64
	Status           string
	CreatedAt        time.Time
	ResolvedAt       *time.Time
}

// Job represents a queued async job
type Job struct {
	ID               int64
	WorkflowRunID    *int64
	CommandRequestID *int64
	JobType          string
	LockKey          string
	Status           string
	Priority         int
	RunAfter         time.Time
	AttemptCount     int
	MaxAttempts      int
}

// PullRequest operations
func (s *Store) GetPullRequestByNumber(ctx context.Context, repositoryID int64, number int) (*PullRequest, error) {
	var pr PullRequest
	err := s.db.QueryRowContext(ctx,
		`SELECT id, repository_id, repo_binding_id, provider, provider_pr_node_id, number, title, base_branch, head_branch, state, url
		 FROM pull_requests WHERE repository_id = ? AND number = ?`,
		repositoryID, number,
	).Scan(&pr.ID, &pr.RepositoryID, &pr.RepoBindingID, &pr.Provider, &pr.ProviderPRNodeID, &pr.Number, &pr.Title, &pr.BaseBranch, &pr.HeadBranch, &pr.State, &pr.URL)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

func (s *Store) GetPullRequestByID(ctx context.Context, pullRequestID int64) (*PullRequest, error) {
	var pr PullRequest
	err := s.db.QueryRowContext(ctx,
		`SELECT id, repository_id, repo_binding_id, provider, provider_pr_node_id, number, title, base_branch, head_branch, state, url
		 FROM pull_requests WHERE id = ?`,
		pullRequestID,
	).Scan(&pr.ID, &pr.RepositoryID, &pr.RepoBindingID, &pr.Provider, &pr.ProviderPRNodeID, &pr.Number, &pr.Title, &pr.BaseBranch, &pr.HeadBranch, &pr.State, &pr.URL)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

func (s *Store) SavePullRequest(ctx context.Context, pr *PullRequest) error {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO pull_requests (repository_id, repo_binding_id, provider, provider_pr_node_id, number, title, base_branch, head_branch, state, url)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(repository_id, number) DO UPDATE SET
		 repo_binding_id = excluded.repo_binding_id,
		 provider_pr_node_id = excluded.provider_pr_node_id,
		 title = excluded.title,
		 base_branch = excluded.base_branch,
		 head_branch = excluded.head_branch,
		 state = excluded.state,
		 url = excluded.url`,
		pr.RepositoryID, pr.RepoBindingID, pr.Provider, pr.ProviderPRNodeID, pr.Number, pr.Title, pr.BaseBranch, pr.HeadBranch, pr.State, pr.URL,
	)
	if err != nil {
		return err
	}

	if pr.ID == 0 {
		if id, _ := result.LastInsertId(); id > 0 {
			pr.ID = id
		} else {
			existing, err := s.GetPullRequestByNumber(ctx, pr.RepositoryID, pr.Number)
			if err != nil {
				return err
			}
			if existing != nil {
				pr.ID = existing.ID
			}
		}
	}
	return nil
}

// CommandRequest operations
func (s *Store) GetCommandRequestByDedupeKey(ctx context.Context, dedupeKey string) (*CommandRequest, error) {
	var req CommandRequest
	err := s.db.QueryRowContext(ctx,
		`SELECT id, pull_request_id, comment_node_id, command_name, command_args, requested_agent, actor_login, authorization_status, dedupe_key, workflow_run_id, status, change_name, alias, prompt_tail, request_id
		 FROM command_requests WHERE dedupe_key = ?`,
		dedupeKey,
	).Scan(&req.ID, &req.PullRequestID, &req.CommentNodeID, &req.CommandName, &req.CommandArgs, &req.RequestedAgent, &req.ActorLogin, &req.AuthorizationStatus, &req.DedupeKey, &req.WorkflowRunID, &req.Status, &req.ChangeName, &req.Alias, &req.PromptTail, &req.RequestID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (s *Store) GetCommandRequestByID(ctx context.Context, requestID int64) (*CommandRequest, error) {
	var req CommandRequest
	err := s.db.QueryRowContext(ctx,
		`SELECT id, pull_request_id, comment_node_id, command_name, command_args, requested_agent, actor_login, authorization_status, dedupe_key, workflow_run_id, status, change_name, alias, prompt_tail, request_id
		 FROM command_requests WHERE id = ?`,
		requestID,
	).Scan(&req.ID, &req.PullRequestID, &req.CommentNodeID, &req.CommandName, &req.CommandArgs, &req.RequestedAgent, &req.ActorLogin, &req.AuthorizationStatus, &req.DedupeKey, &req.WorkflowRunID, &req.Status, &req.ChangeName, &req.Alias, &req.PromptTail, &req.RequestID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (s *Store) SaveCommandRequest(ctx context.Context, req *CommandRequest) error {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO command_requests (pull_request_id, comment_node_id, command_name, command_args, requested_agent, actor_login, authorization_status, dedupe_key, workflow_run_id, status, change_name, alias, prompt_tail, request_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(dedupe_key) DO UPDATE SET
		 status = excluded.status,
		 workflow_run_id = excluded.workflow_run_id`,
		req.PullRequestID, req.CommentNodeID, req.CommandName, req.CommandArgs, req.RequestedAgent, req.ActorLogin, req.AuthorizationStatus, req.DedupeKey, req.WorkflowRunID, req.Status, req.ChangeName, req.Alias, req.PromptTail, req.RequestID,
	)
	if err != nil {
		return err
	}

	if req.ID == 0 {
		if id, _ := result.LastInsertId(); id > 0 {
			req.ID = id
		} else {
			existing, err := s.GetCommandRequestByDedupeKey(ctx, req.DedupeKey)
			if err != nil {
				return err
			}
			if existing != nil {
				req.ID = existing.ID
			}
		}
	}
	return nil
}

func (s *Store) UpdateCommandRequestStatus(ctx context.Context, requestID int64, status string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE command_requests SET status = ? WHERE id = ?`,
		status, requestID,
	)
	return err
}

// WorkflowRun operations
func (s *Store) CreateWorkflowRun(ctx context.Context, run *WorkflowRun) error {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO workflow_runs (work_item_id, repository_id, trigger_event_id, run_type, status, status_reason, change_name, branch_name, worktree_path, requested_by_type, requested_by_login, attempt_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.WorkItemID, run.RepositoryID, run.TriggerEventID, run.RunType, run.Status, run.StatusReason, run.ChangeName, run.BranchName, run.WorktreePath, run.RequestedByType, run.RequestedByLogin, run.AttemptCount,
	)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	run.ID = id
	return nil
}

func (s *Store) GetWorkflowRun(ctx context.Context, runID int64) (*WorkflowRun, error) {
	var run WorkflowRun
	err := s.db.QueryRowContext(ctx,
		`SELECT id, work_item_id, repository_id, trigger_event_id, run_type, status, COALESCE(status_reason, ''), change_name, branch_name, worktree_path, requested_by_type, requested_by_login, attempt_count
		 FROM workflow_runs WHERE id = ?`,
		runID,
	).Scan(&run.ID, &run.WorkItemID, &run.RepositoryID, &run.TriggerEventID, &run.RunType, &run.Status, &run.StatusReason, &run.ChangeName, &run.BranchName, &run.WorktreePath, &run.RequestedByType, &run.RequestedByLogin, &run.AttemptCount)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &run, nil
}

func (s *Store) UpdateWorkflowRunStatus(ctx context.Context, runID int64, status, reason string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE workflow_runs SET status = ?, status_reason = ? WHERE id = ?`,
		status, reason, runID,
	)
	return err
}

func (s *Store) GetPullRequestByBindingID(ctx context.Context, bindingID int64) (*PullRequest, error) {
	var pr PullRequest
	err := s.db.QueryRowContext(ctx,
		`SELECT id, repository_id, repo_binding_id, provider, provider_pr_node_id, number, title, base_branch, head_branch, state, url
		 FROM pull_requests WHERE repo_binding_id = ? ORDER BY id DESC LIMIT 1`,
		bindingID,
	).Scan(&pr.ID, &pr.RepositoryID, &pr.RepoBindingID, &pr.Provider, &pr.ProviderPRNodeID, &pr.Number, &pr.Title, &pr.BaseBranch, &pr.HeadBranch, &pr.State, &pr.URL)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &pr, nil
}

// WorkflowStep operations
func (s *Store) CreateWorkflowStep(ctx context.Context, step *WorkflowStep) error {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO workflow_steps (workflow_run_id, step_name, step_order, status, executor, command_line, tool_version, attempt_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		step.WorkflowRunID, step.StepName, step.StepOrder, step.Status, step.Executor, step.CommandLine, step.ToolVersion, step.AttemptCount,
	)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	step.ID = id
	return nil
}

// GetActiveBindingsByPullRequestID returns active repo bindings linked to a pull request.
func (s *Store) GetActiveBindingsByPullRequestID(ctx context.Context, pullRequestID int64) ([]*RepoBinding, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT b.id, b.work_item_id, b.repository_id, b.branch_name, b.change_name, b.binding_status, b.last_head_sha, b.created_at, b.updated_at
		 FROM repo_bindings b
		 JOIN pull_requests pr ON pr.head_branch = b.branch_name
		 WHERE pr.id = ? AND b.binding_status = 'active'
		 ORDER BY b.change_name ASC`,
		pullRequestID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bindings []*RepoBinding
	for rows.Next() {
		var b RepoBinding
		if err := rows.Scan(&b.ID, &b.WorkItemID, &b.RepositoryID, &b.BranchName, &b.ChangeName, &b.BindingStatus, &b.LastHeadSHA, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		bindings = append(bindings, &b)
	}
	return bindings, rows.Err()
}

// PendingPermissionRequest operations

func (s *Store) CreatePendingPermissionRequest(ctx context.Context, req *PendingPermissionRequest) error {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO pending_permission_requests (request_id, session_id, command_request_id, pull_request_id, repository_id, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		req.RequestID, req.SessionID, req.CommandRequestID, req.PullRequestID, req.RepositoryID, req.Status, time.Now(),
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	req.ID = id
	return nil
}

func (s *Store) GetPendingPermissionRequestByID(ctx context.Context, requestID string) (*PendingPermissionRequest, error) {
	var req PendingPermissionRequest
	var resolvedAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT id, request_id, session_id, command_request_id, pull_request_id, repository_id, status, created_at, resolved_at
		 FROM pending_permission_requests WHERE request_id = ?`,
		requestID,
	).Scan(&req.ID, &req.RequestID, &req.SessionID, &req.CommandRequestID, &req.PullRequestID, &req.RepositoryID, &req.Status, &req.CreatedAt, &resolvedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if resolvedAt.Valid {
		req.ResolvedAt = &resolvedAt.Time
	}
	return &req, nil
}

func (s *Store) ResolvePendingPermissionRequest(ctx context.Context, requestID, status string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE pending_permission_requests SET status = ?, resolved_at = ? WHERE request_id = ?`,
		status, time.Now(), requestID,
	)
	return err
}
