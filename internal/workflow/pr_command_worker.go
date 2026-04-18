package workflow

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mngeow/heimdall/internal/store"
)

// PRCommandWorker processes queued PR command jobs.
type PRCommandWorker struct {
	queue    *store.JobQueue
	executor *PRCommandExecutor
	logger   *slog.Logger
}

// NewPRCommandWorker creates a new PR command worker.
func NewPRCommandWorker(queue *store.JobQueue, executor *PRCommandExecutor, logger *slog.Logger) *PRCommandWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &PRCommandWorker{
		queue:    queue,
		executor: executor,
		logger:   logger,
	}
}

// ProcessJob dequeues and executes the next available PR command job.
func (w *PRCommandWorker) ProcessJob(ctx context.Context) error {
	job, err := w.queue.Dequeue(ctx)
	if err != nil {
		return fmt.Errorf("failed to dequeue job: %w", err)
	}
	if job == nil {
		return nil
	}

	logger := w.logger.With("job_id", job.ID, "job_type", job.JobType)
	logger.Info("processing PR command job")

	if err := w.executeJob(ctx, job); err != nil {
		logger.Error("job execution failed", "error", err)
		if failErr := w.queue.Fail(ctx, job.ID, 0); failErr != nil {
			logger.Error("failed to mark job as failed", "error", failErr)
		}
		return nil
	}

	logger.Info("PR command job completed")
	return nil
}

func (w *PRCommandWorker) executeJob(ctx context.Context, job *store.Job) error {
	if job.CommandRequestID == nil {
		return fmt.Errorf("job %d has no command request", job.ID)
	}

	req, err := w.executor.store.GetCommandRequestByID(ctx, *job.CommandRequestID)
	if err != nil {
		if updateErr := w.executor.store.UpdateCommandRequestStatus(ctx, *job.CommandRequestID, "failed"); updateErr != nil {
			w.logger.Error("failed to mark failed command request status", "error", updateErr)
		}
		return fmt.Errorf("failed to load command request %d: %w", *job.CommandRequestID, err)
	}
	if req == nil {
		if updateErr := w.executor.store.UpdateCommandRequestStatus(ctx, *job.CommandRequestID, "failed"); updateErr != nil {
			w.logger.Error("failed to mark failed command request status", "error", updateErr)
		}
		return fmt.Errorf("command request %d not found", *job.CommandRequestID)
	}

	pr, err := w.executor.store.GetPullRequestByID(ctx, req.PullRequestID)
	if err != nil {
		if updateErr := w.executor.store.UpdateCommandRequestStatus(ctx, req.ID, "failed"); updateErr != nil {
			w.logger.Error("failed to mark failed command request status", "error", updateErr)
		}
		return fmt.Errorf("failed to load pull request %d: %w", req.PullRequestID, err)
	}
	if pr == nil {
		if updateErr := w.executor.store.UpdateCommandRequestStatus(ctx, req.ID, "failed"); updateErr != nil {
			w.logger.Error("failed to mark failed command request status", "error", updateErr)
		}
		return fmt.Errorf("pull request %d not found", req.PullRequestID)
	}

	repo, err := w.executor.store.GetRepositoryByID(ctx, pr.RepositoryID)
	if err != nil {
		if updateErr := w.executor.store.UpdateCommandRequestStatus(ctx, req.ID, "failed"); updateErr != nil {
			w.logger.Error("failed to mark failed command request status", "error", updateErr)
		}
		return fmt.Errorf("failed to load repository %d: %w", pr.RepositoryID, err)
	}
	if repo == nil {
		if updateErr := w.executor.store.UpdateCommandRequestStatus(ctx, req.ID, "failed"); updateErr != nil {
			w.logger.Error("failed to mark failed command request status", "error", updateErr)
		}
		return fmt.Errorf("repository %d not found", pr.RepositoryID)
	}

	if err := w.executor.store.UpdateCommandRequestStatus(ctx, req.ID, "running"); err != nil {
		return fmt.Errorf("failed to mark command request %d as running: %w", req.ID, err)
	}

	execReq := ExecutionRequest{
		Kind:       req.CommandName,
		ChangeName: req.ChangeName,
		Agent:      req.RequestedAgent,
		PromptTail: req.PromptTail,
		Alias:      req.Alias,
		RequestID:  req.RequestID,
	}

	var execErr error
	switch job.JobType {
	case "pr_command_status":
		execErr = w.executor.ExecuteStatus(ctx, pr, repo)
	case "pr_command_refine":
		execErr = w.executor.ExecuteRefine(ctx, execReq, pr, repo)
	case "pr_command_apply":
		execErr = w.executor.ExecuteApply(ctx, execReq, pr, repo)
	case "pr_command_opencode":
		execErr = w.executor.ExecuteOpencode(ctx, execReq, pr, repo)
	case "pr_command_approve":
		execErr = w.executor.ExecuteApprove(ctx, execReq, pr, repo)
	default:
		execErr = fmt.Errorf("unknown job type: %s", job.JobType)
	}

	if execErr != nil {
		status := "blocked"
		if strings.Contains(execErr.Error(), "rejected:") || strings.Contains(execErr.Error(), "no active") {
			status = "rejected"
		}
		if err := w.executor.store.UpdateCommandRequestStatus(ctx, req.ID, status); err != nil {
			return fmt.Errorf("execution failed and command request status update also failed: %w", err)
		}
		if failErr := w.queue.Fail(ctx, job.ID, 0); failErr != nil {
			w.logger.Error("failed to mark job as failed", "error", failErr)
		}
		return execErr
	}

	if err := w.executor.store.UpdateCommandRequestStatus(ctx, req.ID, "completed"); err != nil {
		return fmt.Errorf("failed to mark command request %d as completed: %w", req.ID, err)
	}
	if err := w.queue.Complete(ctx, job.ID); err != nil {
		w.logger.Error("failed to mark job as completed", "job_id", job.ID, "error", err)
	}

	return nil
}
