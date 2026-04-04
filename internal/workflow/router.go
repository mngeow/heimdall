package workflow

import (
	"context"
	"fmt"
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

// GenerateBranchName creates a deterministic branch name
func GenerateBranchName(issueKey, slug string) string {
	return fmt.Sprintf("symphony/%s-%s", issueKey, slug)
}

// GenerateChangeName creates a deterministic OpenSpec change name
func GenerateChangeName(issueKey, slug string) string {
	return fmt.Sprintf("%s-%s", issueKey, slug)
}

// Slugify creates a URL-safe slug from a title
func Slugify(title string) string {
	// Simple slugification - replace spaces with hyphens, lowercase
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// EnqueueProposeWorkflow creates a propose workflow run for a work item
func EnqueueProposeWorkflow(ctx context.Context, s *store.Store, queue *store.JobQueue, workItemID, repositoryID int64, changeName, branchName string) error {
	// Create workflow run
	run := &store.WorkflowRun{
		WorkItemID:      workItemID,
		RepositoryID:    repositoryID,
		RunType:         "propose",
		Status:          "queued",
		ChangeName:      changeName,
		BranchName:      branchName,
		WorktreePath:    fmt.Sprintf("/var/lib/symphony/worktrees/linear/%s", changeName),
		RequestedByType: "system",
	}

	if err := s.CreateWorkflowRun(ctx, run); err != nil {
		return fmt.Errorf("failed to create workflow run: %w", err)
	}

	// Enqueue job
	job := &store.Job{
		WorkflowRunID: &run.ID,
		JobType:       "propose",
		LockKey:       store.CreateIssueLockKey("linear", changeName),
		Status:        "queued",
	}

	if err := queue.Enqueue(ctx, job); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	return nil
}
