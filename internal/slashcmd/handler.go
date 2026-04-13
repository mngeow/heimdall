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
	Name    string
	Args    []string
	Agent   string
	Raw     string
	IsValid bool
	Error   string
}

// Parse parses a comment body for commands
func (p *Parser) Parse(comment string) *Command {
	lines := strings.Split(comment, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for /heimdall commands
		if strings.HasPrefix(line, "/heimdall ") {
			return p.parseHeimdallCommand(line)
		}

		// Check for /opsx-apply
		if strings.HasPrefix(line, "/opsx-apply") {
			return p.parseOpsxApplyCommand(line)
		}

		// Check for /opsx-archive
		if strings.HasPrefix(line, "/opsx-archive") {
			return p.parseOpsxArchiveCommand(line)
		}
	}

	return nil
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
	case "refine":
		instruction := strings.TrimSpace(strings.TrimPrefix(line, "/heimdall refine"))
		return &Command{
			Name:    "refine",
			Args:    []string{instruction},
			Raw:     line,
			IsValid: true,
		}
	default:
		return &Command{
			Name:    subcommand,
			IsValid: false,
			Error:   fmt.Sprintf("unknown subcommand: %s", subcommand),
		}
	}
}

func (p *Parser) parseOpsxApplyCommand(line string) *Command {
	parts := strings.Fields(line)
	cmd := &Command{
		Name:    "apply",
		Raw:     line,
		IsValid: true,
	}

	// Parse arguments
	for i, part := range parts {
		if part == "--agent" && i+1 < len(parts) {
			cmd.Agent = parts[i+1]
		}
	}

	// Agent is required
	if cmd.Agent == "" {
		cmd.IsValid = false
		cmd.Error = "missing --agent flag"
	}

	return cmd
}

func (p *Parser) parseOpsxArchiveCommand(line string) *Command {
	return &Command{
		Name:    "archive",
		Raw:     line,
		IsValid: true,
	}
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

	// Check agent authorization for apply commands
	if cmd.Name == "apply" && cmd.Agent != "" {
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
	h.logger.Info("handling refine command", "pr_id", prID, "instruction", cmd.Args)
	return nil
}

func (h *Handler) handleApply(ctx context.Context, cmd *Command, prID int64, actor string) error {
	// TODO: Enqueue apply workflow
	h.logger.Info("handling apply command", "pr_id", prID, "agent", cmd.Agent)
	return nil
}
