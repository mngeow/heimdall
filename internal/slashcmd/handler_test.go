package slashcmd

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/mngeow/symphony/internal/config"
	"github.com/mngeow/symphony/internal/store"
)

func TestParser(t *testing.T) {
	parser := NewParser(nil)

	tests := []struct {
		name     string
		comment  string
		expected string
		isValid  bool
	}{
		{
			name:     "status command",
			comment:  "/symphony status",
			expected: "status",
			isValid:  true,
		},
		{
			name:     "refine command",
			comment:  "/symphony refine Add error handling",
			expected: "refine",
			isValid:  true,
		},
		{
			name:     "apply command with agent",
			comment:  "/opsx-apply --agent gpt-5.4",
			expected: "apply",
			isValid:  true,
		},
		{
			name:     "apply command without agent",
			comment:  "/opsx-apply",
			expected: "apply",
			isValid:  false,
		},
		{
			name:     "no command",
			comment:  "This is just a regular comment",
			expected: "",
			isValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := parser.Parse(tt.comment)
			if tt.expected == "" {
				if cmd != nil {
					t.Errorf("expected nil command, got %v", cmd)
				}
				return
			}

			if cmd == nil {
				t.Fatalf("expected command, got nil")
			}

			if cmd.Name != tt.expected {
				t.Errorf("expected command %q, got %q", tt.expected, cmd.Name)
			}

			if cmd.IsValid != tt.isValid {
				t.Errorf("expected IsValid=%v, got %v", tt.isValid, cmd.IsValid)
			}
		})
	}
}

func TestAuthorizer(t *testing.T) {
	repoConfig := config.RepoConfig{
		AllowedUsers:  []string{"alice", "bob"},
		AllowedAgents: []string{"gpt-5.4", "claude"},
	}

	authorizer := NewAuthorizer(repoConfig, nil)

	tests := []struct {
		name     string
		actor    string
		agent    string
		expected bool
	}{
		{
			name:     "allowed user",
			actor:    "alice",
			expected: true,
		},
		{
			name:     "disallowed user",
			actor:    "charlie",
			expected: false,
		},
		{
			name:     "allowed user with allowed agent",
			actor:    "alice",
			agent:    "gpt-5.4",
			expected: true,
		},
		{
			name:     "allowed user with disallowed agent",
			actor:    "alice",
			agent:    "unauthorized",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Command{Name: "apply", Agent: tt.agent}
			result := authorizer.Authorize(tt.actor, cmd)
			if result.Authorized != tt.expected {
				t.Errorf("expected Authorized=%v, got %v (reason: %s)", tt.expected, result.Authorized, result.Reason)
			}
		})
	}
}

func TestIntakeProcess(t *testing.T) {
	ctx := context.Background()
	runtimeStore, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	defer runtimeStore.Close()

	if err := runtimeStore.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	queue := store.NewJobQueue(runtimeStore)
	intake := NewIntake(runtimeStore, queue, slog.Default())
	repoConfig := config.RepoConfig{
		AllowedUsers:  []string{"alice"},
		AllowedAgents: []string{"gpt-5.4"},
	}
	pr := &store.PullRequest{ID: 11, Number: 42}

	t.Run("authorized command queues once", func(t *testing.T) {
		result, err := intake.Process(ctx, repoConfig, pr, "IC_1", "alice", "/opsx-apply --agent gpt-5.4")
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if result.Status != "queued" {
			t.Fatalf("expected queued status, got %q", result.Status)
		}
		if result.Job == nil {
			t.Fatal("expected job to be enqueued")
		}

		job, err := queue.Dequeue(ctx)
		if err != nil {
			t.Fatalf("Dequeue() error = %v", err)
		}
		if job == nil || job.JobType != "pr_command_apply" {
			t.Fatalf("expected pr_command_apply job, got %#v", job)
		}
	})

	t.Run("duplicate observation is ignored", func(t *testing.T) {
		result, err := intake.Process(ctx, repoConfig, pr, "IC_1", "alice", "/opsx-apply --agent gpt-5.4")
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if !result.Duplicate || result.Status != "duplicate" {
			t.Fatalf("expected duplicate result, got %+v", result)
		}
	})

	t.Run("edited command stays duplicate by identity", func(t *testing.T) {
		result, err := intake.Process(ctx, repoConfig, pr, "IC_1", "alice", "/symphony refine updated text")
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if !result.Duplicate {
			t.Fatalf("expected edited comment to remain duplicate, got %+v", result)
		}
	})

	t.Run("unauthorized command is rejected", func(t *testing.T) {
		result, err := intake.Process(ctx, repoConfig, pr, "IC_2", "mallory", "/symphony status")
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if result.Status != "rejected" {
			t.Fatalf("expected rejected status, got %q", result.Status)
		}

		stored, err := runtimeStore.GetCommandRequestByDedupeKey(ctx, CommandDedupeKey("IC_2"))
		if err != nil {
			t.Fatalf("GetCommandRequestByDedupeKey() error = %v", err)
		}
		if stored == nil || stored.AuthorizationStatus != "rejected" {
			t.Fatalf("expected rejected request to be persisted, got %#v", stored)
		}
	})

	t.Run("plain comment is ignored", func(t *testing.T) {
		result, err := intake.Process(ctx, repoConfig, pr, "IC_3", "alice", "looks good to me")
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if result.Status != "ignored" {
			t.Fatalf("expected ignored status, got %q", result.Status)
		}

		ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
		defer cancel()
		job, err := queue.Dequeue(ctx)
		if err != nil {
			t.Fatalf("Dequeue() error = %v", err)
		}
		if job != nil {
			t.Fatalf("expected no queued job for plain comment, got %#v", job)
		}
	})
}
