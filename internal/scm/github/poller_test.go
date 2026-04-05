package github

import (
	"context"
	"strconv"
	"testing"
	"time"

	gh "github.com/google/go-github/v57/github"
	"github.com/mngeow/symphony/internal/store"
)

func TestPollerFiltersManagedCommentsAndReconcilesPullRequests(t *testing.T) {
	ctx := context.Background()
	runtimeStore := newPollerTestStore(t, ctx)

	repository := &store.Repository{
		Provider:        "github",
		RepoRef:         "github.com/acme/platform",
		Owner:           "acme",
		Name:            "platform",
		DefaultBranch:   "main",
		BranchPrefix:    "symphony",
		LocalMirrorPath: "/tmp/platform.git",
		IsActive:        true,
	}
	if err := runtimeStore.SaveRepository(ctx, repository); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	bindingID := int64(7)
	pullRequest := &store.PullRequest{
		RepositoryID:  repository.ID,
		RepoBindingID: &bindingID,
		Provider:      "github",
		Number:        42,
		Title:         "Old title",
		BaseBranch:    "main",
		HeadBranch:    "symphony/eng-123-add-rate-limiting",
		State:         "open",
		URL:           "https://github.com/acme/platform/pull/42",
	}
	if err := runtimeStore.SavePullRequest(ctx, pullRequest); err != nil {
		t.Fatalf("SavePullRequest() error = %v", err)
	}

	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)
	api := &fakePollAPI{
		comments: []*gh.IssueComment{
			newIssueComment(42, "IC_1", "/symphony status", "alice", now.Add(-30*time.Second)),
			newIssueComment(99, "IC_2", "/symphony status", "bob", now.Add(-30*time.Second)),
		},
		pullRequests: map[int]*gh.PullRequest{
			42: {
				NodeID:  gh.String("PR_node_42"),
				Title:   gh.String("Updated title"),
				Base:    &gh.PullRequestBranch{Ref: gh.String("main")},
				Head:    &gh.PullRequestBranch{Ref: gh.String("symphony/eng-123-add-rate-limiting")},
				State:   gh.String("closed"),
				HTMLURL: gh.String("https://github.com/acme/platform/pull/42"),
			},
		},
	}

	poller := NewPoller(api, runtimeStore, 2*time.Minute)
	poller.now = func() time.Time { return now }

	result, err := poller.Poll(ctx)
	if err != nil {
		t.Fatalf("Poll() error = %v", err)
	}

	if len(result.Commands) != 1 {
		t.Fatalf("expected 1 discovered command, got %d", len(result.Commands))
	}
	if result.Commands[0].CommentNodeID != "IC_1" {
		t.Fatalf("expected managed comment IC_1, got %q", result.Commands[0].CommentNodeID)
	}
	if len(result.Reconciled) != 1 {
		t.Fatalf("expected 1 reconciled pull request, got %d", len(result.Reconciled))
	}

	checkpoint, err := runtimeStore.GetGitHubPollCheckpoint(ctx, repository.RepoRef)
	if err != nil {
		t.Fatalf("GetGitHubPollCheckpoint() error = %v", err)
	}
	if checkpoint == nil || !checkpoint.Equal(now) {
		t.Fatalf("expected checkpoint %s, got %v", now, checkpoint)
	}

	storedPR, err := runtimeStore.GetPullRequestByNumber(ctx, repository.ID, 42)
	if err != nil {
		t.Fatalf("GetPullRequestByNumber() error = %v", err)
	}
	if storedPR.State != "closed" {
		t.Fatalf("expected reconciled pull request state closed, got %q", storedPR.State)
	}
}

func TestPollerUsesCheckpointOverlapWindow(t *testing.T) {
	ctx := context.Background()
	runtimeStore := newPollerTestStore(t, ctx)

	repository := &store.Repository{
		Provider:        "github",
		RepoRef:         "github.com/acme/platform",
		Owner:           "acme",
		Name:            "platform",
		DefaultBranch:   "main",
		BranchPrefix:    "symphony",
		LocalMirrorPath: "/tmp/platform.git",
		IsActive:        true,
	}
	if err := runtimeStore.SaveRepository(ctx, repository); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	bindingID := int64(7)
	if err := runtimeStore.SavePullRequest(ctx, &store.PullRequest{
		RepositoryID:  repository.ID,
		RepoBindingID: &bindingID,
		Provider:      "github",
		Number:        42,
		Title:         "Tracked PR",
		BaseBranch:    "main",
		HeadBranch:    "symphony/eng-123-add-rate-limiting",
		State:         "open",
		URL:           "https://github.com/acme/platform/pull/42",
	}); err != nil {
		t.Fatalf("SavePullRequest() error = %v", err)
	}

	lastCheckpoint := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)
	if err := runtimeStore.SetGitHubPollCheckpoint(ctx, repository.RepoRef, lastCheckpoint); err != nil {
		t.Fatalf("SetGitHubPollCheckpoint() error = %v", err)
	}

	api := &fakePollAPI{
		pullRequests: map[int]*gh.PullRequest{
			42: {
				NodeID:  gh.String("PR_node_42"),
				Title:   gh.String("Tracked PR"),
				Base:    &gh.PullRequestBranch{Ref: gh.String("main")},
				Head:    &gh.PullRequestBranch{Ref: gh.String("symphony/eng-123-add-rate-limiting")},
				State:   gh.String("open"),
				HTMLURL: gh.String("https://github.com/acme/platform/pull/42"),
			},
		},
	}

	poller := NewPoller(api, runtimeStore, 2*time.Minute)
	poller.now = func() time.Time { return lastCheckpoint.Add(5 * time.Minute) }

	if _, err := poller.Poll(ctx); err != nil {
		t.Fatalf("Poll() error = %v", err)
	}

	wantSince := lastCheckpoint.Add(-2 * time.Minute)
	if len(api.sinceCalls) != 1 || !api.sinceCalls[0].Equal(wantSince) {
		t.Fatalf("expected poll since %s, got %v", wantSince, api.sinceCalls)
	}
}

func TestPollerIgnoresUnlabeledPullRequestsWhenMonitorLabelConfigured(t *testing.T) {
	ctx := context.Background()
	runtimeStore := newPollerTestStore(t, ctx)

	repository := &store.Repository{
		Provider:        "github",
		RepoRef:         "github.com/acme/platform",
		Owner:           "acme",
		Name:            "platform",
		DefaultBranch:   "main",
		BranchPrefix:    "symphony",
		PRMonitorLabel:  "symphony-monitored",
		LocalMirrorPath: "/tmp/platform.git",
		IsActive:        true,
	}
	if err := runtimeStore.SaveRepository(ctx, repository); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	bindingID := int64(7)
	if err := runtimeStore.SavePullRequest(ctx, &store.PullRequest{
		RepositoryID:  repository.ID,
		RepoBindingID: &bindingID,
		Provider:      "github",
		Number:        42,
		Title:         "Tracked PR",
		BaseBranch:    "main",
		HeadBranch:    "symphony/eng-123-add-rate-limiting",
		State:         "open",
		URL:           "https://github.com/acme/platform/pull/42",
	}); err != nil {
		t.Fatalf("SavePullRequest() error = %v", err)
	}

	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)
	api := &fakePollAPI{
		comments: []*gh.IssueComment{
			newIssueComment(42, "IC_1", "/symphony status", "alice", now.Add(-30*time.Second)),
		},
		pullRequests: map[int]*gh.PullRequest{
			42: {
				NodeID:  gh.String("PR_node_42"),
				Title:   gh.String("Updated title"),
				Base:    &gh.PullRequestBranch{Ref: gh.String("main")},
				Head:    &gh.PullRequestBranch{Ref: gh.String("symphony/eng-123-add-rate-limiting")},
				State:   gh.String("open"),
				HTMLURL: gh.String("https://github.com/acme/platform/pull/42"),
			},
		},
	}

	poller := NewPoller(api, runtimeStore, 2*time.Minute)
	poller.now = func() time.Time { return now }

	result, err := poller.Poll(ctx)
	if err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	if len(result.Commands) != 0 {
		t.Fatalf("expected no discovered commands for unlabeled PR, got %d", len(result.Commands))
	}
	if len(result.Reconciled) != 0 {
		t.Fatalf("expected no reconciled PRs for unlabeled PR, got %d", len(result.Reconciled))
	}

	storedPR, err := runtimeStore.GetPullRequestByNumber(ctx, repository.ID, 42)
	if err != nil {
		t.Fatalf("GetPullRequestByNumber() error = %v", err)
	}
	if storedPR.Title != "Tracked PR" {
		t.Fatalf("expected PR title to remain unchanged, got %q", storedPR.Title)
	}
	if len(api.sinceCalls) != 0 {
		t.Fatalf("expected no issue comment polling when no labeled PRs are eligible, got %v", api.sinceCalls)
	}
}

type fakePollAPI struct {
	comments     []*gh.IssueComment
	pullRequests map[int]*gh.PullRequest
	sinceCalls   []time.Time
}

func (f *fakePollAPI) ListIssueCommentsSince(_ context.Context, owner, repo string, since time.Time) ([]*gh.IssueComment, error) {
	f.sinceCalls = append(f.sinceCalls, since)
	return f.comments, nil
}

func (f *fakePollAPI) GetPullRequest(_ context.Context, owner, repo string, number int) (*gh.PullRequest, error) {
	return f.pullRequests[number], nil
}

func newIssueComment(number int, nodeID, body, actor string, when time.Time) *gh.IssueComment {
	return &gh.IssueComment{
		NodeID:    gh.String(nodeID),
		Body:      gh.String(body),
		IssueURL:  gh.String("https://api.github.com/repos/acme/platform/issues/" + strconv.Itoa(number)),
		User:      &gh.User{Login: gh.String(actor)},
		CreatedAt: &gh.Timestamp{Time: when},
		UpdatedAt: &gh.Timestamp{Time: when},
	}
}

func newPollerTestStore(t *testing.T, ctx context.Context) *store.Store {
	t.Helper()
	runtimeStore, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	t.Cleanup(func() {
		runtimeStore.Close()
	})
	if err := runtimeStore.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	return runtimeStore
}
