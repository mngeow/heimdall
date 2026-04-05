package repo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrNoChanges = errors.New("no repository changes to commit")

// Manager handles git repository operations
type Manager struct {
	basePath string
}

// NewManager creates a new repository manager
func NewManager(basePath string) *Manager {
	return &Manager{basePath: basePath}
}

// EnsureBareMirror ensures a bare mirror exists for a repository at the configured path.
func (m *Manager) EnsureBareMirror(ctx context.Context, mirrorPath, owner, name, token string) error {
	// Check if mirror exists
	if _, err := os.Stat(mirrorPath); err == nil {
		// Mirror exists, fetch updates
		cmd := exec.CommandContext(ctx, "git", "-C", mirrorPath, "fetch", "--prune", authenticatedRepoURL(owner, name, token))
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to fetch mirror: %w (output: %s)", err, string(output))
		}
		return nil
	}

	// Create mirror directory
	if err := os.MkdirAll(filepath.Dir(mirrorPath), 0755); err != nil {
		return fmt.Errorf("failed to create mirror directory: %w", err)
	}

	// Clone mirror
	repoURL := authenticatedRepoURL(owner, name, token)
	cmd := exec.CommandContext(ctx, "git", "clone", "--mirror", repoURL, mirrorPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone mirror: %w (output: %s)", err, string(output))
	}

	return nil
}

// CreateWorktree creates a new worktree for a workflow run.
func (m *Manager) CreateWorktree(ctx context.Context, mirrorPath, baseBranch, branchName, worktreePath string) error {
	// Ensure worktree directory exists
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return fmt.Errorf("failed to create worktree directory: %w", err)
	}

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		// Remove existing worktree
		cmd := exec.CommandContext(ctx, "git", "-C", mirrorPath, "worktree", "remove", "-f", worktreePath)
		cmd.CombinedOutput() // Ignore errors
	}

	// Create worktree
	baseRef := fmt.Sprintf("refs/heads/%s", strings.TrimSpace(baseBranch))
	cmd := exec.CommandContext(ctx, "git", "-C", mirrorPath, "worktree", "add", "-B", branchName, worktreePath, baseRef)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create worktree: %w (output: %s)", err, string(output))
	}

	return nil
}

// HasChanges reports whether the worktree contains repository changes.
func (m *Manager) HasChanges(ctx context.Context, worktreePath string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", worktreePath, "status", "--porcelain")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to inspect worktree changes: %w (output: %s)", err, string(output))
	}
	return strings.TrimSpace(string(output)) != "", nil
}

// CommitAll stages all changes, validates a non-empty diff, and creates a commit.
func (m *Manager) CommitAll(ctx context.Context, worktreePath, message string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", worktreePath, "add", "-A")
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to stage changes: %w (output: %s)", err, string(output))
	}

	cmd = exec.CommandContext(ctx, "git", "-C", worktreePath, "diff", "--cached", "--quiet")
	if err := cmd.Run(); err == nil {
		return "", ErrNoChanges
	}

	cmd = exec.CommandContext(ctx, "git", "-C", worktreePath, "-c", "user.name=Symphony", "-c", "user.email=symphony@localhost", "commit", "-m", message)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to commit: %w (output: %s)", err, string(output))
	}

	return m.GetHeadSHA(ctx, worktreePath)
}

// PushBranch pushes the branch to GitHub by using an installation token without mutating git config.
func (m *Manager) PushBranch(ctx context.Context, worktreePath, owner, name, branchName, token string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", worktreePath, "push", authenticatedRepoURL(owner, name, token), "HEAD:refs/heads/"+branchName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push: %w (output: %s)", err, string(output))
	}
	return nil
}

// GetCurrentBranch returns the current branch name
func (m *Manager) GetCurrentBranch(ctx context.Context, worktreePath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", worktreePath, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetHeadSHA returns the current HEAD commit SHA
func (m *Manager) GetHeadSHA(ctx context.Context, worktreePath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", worktreePath, "rev-parse", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func authenticatedRepoURL(owner, name, token string) string {
	return fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, owner, name)
}
