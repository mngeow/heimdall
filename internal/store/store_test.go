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
			StateName:          "In Progress",
			LifecycleBucket:    "active",
			Team:               "ENG",
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
		}
	})

	t.Run("Repository", func(t *testing.T) {
		repo := &Repository{
			Provider:        "github",
			RepoRef:         "github.com/test/repo",
			Owner:           "test",
			Name:            "repo",
			DefaultBranch:   "main",
			LocalMirrorPath: "/var/lib/symphony/repos/github.com/test/repo.git",
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
		}
	})
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
