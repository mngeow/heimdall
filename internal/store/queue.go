package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// JobQueue manages the async job queue
type JobQueue struct {
	store *Store
}

// NewJobQueue creates a new job queue
func NewJobQueue(store *Store) *JobQueue {
	return &JobQueue{store: store}
}

// Enqueue adds a job to the queue
func (q *JobQueue) Enqueue(ctx context.Context, job *Job) error {
	if job.RunAfter.IsZero() {
		job.RunAfter = time.Now()
	}
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 10
	}
	if job.Priority == 0 {
		job.Priority = 100
	}

	result, err := q.store.db.ExecContext(ctx,
		`INSERT INTO jobs (workflow_run_id, command_request_id, job_type, lock_key, status, priority, run_after, attempt_count, max_attempts)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.WorkflowRunID, job.CommandRequestID, job.JobType, job.LockKey, job.Status, job.Priority, job.RunAfter, job.AttemptCount, job.MaxAttempts,
	)
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	id, _ := result.LastInsertId()
	job.ID = id
	return nil
}

// Dequeue retrieves the next available job, respecting lock keys
func (q *JobQueue) Dequeue(ctx context.Context) (*Job, error) {
	// Start a transaction to ensure atomicity
	tx, err := q.store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Find the next available job that doesn't conflict with running jobs by lock key
	var job Job
	err = tx.QueryRowContext(ctx,
		`SELECT j.id, j.workflow_run_id, j.command_request_id, j.job_type, j.lock_key, j.status, j.priority, j.run_after, j.attempt_count, j.max_attempts
		 FROM jobs j
		 WHERE j.status = 'queued' 
		   AND j.run_after <= ?
		   AND j.attempt_count < j.max_attempts
		   AND NOT EXISTS (
			 SELECT 1 FROM jobs 
			 WHERE lock_key = j.lock_key 
			   AND status = 'running'
			   AND id != j.id
		   )
		 ORDER BY j.priority ASC, j.run_after ASC
		 LIMIT 1`,
		time.Now(),
	).Scan(&job.ID, &job.WorkflowRunID, &job.CommandRequestID, &job.JobType, &job.LockKey, &job.Status, &job.Priority, &job.RunAfter, &job.AttemptCount, &job.MaxAttempts)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job: %w", err)
	}

	// Mark as running
	_, err = tx.ExecContext(ctx,
		`UPDATE jobs SET status = 'running' WHERE id = ?`,
		job.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mark job as running: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	job.Status = "running"
	return &job, nil
}

// Complete marks a job as completed
func (q *JobQueue) Complete(ctx context.Context, jobID int64) error {
	_, err := q.store.db.ExecContext(ctx,
		`UPDATE jobs SET status = 'completed' WHERE id = ?`,
		jobID,
	)
	return err
}

// Fail marks a job as failed and schedules a retry if attempts remain
func (q *JobQueue) Fail(ctx context.Context, jobID int64, retryDelay time.Duration) error {
	_, err := q.store.db.ExecContext(ctx,
		`UPDATE jobs 
		 SET status = 'queued', 
		     attempt_count = attempt_count + 1,
		     run_after = ?
		 WHERE id = ? AND attempt_count < max_attempts`,
		time.Now().Add(retryDelay), jobID,
	)
	if err != nil {
		return err
	}

	// If no rows updated, max attempts exceeded - mark as dead
	_, err = q.store.db.ExecContext(ctx,
		`UPDATE jobs SET status = 'dead' WHERE id = ? AND attempt_count >= max_attempts`,
		jobID,
	)
	return err
}

// CreateIssueLockKey creates a lock key for issue-scoped operations
func CreateIssueLockKey(provider, workItemKey string) string {
	return fmt.Sprintf("issue:%s:%s", provider, workItemKey)
}

// CreateRepoLockKey creates a lock key for repo-scoped operations
func CreateRepoLockKey(repoRef string) string {
	return fmt.Sprintf("repo:%s", repoRef)
}

// CreatePullRequestLockKey creates a lock key for pull-request-scoped operations.
func CreatePullRequestLockKey(pullRequestID int64) string {
	return fmt.Sprintf("pr:%d", pullRequestID)
}
