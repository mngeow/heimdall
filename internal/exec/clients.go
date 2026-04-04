package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
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

// Instructions represents OpenSpec instructions
type Instructions struct {
	ArtifactID   string   `json:"artifact_id"`
	Type         string   `json:"type"`
	OutputPath   string   `json:"output_path"`
	Dependencies []string `json:"dependencies"`
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
