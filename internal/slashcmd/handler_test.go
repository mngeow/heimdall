package slashcmd

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/mngeow/heimdall/internal/config"
	"github.com/mngeow/heimdall/internal/store"
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
			comment:  "/heimdall status",
			expected: "status",
			isValid:  true,
		},
		{
			name:     "refine command with agent",
			comment:  "/heimdall refine --agent gpt-5.4 -- Add error handling",
			expected: "refine",
			isValid:  true,
		},
		{
			name:     "refine command without agent",
			comment:  "/heimdall refine Add error handling",
			expected: "refine",
			isValid:  false,
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
			name:     "opencode command with alias and agent",
			comment:  "/heimdall opencode explore-change --agent gpt-5.4 -- Compare options",
			expected: "opencode",
			isValid:  true,
		},
		{
			name:     "approve command with request id",
			comment:  "/heimdall approve perm_123",
			expected: "approve",
			isValid:  true,
		},
		{
			name:     "approve command without request id",
			comment:  "/heimdall approve",
			expected: "approve",
			isValid:  false,
		},
		{
			name:     "no command",
			comment:  "This is just a regular comment",
			expected: "",
			isValid:  false,
		},
	}

	promptTests := []struct {
		name           string
		comment        string
		wantPrompt     string
		wantChangeName string
	}{
		{
			name:           "inline prompt after separator",
			comment:        "/heimdall refine --agent gpt-5.4 -- Add error handling",
			wantPrompt:     "Add error handling",
			wantChangeName: "",
		},
		{
			name:           "multiline prompt after trailing separator",
			comment:        "/heimdall refine --agent gpt-5.4 --\nGood. But I also want you to include the following:\n1. duckduckgo search tool\n2. Expose this agent via a simple fastapi application",
			wantPrompt:     "Good. But I also want you to include the following:\n1. duckduckgo search tool\n2. Expose this agent via a simple fastapi application",
			wantChangeName: "",
		},
		{
			name:           "inline prompt with explicit change name",
			comment:        "/heimdall refine my-change --agent gpt-5.4 -- Add error handling",
			wantPrompt:     "Add error handling",
			wantChangeName: "my-change",
		},
		{
			name:           "no prompt",
			comment:        "/heimdall refine --agent gpt-5.4",
			wantPrompt:     "",
			wantChangeName: "",
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

	for _, tt := range promptTests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := parser.Parse(tt.comment)
			if cmd == nil {
				t.Fatalf("expected command, got nil")
			}
			if cmd.PromptTail != tt.wantPrompt {
				t.Errorf("PromptTail = %q, want %q", cmd.PromptTail, tt.wantPrompt)
			}
			if cmd.ChangeName != tt.wantChangeName {
				t.Errorf("ChangeName = %q, want %q", cmd.ChangeName, tt.wantChangeName)
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

func TestAuthorizerAlias(t *testing.T) {
	repoConfig := config.RepoConfig{
		AllowedUsers:  []string{"alice"},
		AllowedAgents: []string{"gpt-5.4"},
		OpencodeAliases: map[string]config.OpencodeCommandAlias{
			"explore-change": {Name: "explore-change", Command: "opsx-explore", PermissionProfile: "readonly"},
		},
	}
	authorizer := NewAuthorizer(repoConfig, nil)

	t.Run("allowed alias", func(t *testing.T) {
		cmd := &Command{Name: "opencode", Agent: "gpt-5.4", Alias: "explore-change"}
		result := authorizer.Authorize("alice", cmd)
		if !result.Authorized {
			t.Fatalf("expected authorized, got rejected: %s", result.Reason)
		}
	})

	t.Run("disallowed alias", func(t *testing.T) {
		cmd := &Command{Name: "opencode", Agent: "gpt-5.4", Alias: "unknown"}
		result := authorizer.Authorize("alice", cmd)
		if result.Authorized {
			t.Fatal("expected rejected for unknown alias")
		}
	})
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
		result, err := intake.Process(ctx, repoConfig, pr, "IC_1", "alice", "/heimdall refine updated text")
		if err != nil {
			t.Fatalf("Process() error = %v", err)
		}
		if !result.Duplicate {
			t.Fatalf("expected edited comment to remain duplicate, got %+v", result)
		}
	})

	t.Run("unauthorized command is rejected", func(t *testing.T) {
		result, err := intake.Process(ctx, repoConfig, pr, "IC_2", "mallory", "/heimdall status")
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
