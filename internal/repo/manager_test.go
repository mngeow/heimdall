package repo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCreateWorktreeFromConfiguredMirror(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourcePath, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	runGit(t, ctx, tempDir, "init", "--initial-branch=main", sourcePath)
	runGit(t, ctx, sourcePath, "config", "user.name", "Test User")
	runGit(t, ctx, sourcePath, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(sourcePath, "README.md"), []byte("bootstrap\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	runGit(t, ctx, sourcePath, "add", "README.md")
	runGit(t, ctx, sourcePath, "commit", "-m", "initial")

	mirrorPath := filepath.Join(tempDir, "platform.git")
	runGit(t, ctx, tempDir, "clone", "--mirror", sourcePath, mirrorPath)

	manager := NewManager("")
	worktreePath := filepath.Join(tempDir, "worktrees", "heimdall-ENG-123-add-rate-limiting")
	if err := manager.CreateWorktree(ctx, mirrorPath, "main", "heimdall/ENG-123-add-rate-limiting", worktreePath); err != nil {
		t.Fatalf("CreateWorktree() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(worktreePath, "README.md")); err != nil {
		t.Fatalf("expected README.md in worktree, got error %v", err)
	}

	branch, err := manager.GetCurrentBranch(ctx, worktreePath)
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}
	if branch != "heimdall/ENG-123-add-rate-limiting" {
		t.Fatalf("expected bootstrap branch, got %q", branch)
	}
}

func runGit(t *testing.T, ctx context.Context, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
