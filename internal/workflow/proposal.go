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

type proposalRepoManager interface {
	EnsureBareMirror(context.Context, string, string, string, string) error
	CreateWorktree(context.Context, string, string, string, string) error
	HasChanges(context.Context, string) (bool, error)
	CommitAll(context.Context, string, string) (string, error)
	PushBranch(context.Context, string, string, string, string, string) error
}

type proposalGitHubClient interface {
	GetInstallationToken(context.Context) (string, error)
	FindOpenPullRequestByHead(context.Context, string, string, string, string) (*gh.PullRequest, error)
	CreatePullRequest(context.Context, string, string, string, string, string, string) (*gh.PullRequest, error)
	EnsurePRMonitorLabel(context.Context, string, string, string) error
	AddPRMonitorLabel(context.Context, string, string, int, string) error
}

type proposalOpenSpecClient interface {
	SetWorktreePath(worktreePath string)
	ListChanges(ctx context.Context) ([]string, error)
	GetStatus(ctx context.Context, name string) (*exec.ChangeStatus, error)
	GetApplyInstructions(ctx context.Context, changeName string) (*exec.ApplyInstructions, error)
}

// ProposalWorkflow handles the activation-triggered OpenSpec proposal pull request workflow.
type ProposalWorkflow struct {
	store    *store.Store
	repoMgr  proposalRepoManager
	github   proposalGitHubClient
	openspec proposalOpenSpecClient
	proposal exec.ProposalRunner
	logger   *slog.Logger
}

// NewProposalWorkflow creates a new proposal workflow executor.
func NewProposalWorkflow(store *store.Store, repoMgr proposalRepoManager, github proposalGitHubClient, openspec proposalOpenSpecClient, proposal exec.ProposalRunner, logger *slog.Logger) *ProposalWorkflow {
	if logger == nil {
		logger = slog.Default()
	}
	return &ProposalWorkflow{
		store:    store,
		repoMgr:  repoMgr,
		github:   github,
		openspec: openspec,
		proposal: proposal,
		logger:   logger,
	}
}

// Execute runs the proposal workflow to create or reuse a pull request from an activated issue.
func (w *ProposalWorkflow) Execute(ctx context.Context, runID int64) error {
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
	logger.Info("starting activation proposal workflow", "step", "workflow_start")

	// Scope OpenSpec commands to the run's repository worktree
	w.openspec.SetWorktreePath(run.WorktreePath)

	if err := w.store.UpdateWorkflowRunStatus(ctx, run.ID, "running", ""); err != nil {
		return fmt.Errorf("failed to mark workflow run %d running: %w", run.ID, err)
	}

	stepOrder := 0
	nextStepOrder := func() int {
		stepOrder++
		return stepOrder
	}

	// Reconcile existing binding before doing any work
	existingBinding, err := w.store.GetActiveBinding(ctx, workItem.ID, repository.ID)
	if err != nil {
		return w.failRun(ctx, run.ID, logger, nextStepOrder(), "reconcile_binding", "sqlite", "repo_bindings select", "failed to reconcile existing binding", err)
	}
	if existingBinding != nil {
		logger.Info("reusing existing active binding", "step", "reuse_binding", "branch", existingBinding.BranchName, "change", existingBinding.ChangeName)
		if err := w.recordCompletedStep(ctx, run.ID, logger, nextStepOrder(), "reconcile_binding", "sqlite", "repo_bindings select"); err != nil {
			return err
		}
		if err := w.store.UpdateWorkflowRunStatus(ctx, run.ID, "completed", ""); err != nil {
			return fmt.Errorf("failed to mark workflow run %d completed: %w", run.ID, err)
		}
		logger.Info("activation proposal workflow completed", "step", "workflow_complete", "reused_binding", true)
		return nil
	}
	if err := w.recordCompletedStep(ctx, run.ID, logger, nextStepOrder(), "reconcile_binding", "sqlite", "repo_bindings select"); err != nil {
		return err
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

	proposalResult, err := w.runProposalStep(ctx, run, workItem, repository, logger, nextStepOrder())
	if err != nil {
		return err
	}

	hasChanges, err := w.repoMgr.HasChanges(ctx, run.WorktreePath)
	if err != nil {
		return w.failRun(ctx, run.ID, logger, nextStepOrder(), "detect_changes", "git", "git status --porcelain", "failed to inspect proposal worktree changes", err)
	}
	if !hasChanges {
		return w.failRun(ctx, run.ID, logger, nextStepOrder(), "detect_changes", "git", "git status --porcelain", "proposal execution produced no file changes", repo.ErrNoChanges)
	}
	if err := w.recordCompletedStep(ctx, run.ID, logger, nextStepOrder(), "detect_changes", "git", "git status --porcelain"); err != nil {
		return err
	}

	// Discover the change that opencode created
	changeName, err := w.discoverCreatedChange(ctx, run.ID, logger, nextStepOrder(), proposalResult.ChangesBefore)
	if err != nil {
		return err
	}

	applyInstructions, err := w.openspec.GetApplyInstructions(ctx, changeName)
	if err != nil {
		return w.failRun(ctx, run.ID, logger, nextStepOrder(), "openspec_apply_instructions", "openspec", fmt.Sprintf("openspec instructions apply --change %s --json", changeName), "failed to get apply instructions", err)
	}
	if err := w.recordCompletedStep(ctx, run.ID, logger, nextStepOrder(), "openspec_apply_instructions", "openspec", fmt.Sprintf("openspec instructions apply --change %s --json", changeName)); err != nil {
		return err
	}

	if applyInstructions.State == "blocked" {
		return w.failRun(ctx, run.ID, logger, nextStepOrder(), "check_change_ready", "openspec", "openspec status", "change is blocked: missing required artifacts", fmt.Errorf("change %s is blocked", changeName))
	}

	commitMessage := fmt.Sprintf("docs(openspec): propose %s via heimdall", changeName)
	commitSHA, err := w.commitStep(ctx, run.ID, run.WorktreePath, commitMessage, logger, nextStepOrder())
	if err != nil {
		return err
	}

	binding := &store.RepoBinding{
		WorkItemID:    workItem.ID,
		RepositoryID:  repository.ID,
		BranchName:    run.BranchName,
		ChangeName:    changeName,
		BindingStatus: "pending",
		LastHeadSHA:   commitSHA,
	}
	if err := w.store.SaveRepoBinding(ctx, binding); err != nil {
		return w.failRun(ctx, run.ID, logger, nextStepOrder(), "save_binding", "sqlite", "repo_bindings upsert", "failed to save proposal repository binding", err)
	}
	if err := w.recordCompletedStep(ctx, run.ID, logger, nextStepOrder(), "save_binding", "sqlite", "repo_bindings upsert"); err != nil {
		return err
	}

	if err := w.executeStep(ctx, run.ID, logger, nextStepOrder(), "push_branch", "git", "git push", func() error {
		return w.repoMgr.PushBranch(ctx, run.WorktreePath, repository.Owner, repository.Name, run.BranchName, installationToken)
	}); err != nil {
		return err
	}

	pullRequest, reused, err := w.ensurePullRequest(ctx, repository, workItem, run, binding, proposalResult, logger, nextStepOrder())
	if err != nil {
		return err
	}

	binding.BindingStatus = "active"
	binding.LastHeadSHA = commitSHA
	if err := w.store.SaveRepoBinding(ctx, binding); err != nil {
		return w.failRun(ctx, run.ID, logger, nextStepOrder(), "activate_binding", "sqlite", "repo_bindings upsert", "failed to activate proposal repository binding", err)
	}
	if err := w.recordCompletedStep(ctx, run.ID, logger, nextStepOrder(), "activate_binding", "sqlite", "repo_bindings upsert"); err != nil {
		return err
	}

	if err := w.store.UpdateWorkflowRunStatus(ctx, run.ID, "completed", ""); err != nil {
		return fmt.Errorf("failed to mark workflow run %d completed: %w", run.ID, err)
	}

	logger.Info(
		"activation proposal workflow completed",
		"step", "workflow_complete",
		"pull_request_number", pullRequest.GetNumber(),
		"pull_request_reused", reused,
		"commit_sha", commitSHA,
	)
	return nil
}

func (w *ProposalWorkflow) runProposalStep(ctx context.Context, run *store.WorkflowRun, workItem *store.WorkItem, repository *store.Repository, logger *slog.Logger, stepOrder int) (*exec.ProposalResult, error) {
	logger.Info(
		"prepared issue seed for activation proposal",
		"step", "issue_seeding",
		"issue_title_present", strings.TrimSpace(workItem.Title) != "",
		"issue_description_present", strings.TrimSpace(workItem.Description) != "",
	)

	// Capture changes before running opencode to detect what was created
	changesBefore, err := w.openspec.ListChanges(ctx)
	if err != nil {
		return nil, w.failRun(ctx, run.ID, logger, stepOrder, "list_changes_before", "openspec", "openspec list --json", "failed to list changes before proposal", err)
	}

	logger.Info("running activation proposal opencode step", "step", "run_proposal_prompt", "agent", repository.DefaultSpecWritingAgent)
	result, err := w.proposal.RunProposal(ctx, exec.ProposalRequest{
		WorktreePath: run.WorktreePath,
		IssueKey:     workItem.WorkItemKey,
		IssueTitle:   workItem.Title,
		Description:  workItem.Description,
		Agent:        repository.DefaultSpecWritingAgent,
	})
	if err != nil {
		return nil, w.failRun(ctx, run.ID, logger, stepOrder, "run_proposal_prompt", "opencode", fmt.Sprintf("opencode run --agent %s", repository.DefaultSpecWritingAgent), "failed to execute activation proposal prompt", err)
	}
	if err := w.recordCompletedStep(ctx, run.ID, logger, stepOrder, "run_proposal_prompt", "opencode", fmt.Sprintf("opencode run --agent %s", repository.DefaultSpecWritingAgent)); err != nil {
		return nil, err
	}

	// Store the before-changes list for later discovery
	result.ChangesBefore = changesBefore
	return result, nil
}

func (w *ProposalWorkflow) discoverCreatedChange(ctx context.Context, runID int64, logger *slog.Logger, stepOrder int, changesBefore []string) (string, error) {
	logger.Info("discovering created OpenSpec change", "step", "discover_change")

	changesAfter, err := w.openspec.ListChanges(ctx)
	if err != nil {
		return "", w.failRun(ctx, runID, logger, stepOrder, "discover_change", "openspec", "openspec list --json", "failed to list changes after proposal", err)
	}

	// Find the newly created change by comparing before and after
	beforeSet := make(map[string]bool)
	for _, c := range changesBefore {
		beforeSet[c] = true
	}

	var newChanges []string
	for _, c := range changesAfter {
		if !beforeSet[c] {
			newChanges = append(newChanges, c)
		}
	}

	if len(newChanges) == 0 {
		return "", w.failRun(ctx, runID, logger, stepOrder, "discover_change", "openspec", "openspec list --json", "no new change was created by opencode", fmt.Errorf("no new change detected"))
	}

	if len(newChanges) > 1 {
		logger.Warn("multiple new changes detected, using first", "changes", newChanges)
	}

	changeName := newChanges[0]
	if err := w.recordCompletedStep(ctx, runID, logger, stepOrder, "discover_change", "openspec", fmt.Sprintf("openspec list --json (found: %s)", changeName)); err != nil {
		return "", err
	}

	logger.Info("discovered created change", "change_name", changeName)
	return changeName, nil
}

func (w *ProposalWorkflow) commitStep(ctx context.Context, runID int64, worktreePath, message string, logger *slog.Logger, stepOrder int) (string, error) {
	logger.Info("committing proposal change", "step", "commit_changes")
	commitSHA, err := w.repoMgr.CommitAll(ctx, worktreePath, message)
	if err != nil {
		reason := "failed to commit proposal change"
		if err == repo.ErrNoChanges {
			reason = "proposal execution produced no file changes"
		}
		return "", w.failRun(ctx, runID, logger, stepOrder, "commit_changes", "git", "git add -A && git commit", reason, err)
	}
	if err := w.recordCompletedStep(ctx, runID, logger, stepOrder, "commit_changes", "git", "git add -A && git commit"); err != nil {
		return "", err
	}
	return commitSHA, nil
}

func (w *ProposalWorkflow) ensurePullRequest(ctx context.Context, repository *store.Repository, workItem *store.WorkItem, run *store.WorkflowRun, binding *store.RepoBinding, proposalResult *exec.ProposalResult, logger *slog.Logger, stepOrder int) (*gh.PullRequest, bool, error) {
	logger.Info("reconciling proposal pull request", "step", "ensure_pull_request")
	pullRequest, err := w.github.FindOpenPullRequestByHead(ctx, repository.Owner, repository.Name, run.BranchName, repository.DefaultBranch)
	if err != nil {
		return nil, false, w.failRun(ctx, run.ID, logger, stepOrder, "ensure_pull_request", "github", "pull_requests.list", "failed to reconcile existing proposal pull request", err)
	}

	reused := pullRequest != nil
	if pullRequest == nil {
		pullRequest, err = w.github.CreatePullRequest(ctx, repository.Owner, repository.Name, buildProposalPRTitle(workItem), run.BranchName, repository.DefaultBranch, buildProposalPRBody(workItem, binding.ChangeName, proposalResult))
		if err != nil {
			return nil, false, w.failRun(ctx, run.ID, logger, stepOrder, "ensure_pull_request", "github", "pull_requests.create", "failed to create proposal pull request", err)
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
		return nil, false, w.failRun(ctx, run.ID, logger, stepOrder, "ensure_pull_request", "sqlite", "pull_requests upsert", "failed to save proposal pull request record", err)
	}
	if err := w.recordCompletedStep(ctx, run.ID, logger, stepOrder, "ensure_pull_request", "github", map[bool]string{true: "pull_requests.list", false: "pull_requests.create"}[reused]); err != nil {
		return nil, false, err
	}

	return pullRequest, reused, nil
}

func (w *ProposalWorkflow) executeStep(ctx context.Context, runID int64, logger *slog.Logger, stepOrder int, stepName, executor, command string, fn func() error) error {
	logger.Info("starting workflow step", "step", stepName, "executor", executor)
	if err := fn(); err != nil {
		return w.failRun(ctx, runID, logger, stepOrder, stepName, executor, command, fmt.Sprintf("workflow step %s failed", stepName), err)
	}
	return w.recordCompletedStep(ctx, runID, logger, stepOrder, stepName, executor, command)
}

func (w *ProposalWorkflow) recordCompletedStep(ctx context.Context, runID int64, logger *slog.Logger, stepOrder int, stepName, executor, command string) error {
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

func (w *ProposalWorkflow) failRun(ctx context.Context, runID int64, logger *slog.Logger, stepOrder int, stepName, executor, command, reason string, stepErr error) error {
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
	logger.Error("activation proposal workflow step failed", "step", stepName, "executor", executor, "outcome", status, "reason", reason, "error", stepErr)
	return fmt.Errorf("%s: %w", reason, stepErr)
}

func buildProposalPRTitle(workItem *store.WorkItem) string {
	return fmt.Sprintf("[%s] OpenSpec proposal for %s", workItem.WorkItemKey, workItem.Title)
}

func buildProposalPRBody(workItem *store.WorkItem, changeName string, proposalResult *exec.ProposalResult) string {
	description := strings.TrimSpace(workItem.Description)
	if description == "" {
		description = "No issue description provided."
	}

	summary := "Generated OpenSpec proposal artifacts from the activation seed."
	if proposalResult != nil && strings.TrimSpace(proposalResult.Summary) != "" {
		summary = strings.TrimSpace(proposalResult.Summary)
	}

	return fmt.Sprintf("## Source Issue\n- Key: %s\n- Title: %s\n\n## Description\n> %s\n\n## OpenSpec Change\n- Change: `%s`\n\n## Proposal Summary\n- %s\n", workItem.WorkItemKey, workItem.Title, strings.ReplaceAll(description, "\n", "\n> "), changeName, summary)
}
