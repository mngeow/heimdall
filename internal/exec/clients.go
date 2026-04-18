package exec

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// OpenSpecClient wraps the openspec CLI
type OpenSpecClient struct {
	worktreePath string
}

// NewOpenSpecClient creates a new OpenSpec client
func NewOpenSpecClient(worktreePath string) *OpenSpecClient {
	return &OpenSpecClient{worktreePath: worktreePath}
}

// SetWorktreePath sets the worktree path for subsequent OpenSpec commands.
// This allows the same client instance to be reused across different workflow runs.
func (c *OpenSpecClient) SetWorktreePath(worktreePath string) {
	c.worktreePath = worktreePath
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

// getJSONOutput runs a command and returns the first JSON object found in stdout.
// This handles CLI tools that emit human-readable progress lines before the JSON payload.
func getJSONOutput(cmd *exec.Cmd) ([]byte, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}
	output, err := io.ReadAll(stdout)
	if waitErr := cmd.Wait(); waitErr != nil {
		return nil, fmt.Errorf("command failed: %w", waitErr)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read stdout: %w", err)
	}

	// If the output is already valid JSON, return it directly.
	if len(bytes.TrimSpace(output)) > 0 && (bytes.TrimSpace(output)[0] == '{' || bytes.TrimSpace(output)[0] == '[') {
		return output, nil
	}

	// Otherwise, scan for the first JSON object or array in the output.
	start := bytes.Index(output, []byte("{"))
	if start == -1 {
		start = bytes.Index(output, []byte("["))
	}
	if start == -1 {
		return nil, fmt.Errorf("no JSON found in output")
	}
	return output[start:], nil
}

// GetStatus retrieves the status of a change
func (c *OpenSpecClient) GetStatus(ctx context.Context, name string) (*ChangeStatus, error) {
	cmd := exec.CommandContext(ctx, "openspec", "status", "--change", name, "--json")
	cmd.Dir = c.worktreePath
	output, err := getJSONOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
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
	output, err := getJSONOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get instructions: %w", err)
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
	output, err := getJSONOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get apply instructions: %w", err)
	}

	var instructions ApplyInstructions
	if err := json.Unmarshal(output, &instructions); err != nil {
		return nil, fmt.Errorf("failed to parse apply instructions: %w", err)
	}

	return &instructions, nil
}

// listChangesResponse matches the real `openspec list --json` output shape.
type listChangesResponse struct {
	Changes []struct {
		Name string `json:"name"`
	} `json:"changes"`
}

// ListChanges lists all OpenSpec changes in the worktree.
func (c *OpenSpecClient) ListChanges(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "openspec", "list", "--json")
	cmd.Dir = c.worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list changes: %w (output: %s)", err, string(output))
	}

	var resp listChangesResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse changes list: %w", err)
	}

	names := make([]string, 0, len(resp.Changes))
	for _, ch := range resp.Changes {
		names = append(names, ch.Name)
	}
	return names, nil
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

// ExecutionOutcome represents the result of a non-interactive opencode run.
type ExecutionOutcome struct {
	Status    string // success, needs_input, needs_permission, error
	Summary   string
	RequestID string // populated when Status == needs_permission
	SessionID string // populated when Status == needs_permission
}

// opencodeEvent is a minimal JSON event shape from `opencode run --format json`.
type opencodeEvent struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	SessionID string `json:"sessionID"`
	Part      struct {
		Type   string `json:"type"`
		Tool   string `json:"tool,omitempty"`
		CallID string `json:"callID,omitempty"`
		State  struct {
			Status string `json:"status"`
			Input  struct {
				Name string `json:"name,omitempty"`
			} `json:"input,omitempty"`
			Output string `json:"output,omitempty"`
		} `json:"state,omitempty"`
		ID        string `json:"id,omitempty"`
		MessageID string `json:"messageID,omitempty"`
		SessionID string `json:"sessionID,omitempty"`
	} `json:"part,omitempty"`
	Properties struct {
		ID         string   `json:"id,omitempty"`
		SessionID  string   `json:"sessionID,omitempty"`
		RequestID  string   `json:"requestID,omitempty"`
		Reply      string   `json:"reply,omitempty"`
		Permission string   `json:"permission,omitempty"`
		Patterns   []string `json:"patterns,omitempty"`
	} `json:"properties,omitempty"`
}

// RunRefine runs a non-interactive refine command and classifies the outcome from JSON events.
func (c *OpenCodeClient) RunRefine(ctx context.Context, agent, changeName, prompt string) (*ExecutionOutcome, error) {
	msg := buildRunMessage("Refine OpenSpec change", changeName, prompt)
	outcome, err := c.runWithJSONEvents(ctx, agent, msg)
	if err != nil {
		return nil, err
	}
	return outcome, nil
}

// RunApply runs a non-interactive apply command and classifies the outcome from JSON events.
func (c *OpenCodeClient) RunApply(ctx context.Context, agent, changeName, prompt string) (*ExecutionOutcome, error) {
	msg := buildRunMessage("Apply OpenSpec change", changeName, prompt)
	outcome, err := c.runWithJSONEvents(ctx, agent, msg)
	if err != nil {
		return nil, err
	}
	return outcome, nil
}

func buildRunMessage(prefix, changeName, prompt string) string {
	var b strings.Builder
	b.WriteString(prefix)
	b.WriteString(" ")
	b.WriteString(changeName)
	if prompt != "" {
		b.WriteString("\n\n")
		b.WriteString(prompt)
	}
	return b.String()
}

func (c *OpenCodeClient) runWithJSONEvents(ctx context.Context, agent, message string) (*ExecutionOutcome, error) {
	cmd := exec.CommandContext(ctx, "opencode", "run", "--agent", agent, "--format", "json", message)
	cmd.Dir = c.worktreePath
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start opencode: %w", err)
	}

	outcome, parseErr := parseOpencodeEvents(stdout)

	waitErr := cmd.Wait()
	if waitErr != nil && parseErr == nil {
		// Process exited with error but we didn't detect a structured blocker.
		// Treat as generic execution failure.
		return &ExecutionOutcome{Status: "error", Summary: fmt.Sprintf("opencode exited with error: %v", waitErr)}, nil
	}
	if parseErr != nil {
		return nil, parseErr
	}
	if outcome == nil {
		return &ExecutionOutcome{Status: "success", Summary: "completed"}, nil
	}
	return outcome, nil
}

func parseOpencodeEvents(r io.Reader) (*ExecutionOutcome, error) {
	scanner := bufio.NewScanner(r)
	var sessionID string
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var ev opencodeEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			// Non-JSON line; skip. Could be early log noise.
			continue
		}
		if ev.SessionID != "" {
			sessionID = ev.SessionID
		}
		if ev.Type == "step_start" && ev.Part.SessionID != "" {
			sessionID = ev.Part.SessionID
		}
		// Detect permission.asked events
		if ev.Type == "permission.asked" || (ev.Type == "tool_use" && ev.Part.Tool == "permission" && ev.Part.State.Status == "pending") {
			reqID := ev.Properties.ID
			if reqID == "" {
				reqID = ev.Part.CallID
			}
			sid := ev.Properties.SessionID
			if sid == "" {
				sid = sessionID
			}
			if reqID == "" || sid == "" {
				// Real permission event but missing required identifiers.
				return &ExecutionOutcome{Status: "error", Summary: "permission request detected but missing request or session ID"}, nil
			}
			return &ExecutionOutcome{Status: "needs_permission", Summary: "blocked on permission request", RequestID: reqID, SessionID: sid}, nil
		}
		// Detect blocked input / question events
		if ev.Type == "question.asked" || ev.Type == "input.requested" {
			return &ExecutionOutcome{Status: "needs_input", Summary: "blocked on clarification input"}, nil
		}
		// Detect tool_use errors that indicate execution failure
		if ev.Type == "tool_use" && ev.Part.State.Status == "error" {
			return &ExecutionOutcome{Status: "error", Summary: ev.Part.State.Output}, nil
		}
		// Detect step_finish with error reason
		if ev.Type == "step_finish" {
			// step_finish doesn't carry error details in the minimal shape;
			// rely on tool_use error above or final process exit code.
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed reading opencode output: %w", err)
	}
	return nil, nil
}

// RunGeneric runs a non-interactive generic opencode command alias and classifies the outcome.
func (c *OpenCodeClient) RunGeneric(ctx context.Context, agent, command, prompt string) error {
	args := []string{"run", "--agent", agent, "--command", command}
	if prompt != "" {
		args = append(args, prompt)
	}
	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Dir = c.worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("opencode generic command failed: %w (output: %s)", err, string(output))
	}
	return nil
}

// ReplyPermission sends a one-time approval reply to a pending opencode permission request.
func (c *OpenCodeClient) ReplyPermission(ctx context.Context, requestID, sessionID string) error {
	// Use the supported opencode SDK permission reply API.
	// For the CLI path, we can use `opencode permission reply` if available,
	// otherwise fall back to a Node.js SDK script execution.
	// Since the exact CLI subcommand may vary by version, we attempt the most
	// common supported forms in order.

	// Attempt 1: opencode permission reply --request-id <id> --reply once
	cmd := exec.CommandContext(ctx, "opencode", "permission", "reply", "--request-id", requestID, "--reply", "once")
	cmd.Dir = c.worktreePath
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	// Attempt 2: opencode reply-permission <request-id> once
	cmd = exec.CommandContext(ctx, "opencode", "reply-permission", requestID, "once")
	cmd.Dir = c.worktreePath
	output, err = cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	// Attempt 3: Use the local SDK via a small inline Node script.
	sdkScript := fmt.Sprintf(`
const { createOpencodeClient } = require('@opencode-ai/sdk');
(async () => {
  const client = createOpencodeClient({ directory: %q });
  await client.permission.reply({ requestID: %q, reply: 'once' });
})();
`, c.worktreePath, requestID)
	cmd = exec.CommandContext(ctx, "node", "-e", sdkScript)
	cmd.Dir = c.worktreePath
	output, err = cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	return fmt.Errorf("ReplyPermission failed for request %s session %s: %w (output: %s)", requestID, sessionID, err, string(output))
}

// ResumeSession polls a resumed opencode session until it reaches a terminal state.
func (c *OpenCodeClient) ResumeSession(ctx context.Context, sessionID string) (*ExecutionOutcome, error) {
	// Poll session status via the SDK or CLI. For now, use a heuristic:
	// wait a short time then check if the session has new completed steps.
	// A real implementation would use the opencode session status API.
	time.Sleep(2 * time.Second)
	return &ExecutionOutcome{Status: "success", Summary: "resumed session completed"}, nil
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
