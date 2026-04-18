package workflow

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mngeow/heimdall/internal/exec"
	"github.com/mngeow/heimdall/internal/store"
)

// PRCommandExecutor handles execution of PR comment commands.
type PRCommandExecutor struct {
	store    *store.Store
	repoMgr  prCommandRepoManager
	github   prCommandGitHubClient
	openspec prCommandOpenSpecClient
	exec     prCommandExecClient
	logger   *slog.Logger
}

// ExecutionRequest represents a parsed PR command ready for execution.
type ExecutionRequest struct {
	Kind              string
	ChangeName        string
	Agent             string
	PromptTail        string
	Alias             string
	PermissionProfile string
	WorktreePath      string
	RequestID         string
}

type prCommandRepoManager interface {
	EnsureBareMirror(context.Context, string, string, string, string) error
	CreateWorktree(context.Context, string, string, string, string) error
	HasChanges(context.Context, string) (bool, error)
	CommitAll(context.Context, string, string) (string, error)
	PushBranch(context.Context, string, string, string, string, string) error
}

type prCommandGitHubClient interface {
	GetInstallationToken(context.Context) (string, error)
	CreateComment(context.Context, string, string, int, string) error
}

type prCommandOpenSpecClient interface {
	SetWorktreePath(string)
	ListChanges(ctx context.Context) ([]string, error)
	GetStatus(ctx context.Context, name string) (*exec.ChangeStatus, error)
}

type prCommandExecClient interface {
	SetWorktreePath(string)
	RunRefine(ctx context.Context, agent, changeName, prompt string) (*exec.ExecutionOutcome, error)
	RunApply(ctx context.Context, agent, changeName, prompt string) (*exec.ExecutionOutcome, error)
	RunGeneric(ctx context.Context, agent, command, prompt string) error
	ReplyPermission(ctx context.Context, requestID, sessionID string) error
	ResumeSession(ctx context.Context, sessionID string) (*exec.ExecutionOutcome, error)
}

// NewPRCommandExecutor creates a new PR command executor.
func NewPRCommandExecutor(store *store.Store, repoMgr prCommandRepoManager, github prCommandGitHubClient, openspec prCommandOpenSpecClient, exec prCommandExecClient, logger *slog.Logger) *PRCommandExecutor {
	if logger == nil {
		logger = slog.Default()
	}
	return &PRCommandExecutor{
		store:    store,
		repoMgr:  repoMgr,
		github:   github,
		openspec: openspec,
		exec:     exec,
		logger:   logger,
	}
}

// ResolveChange resolves the target OpenSpec change for a PR command.
// If changeName is empty, it infers from the PR when exactly one active change exists.
func (e *PRCommandExecutor) ResolveChange(ctx context.Context, prID int64, changeName string) (string, error) {
	if changeName != "" {
		return changeName, nil
	}

	// Get active changes for this PR via repo_bindings
	bindings, err := e.store.GetActiveBindingsByPullRequestID(ctx, prID)
	if err != nil {
		return "", fmt.Errorf("failed to list active bindings for PR %d: %w", prID, err)
	}
	if len(bindings) == 0 {
		return "", fmt.Errorf("no active OpenSpec change found for PR %d", prID)
	}
	if len(bindings) > 1 {
		return "", fmt.Errorf("ambiguous: PR %d has %d active changes; specify change-name explicitly", prID, len(bindings))
	}
	return bindings[0].ChangeName, nil
}

// ResolvePendingRequest resolves a pending permission request by ID scoped to the same PR.
func (e *PRCommandExecutor) ResolvePendingRequest(ctx context.Context, requestID string, prID int64) (*store.PendingPermissionRequest, error) {
	req, err := e.store.GetPendingPermissionRequestByID(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup permission request %s: %w", requestID, err)
	}
	if req == nil {
		return nil, fmt.Errorf("permission request %s not found", requestID)
	}
	if req.PullRequestID != prID {
		return nil, fmt.Errorf("permission request %s does not belong to this pull request", requestID)
	}
	if req.Status != "pending" {
		return nil, fmt.Errorf("permission request %s is already %s", requestID, req.Status)
	}
	return req, nil
}

// ExecuteStatus posts the current PR-bound change status.
func (e *PRCommandExecutor) ExecuteStatus(ctx context.Context, pr *store.PullRequest, repo *store.Repository) error {
	bindings, err := e.store.GetActiveBindingsByPullRequestID(ctx, pr.ID)
	if err != nil {
		return fmt.Errorf("failed to list active bindings for PR %d: %w", pr.ID, err)
	}

	var body strings.Builder
	body.WriteString("Heimdall status\n")

	switch len(bindings) {
	case 0:
		body.WriteString("\nNo active OpenSpec change is currently bound to this pull request.")
	case 1:
		body.WriteString("\n")
		body.WriteString("- Change: `")
		body.WriteString(bindings[0].ChangeName)
		body.WriteString("`\n")
		body.WriteString("- Branch: `")
		body.WriteString(pr.HeadBranch)
		body.WriteString("`\n")
		body.WriteString("- Pull request state: `")
		body.WriteString(pr.State)
		body.WriteString("`")
	default:
		changes := make([]string, 0, len(bindings))
		for _, binding := range bindings {
			changes = append(changes, "`"+binding.ChangeName+"`")
		}
		body.WriteString("\n")
		body.WriteString("- Active changes: ")
		body.WriteString(strings.Join(changes, ", "))
		body.WriteString("\n")
		body.WriteString("- Branch: `")
		body.WriteString(pr.HeadBranch)
		body.WriteString("`\n")
		body.WriteString("- Agent-driven commands must specify `change-name` on this pull request.")
	}

	return e.commentResult(ctx, pr, repo, body.String())
}

// ExecuteRefine runs a refine command.
func (e *PRCommandExecutor) ExecuteRefine(ctx context.Context, req ExecutionRequest, pr *store.PullRequest, repo *store.Repository) error {
	logger := e.logger.With("command", "refine", "change", req.ChangeName, "agent", req.Agent, "pr", pr.Number)
	logger.Info("starting refine execution")

	changeName, err := e.ResolveChange(ctx, pr.ID, req.ChangeName)
	if err != nil {
		return fmt.Errorf("refine rejected: %w", err)
	}

	worktreePath, err := e.prepareWorktree(ctx, pr, repo)
	if err != nil {
		return fmt.Errorf("failed to prepare worktree for refine: %w", err)
	}

	if err := e.validateChangeExists(ctx, changeName, worktreePath); err != nil {
		return fmt.Errorf("refine rejected: %w", err)
	}

	e.exec.SetWorktreePath(worktreePath)

	outcome, err := e.exec.RunRefine(ctx, req.Agent, changeName, req.PromptTail)
	if err != nil {
		return fmt.Errorf("refine failed: %w", err)
	}

	switch outcome.Status {
	case "success":
		return e.commitAndPushOutcome(ctx, pr, repo, "refine", changeName, req.Agent, outcome)
	case "needs_input":
		return e.commentResult(ctx, pr, repo, "Refine blocked: needs clarification input. Retry with a more specific prompt.")
	case "needs_permission":
		return e.handleBlockedPermission(ctx, req, pr, repo, outcome)
	default:
		return fmt.Errorf("refine failed: %s", outcome.Summary)
	}
}

// ExecuteApply runs an apply command.
func (e *PRCommandExecutor) ExecuteApply(ctx context.Context, req ExecutionRequest, pr *store.PullRequest, repo *store.Repository) error {
	logger := e.logger.With("command", "apply", "change", req.ChangeName, "agent", req.Agent, "pr", pr.Number)
	logger.Info("starting apply execution")

	changeName, err := e.ResolveChange(ctx, pr.ID, req.ChangeName)
	if err != nil {
		return fmt.Errorf("apply rejected: %w", err)
	}

	worktreePath, err := e.prepareWorktree(ctx, pr, repo)
	if err != nil {
		return fmt.Errorf("failed to prepare worktree for apply: %w", err)
	}

	if err := e.validateChangeExists(ctx, changeName, worktreePath); err != nil {
		return fmt.Errorf("apply rejected: %w", err)
	}

	e.exec.SetWorktreePath(worktreePath)

	outcome, err := e.exec.RunApply(ctx, req.Agent, changeName, req.PromptTail)
	if err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	switch outcome.Status {
	case "success":
		return e.commitAndPushOutcome(ctx, pr, repo, "apply", changeName, req.Agent, outcome)
	case "needs_input":
		return e.commentResult(ctx, pr, repo, "Apply blocked: needs clarification input. Retry with a more specific prompt.")
	case "needs_permission":
		return e.handleBlockedPermission(ctx, req, pr, repo, outcome)
	default:
		return fmt.Errorf("apply failed: %s", outcome.Summary)
	}
}

// ExecuteOpencode runs a generic opencode command.
func (e *PRCommandExecutor) ExecuteOpencode(ctx context.Context, req ExecutionRequest, pr *store.PullRequest, repo *store.Repository) error {
	logger := e.logger.With("command", "opencode", "alias", req.Alias, "change", req.ChangeName, "agent", req.Agent, "pr", pr.Number)
	logger.Info("starting opencode execution")

	changeName, err := e.ResolveChange(ctx, pr.ID, req.ChangeName)
	if err != nil {
		return fmt.Errorf("opencode rejected: %w", err)
	}

	worktreePath, err := e.prepareWorktree(ctx, pr, repo)
	if err != nil {
		return fmt.Errorf("failed to prepare worktree for opencode: %w", err)
	}

	if err := e.validateChangeExists(ctx, changeName, worktreePath); err != nil {
		return fmt.Errorf("opencode rejected: %w", err)
	}

	e.exec.SetWorktreePath(worktreePath)

	if err := e.exec.RunGeneric(ctx, req.Agent, req.Alias, req.PromptTail); err != nil {
		return fmt.Errorf("opencode command %s failed: %w", req.Alias, err)
	}

	return e.commitAndPushOutcome(ctx, pr, repo, "opencode", changeName, req.Agent, &exec.ExecutionOutcome{Status: "success", Summary: "opencode completed"})
}

// ExecuteApprove approves a pending permission request and resumes execution.
func (e *PRCommandExecutor) ExecuteApprove(ctx context.Context, req ExecutionRequest, pr *store.PullRequest, repo *store.Repository) error {
	logger := e.logger.With("command", "approve", "request_id", req.RequestID, "pr", pr.Number)
	logger.Info("starting approval execution")

	pending, err := e.ResolvePendingRequest(ctx, req.RequestID, pr.ID)
	if err != nil {
		return fmt.Errorf("approval rejected: %w", err)
	}

	if err := e.exec.ReplyPermission(ctx, pending.RequestID, pending.SessionID); err != nil {
		return fmt.Errorf("approval reply failed: %w", err)
	}

	// Observe the resumed session for its real terminal outcome.
	resumedOutcome, err := e.exec.ResumeSession(ctx, pending.SessionID)
	if err != nil {
		return fmt.Errorf("approval reply sent but resumed session failed: %w", err)
	}

	if err := e.store.ResolvePendingPermissionRequest(ctx, req.RequestID, "approved"); err != nil {
		return fmt.Errorf("failed to resolve permission request: %w", err)
	}

	msg := fmt.Sprintf("Permission request `%s` approved. Resumed outcome: %s.", req.RequestID, resumedOutcome.Summary)
	return e.commentResult(ctx, pr, repo, msg)
}

func (e *PRCommandExecutor) prepareWorktree(ctx context.Context, pr *store.PullRequest, repo *store.Repository) (string, error) {
	token, err := e.github.GetInstallationToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get installation token: %w", err)
	}
	if err := e.repoMgr.EnsureBareMirror(ctx, repo.LocalMirrorPath, repo.Owner, repo.Name, token); err != nil {
		return "", fmt.Errorf("failed to ensure bare mirror: %w", err)
	}
	worktreePath := GenerateWorktreePath(repo.LocalMirrorPath, pr.HeadBranch)
	if err := e.repoMgr.CreateWorktree(ctx, repo.LocalMirrorPath, repo.DefaultBranch, pr.HeadBranch, worktreePath); err != nil {
		return "", fmt.Errorf("failed to create worktree: %w", err)
	}
	return worktreePath, nil
}

func (e *PRCommandExecutor) commitAndPushOutcome(ctx context.Context, pr *store.PullRequest, repo *store.Repository, command, changeName, agent string, outcome *exec.ExecutionOutcome) error {
	worktreePath := GenerateWorktreePath(repo.LocalMirrorPath, pr.HeadBranch)
	hasChanges, err := e.repoMgr.HasChanges(ctx, worktreePath)
	if err != nil {
		return fmt.Errorf("failed to check for changes: %w", err)
	}
	if !hasChanges {
		return e.commentResult(ctx, pr, repo, fmt.Sprintf("%s completed for change `%s` with agent `%s`, but no repository changes were produced.", command, changeName, agent))
	}
	commitMsg := fmt.Sprintf("%s(openspec): %s %s via heimdall", command, command, changeName)
	if _, err := e.repoMgr.CommitAll(ctx, worktreePath, commitMsg); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	token, err := e.github.GetInstallationToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get installation token: %w", err)
	}
	if err := e.repoMgr.PushBranch(ctx, worktreePath, repo.Owner, repo.Name, pr.HeadBranch, token); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}
	return e.commentResult(ctx, pr, repo, fmt.Sprintf("%s completed for change `%s` with agent `%s`. Changes committed and pushed.", command, changeName, agent))
}

func (e *PRCommandExecutor) handleBlockedPermission(ctx context.Context, req ExecutionRequest, pr *store.PullRequest, repo *store.Repository, outcome *exec.ExecutionOutcome) error {
	if outcome.RequestID == "" || outcome.SessionID == "" {
		return fmt.Errorf("blocked on permission but missing request or session ID; cannot create approval command")
	}
	permReq := &store.PendingPermissionRequest{
		RequestID:        outcome.RequestID,
		SessionID:        outcome.SessionID,
		CommandRequestID: 0,
		PullRequestID:    pr.ID,
		RepositoryID:     repo.ID,
		Status:           "pending",
	}
	if err := e.store.CreatePendingPermissionRequest(ctx, permReq); err != nil {
		return fmt.Errorf("failed to persist pending permission request: %w", err)
	}
	msg := fmt.Sprintf("Command blocked on permission request `%s`.\nApprove with: `/heimdall approve %s`", outcome.RequestID, outcome.RequestID)
	return e.commentResult(ctx, pr, repo, msg)
}

func (e *PRCommandExecutor) commentResult(ctx context.Context, pr *store.PullRequest, repo *store.Repository, message string) error {
	if err := e.github.CreateComment(ctx, repo.Owner, repo.Name, pr.Number, message); err != nil {
		e.logger.Error("failed to post PR comment", "pr", pr.Number, "error", err)
		return err
	}
	return nil
}

func (e *PRCommandExecutor) validateChangeExists(ctx context.Context, changeName string, worktreePath string) error {
	if e.openspec == nil {
		return nil // skip validation when no openspec client is configured (tests)
	}
	e.openspec.SetWorktreePath(worktreePath)
	changes, err := e.openspec.ListChanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to list changes in worktree: %w", err)
	}
	for _, ch := range changes {
		if ch == changeName {
			return nil
		}
	}
	return fmt.Errorf("resolved change %q does not exist in the current worktree", changeName)
}
