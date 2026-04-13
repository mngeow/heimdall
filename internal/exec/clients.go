package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// OpenSpecClient wraps the openspec CLI
type OpenSpecClient struct {
	worktreePath string
}

// NewOpenSpecClient creates a new OpenSpec client
func NewOpenSpecClient(worktreePath string) *OpenSpecClient {
	return &OpenSpecClient{worktreePath: worktreePath}
}

// ChangeStatus represents the status of an OpenSpec change
type ChangeStatus struct {
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Artifacts []string `json:"artifacts"`
	CanApply  bool     `json:"can_apply"`
	Blockers  []string `json:"blockers,omitempty"`
}

// CreateChange creates a new OpenSpec change
func (c *OpenSpecClient) CreateChange(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "openspec", "new", "change", name)
	cmd.Dir = c.worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create change: %w (output: %s)", err, string(output))
	}
	return nil
}

// GetStatus retrieves the status of a change
func (c *OpenSpecClient) GetStatus(ctx context.Context, name string) (*ChangeStatus, error) {
	cmd := exec.CommandContext(ctx, "openspec", "status", "--change", name, "--json")
	cmd.Dir = c.worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w (output: %s)", err, string(output))
	}

	var status ChangeStatus
	if err := json.Unmarshal(output, &status); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	return &status, nil
}

// GetInstructions retrieves instructions for the next artifact to generate
func (c *OpenSpecClient) GetInstructions(ctx context.Context, changeName string) (*Instructions, error) {
	cmd := exec.CommandContext(ctx, "openspec", "instructions", changeName, "--json")
	cmd.Dir = c.worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get instructions: %w (output: %s)", err, string(output))
	}

	var instructions Instructions
	if err := json.Unmarshal(output, &instructions); err != nil {
		return nil, fmt.Errorf("failed to parse instructions: %w", err)
	}

	return &instructions, nil
}

// GetApplyInstructions retrieves apply instructions for a change.
func (c *OpenSpecClient) GetApplyInstructions(ctx context.Context, changeName string) (*ApplyInstructions, error) {
	cmd := exec.CommandContext(ctx, "openspec", "instructions", "apply", "--change", changeName, "--json")
	cmd.Dir = c.worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get apply instructions: %w (output: %s)", err, string(output))
	}

	var instructions ApplyInstructions
	if err := json.Unmarshal(output, &instructions); err != nil {
		return nil, fmt.Errorf("failed to parse apply instructions: %w", err)
	}

	return &instructions, nil
}

// ListChanges lists all OpenSpec changes in the worktree.
func (c *OpenSpecClient) ListChanges(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "openspec", "list", "--json")
	cmd.Dir = c.worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list changes: %w (output: %s)", err, string(output))
	}

	var changes []string
	if err := json.Unmarshal(output, &changes); err != nil {
		return nil, fmt.Errorf("failed to parse changes list: %w", err)
	}

	return changes, nil
}

// Instructions represents OpenSpec instructions for artifact generation.
type Instructions struct {
	ArtifactID   string   `json:"artifact_id"`
	Type         string   `json:"type"`
	OutputPath   string   `json:"output_path"`
	Dependencies []string `json:"dependencies"`
}

// ApplyInstructions represents OpenSpec apply instructions.
type ApplyInstructions struct {
	ChangeName   string            `json:"changeName"`
	ChangeDir    string            `json:"changeDir"`
	SchemaName   string            `json:"schemaName"`
	ContextFiles map[string]string `json:"contextFiles"`
	Progress     struct {
		Total     int `json:"total"`
		Complete  int `json:"complete"`
		Remaining int `json:"remaining"`
	} `json:"progress"`
	Tasks []struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		Done        bool   `json:"done"`
	} `json:"tasks"`
	State       string `json:"state"`
	Instruction string `json:"instruction"`
}

// ProposalRequest describes the activation-seeded OpenSpec proposal generation request.
type ProposalRequest struct {
	WorktreePath string
	IssueKey     string
	IssueTitle   string
	Description  string
	Agent        string
}

// ProposalResult describes the proposal generation outcome.
type ProposalResult struct {
	Summary       string
	ChangesBefore []string // List of changes before running opencode, used to discover newly created change
}

// ProposalRunner runs the activation proposal generation through opencode.
type ProposalRunner interface {
	RunProposal(context.Context, ProposalRequest) (*ProposalResult, error)
}

// OpenCodeProposalRunner executes activation proposal generation through the local opencode CLI.
type OpenCodeProposalRunner struct{}

// NewOpenCodeProposalRunner creates a runner for activation proposal generation.
func NewOpenCodeProposalRunner() *OpenCodeProposalRunner {
	return &OpenCodeProposalRunner{}
}

// RunProposal executes the activation proposal generation using the configured agent.
func (r *OpenCodeProposalRunner) RunProposal(ctx context.Context, req ProposalRequest) (*ProposalResult, error) {
	prompt := buildProposalPrompt(req)
	cmd := exec.CommandContext(ctx,
		"opencode", "run",
		"--agent", req.Agent,
		"--dir", req.WorktreePath,
		prompt,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run proposal prompt: %w (output: %s)", err, string(output))
	}

	return &ProposalResult{Summary: proposalSummary(req)}, nil
}

// OpenCodeClient wraps the opencode CLI
type OpenCodeClient struct {
	worktreePath string
}

// NewOpenCodeClient creates a new OpenCode client
func NewOpenCodeClient(worktreePath string) *OpenCodeClient {
	return &OpenCodeClient{worktreePath: worktreePath}
}

// GenerateArtifact generates an artifact using opencode
func (c *OpenCodeClient) GenerateArtifact(ctx context.Context, agent, instructions string) error {
	cmd := exec.CommandContext(ctx, "opencode", "generate", "--agent", agent, "--instructions", instructions)
	cmd.Dir = c.worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to generate artifact: %w (output: %s)", err, string(output))
	}
	return nil
}

// Refine refines an artifact using opencode
func (c *OpenCodeClient) Refine(ctx context.Context, agent, artifactPath, instruction string) error {
	cmd := exec.CommandContext(ctx, "opencode", "refine", "--agent", agent, "--file", artifactPath, "--instruction", instruction)
	cmd.Dir = c.worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to refine artifact: %w (output: %s)", err, string(output))
	}
	return nil
}

// Apply applies a change using opencode
func (c *OpenCodeClient) Apply(ctx context.Context, agent, changeName string) error {
	cmd := exec.CommandContext(ctx, "opencode", "apply", "--agent", agent, changeName)
	cmd.Dir = c.worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply change: %w (output: %s)", err, string(output))
	}
	return nil
}

// GetVersion returns the version of the opencode CLI
func (c *OpenCodeClient) GetVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "opencode", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}
	return string(output), nil
}

func buildProposalPrompt(req ProposalRequest) string {
	description := strings.TrimSpace(req.Description)
	if description == "" {
		description = "No issue description was provided."
	}

	return fmt.Sprintf(`You are generating an OpenSpec change proposal for a Heimdall-activated work item.

Work only inside this repository.
Use the local openspec CLI to create a new change with an appropriate name based on the issue context.
Use openspec status and instructions to determine which artifacts are required.
Generate all apply-required artifacts (proposal, design, specs, tasks) before stopping.
Do not implement tasks; only create the proposal artifacts.

Issue key: %s
Issue title: %s
Issue description:
%s
`, req.IssueKey, req.IssueTitle, description)
}

func proposalSummary(req ProposalRequest) string {
	return fmt.Sprintf("Generated OpenSpec proposal artifacts for issue %s from the activation seed.", req.IssueKey)
}
