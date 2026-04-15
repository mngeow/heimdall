package dashboard

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mngeow/heimdall/internal/store"
)

// OverviewSnapshot holds summary counts for the dashboard overview.
type OverviewSnapshot struct {
	TotalWorkItems      int64
	ActiveWorkItems     int64
	ActivePullRequests  int64
	RunningWorkflowRuns int64
	FailedWorkflowRuns  int64
	QueuedJobs          int64
	RetryableJobs       int64
}

// WorkItemQueueRow holds a single row for the work-item queue screen.
type WorkItemQueueRow struct {
	WorkItemID        int64
	WorkItemKey       string
	Title             string
	StateName         string
	LifecycleBucket   string
	Team              string
	LastSeenUpdatedAt *time.Time
	RepoRef           string
	BranchName        string
	ChangeName        string
	BindingStatus     string
	LatestRunStatus   string
}

// PullRequestRow holds a single row for the active pull-request list.
type PullRequestRow struct {
	PullRequestID int64
	RepositoryID  int64
	RepoRef       string
	Number        int
	Title         string
	State         string
	HeadBranch    string
	WorkItemKey   string
	WorkItemTitle string
	ChangeName    string
	URL           string
}

// CommandActivity holds a Heimdall-tracked command/activity entry.
type CommandActivity struct {
	ID                  int64
	CommentNodeID       string
	CommandName         string
	CommandArgs         string
	RequestedAgent      string
	ActorLogin          string
	AuthorizationStatus string
	Status              string
	WorkflowRunID       *int64
	OccurredAt          *time.Time
}

// WorkflowActivity holds a workflow run/step entry.
type WorkflowActivity struct {
	WorkflowRunID int64
	RunType       string
	Status        string
	StepName      string
	StepStatus    string
	Executor      string
}

// AuditActivity holds an audit event entry.
type AuditActivity struct {
	ID         int64
	EventType  string
	Severity   string
	ActorType  string
	ActorLogin string
	AgentName  string
	CommitSHA  string
	Summary    string
	OccurredAt *time.Time
}

// PRDetail holds the full pull-request detail view data.
type PRDetail struct {
	PullRequest PullRequestRow
	Binding     store.RepoBinding
	WorkItem    store.WorkItem
	Commands    []CommandActivity
	Workflows   []WorkflowActivity
	Audits      []AuditActivity
}

// Queries provides dashboard-focused read queries over the SQLite store.
type Queries struct {
	db *sql.DB
}

// NewQueries creates a new dashboard query service.
func NewQueries(db *sql.DB) *Queries {
	return &Queries{db: db}
}

// Overview returns summary counts for the dashboard.
func (q *Queries) Overview(ctx context.Context) (*OverviewSnapshot, error) {
	var snap OverviewSnapshot

	if err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM work_items`).Scan(&snap.TotalWorkItems); err != nil {
		return nil, fmt.Errorf("overview work_items count failed: %w", err)
	}
	if err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM work_items WHERE lifecycle_bucket = 'active'`).Scan(&snap.ActiveWorkItems); err != nil {
		return nil, fmt.Errorf("overview active work_items count failed: %w", err)
	}
	if err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pull_requests WHERE state = 'open' AND repo_binding_id IS NOT NULL`).Scan(&snap.ActivePullRequests); err != nil {
		return nil, fmt.Errorf("overview active pr count failed: %w", err)
	}
	if err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM workflow_runs WHERE status = 'running'`).Scan(&snap.RunningWorkflowRuns); err != nil {
		return nil, fmt.Errorf("overview running runs count failed: %w", err)
	}
	if err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM workflow_runs WHERE status = 'failed'`).Scan(&snap.FailedWorkflowRuns); err != nil {
		return nil, fmt.Errorf("overview failed runs count failed: %w", err)
	}
	if err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM jobs WHERE status = 'queued'`).Scan(&snap.QueuedJobs); err != nil {
		return nil, fmt.Errorf("overview queued jobs count failed: %w", err)
	}
	if err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM jobs WHERE status = 'queued' AND attempt_count > 0`).Scan(&snap.RetryableJobs); err != nil {
		return nil, fmt.Errorf("overview retryable jobs count failed: %w", err)
	}

	return &snap, nil
}

// WorkItemQueue returns a filterable list of work items with binding/run context.
func (q *Queries) WorkItemQueue(ctx context.Context, filterStatus, filterBucket string) ([]WorkItemQueueRow, error) {
	query := `
SELECT
	wi.id,
	wi.work_item_key,
	wi.title,
	wi.state_name,
	wi.lifecycle_bucket,
	wi.team_key,
	wi.last_seen_updated_at,
	COALESCE(r.repo_ref, '') AS repo_ref,
	COALESCE(rb.branch_name, '') AS branch_name,
	COALESCE(rb.change_name, '') AS change_name,
	COALESCE(rb.binding_status, '') AS binding_status,
	COALESCE(wr.status, '') AS latest_run_status
FROM work_items wi
LEFT JOIN repo_bindings rb ON rb.work_item_id = wi.id AND rb.binding_status = 'active'
LEFT JOIN repositories r ON r.id = rb.repository_id
LEFT JOIN (
	SELECT work_item_id, status FROM workflow_runs
	WHERE id IN (
		SELECT MAX(id) FROM workflow_runs GROUP BY work_item_id
	)
) wr ON wr.work_item_id = wi.id
WHERE 1=1
`
	args := []any{}
	if filterStatus != "" {
		query += ` AND wi.state_name = ?`
		args = append(args, filterStatus)
	}
	if filterBucket != "" {
		query += ` AND wi.lifecycle_bucket = ?`
		args = append(args, filterBucket)
	}
	query += ` ORDER BY wi.last_seen_updated_at IS NULL, wi.last_seen_updated_at DESC, wi.id DESC`

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("work item queue query failed: %w", err)
	}
	defer rows.Close()

	var results []WorkItemQueueRow
	for rows.Next() {
		var row WorkItemQueueRow
		if err := rows.Scan(
			&row.WorkItemID, &row.WorkItemKey, &row.Title, &row.StateName, &row.LifecycleBucket,
			&row.Team, &row.LastSeenUpdatedAt, &row.RepoRef, &row.BranchName, &row.ChangeName,
			&row.BindingStatus, &row.LatestRunStatus,
		); err != nil {
			return nil, fmt.Errorf("work item queue scan failed: %w", err)
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// ActivePullRequests returns active Heimdall-managed pull requests.
func (q *Queries) ActivePullRequests(ctx context.Context) ([]PullRequestRow, error) {
	query := `
SELECT
	pr.id,
	pr.repository_id,
	r.repo_ref,
	pr.number,
	pr.title,
	pr.state,
	pr.head_branch,
	wi.work_item_key,
	wi.title,
	COALESCE(rb.change_name, '') AS change_name,
	pr.url
FROM pull_requests pr
JOIN repositories r ON r.id = pr.repository_id
LEFT JOIN repo_bindings rb ON rb.id = pr.repo_binding_id
LEFT JOIN work_items wi ON wi.id = rb.work_item_id
WHERE pr.state = 'open'
  AND pr.repo_binding_id IS NOT NULL
ORDER BY pr.id DESC
`
	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("active pull requests query failed: %w", err)
	}
	defer rows.Close()

	var results []PullRequestRow
	for rows.Next() {
		var row PullRequestRow
		if err := rows.Scan(
			&row.PullRequestID, &row.RepositoryID, &row.RepoRef, &row.Number, &row.Title,
			&row.State, &row.HeadBranch, &row.WorkItemKey, &row.WorkItemTitle, &row.ChangeName, &row.URL,
		); err != nil {
			return nil, fmt.Errorf("active pull requests scan failed: %w", err)
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// PullRequestDetail returns the full detail view for a single PR.
func (q *Queries) PullRequestDetail(ctx context.Context, prID int64) (*PRDetail, error) {
	prQuery := `
SELECT
	pr.id,
	pr.repository_id,
	r.repo_ref,
	pr.number,
	pr.title,
	pr.state,
	pr.head_branch,
	wi.work_item_key,
	wi.title,
	COALESCE(rb.change_name, '') AS change_name,
	pr.url
FROM pull_requests pr
JOIN repositories r ON r.id = pr.repository_id
LEFT JOIN repo_bindings rb ON rb.id = pr.repo_binding_id
LEFT JOIN work_items wi ON wi.id = rb.work_item_id
WHERE pr.id = ?
`
	var row PullRequestRow
	if err := q.db.QueryRowContext(ctx, prQuery, prID).Scan(
		&row.PullRequestID, &row.RepositoryID, &row.RepoRef, &row.Number, &row.Title,
		&row.State, &row.HeadBranch, &row.WorkItemKey, &row.WorkItemTitle, &row.ChangeName, &row.URL,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("pr detail query failed: %w", err)
	}

	detail := &PRDetail{PullRequest: row}

	// Binding
	if err := q.db.QueryRowContext(ctx,
		`SELECT id, work_item_id, repository_id, branch_name, change_name, binding_status, last_head_sha, created_at, updated_at FROM repo_bindings
		 WHERE id = (SELECT repo_binding_id FROM pull_requests WHERE id = ?)`,
		prID,
	).Scan(
		&detail.Binding.ID, &detail.Binding.WorkItemID, &detail.Binding.RepositoryID,
		&detail.Binding.BranchName, &detail.Binding.ChangeName, &detail.Binding.BindingStatus,
		&detail.Binding.LastHeadSHA, &detail.Binding.CreatedAt, &detail.Binding.UpdatedAt,
	); err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("pr detail binding query failed: %w", err)
	}

	// WorkItem
	if detail.Binding.WorkItemID != 0 {
		var project sql.NullString
		var labelsJSON string
		if err := q.db.QueryRowContext(ctx,
			`SELECT id, provider, provider_work_item_id, work_item_key, title, description, state_name, lifecycle_bucket, project_name, team_key, labels_json, last_seen_updated_at FROM work_items WHERE id = ?`,
			detail.Binding.WorkItemID,
		).Scan(
			&detail.WorkItem.ID, &detail.WorkItem.Provider, &detail.WorkItem.ProviderWorkItemID, &detail.WorkItem.WorkItemKey,
			&detail.WorkItem.Title, &detail.WorkItem.Description, &detail.WorkItem.StateName, &detail.WorkItem.LifecycleBucket,
			&project, &detail.WorkItem.Team, &labelsJSON, &detail.WorkItem.LastSeenUpdatedAt,
		); err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("pr detail work item query failed: %w", err)
		}
		if project.Valid {
			detail.WorkItem.Project = project.String
		}
	}

	// Commands
	cmdRows, err := q.db.QueryContext(ctx,
		`SELECT id, comment_node_id, command_name, command_args, requested_agent, actor_login, authorization_status, status, workflow_run_id
		 FROM command_requests WHERE pull_request_id = ? ORDER BY id DESC LIMIT 50`,
		prID,
	)
	if err != nil {
		return nil, fmt.Errorf("pr detail commands query failed: %w", err)
	}
	defer cmdRows.Close()
	for cmdRows.Next() {
		var ca CommandActivity
		if err := cmdRows.Scan(&ca.ID, &ca.CommentNodeID, &ca.CommandName, &ca.CommandArgs, &ca.RequestedAgent, &ca.ActorLogin, &ca.AuthorizationStatus, &ca.Status, &ca.WorkflowRunID); err != nil {
			return nil, fmt.Errorf("pr detail commands scan failed: %w", err)
		}
		detail.Commands = append(detail.Commands, ca)
	}

	// Workflow activity (runs + latest step)
	wfRows, err := q.db.QueryContext(ctx, `
SELECT
	wr.id,
	wr.run_type,
	wr.status,
	COALESCE(ws.step_name, ''),
	COALESCE(ws.status, ''),
	COALESCE(ws.executor, '')
FROM workflow_runs wr
LEFT JOIN workflow_steps ws ON ws.workflow_run_id = wr.id
WHERE wr.id IN (
	SELECT workflow_run_id FROM command_requests WHERE pull_request_id = ?
	UNION
	SELECT id FROM workflow_runs WHERE work_item_id = ?
)
ORDER BY wr.id DESC, ws.step_order ASC
LIMIT 50`, prID, detail.Binding.WorkItemID)
	if err != nil {
		return nil, fmt.Errorf("pr detail workflows query failed: %w", err)
	}
	defer wfRows.Close()
	for wfRows.Next() {
		var wa WorkflowActivity
		if err := wfRows.Scan(&wa.WorkflowRunID, &wa.RunType, &wa.Status, &wa.StepName, &wa.StepStatus, &wa.Executor); err != nil {
			return nil, fmt.Errorf("pr detail workflows scan failed: %w", err)
		}
		detail.Workflows = append(detail.Workflows, wa)
	}

	// Audit events
	auditRows, err := q.db.QueryContext(ctx,
		`SELECT id, event_type, severity, actor_type, actor_login, agent_name, commit_sha, summary, occurred_at
		 FROM audit_events
		 WHERE workflow_run_id IN (SELECT workflow_run_id FROM command_requests WHERE pull_request_id = ?)
		    OR command_request_id IN (SELECT id FROM command_requests WHERE pull_request_id = ?)
		 ORDER BY occurred_at DESC LIMIT 50`,
		prID, prID,
	)
	if err != nil {
		return nil, fmt.Errorf("pr detail audits query failed: %w", err)
	}
	defer auditRows.Close()
	for auditRows.Next() {
		var aa AuditActivity
		var occurred sql.NullTime
		if err := auditRows.Scan(&aa.ID, &aa.EventType, &aa.Severity, &aa.ActorType, &aa.ActorLogin, &aa.AgentName, &aa.CommitSHA, &aa.Summary, &occurred); err != nil {
			return nil, fmt.Errorf("pr detail audits scan failed: %w", err)
		}
		if occurred.Valid {
			aa.OccurredAt = &occurred.Time
		}
		detail.Audits = append(detail.Audits, aa)
	}

	return detail, nil
}

// DistinctWorkItemStatuses returns distinct state names for filter UI.
func (q *Queries) DistinctWorkItemStatuses(ctx context.Context) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT DISTINCT state_name FROM work_items ORDER BY state_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vals []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	return vals, rows.Err()
}

// DistinctWorkItemBuckets returns distinct lifecycle buckets for filter UI.
func (q *Queries) DistinctWorkItemBuckets(ctx context.Context) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT DISTINCT lifecycle_bucket FROM work_items ORDER BY lifecycle_bucket`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vals []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	return vals, rows.Err()
}
