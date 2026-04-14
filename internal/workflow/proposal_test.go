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

type fakeProposalRepoManager struct {
	hasChanges bool
	commitSHA  string
	steps      []string
}

func (f *fakeProposalRepoManager) EnsureBareMirror(context.Context, string, string, string, string) error {
	f.steps = append(f.steps, "ensure_mirror")
	return nil
}

func (f *fakeProposalRepoManager) CreateWorktree(context.Context, string, string, string, string) error {
	f.steps = append(f.steps, "create_worktree")
	return nil
}

func (f *fakeProposalRepoManager) HasChanges(context.Context, string) (bool, error) {
	f.steps = append(f.steps, "has_changes")
	return f.hasChanges, nil
}

func (f *fakeProposalRepoManager) CommitAll(context.Context, string, string) (string, error) {
	f.steps = append(f.steps, "commit")
	if !f.hasChanges {
		return "", errors.New("commit should not run without changes")
	}
	return f.commitSHA, nil
}

func (f *fakeProposalRepoManager) PushBranch(context.Context, string, string, string, string, string) error {
	f.steps = append(f.steps, "push")
	return nil
}

type fakeProposalGitHubClient struct {
	token          string
	existingPR     *gh.PullRequest
	createdPR      *gh.PullRequest
	createCalls    int
	findCalls      int
	ensureLabel    []string
	addLabelCalls  []int
	installationOK bool
}

func (f *fakeProposalGitHubClient) GetInstallationToken(context.Context) (string, error) {
	if !f.installationOK {
		return "", errors.New("installation token error")
	}
	return f.token, nil
}

func (f *fakeProposalGitHubClient) FindOpenPullRequestByHead(context.Context, string, string, string, string) (*gh.PullRequest, error) {
	f.findCalls++
	return f.existingPR, nil
}

func (f *fakeProposalGitHubClient) CreatePullRequest(context.Context, string, string, string, string, string, string) (*gh.PullRequest, error) {
	f.createCalls++
	return f.createdPR, nil
}

func (f *fakeProposalGitHubClient) EnsurePRMonitorLabel(_ context.Context, _, _, label string) error {
	f.ensureLabel = append(f.ensureLabel, label)
	return nil
}

func (f *fakeProposalGitHubClient) AddPRMonitorLabel(_ context.Context, _, _ string, number int, _ string) error {
	f.addLabelCalls = append(f.addLabelCalls, number)
	return nil
}

type fakeOpenSpecClient struct {
	createChangeCalls         int
	getStatusCalls            int
	getApplyInstructionsCalls int
	listChangesCalls          int
	worktreePath              string
	applyInstructions         *exec.ApplyInstructions
	changesBefore             []string
	changesAfter              []string
}

func (f *fakeOpenSpecClient) SetWorktreePath(worktreePath string) {
	f.worktreePath = worktreePath
}

func (f *fakeOpenSpecClient) CreateChange(_ context.Context, _ string) error {
	f.createChangeCalls++
	return nil
}

func (f *fakeOpenSpecClient) GetStatus(_ context.Context, _ string) (*exec.ChangeStatus, error) {
	f.getStatusCalls++
	return &exec.ChangeStatus{Name: "eng-123-add-rate-limiting-for-api-requests", Status: "in-progress"}, nil
}

func (f *fakeOpenSpecClient) GetApplyInstructions(_ context.Context, _ string) (*exec.ApplyInstructions, error) {
	f.getApplyInstructionsCalls++
	if f.applyInstructions != nil {
		return f.applyInstructions, nil
	}
	return &exec.ApplyInstructions{State: "ready"}, nil
}

func (f *fakeOpenSpecClient) ListChanges(_ context.Context) ([]string, error) {
	f.listChangesCalls++
	if f.listChangesCalls == 1 {
		return f.changesBefore, nil
	}
	return f.changesAfter, nil
}

type fakeProposalRunner struct {
	request exec.ProposalRequest
	result  *exec.ProposalResult
	err     error
}

func (f *fakeProposalRunner) RunProposal(_ context.Context, req exec.ProposalRequest) (*exec.ProposalResult, error) {
	f.request = req
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func TestProposalWorkflowExecuteSuccessReusesExistingPullRequest(t *testing.T) {
	ctx := context.Background()
	runtimeStore := testWorkflowStore(t)
	workItem, repository, run := seedProposalRun(t, ctx, runtimeStore)
	repository.PRMonitorLabel = "heimdall-monitored"
	repository.DefaultSpecWritingAgent = "gpt-5.4"
	if err := runtimeStore.SaveRepository(ctx, repository); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	repoMgr := &fakeProposalRepoManager{hasChanges: true, commitSHA: "abc123"}
	githubClient := &fakeProposalGitHubClient{
		token:          "installation-token",
		existingPR:     &gh.PullRequest{Number: gh.Int(42), NodeID: gh.String("PR_node_42"), Title: gh.String("[ENG-123] OpenSpec proposal for Add rate limiting"), State: gh.String("open"), HTMLURL: gh.String("https://example.test/pr/42")},
		installationOK: true,
	}
	openSpecClient := &fakeOpenSpecClient{
		applyInstructions: &exec.ApplyInstructions{State: "ready"},
		changesBefore:     []string{},
		changesAfter:      []string{"eng-123-add-rate-limiting"},
	}
	proposalRunner := &fakeProposalRunner{result: &exec.ProposalResult{Summary: "Generated OpenSpec proposal artifacts for issue ENG-123 from the activation seed."}}
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	workflow := NewProposalWorkflow(runtimeStore, repoMgr, githubClient, openSpecClient, proposalRunner, logger)
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
	if binding.ChangeName != "eng-123-add-rate-limiting" {
		t.Fatalf("expected binding change name eng-123-add-rate-limiting, got %q", binding.ChangeName)
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
	if proposalRunner.request.IssueKey != workItem.WorkItemKey || proposalRunner.request.IssueTitle != workItem.Title {
		t.Fatalf("expected proposal request to include issue seed, got %#v", proposalRunner.request)
	}
	if proposalRunner.request.Agent != "gpt-5.4" {
		t.Fatalf("expected proposal agent gpt-5.4, got %q", proposalRunner.request.Agent)
	}

	logOutput := logs.String()
	if !strings.Contains(logOutput, "run_proposal_prompt") || !strings.Contains(logOutput, "workflow_complete") {
		t.Fatalf("expected step-level workflow logs, got %s", logOutput)
	}
	if strings.Contains(logOutput, workItem.Description) {
		t.Fatalf("expected logs to omit raw issue description, got %s", logOutput)
	}
	if strings.Contains(logOutput, githubClient.token) {
		t.Fatalf("expected logs to omit installation token, got %s", logOutput)
	}
}

func TestProposalWorkflowExecuteBlocksWhenNoChangesProduced(t *testing.T) {
	ctx := context.Background()
	runtimeStore := testWorkflowStore(t)
	_, _, run := seedProposalRun(t, ctx, runtimeStore)

	repoMgr := &fakeProposalRepoManager{hasChanges: false}
	githubClient := &fakeProposalGitHubClient{token: "installation-token", installationOK: true}
	openSpecClient := &fakeOpenSpecClient{
		applyInstructions: &exec.ApplyInstructions{State: "ready"},
		changesBefore:     []string{},
		changesAfter:      []string{"eng-123-add-rate-limiting"},
	}
	proposalRunner := &fakeProposalRunner{result: &exec.ProposalResult{Summary: "No-op", ChangesBefore: []string{}}}
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	workflow := NewProposalWorkflow(runtimeStore, repoMgr, githubClient, openSpecClient, proposalRunner, logger)
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
	if retrievedRun.StatusReason != "proposal execution produced no file changes" {
		t.Fatalf("expected no-change reason, got %q", retrievedRun.StatusReason)
	}
	if !strings.Contains(logs.String(), "detect_changes") {
		t.Fatalf("expected detect_changes log entry, got %s", logs.String())
	}
}

func TestProposalWorkflowExecuteReusesExistingBinding(t *testing.T) {
	ctx := context.Background()
	runtimeStore := testWorkflowStore(t)
	workItem, repository, run := seedProposalRun(t, ctx, runtimeStore)

	// Pre-create an active binding
	binding := &store.RepoBinding{
		WorkItemID:    workItem.ID,
		RepositoryID:  repository.ID,
		BranchName:    run.BranchName,
		ChangeName:    run.ChangeName,
		BindingStatus: "active",
	}
	if err := runtimeStore.SaveRepoBinding(ctx, binding); err != nil {
		t.Fatalf("SaveRepoBinding() error = %v", err)
	}

	repoMgr := &fakeProposalRepoManager{hasChanges: true, commitSHA: "abc123"}
	githubClient := &fakeProposalGitHubClient{installationOK: true}
	openSpecClient := &fakeOpenSpecClient{}
	proposalRunner := &fakeProposalRunner{}
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	workflow := NewProposalWorkflow(runtimeStore, repoMgr, githubClient, openSpecClient, proposalRunner, logger)
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
	if repoMgr.steps != nil {
		t.Fatalf("expected no repo manager steps when binding is reused, got %v", repoMgr.steps)
	}
}

func TestProposalWorkflowSetsOpenSpecWorktreePath(t *testing.T) {
	ctx := context.Background()
	runtimeStore := testWorkflowStore(t)
	workItem, repository, run := seedProposalRun(t, ctx, runtimeStore)

	_ = workItem
	_ = repository
	repoMgr := &fakeProposalRepoManager{hasChanges: true, commitSHA: "abc123"}
	githubClient := &fakeProposalGitHubClient{token: "installation-token", installationOK: true}
	openSpecClient := &fakeOpenSpecClient{
		applyInstructions: &exec.ApplyInstructions{State: "ready"},
		changesBefore:     []string{},
		changesAfter:      []string{"eng-123-add-rate-limiting"},
	}
	proposalRunner := &fakeProposalRunner{result: &exec.ProposalResult{Summary: "Generated OpenSpec proposal artifacts for issue ENG-123 from the activation seed."}}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	workflow := NewProposalWorkflow(runtimeStore, repoMgr, githubClient, openSpecClient, proposalRunner, logger)
	if err := workflow.Execute(ctx, run.ID); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	expectedWorktreePath := run.WorktreePath
	if openSpecClient.worktreePath != expectedWorktreePath {
		t.Fatalf("expected OpenSpec client worktree path %q, got %q", expectedWorktreePath, openSpecClient.worktreePath)
	}
}

func TestProposalWorkflowBlocksWhenApplyInstructionsBlocked(t *testing.T) {
	ctx := context.Background()
	runtimeStore := testWorkflowStore(t)
	_, _, run := seedProposalRun(t, ctx, runtimeStore)

	repoMgr := &fakeProposalRepoManager{hasChanges: true, commitSHA: "abc123"}
	githubClient := &fakeProposalGitHubClient{token: "installation-token", installationOK: true}
	openSpecClient := &fakeOpenSpecClient{
		applyInstructions: &exec.ApplyInstructions{State: "blocked"},
		changesBefore:     []string{},
		changesAfter:      []string{"eng-123-add-rate-limiting"},
	}
	proposalRunner := &fakeProposalRunner{result: &exec.ProposalResult{Summary: "Generated OpenSpec proposal artifacts for issue ENG-123 from the activation seed."}}
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	workflow := NewProposalWorkflow(runtimeStore, repoMgr, githubClient, openSpecClient, proposalRunner, logger)
	err := workflow.Execute(ctx, run.ID)
	if err == nil {
		t.Fatal("expected Execute() to fail when apply instructions are blocked")
	}

	retrievedRun, getErr := runtimeStore.GetWorkflowRun(ctx, run.ID)
	if getErr != nil {
		t.Fatalf("GetWorkflowRun() error = %v", getErr)
	}
	if retrievedRun.Status != "failed" {
		t.Fatalf("expected failed workflow run, got %q", retrievedRun.Status)
	}
	if !strings.Contains(retrievedRun.StatusReason, "blocked") {
		t.Fatalf("expected blocked reason, got %q", retrievedRun.StatusReason)
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

func seedProposalRun(t *testing.T, ctx context.Context, runtimeStore *store.Store) (*store.WorkItem, *store.Repository, *store.WorkflowRun) {
	t.Helper()
	repository := &store.Repository{
		Provider:                "github",
		RepoRef:                 "github.com/acme/platform",
		Owner:                   "acme",
		Name:                    "platform",
		DefaultBranch:           "main",
		BranchPrefix:            "heimdall",
		LocalMirrorPath:         "/var/lib/heimdall/repos/github.com/acme/platform.git",
		DefaultSpecWritingAgent: "gpt-5.4",
		IsActive:                true,
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

	branchName := GenerateBranchName(repository.BranchPrefix, workItem.WorkItemKey, workItem.Title)
	run, err := CreateProposalWorkflowRun(ctx, runtimeStore, workItem.ID, repository, branchName)
	if err != nil {
		t.Fatalf("CreateProposalWorkflowRun() error = %v", err)
	}

	return workItem, repository, run
}
