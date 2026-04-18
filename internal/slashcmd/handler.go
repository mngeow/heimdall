package slashcmd

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mngeow/heimdall/internal/config"
	"github.com/mngeow/heimdall/internal/store"
)

// Parser handles parsing of PR comment commands
type Parser struct {
	logger *slog.Logger
}

// NewParser creates a new command parser
func NewParser(logger *slog.Logger) *Parser {
	return &Parser{logger: logger}
}

// Command represents a parsed command
type Command struct {
	Name       string
	Args       []string
	Agent      string
	ChangeName string
	Alias      string
	PromptTail string
	RequestID  string
	Raw        string
	IsValid    bool
	Error      string
}

// Parse parses a comment body for commands.
// It finds the first command line, then preserves the rest of the comment
// as prompt tail when the command uses a standalone "--" separator.
func (p *Parser) Parse(comment string) *Command {
	lines := strings.Split(comment, "\n")

	var commandLine string
	var commandLineIndex int

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "/heimdall ") {
			commandLine = trimmed
			commandLineIndex = i
			break
		}
		if strings.HasPrefix(trimmed, "/opsx-apply") {
			commandLine = trimmed
			commandLineIndex = i
			break
		}
		if strings.HasPrefix(trimmed, "/opsx-archive") {
			commandLine = trimmed
			commandLineIndex = i
			break
		}
	}

	if commandLine == "" {
		return nil
	}

	var cmd *Command
	if strings.HasPrefix(commandLine, "/heimdall ") {
		cmd = p.parseHeimdallCommand(commandLine)
	} else if strings.HasPrefix(commandLine, "/opsx-apply") {
		cmd = p.parseOpsxApplyCommand(commandLine)
	} else if strings.HasPrefix(commandLine, "/opsx-archive") {
		cmd = p.parseOpsxArchiveCommand(commandLine)
	}

	if cmd == nil {
		return nil
	}

	// Preserve multiline prompt tail after the first standalone "--"
	// on the command line, or when the command line ends with "--"
	// and the prompt continues on later lines.
	if idx := strings.Index(commandLine, " -- "); idx != -1 {
		cmd.PromptTail = strings.TrimSpace(commandLine[idx+4:])
	} else if strings.HasSuffix(commandLine, " --") {
		// Prompt continues on subsequent lines
		if commandLineIndex+1 < len(lines) {
			promptLines := lines[commandLineIndex+1:]
			cmd.PromptTail = strings.TrimSpace(strings.Join(promptLines, "\n"))
		}
	}

	return cmd
}

func (p *Parser) parseHeimdallCommand(line string) *Command {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return &Command{IsValid: false, Error: "missing subcommand"}
	}

	subcommand := parts[1]

	switch subcommand {
	case "status":
		return &Command{
			Name:    "status",
			Raw:     line,
			IsValid: true,
		}
	case "refine", "apply", "opencode":
		return p.parseAgentDrivenCommand(subcommand, line)
	case "approve":
		return p.parseApproveCommand(line)
	default:
		return &Command{
			Name:    subcommand,
			IsValid: false,
			Error:   fmt.Sprintf("unknown subcommand: %s", subcommand),
		}
	}
}

func (p *Parser) parseAgentDrivenCommand(name, line string) *Command {
	cmd := &Command{
		Name:    name,
		Raw:     line,
		IsValid: true,
	}

	// Split on standalone "--" to separate arguments from prompt tail for inline prompts
	var argsLine string
	if idx := strings.Index(line, " -- "); idx != -1 {
		argsLine = line[:idx]
	} else {
		argsLine = line
	}
	// Remove trailing " --" so it does not interfere with positional parsing
	if strings.HasSuffix(argsLine, " --") {
		argsLine = strings.TrimSuffix(argsLine, " --")
	}

	parts := strings.Fields(argsLine)
	if name == "opencode" && len(parts) >= 3 {
		cmd.Alias = parts[2]
	}

	for i, part := range parts {
		if part == "--agent" && i+1 < len(parts) {
			cmd.Agent = parts[i+1]
		}
	}

	// Extract optional change-name (first positional after subcommand/alias)
	posIndex := 2
	if name == "opencode" {
		posIndex = 3
	}
	for i := posIndex; i < len(parts); i++ {
		if strings.HasPrefix(parts[i], "--") {
			// Skip flags and their values
			if parts[i] == "--agent" && i+1 < len(parts) {
				i++ // skip the agent value too
			}
			continue
		}
		cmd.ChangeName = parts[i]
		break
	}

	if cmd.Agent == "" {
		cmd.IsValid = false
		cmd.Error = "missing --agent flag"
	}

	return cmd
}

func (p *Parser) parseOpsxApplyCommand(line string) *Command {
	cmd := p.parseAgentDrivenCommand("apply", line)
	cmd.Raw = line
	return cmd
}

func (p *Parser) parseOpsxArchiveCommand(line string) *Command {
	return &Command{
		Name:    "archive",
		Raw:     line,
		IsValid: true,
	}
}

func (p *Parser) parseApproveCommand(line string) *Command {
	parts := strings.Fields(line)
	cmd := &Command{
		Name:    "approve",
		Raw:     line,
		IsValid: true,
	}
	if len(parts) >= 3 {
		cmd.RequestID = parts[2]
	} else {
		cmd.IsValid = false
		cmd.Error = "missing permission request ID"
	}
	return cmd
}

// Authorizer handles command authorization
type Authorizer struct {
	repoConfig config.RepoConfig
	logger     *slog.Logger
}

// NewAuthorizer creates a new command authorizer
func NewAuthorizer(repoConfig config.RepoConfig, logger *slog.Logger) *Authorizer {
	return &Authorizer{
		repoConfig: repoConfig,
		logger:     logger,
	}
}

// AuthorizationResult represents the result of an authorization check
type AuthorizationResult struct {
	Authorized bool
	Reason     string
}

// Authorize checks if a user is authorized to execute a command
func (a *Authorizer) Authorize(actor string, cmd *Command) *AuthorizationResult {
	// Check if user is in allowed list
	allowed := false
	for _, user := range a.repoConfig.AllowedUsers {
		if user == actor {
			allowed = true
			break
		}
	}

	if !allowed {
		return &AuthorizationResult{
			Authorized: false,
			Reason:     fmt.Sprintf("user %s is not in allowed users list", actor),
		}
	}

	// Agent authorization for agent-driven commands
	if (cmd.Name == "refine" || cmd.Name == "apply" || cmd.Name == "opencode") && cmd.Agent != "" {
		agentAllowed := false
		for _, agent := range a.repoConfig.AllowedAgents {
			if agent == cmd.Agent {
				agentAllowed = true
				break
			}
		}

		if !agentAllowed {
			return &AuthorizationResult{
				Authorized: false,
				Reason:     fmt.Sprintf("agent %s is not in allowed agents list", cmd.Agent),
			}
		}
	}

	// Alias authorization for opencode commands
	if cmd.Name == "opencode" && cmd.Alias != "" {
		if _, ok := a.repoConfig.OpencodeAliases[cmd.Alias]; !ok {
			return &AuthorizationResult{
				Authorized: false,
				Reason:     fmt.Sprintf("alias %s is not configured for this repository", cmd.Alias),
			}
		}
	}

	return &AuthorizationResult{Authorized: true}
}

// Handler handles command execution
type Handler struct {
	store  *store.Store
	queue  *store.JobQueue
	logger *slog.Logger
}

// NewHandler creates a new command handler
func NewHandler(store *store.Store, queue *store.JobQueue, logger *slog.Logger) *Handler {
	return &Handler{
		store:  store,
		queue:  queue,
		logger: logger,
	}
}

// Handle processes a command
func (h *Handler) Handle(ctx context.Context, cmd *Command, prID int64, actor string) error {
	h.logger.Info("handling command", "command", cmd.Name, "actor", actor)

	switch cmd.Name {
	case "status":
		return h.handleStatus(ctx, prID)
	case "refine":
		return h.handleRefine(ctx, cmd, prID, actor)
	case "apply":
		return h.handleApply(ctx, cmd, prID, actor)
	case "opencode":
		return h.handleOpencode(ctx, cmd, prID, actor)
	case "approve":
		return h.handleApprove(ctx, cmd, prID, actor)
	default:
		return fmt.Errorf("unknown command: %s", cmd.Name)
	}
}

func (h *Handler) handleStatus(ctx context.Context, prID int64) error {
	// TODO: Get PR status and post comment
	h.logger.Info("handling status command", "pr_id", prID)
	return nil
}

func (h *Handler) handleRefine(ctx context.Context, cmd *Command, prID int64, actor string) error {
	// TODO: Enqueue refine workflow
	h.logger.Info("handling refine command", "pr_id", prID, "change", cmd.ChangeName, "agent", cmd.Agent, "prompt", cmd.PromptTail)
	return nil
}

func (h *Handler) handleApply(ctx context.Context, cmd *Command, prID int64, actor string) error {
	// TODO: Enqueue apply workflow
	h.logger.Info("handling apply command", "pr_id", prID, "change", cmd.ChangeName, "agent", cmd.Agent, "prompt", cmd.PromptTail)
	return nil
}

func (h *Handler) handleOpencode(ctx context.Context, cmd *Command, prID int64, actor string) error {
	// TODO: Enqueue generic opencode workflow
	h.logger.Info("handling opencode command", "pr_id", prID, "change", cmd.ChangeName, "agent", cmd.Agent, "alias", cmd.Alias, "prompt", cmd.PromptTail)
	return nil
}

func (h *Handler) handleApprove(ctx context.Context, cmd *Command, prID int64, actor string) error {
	// TODO: Enqueue approval workflow
	h.logger.Info("handling approve command", "pr_id", prID, "request_id", cmd.RequestID)
	return nil
}
