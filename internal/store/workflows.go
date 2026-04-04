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
}

// WorkflowRun represents a workflow execution
type WorkflowRun struct {
	ID               int64
	WorkItemID       int64
	RepositoryID     int64
	TriggerEventID   *int64
	RunType          string
	Status           string
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

func (s *Store) SavePullRequest(ctx context.Context, pr *PullRequest) error {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO pull_requests (repository_id, repo_binding_id, provider, provider_pr_node_id, number, title, base_branch, head_branch, state, url)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(repository_id, number) DO UPDATE SET
		 title = excluded.title,
		 state = excluded.state,
		 url = excluded.url`,
		pr.RepositoryID, pr.RepoBindingID, pr.Provider, pr.ProviderPRNodeID, pr.Number, pr.Title, pr.BaseBranch, pr.HeadBranch, pr.State, pr.URL,
	)
	if err != nil {
		return err
	}

	if pr.ID == 0 {
		id, _ := result.LastInsertId()
		pr.ID = id
	}
	return nil
}

// CommandRequest operations
func (s *Store) GetCommandRequestByDedupeKey(ctx context.Context, dedupeKey string) (*CommandRequest, error) {
	var req CommandRequest
	err := s.db.QueryRowContext(ctx,
		`SELECT id, pull_request_id, comment_node_id, command_name, command_args, requested_agent, actor_login, authorization_status, dedupe_key, workflow_run_id, status
		 FROM command_requests WHERE dedupe_key = ?`,
		dedupeKey,
	).Scan(&req.ID, &req.PullRequestID, &req.CommentNodeID, &req.CommandName, &req.CommandArgs, &req.RequestedAgent, &req.ActorLogin, &req.AuthorizationStatus, &req.DedupeKey, &req.WorkflowRunID, &req.Status)

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
		`INSERT INTO command_requests (pull_request_id, comment_node_id, command_name, command_args, requested_agent, actor_login, authorization_status, dedupe_key, workflow_run_id, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(dedupe_key) DO UPDATE SET
		 status = excluded.status,
		 workflow_run_id = excluded.workflow_run_id`,
		req.PullRequestID, req.CommentNodeID, req.CommandName, req.CommandArgs, req.RequestedAgent, req.ActorLogin, req.AuthorizationStatus, req.DedupeKey, req.WorkflowRunID, req.Status,
	)
	if err != nil {
		return err
	}

	if req.ID == 0 {
		id, _ := result.LastInsertId()
		req.ID = id
	}
	return nil
}

// WorkflowRun operations
func (s *Store) CreateWorkflowRun(ctx context.Context, run *WorkflowRun) error {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO workflow_runs (work_item_id, repository_id, trigger_event_id, run_type, status, change_name, branch_name, worktree_path, requested_by_type, requested_by_login, attempt_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.WorkItemID, run.RepositoryID, run.TriggerEventID, run.RunType, run.Status, run.ChangeName, run.BranchName, run.WorktreePath, run.RequestedByType, run.RequestedByLogin, run.AttemptCount,
	)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	run.ID = id
	return nil
}

func (s *Store) UpdateWorkflowRunStatus(ctx context.Context, runID int64, status string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE workflow_runs SET status = ? WHERE id = ?`,
		status, runID,
	)
	return err
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
