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

func TestCreateWorktreeRecoversFromStaleWorktreeRegistration(t *testing.T) {
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

	// First creation should succeed
	if err := manager.CreateWorktree(ctx, mirrorPath, "main", "heimdall/ENG-123-add-rate-limiting", worktreePath); err != nil {
		t.Fatalf("CreateWorktree() first call error = %v", err)
	}

	// Simulate a stale worktree registration by deleting the directory but leaving git metadata
	if err := os.RemoveAll(worktreePath); err != nil {
		t.Fatalf("RemoveAll() error = %v", err)
	}

	// Second creation should recover from the stale registration and succeed
	if err := manager.CreateWorktree(ctx, mirrorPath, "main", "heimdall/ENG-123-add-rate-limiting", worktreePath); err != nil {
		t.Fatalf("CreateWorktree() retry after stale registration error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(worktreePath, "README.md")); err != nil {
		t.Fatalf("expected README.md in recovered worktree, got error %v", err)
	}

	branch, err := manager.GetCurrentBranch(ctx, worktreePath)
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}
	if branch != "heimdall/ENG-123-add-rate-limiting" {
		t.Fatalf("expected bootstrap branch, got %q", branch)
	}
}

func TestParseWorktreeList(t *testing.T) {
	input := `worktree /Users/mngeow/Projects/repo
HEAD 1b6eabefb24114cfa4754b601b8a33452b2ac7f9
branch refs/heads/main

worktree /Users/mngeow/Projects/repo-worktrees/feature-foo
HEAD b74804419e47becea3495af54cd91fa04757961d
branch refs/heads/feature/foo
prunable gitdir file points to non-existent location

worktree /Users/mngeow/Projects/repo-worktrees/feature-bar
HEAD b74804419e47becea3495af54cd91fa04757961d
branch refs/heads/feature/bar
`

	worktrees := parseWorktreeList(input)
	if len(worktrees) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(worktrees))
	}

	cases := []struct {
		index  int
		path   string
		branch string
	}{
		{0, "/Users/mngeow/Projects/repo", "refs/heads/main"},
		{1, "/Users/mngeow/Projects/repo-worktrees/feature-foo", "refs/heads/feature/foo"},
		{2, "/Users/mngeow/Projects/repo-worktrees/feature-bar", "refs/heads/feature/bar"},
	}

	for _, tc := range cases {
		if worktrees[tc.index].path != tc.path {
			t.Errorf("worktree[%d].path = %q, want %q", tc.index, worktrees[tc.index].path, tc.path)
		}
		if worktrees[tc.index].branch != tc.branch {
			t.Errorf("worktree[%d].branch = %q, want %q", tc.index, worktrees[tc.index].branch, tc.branch)
		}
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
