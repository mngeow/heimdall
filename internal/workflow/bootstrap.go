package workflow

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	gh "github.com/google/go-github/v57/github"
	"github.com/mngeow/heimdall/internal/exec"
	"github.com/mngeow/heimdall/internal/repo"
	"github.com/mngeow/heimdall/internal/store"
)

type bootstrapRepoManager interface {
	EnsureBareMirror(context.Context, string, string, string, string) error
	CreateWorktree(context.Context, string, string, string, string) error
	HasChanges(context.Context, string) (bool, error)
	CommitAll(context.Context, string, string) (string, error)
	PushBranch(context.Context, string, string, string, string, string) error
}

type bootstrapGitHubClient interface {
	GetInstallationToken(context.Context) (string, error)
	FindOpenPullRequestByHead(context.Context, string, string, string, string) (*gh.PullRequest, error)
	CreatePullRequest(context.Context, string, string, string, string, string, string) (*gh.PullRequest, error)
	EnsurePRMonitorLabel(context.Context, string, string, string) error
	AddPRMonitorLabel(context.Context, string, string, int, string) error
}

// BootstrapWorkflow handles the activation-triggered bootstrap pull request workflow.
type BootstrapWorkflow struct {
	store     *store.Store
	repoMgr   bootstrapRepoManager
	github    bootstrapGitHubClient
	bootstrap exec.BootstrapRunner
	logger    *slog.Logger
}

// NewBootstrapWorkflow creates a new bootstrap workflow executor.
func NewBootstrapWorkflow(store *store.Store, repoMgr bootstrapRepoManager, github bootstrapGitHubClient, bootstrap exec.BootstrapRunner, logger *slog.Logger) *BootstrapWorkflow {
	if logger == nil {
		logger = slog.Default()
	}
	return &BootstrapWorkflow{
		store:     store,
		repoMgr:   repoMgr,
		github:    github,
		bootstrap: bootstrap,
		logger:    logger,
	}
}

// Execute runs the bootstrap workflow to create or reuse a pull request from an activated issue.
func (w *BootstrapWorkflow) Execute(ctx context.Context, runID int64) error {
	run, err := w.store.GetWorkflowRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to load workflow run: %w", err)
	}
	if run == nil {
		return fmt.Errorf("workflow run %d not found", runID)
	}

	workItem, err := w.store.GetWorkItemByID(ctx, run.WorkItemID)
	if err != nil {
		return fmt.Errorf("failed to load work item for run %d: %w", runID, err)
	}
	if workItem == nil {
		return fmt.Errorf("work item %d not found for run %d", run.WorkItemID, runID)
	}

	repository, err := w.store.GetRepositoryByID(ctx, run.RepositoryID)
	if err != nil {
		return fmt.Errorf("failed to load repository for run %d: %w", runID, err)
	}
	if repository == nil {
		return fmt.Errorf("repository %d not found for run %d", run.RepositoryID, runID)
	}

	logger := w.logger.With(
		"workflow_run_id", run.ID,
		"work_item_key", workItem.WorkItemKey,
		"repository", repository.RepoRef,
		"branch", run.BranchName,
		"worktree_path", run.WorktreePath,
	)
	logger.Info("starting activation bootstrap workflow", "step", "workflow_start")

	if err := w.store.UpdateWorkflowRunStatus(ctx, run.ID, "running", ""); err != nil {
		return fmt.Errorf("failed to mark workflow run %d running: %w", run.ID, err)
	}

	stepOrder := 0
	nextStepOrder := func() int {
		stepOrder++
		return stepOrder
	}

	installationToken, err := w.github.GetInstallationToken(ctx)
	if err != nil {
		return w.failRun(ctx, run.ID, logger, nextStepOrder(), "get_installation_token", "github", "GetInstallationToken", "failed to mint GitHub installation token", err)
	}

	if err := w.executeStep(ctx, run.ID, logger, nextStepOrder(), "ensure_mirror", "git", "git clone --mirror|fetch --prune", func() error {
		return w.repoMgr.EnsureBareMirror(ctx, repository.LocalMirrorPath, repository.Owner, repository.Name, installationToken)
	}); err != nil {
		return err
	}

	if err := w.executeStep(ctx, run.ID, logger, nextStepOrder(), "create_worktree", "git", "git worktree add", func() error {
		return w.repoMgr.CreateWorktree(ctx, repository.LocalMirrorPath, repository.DefaultBranch, run.BranchName, run.WorktreePath)
	}); err != nil {
		return err
	}

	bootstrapResult, err := w.runBootstrapStep(ctx, run, workItem, logger, nextStepOrder())
	if err != nil {
		return err
	}

	hasChanges, err := w.repoMgr.HasChanges(ctx, run.WorktreePath)
	if err != nil {
		return w.failRun(ctx, run.ID, logger, nextStepOrder(), "detect_changes", "git", "git status --porcelain", "failed to inspect bootstrap worktree changes", err)
	}
	if !hasChanges {
		return w.failRun(ctx, run.ID, logger, nextStepOrder(), "detect_changes", "git", "git status --porcelain", "bootstrap execution produced no file changes", repo.ErrNoChanges)
	}
	if err := w.recordCompletedStep(ctx, run.ID, logger, nextStepOrder(), "detect_changes", "git", "git status --porcelain"); err != nil {
		return err
	}

	commitMessage := fmt.Sprintf("docs: bootstrap %s via heimdall", strings.ToLower(workItem.WorkItemKey))
	commitSHA, err := w.commitStep(ctx, run.ID, run.WorktreePath, commitMessage, logger, nextStepOrder())
	if err != nil {
		return err
	}

	binding := &store.RepoBinding{
		WorkItemID:    workItem.ID,
		RepositoryID:  repository.ID,
		BranchName:    run.BranchName,
		ChangeName:    run.ChangeName,
		BindingStatus: "pending",
		LastHeadSHA:   commitSHA,
	}
	if err := w.store.SaveRepoBinding(ctx, binding); err != nil {
		return w.failRun(ctx, run.ID, logger, nextStepOrder(), "save_binding", "sqlite", "repo_bindings upsert", "failed to save bootstrap repository binding", err)
	}
	if err := w.recordCompletedStep(ctx, run.ID, logger, nextStepOrder(), "save_binding", "sqlite", "repo_bindings upsert"); err != nil {
		return err
	}

	if err := w.executeStep(ctx, run.ID, logger, nextStepOrder(), "push_branch", "git", "git push", func() error {
		return w.repoMgr.PushBranch(ctx, run.WorktreePath, repository.Owner, repository.Name, run.BranchName, installationToken)
	}); err != nil {
		return err
	}

	pullRequest, reused, err := w.ensurePullRequest(ctx, repository, workItem, run, binding, bootstrapResult, logger, nextStepOrder())
	if err != nil {
		return err
	}

	binding.BindingStatus = "active"
	binding.LastHeadSHA = commitSHA
	if err := w.store.SaveRepoBinding(ctx, binding); err != nil {
		return w.failRun(ctx, run.ID, logger, nextStepOrder(), "activate_binding", "sqlite", "repo_bindings upsert", "failed to activate bootstrap repository binding", err)
	}
	if err := w.recordCompletedStep(ctx, run.ID, logger, nextStepOrder(), "activate_binding", "sqlite", "repo_bindings upsert"); err != nil {
		return err
	}

	if err := w.store.UpdateWorkflowRunStatus(ctx, run.ID, "completed", ""); err != nil {
		return fmt.Errorf("failed to mark workflow run %d completed: %w", run.ID, err)
	}

	logger.Info(
		"activation bootstrap workflow completed",
		"step", "workflow_complete",
		"pull_request_number", pullRequest.GetNumber(),
		"pull_request_reused", reused,
		"commit_sha", commitSHA,
	)
	return nil
}

func (w *BootstrapWorkflow) runBootstrapStep(ctx context.Context, run *store.WorkflowRun, workItem *store.WorkItem, logger *slog.Logger, stepOrder int) (*exec.BootstrapResult, error) {
	logger.Info(
		"prepared issue seed for activation bootstrap",
		"step", "issue_seeding",
		"issue_title_present", strings.TrimSpace(workItem.Title) != "",
		"issue_description_present", strings.TrimSpace(workItem.Description) != "",
	)
	logger.Info("running activation bootstrap opencode step", "step", "run_bootstrap_prompt")
	result, err := w.bootstrap.RunBootstrap(ctx, exec.BootstrapRequest{
		WorktreePath: run.WorktreePath,
		IssueKey:     workItem.WorkItemKey,
		IssueTitle:   workItem.Title,
		Description:  workItem.Description,
		BranchName:   run.BranchName,
	})
	if err != nil {
		return nil, w.failRun(ctx, run.ID, logger, stepOrder, "run_bootstrap_prompt", "opencode", "opencode run --agent general --model openai/gpt-5.4", "failed to execute activation bootstrap prompt", err)
	}
	if err := w.recordCompletedStep(ctx, run.ID, logger, stepOrder, "run_bootstrap_prompt", "opencode", "opencode run --agent general --model openai/gpt-5.4"); err != nil {
		return nil, err
	}
	return result, nil
}

func (w *BootstrapWorkflow) commitStep(ctx context.Context, runID int64, worktreePath, message string, logger *slog.Logger, stepOrder int) (string, error) {
	logger.Info("committing bootstrap change", "step", "commit_changes")
	commitSHA, err := w.repoMgr.CommitAll(ctx, worktreePath, message)
	if err != nil {
		reason := "failed to commit bootstrap change"
		if err == repo.ErrNoChanges {
			reason = "bootstrap execution produced no file changes"
		}
		return "", w.failRun(ctx, runID, logger, stepOrder, "commit_changes", "git", "git add -A && git commit", reason, err)
	}
	if err := w.recordCompletedStep(ctx, runID, logger, stepOrder, "commit_changes", "git", "git add -A && git commit"); err != nil {
		return "", err
	}
	return commitSHA, nil
}

func (w *BootstrapWorkflow) ensurePullRequest(ctx context.Context, repository *store.Repository, workItem *store.WorkItem, run *store.WorkflowRun, binding *store.RepoBinding, bootstrapResult *exec.BootstrapResult, logger *slog.Logger, stepOrder int) (*gh.PullRequest, bool, error) {
	logger.Info("reconciling bootstrap pull request", "step", "ensure_pull_request")
	pullRequest, err := w.github.FindOpenPullRequestByHead(ctx, repository.Owner, repository.Name, run.BranchName, repository.DefaultBranch)
	if err != nil {
		return nil, false, w.failRun(ctx, run.ID, logger, stepOrder, "ensure_pull_request", "github", "pull_requests.list", "failed to reconcile existing bootstrap pull request", err)
	}

	reused := pullRequest != nil
	if pullRequest == nil {
		pullRequest, err = w.github.CreatePullRequest(ctx, repository.Owner, repository.Name, buildBootstrapPRTitle(workItem), run.BranchName, repository.DefaultBranch, buildBootstrapPRBody(workItem, bootstrapResult))
		if err != nil {
			return nil, false, w.failRun(ctx, run.ID, logger, stepOrder, "ensure_pull_request", "github", "pull_requests.create", "failed to create bootstrap pull request", err)
		}
	}

	if repository.PRMonitorLabel != "" {
		if err := w.github.EnsurePRMonitorLabel(ctx, repository.Owner, repository.Name, repository.PRMonitorLabel); err != nil {
			return nil, false, w.failRun(ctx, run.ID, logger, stepOrder, "ensure_pull_request", "github", "issues.create_label", "failed to reconcile PR monitor label", err)
		}
		if err := w.github.AddPRMonitorLabel(ctx, repository.Owner, repository.Name, pullRequest.GetNumber(), repository.PRMonitorLabel); err != nil {
			return nil, false, w.failRun(ctx, run.ID, logger, stepOrder, "ensure_pull_request", "github", "issues.add_labels", "failed to apply PR monitor label", err)
		}
	}

	prRecord := &store.PullRequest{
		RepositoryID:     repository.ID,
		RepoBindingID:    &binding.ID,
		Provider:         "github",
		ProviderPRNodeID: pullRequest.GetNodeID(),
		Number:           pullRequest.GetNumber(),
		Title:            pullRequest.GetTitle(),
		BaseBranch:       repository.DefaultBranch,
		HeadBranch:       run.BranchName,
		State:            strings.ToLower(pullRequest.GetState()),
		URL:              pullRequest.GetHTMLURL(),
	}
	if prRecord.State == "" {
		prRecord.State = "open"
	}
	if err := w.store.SavePullRequest(ctx, prRecord); err != nil {
		return nil, false, w.failRun(ctx, run.ID, logger, stepOrder, "ensure_pull_request", "sqlite", "pull_requests upsert", "failed to save bootstrap pull request record", err)
	}
	if err := w.recordCompletedStep(ctx, run.ID, logger, stepOrder, "ensure_pull_request", "github", map[bool]string{true: "pull_requests.list", false: "pull_requests.create"}[reused]); err != nil {
		return nil, false, err
	}

	return pullRequest, reused, nil
}

func (w *BootstrapWorkflow) executeStep(ctx context.Context, runID int64, logger *slog.Logger, stepOrder int, stepName, executor, command string, fn func() error) error {
	logger.Info("starting workflow step", "step", stepName, "executor", executor)
	if err := fn(); err != nil {
		return w.failRun(ctx, runID, logger, stepOrder, stepName, executor, command, fmt.Sprintf("workflow step %s failed", stepName), err)
	}
	return w.recordCompletedStep(ctx, runID, logger, stepOrder, stepName, executor, command)
}

func (w *BootstrapWorkflow) recordCompletedStep(ctx context.Context, runID int64, logger *slog.Logger, stepOrder int, stepName, executor, command string) error {
	if err := w.store.CreateWorkflowStep(ctx, &store.WorkflowStep{
		WorkflowRunID: runID,
		StepName:      stepName,
		StepOrder:     stepOrder,
		Status:        "completed",
		Executor:      executor,
		CommandLine:   command,
	}); err != nil {
		return fmt.Errorf("failed to record workflow step %s: %w", stepName, err)
	}
	logger.Info("completed workflow step", "step", stepName, "executor", executor, "outcome", "completed")
	return nil
}

func (w *BootstrapWorkflow) failRun(ctx context.Context, runID int64, logger *slog.Logger, stepOrder int, stepName, executor, command, reason string, stepErr error) error {
	status := "failed"
	if stepErr == repo.ErrNoChanges {
		status = "blocked"
	}
	_ = w.store.CreateWorkflowStep(ctx, &store.WorkflowStep{
		WorkflowRunID: runID,
		StepName:      stepName,
		StepOrder:     stepOrder,
		Status:        status,
		Executor:      executor,
		CommandLine:   command,
	})
	_ = w.store.UpdateWorkflowRunStatus(ctx, runID, status, reason)
	logger.Error("activation bootstrap workflow step failed", "step", stepName, "executor", executor, "outcome", status, "reason", reason, "error", stepErr)
	return fmt.Errorf("%s: %w", reason, stepErr)
}

func buildBootstrapPRTitle(workItem *store.WorkItem) string {
	return fmt.Sprintf("[%s] Bootstrap PR for %s", workItem.WorkItemKey, workItem.Title)
}

func buildBootstrapPRBody(workItem *store.WorkItem, bootstrapResult *exec.BootstrapResult) string {
	description := strings.TrimSpace(workItem.Description)
	if description == "" {
		description = "No issue description provided."
	}

	summary := "Created a temporary bootstrap file change from the activation seed."
	if bootstrapResult != nil && strings.TrimSpace(bootstrapResult.Summary) != "" {
		summary = strings.TrimSpace(bootstrapResult.Summary)
	}

	return fmt.Sprintf("## Source Issue\n- Key: %s\n- Title: %s\n\n## Description\n> %s\n\n## Bootstrap Summary\n- %s\n", workItem.WorkItemKey, workItem.Title, strings.ReplaceAll(description, "\n", "\n> "), summary)
}
