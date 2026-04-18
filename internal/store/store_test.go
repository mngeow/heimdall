package store

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestStore(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store for testing
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Run migrations
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	t.Run("ProviderCursor", func(t *testing.T) {
		cursor := &ProviderCursor{
			Provider:    "linear",
			ScopeKey:    "team:ENG",
			CursorValue: "cursor123",
			CursorKind:  "updated_at",
		}

		// Save cursor
		if err := store.SetProviderCursor(ctx, cursor); err != nil {
			t.Errorf("failed to set cursor: %v", err)
		}

		// Retrieve cursor
		retrieved, err := store.GetProviderCursor(ctx, "linear", "team:ENG")
		if err != nil {
			t.Errorf("failed to get cursor: %v", err)
		}
		if retrieved == nil {
			t.Error("expected cursor, got nil")
		} else if retrieved.CursorValue != "cursor123" {
			t.Errorf("expected cursor123, got %s", retrieved.CursorValue)
		}
	})

	t.Run("WorkItem", func(t *testing.T) {
		item := &WorkItem{
			Provider:           "linear",
			ProviderWorkItemID: "linear-123",
			WorkItemKey:        "ENG-123",
			Title:              "Test Issue",
			Description:        "Detailed description",
			StateName:          "In Progress",
			LifecycleBucket:    "active",
			Project:            "Core Platform",
			Team:               "ENG",
			Labels:             []string{"backend", "urgent"},
		}

		// Save work item
		if err := store.SaveWorkItem(ctx, item); err != nil {
			t.Errorf("failed to save work item: %v", err)
		}

		// Retrieve work item
		retrieved, err := store.GetWorkItemByKey(ctx, "linear", "ENG-123")
		if err != nil {
			t.Errorf("failed to get work item: %v", err)
		}
		if retrieved == nil {
			t.Error("expected work item, got nil")
		} else if retrieved.Title != "Test Issue" {
			t.Errorf("expected 'Test Issue', got %s", retrieved.Title)
		} else if retrieved.Description != "Detailed description" {
			t.Errorf("expected description to round-trip, got %q", retrieved.Description)
		} else if retrieved.Project != "Core Platform" {
			t.Errorf("expected project to round-trip, got %q", retrieved.Project)
		} else if len(retrieved.Labels) != 2 || retrieved.Labels[0] != "backend" || retrieved.Labels[1] != "urgent" {
			t.Errorf("expected labels to round-trip, got %#v", retrieved.Labels)
		}
	})

	t.Run("Repository", func(t *testing.T) {
		repo := &Repository{
			Provider:        "github",
			RepoRef:         "github.com/test/repo",
			Owner:           "test",
			Name:            "repo",
			DefaultBranch:   "main",
			LocalMirrorPath: "/var/lib/heimdall/repos/github.com/test/repo.git",
			IsActive:        true,
		}

		// Save repository
		if err := store.SaveRepository(ctx, repo); err != nil {
			t.Errorf("failed to save repository: %v", err)
		}

		// Retrieve repository
		retrieved, err := store.GetRepositoryByRef(ctx, "github.com/test/repo")
		if err != nil {
			t.Errorf("failed to get repository: %v", err)
		}
		if retrieved == nil {
			t.Error("expected repository, got nil")
		} else if retrieved.Owner != "test" {
			t.Errorf("expected owner 'test', got %s", retrieved.Owner)
		} else if retrieved.PRMonitorLabel != "" {
			t.Errorf("expected empty PR monitor label by default, got %q", retrieved.PRMonitorLabel)
		}
	})

	t.Run("RepositoryPRMonitorLabel", func(t *testing.T) {
		repo := &Repository{
			Provider:        "github",
			RepoRef:         "github.com/test/monitored",
			Owner:           "test",
			Name:            "monitored",
			DefaultBranch:   "main",
			BranchPrefix:    "heimdall",
			PRMonitorLabel:  "heimdall-monitored",
			LocalMirrorPath: "/var/lib/heimdall/repos/github.com/test/monitored.git",
			IsActive:        true,
		}

		if err := store.SaveRepository(ctx, repo); err != nil {
			t.Errorf("failed to save repository: %v", err)
		}

		retrieved, err := store.GetRepositoryByRef(ctx, repo.RepoRef)
		if err != nil {
			t.Errorf("failed to get repository: %v", err)
		}
		if retrieved == nil {
			t.Error("expected repository, got nil")
		} else if retrieved.PRMonitorLabel != "heimdall-monitored" {
			t.Errorf("expected PR monitor label to round-trip, got %q", retrieved.PRMonitorLabel)
		}
	})
}

func TestWorkflowRunStatusReason(t *testing.T) {
	ctx := context.Background()
	runtimeStore, err := New(":memory:")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer runtimeStore.Close()

	if err := runtimeStore.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if err := runtimeStore.SaveRepository(ctx, &Repository{
		Provider:        "github",
		RepoRef:         "github.com/test/repo",
		Owner:           "test",
		Name:            "repo",
		DefaultBranch:   "main",
		BranchPrefix:    "heimdall",
		LocalMirrorPath: "/tmp/test-repo.git",
		IsActive:        true,
	}); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	workItem := &WorkItem{
		Provider:           "linear",
		ProviderWorkItemID: "linear-123",
		WorkItemKey:        "ENG-123",
		Title:              "Test Issue",
		StateName:          "In Progress",
		LifecycleBucket:    "active",
	}
	if err := runtimeStore.SaveWorkItem(ctx, workItem); err != nil {
		t.Fatalf("SaveWorkItem() error = %v", err)
	}
	repoRecord, err := runtimeStore.GetRepositoryByRef(ctx, "github.com/test/repo")
	if err != nil {
		t.Fatalf("GetRepositoryByRef() error = %v", err)
	}

	run := &WorkflowRun{
		WorkItemID:      workItem.ID,
		RepositoryID:    repoRecord.ID,
		RunType:         "bootstrap_pull_request",
		Status:          "queued",
		ChangeName:      "ENG-123-test-issue",
		BranchName:      "heimdall/ENG-123-test-issue",
		WorktreePath:    "/tmp/worktree",
		RequestedByType: "system",
	}
	if err := runtimeStore.CreateWorkflowRun(ctx, run); err != nil {
		t.Fatalf("CreateWorkflowRun() error = %v", err)
	}

	if err := runtimeStore.UpdateWorkflowRunStatus(ctx, run.ID, "blocked", "bootstrap execution produced no file changes"); err != nil {
		t.Fatalf("UpdateWorkflowRunStatus() error = %v", err)
	}

	retrieved, err := runtimeStore.GetWorkflowRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetWorkflowRun() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected workflow run, got nil")
	}
	if retrieved.Status != "blocked" {
		t.Fatalf("expected blocked status, got %q", retrieved.Status)
	}
	if retrieved.StatusReason != "bootstrap execution produced no file changes" {
		t.Fatalf("expected blocked reason to round-trip, got %q", retrieved.StatusReason)
	}
}

func TestJobQueue(t *testing.T) {
	ctx := context.Background()

	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	queue := NewJobQueue(store)

	t.Run("EnqueueAndDequeue", func(t *testing.T) {
		job := &Job{
			JobType:     "propose",
			LockKey:     "issue:linear:ENG-123",
			Status:      "queued",
			Priority:    100,
			RunAfter:    time.Now(),
			MaxAttempts: 3,
		}

		// Enqueue job
		if err := queue.Enqueue(ctx, job); err != nil {
			t.Errorf("failed to enqueue: %v", err)
		}

		// Dequeue job
		dequeued, err := queue.Dequeue(ctx)
		if err != nil {
			t.Errorf("failed to dequeue: %v", err)
		}
		if dequeued == nil {
			t.Error("expected job, got nil")
		} else if dequeued.JobType != "propose" {
			t.Errorf("expected job type 'propose', got %s", dequeued.JobType)
		}

		// Complete job
		if err := queue.Complete(ctx, dequeued.ID); err != nil {
			t.Errorf("failed to complete: %v", err)
		}
	})
}

func TestPendingPermissionRequest(t *testing.T) {
	ctx := context.Background()
	runtimeStore, err := New(":memory:")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer runtimeStore.Close()

	if err := runtimeStore.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if err := runtimeStore.SaveRepository(ctx, &Repository{
		Provider:        "github",
		RepoRef:         "github.com/test/repo",
		Owner:           "test",
		Name:            "repo",
		DefaultBranch:   "main",
		BranchPrefix:    "heimdall",
		LocalMirrorPath: "/tmp/test-repo.git",
		IsActive:        true,
	}); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	repoRecord, err := runtimeStore.GetRepositoryByRef(ctx, "github.com/test/repo")
	if err != nil {
		t.Fatalf("GetRepositoryByRef() error = %v", err)
	}

	pr := &PullRequest{
		RepositoryID: repoRecord.ID,
		Number:       42,
		HeadBranch:   "heimdall/test",
		BaseBranch:   "main",
		State:        "open",
	}
	if err := runtimeStore.SavePullRequest(ctx, pr); err != nil {
		t.Fatalf("SavePullRequest() error = %v", err)
	}

	t.Run("CreateAndGetPendingPermissionRequest", func(t *testing.T) {
		req := &PendingPermissionRequest{
			RequestID:        "perm_123",
			SessionID:        "sess_456",
			CommandRequestID: 1,
			PullRequestID:    pr.ID,
			RepositoryID:     repoRecord.ID,
			Status:           "pending",
		}
		if err := runtimeStore.CreatePendingPermissionRequest(ctx, req); err != nil {
			t.Fatalf("CreatePendingPermissionRequest() error = %v", err)
		}
		if req.ID == 0 {
			t.Fatal("expected pending permission request ID to be set")
		}

		retrieved, err := runtimeStore.GetPendingPermissionRequestByID(ctx, "perm_123")
		if err != nil {
			t.Fatalf("GetPendingPermissionRequestByID() error = %v", err)
		}
		if retrieved == nil {
			t.Fatal("expected pending permission request, got nil")
		}
		if retrieved.RequestID != "perm_123" {
			t.Errorf("RequestID = %q, want %q", retrieved.RequestID, "perm_123")
		}
		if retrieved.Status != "pending" {
			t.Errorf("Status = %q, want %q", retrieved.Status, "pending")
		}
	})

	t.Run("ResolvePendingPermissionRequest", func(t *testing.T) {
		if err := runtimeStore.ResolvePendingPermissionRequest(ctx, "perm_123", "approved"); err != nil {
			t.Fatalf("ResolvePendingPermissionRequest() error = %v", err)
		}

		retrieved, err := runtimeStore.GetPendingPermissionRequestByID(ctx, "perm_123")
		if err != nil {
			t.Fatalf("GetPendingPermissionRequestByID() error = %v", err)
		}
		if retrieved == nil {
			t.Fatal("expected pending permission request, got nil")
		}
		if retrieved.Status != "approved" {
			t.Errorf("Status = %q, want %q", retrieved.Status, "approved")
		}
		if retrieved.ResolvedAt == nil {
			t.Error("expected ResolvedAt to be set")
		}
	})

	t.Run("GetUnknownRequestIDReturnsNil", func(t *testing.T) {
		retrieved, err := runtimeStore.GetPendingPermissionRequestByID(ctx, "perm_unknown")
		if err != nil {
			t.Fatalf("GetPendingPermissionRequestByID() error = %v", err)
		}
		if retrieved != nil {
			t.Errorf("expected nil for unknown request ID, got %+v", retrieved)
		}
	})
}
