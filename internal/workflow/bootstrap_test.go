package workflow

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	gh "github.com/google/go-github/v57/github"
	"github.com/mngeow/heimdall/internal/exec"
	"github.com/mngeow/heimdall/internal/store"
)

type fakeBootstrapRepoManager struct {
	hasChanges bool
	commitSHA  string
	steps      []string
}

func (f *fakeBootstrapRepoManager) EnsureBareMirror(context.Context, string, string, string, string) error {
	f.steps = append(f.steps, "ensure_mirror")
	return nil
}

func (f *fakeBootstrapRepoManager) CreateWorktree(context.Context, string, string, string, string) error {
	f.steps = append(f.steps, "create_worktree")
	return nil
}

func (f *fakeBootstrapRepoManager) HasChanges(context.Context, string) (bool, error) {
	f.steps = append(f.steps, "has_changes")
	return f.hasChanges, nil
}

func (f *fakeBootstrapRepoManager) CommitAll(context.Context, string, string) (string, error) {
	f.steps = append(f.steps, "commit")
	if !f.hasChanges {
		return "", errors.New("commit should not run without changes")
	}
	return f.commitSHA, nil
}

func (f *fakeBootstrapRepoManager) PushBranch(context.Context, string, string, string, string, string) error {
	f.steps = append(f.steps, "push")
	return nil
}

type fakeBootstrapGitHubClient struct {
	token          string
	existingPR     *gh.PullRequest
	createdPR      *gh.PullRequest
	createCalls    int
	findCalls      int
	ensureLabel    []string
	addLabelCalls  []int
	installationOK bool
}

func (f *fakeBootstrapGitHubClient) GetInstallationToken(context.Context) (string, error) {
	if !f.installationOK {
		return "", errors.New("installation token error")
	}
	return f.token, nil
}

func (f *fakeBootstrapGitHubClient) FindOpenPullRequestByHead(context.Context, string, string, string, string) (*gh.PullRequest, error) {
	f.findCalls++
	return f.existingPR, nil
}

func (f *fakeBootstrapGitHubClient) CreatePullRequest(context.Context, string, string, string, string, string, string) (*gh.PullRequest, error) {
	f.createCalls++
	return f.createdPR, nil
}

func (f *fakeBootstrapGitHubClient) EnsurePRMonitorLabel(_ context.Context, _, _, label string) error {
	f.ensureLabel = append(f.ensureLabel, label)
	return nil
}

func (f *fakeBootstrapGitHubClient) AddPRMonitorLabel(_ context.Context, _, _ string, number int, _ string) error {
	f.addLabelCalls = append(f.addLabelCalls, number)
	return nil
}

type fakeBootstrapRunner struct {
	request exec.BootstrapRequest
	result  *exec.BootstrapResult
	err     error
}

func (f *fakeBootstrapRunner) RunBootstrap(_ context.Context, req exec.BootstrapRequest) (*exec.BootstrapResult, error) {
	f.request = req
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func TestBootstrapWorkflowExecuteSuccessReusesExistingPullRequest(t *testing.T) {
	ctx := context.Background()
	runtimeStore := testWorkflowStore(t)
	workItem, repository, run := seedBootstrapRun(t, ctx, runtimeStore)
	repository.PRMonitorLabel = "heimdall-monitored"
	if err := runtimeStore.SaveRepository(ctx, repository); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	repoMgr := &fakeBootstrapRepoManager{hasChanges: true, commitSHA: "abc123"}
	githubClient := &fakeBootstrapGitHubClient{
		token:          "installation-token",
		existingPR:     &gh.PullRequest{Number: gh.Int(42), NodeID: gh.String("PR_node_42"), Title: gh.String("[ENG-123] Bootstrap PR for Add rate limiting"), State: gh.String("open"), HTMLURL: gh.String("https://example.test/pr/42")},
		installationOK: true,
	}
	bootstrapRunner := &fakeBootstrapRunner{result: &exec.BootstrapResult{Summary: "Created or updated .heimdall/bootstrap/ENG-123.md from the activation seed."}}
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	workflow := NewBootstrapWorkflow(runtimeStore, repoMgr, githubClient, bootstrapRunner, logger)
	if err := workflow.Execute(ctx, run.ID); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	retrievedRun, err := runtimeStore.GetWorkflowRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetWorkflowRun() error = %v", err)
	}
	if retrievedRun.Status != "completed" {
		t.Fatalf("expected completed workflow run, got %q", retrievedRun.Status)
	}

	binding, err := runtimeStore.GetActiveBinding(ctx, workItem.ID, repository.ID)
	if err != nil {
		t.Fatalf("GetActiveBinding() error = %v", err)
	}
	if binding == nil {
		t.Fatal("expected active repo binding")
	}
	if binding.LastHeadSHA != "abc123" {
		t.Fatalf("expected binding SHA abc123, got %q", binding.LastHeadSHA)
	}

	pr, err := runtimeStore.GetPullRequestByBindingID(ctx, binding.ID)
	if err != nil {
		t.Fatalf("GetPullRequestByBindingID() error = %v", err)
	}
	if pr == nil || pr.Number != 42 {
		t.Fatalf("expected reused pull request #42, got %#v", pr)
	}
	if githubClient.createCalls != 0 {
		t.Fatalf("expected no PR create calls, got %d", githubClient.createCalls)
	}
	if len(githubClient.ensureLabel) != 1 || githubClient.ensureLabel[0] != "heimdall-monitored" {
		t.Fatalf("expected PR monitor label reconciliation, got %#v", githubClient.ensureLabel)
	}
	if len(githubClient.addLabelCalls) != 1 || githubClient.addLabelCalls[0] != 42 {
		t.Fatalf("expected PR monitor label applied to PR #42, got %#v", githubClient.addLabelCalls)
	}
	if bootstrapRunner.request.IssueKey != workItem.WorkItemKey || bootstrapRunner.request.IssueTitle != workItem.Title {
		t.Fatalf("expected bootstrap request to include issue seed, got %#v", bootstrapRunner.request)
	}

	logOutput := logs.String()
	if !strings.Contains(logOutput, "run_bootstrap_prompt") || !strings.Contains(logOutput, "workflow_complete") {
		t.Fatalf("expected step-level workflow logs, got %s", logOutput)
	}
	if strings.Contains(logOutput, workItem.Description) {
		t.Fatalf("expected logs to omit raw issue description, got %s", logOutput)
	}
	if strings.Contains(logOutput, githubClient.token) {
		t.Fatalf("expected logs to omit installation token, got %s", logOutput)
	}
}

func TestBootstrapWorkflowExecuteBlocksWhenNoChangesProduced(t *testing.T) {
	ctx := context.Background()
	runtimeStore := testWorkflowStore(t)
	_, _, run := seedBootstrapRun(t, ctx, runtimeStore)

	repoMgr := &fakeBootstrapRepoManager{hasChanges: false}
	githubClient := &fakeBootstrapGitHubClient{token: "installation-token", installationOK: true}
	bootstrapRunner := &fakeBootstrapRunner{result: &exec.BootstrapResult{Summary: "No-op"}}
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	workflow := NewBootstrapWorkflow(runtimeStore, repoMgr, githubClient, bootstrapRunner, logger)
	err := workflow.Execute(ctx, run.ID)
	if err == nil {
		t.Fatal("expected Execute() to fail when no changes are produced")
	}

	retrievedRun, getErr := runtimeStore.GetWorkflowRun(ctx, run.ID)
	if getErr != nil {
		t.Fatalf("GetWorkflowRun() error = %v", getErr)
	}
	if retrievedRun.Status != "blocked" {
		t.Fatalf("expected blocked workflow run, got %q", retrievedRun.Status)
	}
	if retrievedRun.StatusReason != "bootstrap execution produced no file changes" {
		t.Fatalf("expected no-change reason, got %q", retrievedRun.StatusReason)
	}
	if !strings.Contains(logs.String(), "detect_changes") {
		t.Fatalf("expected detect_changes log entry, got %s", logs.String())
	}
}

func testWorkflowStore(t *testing.T) *store.Store {
	t.Helper()
	runtimeStore, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	if err := runtimeStore.Migrate(context.Background()); err != nil {
		t.Fatalf("store.Migrate() error = %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeStore.Close()
	})
	return runtimeStore
}

func seedBootstrapRun(t *testing.T, ctx context.Context, runtimeStore *store.Store) (*store.WorkItem, *store.Repository, *store.WorkflowRun) {
	t.Helper()
	repository := &store.Repository{
		Provider:        "github",
		RepoRef:         "github.com/acme/platform",
		Owner:           "acme",
		Name:            "platform",
		DefaultBranch:   "main",
		BranchPrefix:    "heimdall",
		LocalMirrorPath: "/var/lib/heimdall/repos/github.com/acme/platform.git",
		IsActive:        true,
	}
	if err := runtimeStore.SaveRepository(ctx, repository); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	workItem := &store.WorkItem{
		Provider:           "linear",
		ProviderWorkItemID: "linear-issue-1",
		WorkItemKey:        "ENG-123",
		Title:              "Add rate limiting",
		Description:        "Add rate limiting for API requests. SECRET issue details should stay out of logs.",
		StateName:          "In Progress",
		LifecycleBucket:    "active",
		Project:            "Core Platform",
		Team:               "ENG",
	}
	if err := runtimeStore.SaveWorkItem(ctx, workItem); err != nil {
		t.Fatalf("SaveWorkItem() error = %v", err)
	}

	run, err := CreateBootstrapWorkflowRun(ctx, runtimeStore, workItem.ID, repository, GenerateChangeName(workItem.WorkItemKey, "add-rate-limiting-for-api-requests"), GenerateBranchName(repository.BranchPrefix, workItem.WorkItemKey, "add-rate-limiting-for-api-requests"))
	if err != nil {
		t.Fatalf("CreateBootstrapWorkflowRun() error = %v", err)
	}

	return workItem, repository, run
}
