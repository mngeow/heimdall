package workflow

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mngeow/heimdall/internal/config"
	"github.com/mngeow/heimdall/internal/store"
)

// Router resolves the target repository for a work item
type Router struct {
	repos []config.RepoConfig
}

// NewRouter creates a new repository router
func NewRouter(repos []config.RepoConfig) *Router {
	return &Router{repos: repos}
}

// RouteResult represents the result of a routing decision
type RouteResult struct {
	Repository *config.RepoConfig
	Matched    bool
	Reason     string
}

// Resolve determines the target repository for a work item
func (r *Router) Resolve(teamKey string) *RouteResult {
	// If only one repo configured, use it
	if len(r.repos) == 1 {
		return &RouteResult{
			Repository: &r.repos[0],
			Matched:    true,
		}
	}

	// Try to match by team key
	for _, repo := range r.repos {
		for _, key := range repo.LinearTeamKeys {
			if key == teamKey {
				return &RouteResult{
					Repository: &repo,
					Matched:    true,
				}
			}
		}
	}

	return &RouteResult{
		Matched: false,
		Reason:  fmt.Sprintf("no repository mapping found for team: %s", teamKey),
	}
}

// GenerateBranchName creates a deterministic branch name from the issue key and title.
func GenerateBranchName(branchPrefix, issueKey, title string) string {
	prefix := strings.Trim(strings.TrimSpace(branchPrefix), "/")
	if prefix == "" {
		prefix = "heimdall"
	}
	slug := CleanSlug(title)
	if slug == "" {
		slug = CleanSlug(issueKey)
	}
	return fmt.Sprintf("%s/%s-%s", prefix, issueKey, slug)
}

// CleanSlug creates a filesystem-safe, URL-safe slug from free-form text.
// It strips characters that cause issues in branch names and folder paths
// (commas, colons, non-ASCII/non-UTF-8 safe characters, etc.).
func CleanSlug(text string) string {
	var result strings.Builder
	lastDash := false

	for _, r := range strings.ToLower(text) {
		// Keep only ASCII alphanumerics
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			result.WriteRune(r)
			lastDash = false
			continue
		}
		// Any other character becomes a single dash boundary
		if !lastDash && result.Len() > 0 {
			result.WriteRune('-')
			lastDash = true
		}
	}

	return strings.Trim(result.String(), "-")
}

// GenerateWorktreePath creates a deterministic worktree path next to the configured mirror.
func GenerateWorktreePath(localMirrorPath, branchName string) string {
	mirrorDir := filepath.Dir(localMirrorPath)
	mirrorBase := strings.TrimSuffix(filepath.Base(localMirrorPath), filepath.Ext(localMirrorPath))
	// The branchName is already cleaned, but keep the replacement as a safety net.
	branchComponent := strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(branchName)
	return filepath.Join(mirrorDir, mirrorBase+"-worktrees", branchComponent)
}

// CreateProposalWorkflowRun creates a proposal workflow run for an activated work item.
func CreateProposalWorkflowRun(ctx context.Context, s *store.Store, workItemID int64, repository *store.Repository, branchName string) (*store.WorkflowRun, error) {
	run := &store.WorkflowRun{
		WorkItemID:      workItemID,
		RepositoryID:    repository.ID,
		RunType:         "activation_proposal_pull_request",
		Status:          "queued",
		BranchName:      branchName,
		WorktreePath:    GenerateWorktreePath(repository.LocalMirrorPath, branchName),
		RequestedByType: "system",
	}

	if err := s.CreateWorkflowRun(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to create workflow run: %w", err)
	}

	return run, nil
}
