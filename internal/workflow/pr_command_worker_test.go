package workflow

import (
	"context"
	"testing"

	"github.com/mngeow/heimdall/internal/exec"
	"github.com/mngeow/heimdall/internal/store"
)

// fakeJobQueue is an in-memory job queue for testing.
type fakeJobQueue struct {
	jobs      []*store.Job
	completed []int64
	failed    []int64
}

func (q *fakeJobQueue) Enqueue(ctx context.Context, job *store.Job) error {
	q.jobs = append(q.jobs, job)
	return nil
}

func (q *fakeJobQueue) Dequeue(ctx context.Context) (*store.Job, error) {
	for _, job := range q.jobs {
		if job.Status == "queued" {
			job.Status = "running"
			return job, nil
		}
	}
	return nil, nil
}

func (q *fakeJobQueue) Complete(ctx context.Context, jobID int64) error {
	q.completed = append(q.completed, jobID)
	for _, job := range q.jobs {
		if job.ID == jobID {
			job.Status = "completed"
		}
	}
	return nil
}

func (q *fakeJobQueue) Fail(ctx context.Context, jobID int64, retryDelay int) error {
	q.failed = append(q.failed, jobID)
	for _, job := range q.jobs {
		if job.ID == jobID {
			job.Status = "failed"
		}
	}
	return nil
}

func setupWorkerTest(t *testing.T) (context.Context, *store.Store, *store.JobQueue, *recordingFakeGitHubClient, *recordingFakeExecClient) {
	ctx := context.Background()
	runtimeStore, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
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
	execClient := &recordingFakeExecClient{}
	return ctx, runtimeStore, store.NewJobQueue(runtimeStore), gh, execClient
}

func TestWorkerMarksJobCompletedOnSuccess(t *testing.T) {
	ctx, runtimeStore, jq, gh, execClient := setupWorkerTest(t)
	defer runtimeStore.Close()

	// Enqueue a refine command request and job
	cmdReq := &store.CommandRequest{
		PullRequestID:  1,
		CommandName:    "refine",
		Status:         "queued",
		ChangeName:     "ENG-123-test",
		RequestedAgent: "gpt-5.4",
		PromptTail:     "Add error handling",
		DedupeKey:      "test-dedupe-1",
	}
	if err := runtimeStore.SaveCommandRequest(ctx, cmdReq); err != nil {
		t.Fatalf("SaveCommandRequest() error = %v", err)
	}

	job := &store.Job{
		CommandRequestID: &cmdReq.ID,
		JobType:          "pr_command_refine",
		LockKey:          store.CreatePullRequestLockKey(1),
		Status:           "queued",
	}
	if err := jq.Enqueue(ctx, job); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	repoMgr := &recordingFakeRepoManager{hasChangesReturn: true}
	executor := NewPRCommandExecutor(runtimeStore, repoMgr, gh, nil, execClient, nil)
	worker := NewPRCommandWorker(jq, executor, nil)

	if err := worker.ProcessJob(ctx); err != nil {
		t.Fatalf("ProcessJob() error = %v", err)
	}

	// Job should be completed
	completed, err := runtimeStore.GetJobByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJobByID() error = %v", err)
	}
	if completed == nil {
		t.Fatal("expected job to exist")
	}
	if completed.Status != "completed" {
		t.Errorf("job status = %q, want %q", completed.Status, "completed")
	}

	// Command request should also be completed
	req, err := runtimeStore.GetCommandRequestByID(ctx, cmdReq.ID)
	if err != nil {
		t.Fatalf("GetCommandRequestByID() error = %v", err)
	}
	if req.Status != "completed" {
		t.Errorf("command request status = %q, want %q", req.Status, "completed")
	}

	// PR lock should be released — a second job for the same PR should be dequeuable
	job2 := &store.Job{
		CommandRequestID: &cmdReq.ID,
		JobType:          "pr_command_status",
		LockKey:          store.CreatePullRequestLockKey(1),
		Status:           "queued",
	}
	if err := jq.Enqueue(ctx, job2); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	dequeued, err := jq.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue() error = %v", err)
	}
	if dequeued == nil {
		t.Fatal("expected second job to be dequeuable after first completed, got nil")
	}
	if dequeued.ID != job2.ID {
		t.Errorf("dequeued job ID = %d, want %d", dequeued.ID, job2.ID)
	}
}

func TestWorkerMarksJobFailedOnExecutionError(t *testing.T) {
	ctx, runtimeStore, jq, gh, execClient := setupWorkerTest(t)
	defer runtimeStore.Close()

	// Make refine fail
	execClient.refineErr = context.Canceled

	cmdReq := &store.CommandRequest{
		PullRequestID:  1,
		CommandName:    "refine",
		Status:         "queued",
		ChangeName:     "ENG-123-test",
		RequestedAgent: "gpt-5.4",
		DedupeKey:      "test-dedupe-2",
	}
	if err := runtimeStore.SaveCommandRequest(ctx, cmdReq); err != nil {
		t.Fatalf("SaveCommandRequest() error = %v", err)
	}

	job := &store.Job{
		CommandRequestID: &cmdReq.ID,
		JobType:          "pr_command_refine",
		LockKey:          store.CreatePullRequestLockKey(1),
		Status:           "queued",
	}
	if err := jq.Enqueue(ctx, job); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	repoMgr := &recordingFakeRepoManager{hasChangesReturn: true}
	executor := NewPRCommandExecutor(runtimeStore, repoMgr, gh, nil, execClient, nil)
	worker := NewPRCommandWorker(jq, executor, nil)

	// ProcessJob should not return error; it logs and swallows execution errors
	if err := worker.ProcessJob(ctx); err != nil {
		t.Fatalf("ProcessJob() error = %v", err)
	}

	// Command request should be marked failed/blocked
	req, err := runtimeStore.GetCommandRequestByID(ctx, cmdReq.ID)
	if err != nil {
		t.Fatalf("GetCommandRequestByID() error = %v", err)
	}
	if req.Status != "blocked" && req.Status != "failed" {
		t.Errorf("command request status = %q, want blocked or failed", req.Status)
	}
}

func TestWorkerRejectsMissingCommandRequest(t *testing.T) {
	ctx, runtimeStore, jq, _, _ := setupWorkerTest(t)
	defer runtimeStore.Close()

	badID := int64(99999)
	job := &store.Job{
		CommandRequestID: &badID,
		JobType:          "pr_command_refine",
		LockKey:          store.CreatePullRequestLockKey(1),
		Status:           "queued",
	}
	if err := jq.Enqueue(ctx, job); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	executor := NewPRCommandExecutor(runtimeStore, nil, nil, nil, nil, nil)
	worker := NewPRCommandWorker(jq, executor, nil)

	if err := worker.ProcessJob(ctx); err != nil {
		t.Fatalf("ProcessJob() error = %v", err)
	}

	// Job should have been failed (Fail() keeps it queued with incremented attempts if retries remain)
	j, err := runtimeStore.GetJobByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJobByID() error = %v", err)
	}
	if j == nil {
		t.Fatal("expected job to exist")
	}
	if j.Status != "queued" && j.Status != "failed" && j.Status != "dead" {
		t.Errorf("job status = %q, want queued/failed/dead", j.Status)
	}
}

func TestExecuteApproveCallsResumeSessionAndReportsOutcome(t *testing.T) {
	ctx, runtimeStore, _, gh, execClient := setupWorkerTest(t)
	defer runtimeStore.Close()

	permReq := &store.PendingPermissionRequest{
		RequestID:        "perm_123",
		SessionID:        "sess_456",
		CommandRequestID: 1,
		PullRequestID:    1,
		RepositoryID:     1,
		Status:           "pending",
	}
	if err := runtimeStore.CreatePendingPermissionRequest(ctx, permReq); err != nil {
		t.Fatalf("CreatePendingPermissionRequest() error = %v", err)
	}

	// Configure a specific resumed outcome
	execClient.resumeOutcome = &exec.ExecutionOutcome{Status: "success", Summary: "artifacts generated and tasks applied"}
	execClient.resumeErr = nil

	executor := NewPRCommandExecutor(runtimeStore, nil, gh, nil, execClient, nil)
	req := ExecutionRequest{Kind: "approve", RequestID: "perm_123"}
	pr := &store.PullRequest{ID: 1, RepositoryID: 1, Number: 42, HeadBranch: "heimdall/test-branch", BaseBranch: "main", State: "open"}
	repo := &store.Repository{Owner: "test", Name: "repo"}
	if err := executor.ExecuteApprove(ctx, req, pr, repo); err != nil {
		t.Fatalf("ExecuteApprove() error = %v", err)
	}

	if len(execClient.replyCalls) != 1 {
		t.Fatalf("expected 1 ReplyPermission call, got %d", len(execClient.replyCalls))
	}
	if execClient.replyCalls[0].requestID != "perm_123" {
		t.Errorf("ReplyPermission requestID = %q, want %q", execClient.replyCalls[0].requestID, "perm_123")
	}

	if len(gh.comments) != 1 {
		t.Fatalf("expected 1 PR comment, got %d", len(gh.comments))
	}
	comment := gh.comments[0]
	if !contains(comment, "artifacts generated and tasks applied") {
		t.Errorf("expected comment to contain resumed outcome summary, got: %s", comment)
	}
	if !contains(comment, "perm_123") {
		t.Errorf("expected comment to mention request ID, got: %s", comment)
	}
}

func TestExecuteApproveResumeSessionFailure(t *testing.T) {
	ctx, runtimeStore, _, gh, execClient := setupWorkerTest(t)
	defer runtimeStore.Close()

	permReq := &store.PendingPermissionRequest{
		RequestID:        "perm_789",
		SessionID:        "sess_abc",
		CommandRequestID: 1,
		PullRequestID:    1,
		RepositoryID:     1,
		Status:           "pending",
	}
	if err := runtimeStore.CreatePendingPermissionRequest(ctx, permReq); err != nil {
		t.Fatalf("CreatePendingPermissionRequest() error = %v", err)
	}

	execClient.resumeOutcome = nil
	execClient.resumeErr = context.Canceled

	executor := NewPRCommandExecutor(runtimeStore, nil, gh, nil, execClient, nil)
	req := ExecutionRequest{Kind: "approve", RequestID: "perm_789"}
	pr := &store.PullRequest{ID: 1, RepositoryID: 1, Number: 42, HeadBranch: "heimdall/test-branch", BaseBranch: "main", State: "open"}
	repo := &store.Repository{Owner: "test", Name: "repo"}
	err := executor.ExecuteApprove(ctx, req, pr, repo)
	if err == nil {
		t.Fatal("expected error when ResumeSession fails, got nil")
	}
	if !contains(err.Error(), "resumed session failed") {
		t.Errorf("expected error to mention resumed session failure, got: %v", err)
	}
}

func TestValidateChangeExistsRejectsStaleBinding(t *testing.T) {
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

	// Fake openspec client that reports a different change name
	fakeOpenspec := &stubOpenSpecClient{changes: []string{"other-change", "another-change"}}
	executor := NewPRCommandExecutor(runtimeStore, nil, nil, fakeOpenspec, nil, nil)

	err = executor.validateChangeExists(ctx, "stale-change", GenerateWorktreePath(repo.LocalMirrorPath, pr.HeadBranch))
	if err == nil {
		t.Fatal("expected error for stale change, got nil")
	}
	if !contains(err.Error(), "does not exist in the current worktree") {
		t.Errorf("expected stale change error, got: %v", err)
	}
}

func TestValidateChangeExistsAcceptsExistingChange(t *testing.T) {
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

	fakeOpenspec := &stubOpenSpecClient{changes: []string{"existing-change", "other-change"}}
	executor := NewPRCommandExecutor(runtimeStore, nil, nil, fakeOpenspec, nil, nil)

	err = executor.validateChangeExists(ctx, "existing-change", GenerateWorktreePath(repo.LocalMirrorPath, pr.HeadBranch))
	if err != nil {
		t.Fatalf("expected no error for existing change, got: %v", err)
	}
}

// stubOpenSpecClient is a test double for prCommandOpenSpecClient.
type stubOpenSpecClient struct {
	changes []string
}

func (f *stubOpenSpecClient) SetWorktreePath(string) {}
func (f *stubOpenSpecClient) ListChanges(ctx context.Context) ([]string, error) {
	return f.changes, nil
}
func (f *stubOpenSpecClient) GetStatus(ctx context.Context, name string) (*exec.ChangeStatus, error) {
	return nil, nil
}
