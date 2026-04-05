package workflow

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mngeow/symphony/internal/config"
	"github.com/mngeow/symphony/internal/store"
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

// GenerateBranchName creates a deterministic branch name.
func GenerateBranchName(branchPrefix, issueKey, slug string) string {
	prefix := strings.Trim(strings.TrimSpace(branchPrefix), "/")
	if prefix == "" {
		prefix = "symphony"
	}
	return fmt.Sprintf("%s/%s-%s", prefix, issueKey, slug)
}

// GenerateChangeName creates a deterministic OpenSpec change name
func GenerateChangeName(issueKey, slug string) string {
	return fmt.Sprintf("%s-%s", issueKey, slug)
}

// Slugify creates a URL-safe slug from free-form text.
func Slugify(text string) string {
	var result strings.Builder
	lastDash := false

	for _, r := range strings.ToLower(text) {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			result.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && result.Len() > 0 {
			result.WriteRune('-')
			lastDash = true
		}
	}

	slug := strings.Trim(result.String(), "-")
	return slug
}

// SlugFromDescriptionOrTitle returns a description-first slug with title fallback.
func SlugFromDescriptionOrTitle(description, title string) string {
	if slug := Slugify(description); slug != "" {
		return slug
	}
	return Slugify(title)
}

// GenerateWorktreePath creates a deterministic worktree path next to the configured mirror.
func GenerateWorktreePath(localMirrorPath, branchName string) string {
	mirrorDir := filepath.Dir(localMirrorPath)
	mirrorBase := strings.TrimSuffix(filepath.Base(localMirrorPath), filepath.Ext(localMirrorPath))
	branchComponent := strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(branchName)
	return filepath.Join(mirrorDir, mirrorBase+"-worktrees", branchComponent)
}

// CreateBootstrapWorkflowRun creates a bootstrap workflow run for an activated work item.
func CreateBootstrapWorkflowRun(ctx context.Context, s *store.Store, workItemID int64, repository *store.Repository, changeName, branchName string) (*store.WorkflowRun, error) {
	run := &store.WorkflowRun{
		WorkItemID:      workItemID,
		RepositoryID:    repository.ID,
		RunType:         "bootstrap_pull_request",
		Status:          "queued",
		ChangeName:      changeName,
		BranchName:      branchName,
		WorktreePath:    GenerateWorktreePath(repository.LocalMirrorPath, branchName),
		RequestedByType: "system",
	}

	if err := s.CreateWorkflowRun(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to create workflow run: %w", err)
	}

	return run, nil
}
