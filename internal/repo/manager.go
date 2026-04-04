package repo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Manager handles git repository operations
type Manager struct {
	basePath string
}

// NewManager creates a new repository manager
func NewManager(basePath string) *Manager {
	return &Manager{basePath: basePath}
}

// EnsureBareMirror ensures a bare mirror exists for a repository
func (m *Manager) EnsureBareMirror(ctx context.Context, owner, name, token string) (string, error) {
	mirrorPath := filepath.Join(m.basePath, "repos", "github.com", fmt.Sprintf("%s/%s.git", owner, name))

	// Check if mirror exists
	if _, err := os.Stat(mirrorPath); err == nil {
		// Mirror exists, fetch updates
		cmd := exec.CommandContext(ctx, "git", "-C", mirrorPath, "fetch", "--prune", "origin")
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to fetch mirror: %w (output: %s)", err, string(output))
		}
		return mirrorPath, nil
	}

	// Create mirror directory
	if err := os.MkdirAll(filepath.Dir(mirrorPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create mirror directory: %w", err)
	}

	// Clone mirror
	repoURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, owner, name)
	cmd := exec.CommandContext(ctx, "git", "clone", "--mirror", repoURL, mirrorPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to clone mirror: %w (output: %s)", err, string(output))
	}

	return mirrorPath, nil
}

// CreateWorktree creates a new worktree for a workflow run
func (m *Manager) CreateWorktree(ctx context.Context, mirrorPath, branchName, worktreePath string) error {
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
	cmd := exec.CommandContext(ctx, "git", "-C", mirrorPath, "worktree", "add", "-B", branchName, worktreePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create worktree: %w (output: %s)", err, string(output))
	}

	return nil
}

// CommitAndPush commits changes and pushes to the remote
func (m *Manager) CommitAndPush(ctx context.Context, worktreePath, message string) error {
	// Configure git user
	cmd := exec.CommandContext(ctx, "git", "-C", worktreePath, "config", "user.email", "symphony@localhost")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to configure git email: %w (output: %s)", err, string(output))
	}

	cmd = exec.CommandContext(ctx, "git", "-C", worktreePath, "config", "user.name", "Symphony")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to configure git name: %w (output: %s)", err, string(output))
	}

	// Stage all changes
	cmd = exec.CommandContext(ctx, "git", "-C", worktreePath, "add", "-A")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage changes: %w (output: %s)", err, string(output))
	}

	// Check if there are changes to commit
	cmd = exec.CommandContext(ctx, "git", "-C", worktreePath, "diff", "--cached", "--quiet")
	if err := cmd.Run(); err == nil {
		// No changes to commit
		return nil
	}

	// Commit
	cmd = exec.CommandContext(ctx, "git", "-C", worktreePath, "commit", "-m", message)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit: %w (output: %s)", err, string(output))
	}

	// Push
	cmd = exec.CommandContext(ctx, "git", "-C", worktreePath, "push", "origin", "HEAD")
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
