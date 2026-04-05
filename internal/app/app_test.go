package app

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mngeow/symphony/internal/board/linear"
	"github.com/mngeow/symphony/internal/config"
	"github.com/mngeow/symphony/internal/store"
	"github.com/mngeow/symphony/internal/workflow"
)

type stubActivationFlow struct {
	runID int64
	err   error
}

func (s *stubActivationFlow) Execute(_ context.Context, runID int64) error {
	s.runID = runID
	return s.err
}

func TestPollLinearOnceCreatesBootstrapWorkflowRun(t *testing.T) {
	ctx := context.Background()
	runtimeStore, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	defer runtimeStore.Close()
	if err := runtimeStore.Migrate(ctx); err != nil {
		t.Fatalf("store.Migrate() error = %v", err)
	}

	cfg := &config.Config{
		Linear: config.LinearConfig{
			PollInterval: 30 * time.Second,
			ActiveStates: []string{"In Progress"},
			ProjectName:  "Core Platform",
			APIToken:     "linear-token",
		},
		Repos: []config.RepoConfig{{
			Name:            "github.com/acme/platform",
			LocalMirrorPath: "/var/lib/symphony/repos/github.com/acme/platform.git",
			AllowedAgents:   []string{"gpt-5.4"},
			AllowedUsers:    []string{"mngeow"},
			DefaultBranch:   "main",
			BranchPrefix:    "symphony",
		}},
	}
	if err := syncConfiguredRepositories(ctx, runtimeStore, cfg.Repos); err != nil {
		t.Fatalf("syncConfiguredRepositories() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"issues":{"nodes":[{"id":"linear-issue-1","identifier":"ENG-123","title":"Add rate limiting","description":"More details","updatedAt":"2026-04-05T09:59:30Z","state":{"id":"state-1","name":"In Progress","type":"started"},"team":{"id":"team-1","key":"ENG","name":"Engineering"},"project":{"id":"project-1","name":"Core Platform"},"labels":{"nodes":[]}}],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}`))
	}))
	defer server.Close()

	fakeFlow := &stubActivationFlow{}
	application := &App{
		config:         cfg,
		logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		store:          runtimeStore,
		linearProvider: linear.NewProvider(cfg.Linear.APIToken, cfg.Linear.ProjectName, cfg.Linear.ActiveStates, cfg.Linear.PollInterval, runtimeStore, linear.WithEndpoint(server.URL), linear.WithHTTPClient(server.Client()), linear.WithNow(func() time.Time { return time.Date(2026, time.April, 5, 10, 0, 0, 0, time.UTC) })),
		router:         workflow.NewRouter(cfg.Repos),
		activationFlow: fakeFlow,
	}

	if err := application.pollLinearOnce(ctx); err != nil {
		t.Fatalf("pollLinearOnce() error = %v", err)
	}

	snapshot, err := runtimeStore.GetWorkItemByKey(ctx, "linear", "ENG-123")
	if err != nil {
		t.Fatalf("GetWorkItemByKey() error = %v", err)
	}
	if snapshot == nil {
		t.Fatal("expected saved Linear work item snapshot")
	}
	if snapshot.Description != "More details" {
		t.Fatalf("expected saved description, got %q", snapshot.Description)
	}

	if fakeFlow.runID == 0 {
		t.Fatal("expected activation workflow executor to be invoked")
	}

	run, err := runtimeStore.GetWorkflowRun(ctx, fakeFlow.runID)
	if err != nil {
		t.Fatalf("GetWorkflowRun() error = %v", err)
	}
	if run == nil {
		t.Fatal("expected workflow run to be created")
	}
	if run.RunType != "bootstrap_pull_request" {
		t.Fatalf("expected bootstrap run type, got %q", run.RunType)
	}
	if run.BranchName != "symphony/ENG-123-more-details" {
		t.Fatalf("expected description-seeded branch, got %q", run.BranchName)
	}
}
