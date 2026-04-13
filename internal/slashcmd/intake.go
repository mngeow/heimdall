package slashcmd

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mngeow/heimdall/internal/config"
	"github.com/mngeow/heimdall/internal/store"
)

// Intake persists and deduplicates command observations from GitHub polling.
type Intake struct {
	store  *store.Store
	queue  *store.JobQueue
	logger *slog.Logger
	parser *Parser
}

// ProcessResult describes how Symphony handled a polled comment.
type ProcessResult struct {
	Status    string
	Duplicate bool
	Command   *Command
	Request   *store.CommandRequest
	Job       *store.Job
}

// NewIntake creates a new polling intake handler.
func NewIntake(store *store.Store, queue *store.JobQueue, logger *slog.Logger) *Intake {
	return &Intake{
		store:  store,
		queue:  queue,
		logger: logger,
		parser: NewParser(logger),
	}
}

// Process converts a discovered GitHub comment into a persisted command request.
func (i *Intake) Process(ctx context.Context, repoConfig config.RepoConfig, pr *store.PullRequest, commentNodeID, actor, body string) (*ProcessResult, error) {
	dedupeKey := CommandDedupeKey(commentNodeID)
	existing, err := i.store.GetCommandRequestByDedupeKey(ctx, dedupeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check command dedupe key: %w", err)
	}
	if existing != nil {
		return &ProcessResult{Status: "duplicate", Duplicate: true, Request: existing}, nil
	}

	command := i.parser.Parse(body)
	if command == nil {
		return &ProcessResult{Status: "ignored"}, nil
	}

	request := &store.CommandRequest{
		PullRequestID:       pr.ID,
		CommentNodeID:       commentNodeID,
		CommandName:         command.Name,
		CommandArgs:         strings.Join(command.Args, "\n"),
		RequestedAgent:      command.Agent,
		ActorLogin:          actor,
		AuthorizationStatus: "not_checked",
		DedupeKey:           dedupeKey,
		Status:              "ignored",
	}

	if !command.IsValid {
		request.Status = "rejected"
		if err := i.store.SaveCommandRequest(ctx, request); err != nil {
			return nil, fmt.Errorf("failed to save rejected command request: %w", err)
		}
		return &ProcessResult{Status: request.Status, Command: command, Request: request}, nil
	}

	authorizer := NewAuthorizer(repoConfig, i.logger)
	authorization := authorizer.Authorize(actor, command)
	if !authorization.Authorized {
		request.AuthorizationStatus = "rejected"
		request.Status = "rejected"
		if err := i.store.SaveCommandRequest(ctx, request); err != nil {
			return nil, fmt.Errorf("failed to save unauthorized command request: %w", err)
		}
		return &ProcessResult{Status: request.Status, Command: command, Request: request}, nil
	}

	request.AuthorizationStatus = "authorized"
	request.Status = "queued"
	if err := i.store.SaveCommandRequest(ctx, request); err != nil {
		return nil, fmt.Errorf("failed to save command request: %w", err)
	}

	requestID := request.ID
	job := &store.Job{
		CommandRequestID: &requestID,
		JobType:          fmt.Sprintf("pr_command_%s", command.Name),
		LockKey:          store.CreatePullRequestLockKey(pr.ID),
		Status:           "queued",
	}
	if err := i.queue.Enqueue(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to enqueue command job: %w", err)
	}

	return &ProcessResult{Status: request.Status, Command: command, Request: request, Job: job}, nil
}

// CommandDedupeKey returns the durable dedupe key for a GitHub command comment.
func CommandDedupeKey(commentNodeID string) string {
	return "github-comment:" + commentNodeID
}
