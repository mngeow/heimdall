package exec

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestOpenSpecClientListChangesParsesRealResponseShape(t *testing.T) {
	// Create a temporary directory to act as the worktree
	tempDir := t.TempDir()

	// Initialize a minimal git repo so openspec can run (openspec may require git)
	if err := exec.Command("git", "init", tempDir).Run(); err != nil {
		t.Skip("git not available, skipping test")
	}

	// Create a mock openspec script that returns the real CLI response shape
	mockOpenspec := filepath.Join(tempDir, "openspec")
	script := `#!/bin/sh
if [ "$1" = "list" ] && [ "$2" = "--json" ]; then
	echo '{"changes":[{"name":"change-1"},{"name":"change-2"}]}'
else
	echo "unknown command" >&2
	exit 1
fi
`
	if err := os.WriteFile(mockOpenspec, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}

	// Temporarily modify PATH to use our mock
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tempDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	client := NewOpenSpecClient(tempDir)
	changes, err := client.ListChanges(context.Background())
	if err != nil {
		t.Fatalf("ListChanges() error = %v", err)
	}

	want := []string{"change-1", "change-2"}
	if len(changes) != len(want) {
		t.Fatalf("ListChanges() = %v, want %v", changes, want)
	}
	for i, name := range want {
		if changes[i] != name {
			t.Errorf("ListChanges()[%d] = %q, want %q", i, changes[i], name)
		}
	}
}

func TestOpenSpecClientListChangesEmptyResponse(t *testing.T) {
	tempDir := t.TempDir()

	if err := exec.Command("git", "init", tempDir).Run(); err != nil {
		t.Skip("git not available, skipping test")
	}

	mockOpenspec := filepath.Join(tempDir, "openspec")
	script := `#!/bin/sh
if [ "$1" = "list" ] && [ "$2" = "--json" ]; then
	echo '{"changes":[]}'
else
	echo "unknown command" >&2
	exit 1
fi
`
	if err := os.WriteFile(mockOpenspec, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tempDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	client := NewOpenSpecClient(tempDir)
	changes, err := client.ListChanges(context.Background())
	if err != nil {
		t.Fatalf("ListChanges() error = %v", err)
	}

	if len(changes) != 0 {
		t.Fatalf("ListChanges() = %v, want empty slice", changes)
	}
}

func TestParseOpencodeEventsDetectsPermissionAsked(t *testing.T) {
	events := []byte(`{"type":"step_start","timestamp":1,"sessionID":"ses_abc","part":{"id":"p1","messageID":"m1","sessionID":"ses_abc","snapshot":"s1","type":"step-start"}}
{"type":"permission.asked","timestamp":2,"sessionID":"ses_abc","properties":{"id":"perm_123","sessionID":"ses_abc","permission":"write","patterns":["*.md"]}}
`)
	outcome, err := parseOpencodeEvents(bytes.NewReader(events))
	if err != nil {
		t.Fatalf("parseOpencodeEvents() error = %v", err)
	}
	if outcome == nil {
		t.Fatal("expected outcome, got nil")
	}
	if outcome.Status != "needs_permission" {
		t.Errorf("Status = %q, want %q", outcome.Status, "needs_permission")
	}
	if outcome.RequestID != "perm_123" {
		t.Errorf("RequestID = %q, want %q", outcome.RequestID, "perm_123")
	}
	if outcome.SessionID != "ses_abc" {
		t.Errorf("SessionID = %q, want %q", outcome.SessionID, "ses_abc")
	}
}

func TestParseOpencodeEventsIgnoresHelpText(t *testing.T) {
	// Simulate CLI help output as non-JSON lines
	help := []byte(`Usage: opencode run [options] <message>
Options:
  --agent <name>   agent to use
`)
	outcome, err := parseOpencodeEvents(bytes.NewReader(help))
	if err != nil {
		t.Fatalf("parseOpencodeEvents() error = %v", err)
	}
	if outcome != nil {
		t.Fatalf("expected nil outcome for help text, got %+v", outcome)
	}
}

func TestParseOpencodeEventsMissingIDs(t *testing.T) {
	// permission.asked without id should become error, not needs_permission
	events := []byte(`{"type":"permission.asked","timestamp":1,"sessionID":"ses_abc","properties":{"sessionID":"ses_abc"}}
`)
	outcome, err := parseOpencodeEvents(bytes.NewReader(events))
	if err != nil {
		t.Fatalf("parseOpencodeEvents() error = %v", err)
	}
	if outcome == nil {
		t.Fatal("expected error outcome, got nil")
	}
	if outcome.Status != "error" {
		t.Errorf("Status = %q, want %q", outcome.Status, "error")
	}
}

func TestParseOpencodeEventsLargeTextEventDoesNotAbort(t *testing.T) {
	// Build a text event with a payload larger than the old bufio.Scanner default limit (64 KiB).
	largePayload := bytes.Repeat([]byte("x"), 128*1024)
	events := []byte(`{"type":"step_start","timestamp":1,"sessionID":"ses_abc","part":{"id":"p1","messageID":"m1","sessionID":"ses_abc","snapshot":"s1","type":"step-start"}}
{"type":"text","timestamp":2,"sessionID":"ses_abc","part":{"id":"p2","messageID":"m2","sessionID":"ses_abc","type":"text","text":"` + string(largePayload) + `"}}
`)
	outcome, err := parseOpencodeEvents(bytes.NewReader(events))
	if err != nil {
		t.Fatalf("parseOpencodeEvents() error = %v", err)
	}
	if outcome != nil {
		t.Fatalf("expected nil outcome after large text event, got %+v", outcome)
	}
}

func TestParseOpencodeEventsPermissionAfterLargeTextEvent(t *testing.T) {
	largePayload := bytes.Repeat([]byte("y"), 128*1024)
	events := []byte(`{"type":"text","timestamp":1,"sessionID":"ses_abc","part":{"id":"p1","messageID":"m1","sessionID":"ses_abc","type":"text","text":"` + string(largePayload) + `"}}
{"type":"permission.asked","timestamp":2,"sessionID":"ses_abc","properties":{"id":"perm_456","sessionID":"ses_abc","permission":"write","patterns":["*.go"]}}
`)
	outcome, err := parseOpencodeEvents(bytes.NewReader(events))
	if err != nil {
		t.Fatalf("parseOpencodeEvents() error = %v", err)
	}
	if outcome == nil {
		t.Fatal("expected outcome, got nil")
	}
	if outcome.Status != "needs_permission" {
		t.Errorf("Status = %q, want %q", outcome.Status, "needs_permission")
	}
	if outcome.RequestID != "perm_456" {
		t.Errorf("RequestID = %q, want %q", outcome.RequestID, "perm_456")
	}
	if outcome.SessionID != "ses_abc" {
		t.Errorf("SessionID = %q, want %q", outcome.SessionID, "ses_abc")
	}
}

func TestParseOpencodeEventsFinalEventWithoutTrailingNewline(t *testing.T) {
	// Final event line ends at EOF without a trailing newline.
	events := []byte(`{"type":"step_start","timestamp":1,"sessionID":"ses_def","part":{"id":"p1","messageID":"m1","sessionID":"ses_def","snapshot":"s1","type":"step-start"}}
{"type":"permission.asked","timestamp":2,"sessionID":"ses_def","properties":{"id":"perm_789","sessionID":"ses_def","permission":"write","patterns":["*.md"]}}`)
	outcome, err := parseOpencodeEvents(bytes.NewReader(events))
	if err != nil {
		t.Fatalf("parseOpencodeEvents() error = %v", err)
	}
	if outcome == nil {
		t.Fatal("expected outcome, got nil")
	}
	if outcome.Status != "needs_permission" {
		t.Errorf("Status = %q, want %q", outcome.Status, "needs_permission")
	}
	if outcome.RequestID != "perm_789" {
		t.Errorf("RequestID = %q, want %q", outcome.RequestID, "perm_789")
	}
}

func TestOpenSpecClientGetApplyInstructionsParsesJSONWithProgressPrefix(t *testing.T) {
	tempDir := t.TempDir()

	if err := exec.Command("git", "init", tempDir).Run(); err != nil {
		t.Skip("git not available, skipping test")
	}

	mockOpenspec := filepath.Join(tempDir, "openspec")
	script := `#!/bin/sh
if [ "$1" = "instructions" ] && [ "$2" = "apply" ] && [ "$3" = "--change" ] && [ "$4" = "test-change" ] && [ "$5" = "--json" ]; then
	echo '- Generating apply instructions...'
	echo '{"changeName":"test-change","schemaName":"spec-driven","state":"ready","tasks":[{"id":"1","description":"do work","done":false}]}'
else
	echo "unknown command" >&2
	exit 1
fi
`
	if err := os.WriteFile(mockOpenspec, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tempDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	client := NewOpenSpecClient(tempDir)
	instructions, err := client.GetApplyInstructions(context.Background(), "test-change")
	if err != nil {
		t.Fatalf("GetApplyInstructions() error = %v", err)
	}

	if instructions.ChangeName != "test-change" {
		t.Errorf("ChangeName = %q, want %q", instructions.ChangeName, "test-change")
	}
	if instructions.SchemaName != "spec-driven" {
		t.Errorf("SchemaName = %q, want %q", instructions.SchemaName, "spec-driven")
	}
	if instructions.State != "ready" {
		t.Errorf("State = %q, want %q", instructions.State, "ready")
	}
	if len(instructions.Tasks) != 1 {
		t.Fatalf("len(Tasks) = %d, want 1", len(instructions.Tasks))
	}
	if instructions.Tasks[0].ID != "1" {
		t.Errorf("Tasks[0].ID = %q, want %q", instructions.Tasks[0].ID, "1")
	}
}
