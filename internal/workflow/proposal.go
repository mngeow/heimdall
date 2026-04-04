package workflow

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mngeow/symphony/internal/exec"
	"github.com/mngeow/symphony/internal/repo"
	"github.com/mngeow/symphony/internal/scm/github"
	"github.com/mngeow/symphony/internal/store"
)

// ProposalWorkflow handles the proposal generation workflow
type ProposalWorkflow struct {
	store   *store.Store
	repoMgr *repo.Manager
	github  *github.Client
	logger  *slog.Logger
}

// NewProposalWorkflow creates a new proposal workflow
func NewProposalWorkflow(store *store.Store, repoMgr *repo.Manager, github *github.Client, logger *slog.Logger) *ProposalWorkflow {
	return &ProposalWorkflow{
		store:   store,
		repoMgr: repoMgr,
		github:  github,
		logger:  logger,
	}
}

// Execute runs the proposal workflow
func (w *ProposalWorkflow) Execute(ctx context.Context, runID int64) error {
	w.logger.Info("executing proposal workflow", "run_id", runID)

	// Get workflow run
	// TODO: Implement GetWorkflowRun in store
	// For now, assume we have the run details

	// Update status to running
	if err := w.store.UpdateWorkflowRunStatus(ctx, runID, "running"); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Create workflow step for setup
	step := &store.WorkflowStep{
		WorkflowRunID: runID,
		StepName:      "setup_worktree",
		StepOrder:     1,
		Status:        "running",
		Executor:      "git",
	}
	if err := w.store.CreateWorkflowStep(ctx, step); err != nil {
		return fmt.Errorf("failed to create step: %w", err)
	}

	// TODO: Get run details and execute actual workflow
	// 1. Get repository and work item details
	// 2. Ensure bare mirror exists
	// 3. Create worktree
	// 4. Create OpenSpec change
	// 5. Generate artifacts
	// 6. Commit and push
	// 7. Create or update PR
	// 8. Post status comment

	// Mark step complete
	step.Status = "completed"

	// Update workflow run status
	if err := w.store.UpdateWorkflowRunStatus(ctx, runID, "completed"); err != nil {
		return fmt.Errorf("failed to update final status: %w", err)
	}

	w.logger.Info("proposal workflow completed", "run_id", runID)
	return nil
}

// generateArtifacts generates all required artifacts for a proposal
func (w *ProposalWorkflow) generateArtifacts(ctx context.Context, worktreePath, changeName, agent string) error {
	openSpec := exec.NewOpenSpecClient(worktreePath)
	openCode := exec.NewOpenCodeClient(worktreePath)

	// Get change status
	status, err := openSpec.GetStatus(ctx, changeName)
	if err != nil {
		return fmt.Errorf("failed to get change status: %w", err)
	}

	// Generate missing artifacts
	for _, artifact := range status.Artifacts {
		instructions, err := openSpec.GetInstructions(ctx, changeName)
		if err != nil {
			return fmt.Errorf("failed to get instructions: %w", err)
		}

		if err := openCode.GenerateArtifact(ctx, agent, instructions.ArtifactID); err != nil {
			return fmt.Errorf("failed to generate artifact %s: %w", artifact, err)
		}
	}

	return nil
}
