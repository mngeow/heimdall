package slashcmd

import (
	"testing"

	"github.com/mngeow/symphony/internal/config"
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
