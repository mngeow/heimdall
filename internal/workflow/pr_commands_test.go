package workflow

import (
	"context"
	"testing"

	"github.com/mngeow/heimdall/internal/exec"
	"github.com/mngeow/heimdall/internal/store"
)

// recordingFakeRepoManager records every method call for verification.
type recordingFakeRepoManager struct {
	ensureBareMirrorCalls []struct{ mirrorPath, owner, repo, token string }
	createWorktreeCalls   []struct{ mirrorPath, baseBranch, headBranch, worktreePath string }
	hasChangesReturn      bool
	hasChangesCalls       []string
	commitAllCalls        []struct{ worktreePath, message string }
	pushBranchCalls       []struct{ worktreePath, owner, repo, branch, token string }
}

func (f *recordingFakeRepoManager) EnsureBareMirror(_ context.Context, mirrorPath, owner, repo, token string) error {
	f.ensureBareMirrorCalls = append(f.ensureBareMirrorCalls, struct{ mirrorPath, owner, repo, token string }{mirrorPath, owner, repo, token})
	return nil
}
func (f *recordingFakeRepoManager) CreateWorktree(_ context.Context, mirrorPath, baseBranch, headBranch, worktreePath string) error {
	f.createWorktreeCalls = append(f.createWorktreeCalls, struct{ mirrorPath, baseBranch, headBranch, worktreePath string }{mirrorPath, baseBranch, headBranch, worktreePath})
	return nil
}
func (f *recordingFakeRepoManager) HasChanges(_ context.Context, worktreePath string) (bool, error) {
	f.hasChangesCalls = append(f.hasChangesCalls, worktreePath)
	return f.hasChangesReturn, nil
}
func (f *recordingFakeRepoManager) CommitAll(_ context.Context, worktreePath, message string) (string, error) {
	f.commitAllCalls = append(f.commitAllCalls, struct{ worktreePath, message string }{worktreePath, message})
	return "abc123", nil
}
func (f *recordingFakeRepoManager) PushBranch(_ context.Context, worktreePath, owner, repo, branch, token string) error {
	f.pushBranchCalls = append(f.pushBranchCalls, struct{ worktreePath, owner, repo, branch, token string }{worktreePath, owner, repo, branch, token})
	return nil
}

// recordingFakeGitHubClient records CreateComment calls.
type recordingFakeGitHubClient struct {
	comments []string
}

func (f *recordingFakeGitHubClient) GetInstallationToken(_ context.Context) (string, error) {
	return "fake-token", nil
}
func (f *recordingFakeGitHubClient) CreateComment(_ context.Context, owner, repo string, number int, body string) error {
	f.comments = append(f.comments, body)
	return nil
}

// recordingFakeExecClient records execution calls and returns configurable outcomes.
type recordingFakeExecClient struct {
	refineOutcome *exec.ExecutionOutcome
	refineErr     error
	applyOutcome  *exec.ExecutionOutcome
	applyErr      error
	genericErr    error
	replyErr      error
	resumeOutcome *exec.ExecutionOutcome
	resumeErr     error

	refineCalls  []struct{ agent, changeName, prompt string }
	applyCalls   []struct{ agent, changeName, prompt string }
	genericCalls []struct{ agent, command, prompt string }
	replyCalls   []struct{ requestID, sessionID string }
	resumeCalls  []string
}

func (f *recordingFakeExecClient) RunRefine(_ context.Context, agent, changeName, prompt string) (*exec.ExecutionOutcome, error) {
	f.refineCalls = append(f.refineCalls, struct{ agent, changeName, prompt string }{agent, changeName, prompt})
	if f.refineErr != nil {
		return nil, f.refineErr
	}
	if f.refineOutcome != nil {
		return f.refineOutcome, nil
	}
	return &exec.ExecutionOutcome{Status: "success", Summary: "refine completed"}, nil
}
func (f *recordingFakeExecClient) RunApply(_ context.Context, agent, changeName, prompt string) (*exec.ExecutionOutcome, error) {
	f.applyCalls = append(f.applyCalls, struct{ agent, changeName, prompt string }{agent, changeName, prompt})
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	if f.applyOutcome != nil {
		return f.applyOutcome, nil
	}
	return &exec.ExecutionOutcome{Status: "success", Summary: "apply completed"}, nil
}
func (f *recordingFakeExecClient) RunGeneric(_ context.Context, agent, command, prompt string) error {
	f.genericCalls = append(f.genericCalls, struct{ agent, command, prompt string }{agent, command, prompt})
	return f.genericErr
}
func (f *recordingFakeExecClient) ReplyPermission(_ context.Context, requestID, sessionID string) error {
	f.replyCalls = append(f.replyCalls, struct{ requestID, sessionID string }{requestID, sessionID})
	return f.replyErr
}
func (f *recordingFakeExecClient) ResumeSession(_ context.Context, sessionID string) (*exec.ExecutionOutcome, error) {
	f.resumeCalls = append(f.resumeCalls, sessionID)
	if f.resumeErr != nil {
		return nil, f.resumeErr
	}
	if f.resumeOutcome != nil {
		return f.resumeOutcome, nil
	}
	return &exec.ExecutionOutcome{Status: "success", Summary: "resumed session completed"}, nil
}
func (f *recordingFakeExecClient) SetWorktreePath(_ string) {}

func TestPRCommandExecutorResolveChange(t *testing.T) {
	ctx := context.Background()
	runtimeStore, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer runtimeStore.Close()

	if err := runtimeStore.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// Setup repository
	repo := &store.Repository{
		Provider:        "github",
		RepoRef:         "github.com/test/repo",
		Owner:           "test",
		Name:            "repo",
		DefaultBranch:   "main",
		BranchPrefix:    "heimdall",
		LocalMirrorPath: "/tmp/test-repo.git",
		IsActive:        true,
	}
	if err := runtimeStore.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	// Setup PR
	pr := &store.PullRequest{
		RepositoryID: repo.ID,
		Number:       42,
		HeadBranch:   "heimdall/test-branch",
		BaseBranch:   "main",
		State:        "open",
	}
	if err := runtimeStore.SavePullRequest(ctx, pr); err != nil {
		t.Fatalf("SavePullRequest() error = %v", err)
	}

	executor := NewPRCommandExecutor(runtimeStore, nil, nil, nil, nil, nil)

	t.Run("ExplicitChangeName", func(t *testing.T) {
		changeName, err := executor.ResolveChange(ctx, pr.ID, "my-change")
		if err != nil {
			t.Fatalf("ResolveChange() error = %v", err)
		}
		if changeName != "my-change" {
			t.Errorf("ResolveChange() = %q, want %q", changeName, "my-change")
		}
	})

	t.Run("NoActiveBindings", func(t *testing.T) {
		_, err := executor.ResolveChange(ctx, pr.ID, "")
		if err == nil {
			t.Fatal("expected error for no active bindings, got nil")
		}
	})

	t.Run("SingleActiveBinding", func(t *testing.T) {
		binding := &store.RepoBinding{
			WorkItemID:    1,
			RepositoryID:  repo.ID,
			BranchName:    "heimdall/test-branch",
			ChangeName:    "ENG-123-test",
			BindingStatus: "active",
		}
		if err := runtimeStore.SaveRepoBinding(ctx, binding); err != nil {
			t.Fatalf("SaveRepoBinding() error = %v", err)
		}

		changeName, err := executor.ResolveChange(ctx, pr.ID, "")
		if err != nil {
			t.Fatalf("ResolveChange() error = %v", err)
		}
		if changeName != "ENG-123-test" {
			t.Errorf("ResolveChange() = %q, want %q", changeName, "ENG-123-test")
		}
	})

	t.Run("MultipleActiveBindingsAmbiguous", func(t *testing.T) {
		binding2 := &store.RepoBinding{
			WorkItemID:    2,
			RepositoryID:  repo.ID,
			BranchName:    "heimdall/test-branch",
			ChangeName:    "ENG-456-other",
			BindingStatus: "active",
		}
		if err := runtimeStore.SaveRepoBinding(ctx, binding2); err != nil {
			t.Fatalf("SaveRepoBinding() error = %v", err)
		}

		_, err := executor.ResolveChange(ctx, pr.ID, "")
		if err == nil {
			t.Fatal("expected error for ambiguous bindings, got nil")
		}
		if err.Error() != "ambiguous: PR 1 has 2 active changes; specify change-name explicitly" {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestPRCommandExecutorResolvePendingRequest(t *testing.T) {
	ctx := context.Background()
	runtimeStore, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer runtimeStore.Close()

	if err := runtimeStore.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// Setup repository
	repo := &store.Repository{
		Provider:        "github",
		RepoRef:         "github.com/test/repo",
		Owner:           "test",
		Name:            "repo",
		DefaultBranch:   "main",
		BranchPrefix:    "heimdall",
		LocalMirrorPath: "/tmp/test-repo.git",
		IsActive:        true,
	}
	if err := runtimeStore.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	// Setup PR
	pr := &store.PullRequest{
		RepositoryID: repo.ID,
		Number:       42,
		HeadBranch:   "heimdall/test-branch",
		BaseBranch:   "main",
		State:        "open",
	}
	if err := runtimeStore.SavePullRequest(ctx, pr); err != nil {
		t.Fatalf("SavePullRequest() error = %v", err)
	}

	executor := NewPRCommandExecutor(runtimeStore, nil, nil, nil, nil, nil)

	t.Run("UnknownRequestID", func(t *testing.T) {
		_, err := executor.ResolvePendingRequest(ctx, "perm_unknown", pr.ID)
		if err == nil {
			t.Fatal("expected error for unknown request ID, got nil")
		}
	})

	t.Run("WrongPullRequest", func(t *testing.T) {
		req := &store.PendingPermissionRequest{
			RequestID:        "perm_123",
			SessionID:        "sess_456",
			CommandRequestID: 1,
			PullRequestID:    999,
			RepositoryID:     repo.ID,
			Status:           "pending",
		}
		if err := runtimeStore.CreatePendingPermissionRequest(ctx, req); err != nil {
			t.Fatalf("CreatePendingPermissionRequest() error = %v", err)
		}

		_, err := executor.ResolvePendingRequest(ctx, "perm_123", pr.ID)
		if err == nil {
			t.Fatal("expected error for wrong pull request, got nil")
		}
	})

	t.Run("AlreadyResolved", func(t *testing.T) {
		req := &store.PendingPermissionRequest{
			RequestID:        "perm_456",
			SessionID:        "sess_789",
			CommandRequestID: 2,
			PullRequestID:    pr.ID,
			RepositoryID:     repo.ID,
			Status:           "approved",
		}
		if err := runtimeStore.CreatePendingPermissionRequest(ctx, req); err != nil {
			t.Fatalf("CreatePendingPermissionRequest() error = %v", err)
		}
		if err := runtimeStore.ResolvePendingPermissionRequest(ctx, "perm_456", "approved"); err != nil {
			t.Fatalf("ResolvePendingPermissionRequest() error = %v", err)
		}

		_, err := executor.ResolvePendingRequest(ctx, "perm_456", pr.ID)
		if err == nil {
			t.Fatal("expected error for already resolved request, got nil")
		}
	})

	t.Run("ValidPendingRequest", func(t *testing.T) {
		req := &store.PendingPermissionRequest{
			RequestID:        "perm_789",
			SessionID:        "sess_abc",
			CommandRequestID: 3,
			PullRequestID:    pr.ID,
			RepositoryID:     repo.ID,
			Status:           "pending",
		}
		if err := runtimeStore.CreatePendingPermissionRequest(ctx, req); err != nil {
			t.Fatalf("CreatePendingPermissionRequest() error = %v", err)
		}

		retrieved, err := executor.ResolvePendingRequest(ctx, "perm_789", pr.ID)
		if err != nil {
			t.Fatalf("ResolvePendingRequest() error = %v", err)
		}
		if retrieved.RequestID != "perm_789" {
			t.Errorf("RequestID = %q, want %q", retrieved.RequestID, "perm_789")
		}
	})
}

func TestExecuteStatusPostsRealState(t *testing.T) {
	ctx := context.Background()
	runtimeStore, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer runtimeStore.Close()
	if err := runtimeStore.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	repo := &store.Repository{Provider: "github", RepoRef: "github.com/test/repo", Owner: "test", Name: "repo", DefaultBranch: "main", BranchPrefix: "heimdall", LocalMirrorPath: "/tmp/test-repo.git", IsActive: true}
	if err := runtimeStore.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}
	pr := &store.PullRequest{RepositoryID: repo.ID, Number: 42, HeadBranch: "heimdall/test-branch", BaseBranch: "main", State: "open"}
	if err := runtimeStore.SavePullRequest(ctx, pr); err != nil {
		t.Fatalf("SavePullRequest() error = %v", err)
	}
	binding := &store.RepoBinding{WorkItemID: 1, RepositoryID: repo.ID, BranchName: "heimdall/test-branch", ChangeName: "ENG-123-test", BindingStatus: "active"}
	if err := runtimeStore.SaveRepoBinding(ctx, binding); err != nil {
		t.Fatalf("SaveRepoBinding() error = %v", err)
	}

	gh := &recordingFakeGitHubClient{}
	executor := NewPRCommandExecutor(runtimeStore, nil, gh, nil, nil, nil)

	if err := executor.ExecuteStatus(ctx, pr, repo); err != nil {
		t.Fatalf("ExecuteStatus() error = %v", err)
	}

	if len(gh.comments) != 1 {
		t.Fatalf("expected 1 PR comment, got %d", len(gh.comments))
	}
	comment := gh.comments[0]
	if comment == "" {
		t.Fatal("expected non-empty status comment")
	}
	if !contains(comment, "ENG-123-test") {
		t.Errorf("status comment should mention the active change name, got: %s", comment)
	}
	if !contains(comment, "heimdall/test-branch") {
		t.Errorf("status comment should mention the branch, got: %s", comment)
	}
}

func TestExecuteRefinePerformsRealWork(t *testing.T) {
	ctx := context.Background()
	runtimeStore, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer runtimeStore.Close()
	if err := runtimeStore.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	repo := &store.Repository{Provider: "github", RepoRef: "github.com/test/repo", Owner: "test", Name: "repo", DefaultBranch: "main", BranchPrefix: "heimdall", LocalMirrorPath: "/tmp/test-repo.git", IsActive: true}
	if err := runtimeStore.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}
	pr := &store.PullRequest{RepositoryID: repo.ID, Number: 42, HeadBranch: "heimdall/test-branch", BaseBranch: "main", State: "open"}
	if err := runtimeStore.SavePullRequest(ctx, pr); err != nil {
		t.Fatalf("SavePullRequest() error = %v", err)
	}
	binding := &store.RepoBinding{WorkItemID: 1, RepositoryID: repo.ID, BranchName: "heimdall/test-branch", ChangeName: "ENG-123-test", BindingStatus: "active"}
	if err := runtimeStore.SaveRepoBinding(ctx, binding); err != nil {
		t.Fatalf("SaveRepoBinding() error = %v", err)
	}

	gh := &recordingFakeGitHubClient{}
	repoMgr := &recordingFakeRepoManager{hasChangesReturn: true}
	execClient := &recordingFakeExecClient{}
	executor := NewPRCommandExecutor(runtimeStore, repoMgr, gh, nil, execClient, nil)

	req := ExecutionRequest{Kind: "refine", ChangeName: "", Agent: "gpt-5.4", PromptTail: "Add error handling"}
	if err := executor.ExecuteRefine(ctx, req, pr, repo); err != nil {
		t.Fatalf("ExecuteRefine() error = %v", err)
	}

	// Must have resolved change name from binding
	if len(execClient.refineCalls) != 1 {
		t.Fatalf("expected 1 refine call, got %d", len(execClient.refineCalls))
	}
	call := execClient.refineCalls[0]
	if call.changeName != "ENG-123-test" {
		t.Errorf("refine called with changeName=%q, want %q", call.changeName, "ENG-123-test")
	}
	if call.prompt != "Add error handling" {
		t.Errorf("refine called with prompt=%q, want %q", call.prompt, "Add error handling")
	}

	// Must have prepared worktree
	if len(repoMgr.ensureBareMirrorCalls) != 1 {
		t.Errorf("expected EnsureBareMirror to be called, got %d calls", len(repoMgr.ensureBareMirrorCalls))
	}
	if len(repoMgr.createWorktreeCalls) != 1 {
		t.Errorf("expected CreateWorktree to be called, got %d calls", len(repoMgr.createWorktreeCalls))
	}

	// Must have committed and pushed because hasChangesReturn=true
	if len(repoMgr.commitAllCalls) != 1 {
		t.Errorf("expected CommitAll to be called, got %d calls", len(repoMgr.commitAllCalls))
	}
	if len(repoMgr.pushBranchCalls) != 1 {
		t.Errorf("expected PushBranch to be called, got %d calls", len(repoMgr.pushBranchCalls))
	}

	// Must have posted a success comment
	if len(gh.comments) != 1 {
		t.Fatalf("expected 1 PR comment, got %d", len(gh.comments))
	}
	if !contains(gh.comments[0], "refine completed") {
		t.Errorf("expected success comment, got: %s", gh.comments[0])
	}
}

func TestExecuteApplyPerformsRealWork(t *testing.T) {
	ctx := context.Background()
	runtimeStore, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer runtimeStore.Close()
	if err := runtimeStore.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	repo := &store.Repository{Provider: "github", RepoRef: "github.com/test/repo", Owner: "test", Name: "repo", DefaultBranch: "main", BranchPrefix: "heimdall", LocalMirrorPath: "/tmp/test-repo.git", IsActive: true}
	if err := runtimeStore.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}
	pr := &store.PullRequest{RepositoryID: repo.ID, Number: 42, HeadBranch: "heimdall/test-branch", BaseBranch: "main", State: "open"}
	if err := runtimeStore.SavePullRequest(ctx, pr); err != nil {
		t.Fatalf("SavePullRequest() error = %v", err)
	}
	binding := &store.RepoBinding{WorkItemID: 1, RepositoryID: repo.ID, BranchName: "heimdall/test-branch", ChangeName: "ENG-123-test", BindingStatus: "active"}
	if err := runtimeStore.SaveRepoBinding(ctx, binding); err != nil {
		t.Fatalf("SaveRepoBinding() error = %v", err)
	}

	gh := &recordingFakeGitHubClient{}
	repoMgr := &recordingFakeRepoManager{hasChangesReturn: true}
	execClient := &recordingFakeExecClient{}
	executor := NewPRCommandExecutor(runtimeStore, repoMgr, gh, nil, execClient, nil)

	req := ExecutionRequest{Kind: "apply", ChangeName: "", Agent: "gpt-5.4", PromptTail: ""}
	if err := executor.ExecuteApply(ctx, req, pr, repo); err != nil {
		t.Fatalf("ExecuteApply() error = %v", err)
	}

	if len(execClient.applyCalls) != 1 {
		t.Fatalf("expected 1 apply call, got %d", len(execClient.applyCalls))
	}
	if execClient.applyCalls[0].changeName != "ENG-123-test" {
		t.Errorf("apply called with changeName=%q, want %q", execClient.applyCalls[0].changeName, "ENG-123-test")
	}
	if len(repoMgr.commitAllCalls) != 1 {
		t.Errorf("expected CommitAll to be called, got %d calls", len(repoMgr.commitAllCalls))
	}
	if len(gh.comments) != 1 {
		t.Fatalf("expected 1 PR comment, got %d", len(gh.comments))
	}
}

func TestExecuteOpencodePerformsRealWork(t *testing.T) {
	ctx := context.Background()
	runtimeStore, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer runtimeStore.Close()
	if err := runtimeStore.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	repo := &store.Repository{Provider: "github", RepoRef: "github.com/test/repo", Owner: "test", Name: "repo", DefaultBranch: "main", BranchPrefix: "heimdall", LocalMirrorPath: "/tmp/test-repo.git", IsActive: true}
	if err := runtimeStore.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}
	pr := &store.PullRequest{RepositoryID: repo.ID, Number: 42, HeadBranch: "heimdall/test-branch", BaseBranch: "main", State: "open"}
	if err := runtimeStore.SavePullRequest(ctx, pr); err != nil {
		t.Fatalf("SavePullRequest() error = %v", err)
	}
	binding := &store.RepoBinding{WorkItemID: 1, RepositoryID: repo.ID, BranchName: "heimdall/test-branch", ChangeName: "ENG-123-test", BindingStatus: "active"}
	if err := runtimeStore.SaveRepoBinding(ctx, binding); err != nil {
		t.Fatalf("SaveRepoBinding() error = %v", err)
	}

	gh := &recordingFakeGitHubClient{}
	repoMgr := &recordingFakeRepoManager{hasChangesReturn: true}
	execClient := &recordingFakeExecClient{}
	executor := NewPRCommandExecutor(runtimeStore, repoMgr, gh, nil, execClient, nil)

	req := ExecutionRequest{Kind: "opencode", ChangeName: "", Agent: "gpt-5.4", Alias: "explore-change", PromptTail: "Compare options"}
	if err := executor.ExecuteOpencode(ctx, req, pr, repo); err != nil {
		t.Fatalf("ExecuteOpencode() error = %v", err)
	}

	if len(execClient.genericCalls) != 1 {
		t.Fatalf("expected 1 generic call, got %d", len(execClient.genericCalls))
	}
	if execClient.genericCalls[0].command != "explore-change" {
		t.Errorf("generic called with command=%q, want %q", execClient.genericCalls[0].command, "explore-change")
	}
	if execClient.genericCalls[0].prompt != "Compare options" {
		t.Errorf("generic called with prompt=%q, want %q", execClient.genericCalls[0].prompt, "Compare options")
	}
	if len(repoMgr.commitAllCalls) != 1 {
		t.Errorf("expected CommitAll to be called, got %d calls", len(repoMgr.commitAllCalls))
	}
	if len(gh.comments) != 1 {
		t.Fatalf("expected 1 PR comment, got %d", len(gh.comments))
	}
}

func TestExecuteApprovePerformsRealWork(t *testing.T) {
	ctx := context.Background()
	runtimeStore, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer runtimeStore.Close()
	if err := runtimeStore.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	repo := &store.Repository{Provider: "github", RepoRef: "github.com/test/repo", Owner: "test", Name: "repo", DefaultBranch: "main", BranchPrefix: "heimdall", LocalMirrorPath: "/tmp/test-repo.git", IsActive: true}
	if err := runtimeStore.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}
	pr := &store.PullRequest{RepositoryID: repo.ID, Number: 42, HeadBranch: "heimdall/test-branch", BaseBranch: "main", State: "open"}
	if err := runtimeStore.SavePullRequest(ctx, pr); err != nil {
		t.Fatalf("SavePullRequest() error = %v", err)
	}

	permReq := &store.PendingPermissionRequest{
		RequestID:        "perm_123",
		SessionID:        "sess_456",
		CommandRequestID: 1,
		PullRequestID:    pr.ID,
		RepositoryID:     repo.ID,
		Status:           "pending",
	}
	if err := runtimeStore.CreatePendingPermissionRequest(ctx, permReq); err != nil {
		t.Fatalf("CreatePendingPermissionRequest() error = %v", err)
	}

	gh := &recordingFakeGitHubClient{}
	execClient := &recordingFakeExecClient{}
	executor := NewPRCommandExecutor(runtimeStore, nil, gh, nil, execClient, nil)

	req := ExecutionRequest{Kind: "approve", RequestID: "perm_123"}
	if err := executor.ExecuteApprove(ctx, req, pr, repo); err != nil {
		t.Fatalf("ExecuteApprove() error = %v", err)
	}

	if len(execClient.replyCalls) != 1 {
		t.Fatalf("expected 1 ReplyPermission call, got %d", len(execClient.replyCalls))
	}
	if execClient.replyCalls[0].requestID != "perm_123" {
		t.Errorf("ReplyPermission called with requestID=%q, want %q", execClient.replyCalls[0].requestID, "perm_123")
	}

	// Must have resolved the permission request in storage
	resolved, err := runtimeStore.GetPendingPermissionRequestByID(ctx, "perm_123")
	if err != nil {
		t.Fatalf("GetPendingPermissionRequestByID() error = %v", err)
	}
	if resolved == nil || resolved.Status != "approved" {
		t.Errorf("expected permission request to be approved, got %#v", resolved)
	}

	if len(gh.comments) != 1 {
		t.Fatalf("expected 1 PR comment, got %d", len(gh.comments))
	}
	if !contains(gh.comments[0], "perm_123") {
		t.Errorf("expected approval comment to mention request ID, got: %s", gh.comments[0])
	}
	if !contains(gh.comments[0], "resumed") {
		t.Errorf("expected approval comment to mention resumed outcome, got: %s", gh.comments[0])
	}
}

func TestExecuteApproveRejectsMissingIDs(t *testing.T) {
	ctx := context.Background()
	runtimeStore, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer runtimeStore.Close()
	if err := runtimeStore.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	repo := &store.Repository{Provider: "github", RepoRef: "github.com/test/repo", Owner: "test", Name: "repo", DefaultBranch: "main", BranchPrefix: "heimdall", LocalMirrorPath: "/tmp/test-repo.git", IsActive: true}
	if err := runtimeStore.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}
	pr := &store.PullRequest{RepositoryID: repo.ID, Number: 42, HeadBranch: "heimdall/test-branch", BaseBranch: "main", State: "open"}
	if err := runtimeStore.SavePullRequest(ctx, pr); err != nil {
		t.Fatalf("SavePullRequest() error = %v", err)
	}

	gh := &recordingFakeGitHubClient{}
	executor := NewPRCommandExecutor(runtimeStore, nil, gh, nil, &recordingFakeExecClient{}, nil)

	// Simulate blocked permission outcome with empty IDs
	outcome := &exec.ExecutionOutcome{Status: "needs_permission", RequestID: "", SessionID: ""}
	req := ExecutionRequest{Kind: "refine", ChangeName: "test-change", Agent: "gpt-5.4"}
	if err := executor.handleBlockedPermission(ctx, req, pr, repo, outcome); err == nil {
		t.Fatal("expected error for empty permission IDs, got nil")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
