package slashcmd

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mngeow/heimdall/internal/config"
	"github.com/mngeow/heimdall/internal/scm/github"
	"github.com/mngeow/heimdall/internal/store"
)

// Intake persists and deduplicates command observations from GitHub polling.
type Intake struct {
	store     *store.Store
	queue     *store.JobQueue
	logger    *slog.Logger
	parser    *Parser
	github    intakeGitHubClient
	publicURL string
}

type intakeGitHubClient interface {
	AddReaction(ctx context.Context, owner, repo string, commentID int64, content string) error
	CreateComment(ctx context.Context, owner, repo string, number int, body string) error
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
func NewIntake(store *store.Store, queue *store.JobQueue, logger *slog.Logger, githubClient intakeGitHubClient, publicURL string) *Intake {
	return &Intake{
		store:     store,
		queue:     queue,
		logger:    logger,
		parser:    NewParser(logger),
		github:    githubClient,
		publicURL: publicURL,
	}
}

// Process converts a discovered GitHub comment into a persisted command request.
func (i *Intake) Process(ctx context.Context, repoConfig config.RepoConfig, pr *store.PullRequest, commentNodeID string, commentID int64, actor, body string) (*ProcessResult, error) {
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
		ChangeName:          command.ChangeName,
		Alias:               command.Alias,
		PromptTail:          command.PromptTail,
		RequestID:           command.RequestID,
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
	jobType := fmt.Sprintf("pr_command_%s", command.Name)
	if command.Name == "apply" && strings.HasPrefix(body, "/opsx-apply") {
		// Compatibility alias: still stored as apply, job type remains pr_command_apply
	}
	job := &store.Job{
		CommandRequestID: &requestID,
		JobType:          jobType,
		LockKey:          store.CreatePullRequestLockKey(pr.ID),
		Status:           "queued",
	}
	if err := i.queue.Enqueue(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to enqueue command job: %w", err)
	}

	// Post acceptance feedback asynchronously; failures must not prevent execution.
	i.sendAcceptanceFeedback(ctx, repoConfig, pr, commentID, request, command)

	return &ProcessResult{Status: request.Status, Command: command, Request: request, Job: job}, nil
}

func (i *Intake) sendAcceptanceFeedback(ctx context.Context, repoConfig config.RepoConfig, pr *store.PullRequest, commentID int64, req *store.CommandRequest, cmd *Command) {
	owner, repoName, err := github.ParseRepoRef(repoConfig.Name)
	if err != nil {
		i.logger.Error("failed to parse repo ref for feedback", "repo", repoConfig.Name, "error", err)
		return
	}

	if !req.FeedbackReactionPosted {
		if err := i.github.AddReaction(ctx, owner, repoName, commentID, "eyes"); err != nil {
			i.logger.Error("failed to post eyes reaction", "command_request_id", req.ID, "error", err)
		} else {
			req.FeedbackReactionPosted = true
			if saveErr := i.store.SaveCommandRequest(ctx, req); saveErr != nil {
				i.logger.Error("failed to save reaction posted flag", "command_request_id", req.ID, "error", saveErr)
			}
		}
	}

	if isOpencodeBackedCommand(cmd.Name) && !req.FeedbackLinkPosted {
		url := buildCommandRunURL(i.publicURL, req.ID)
		msg := fmt.Sprintf("Heimdall is processing this command. [View live output →](%s)", url)
		if err := i.github.CreateComment(ctx, owner, repoName, pr.Number, msg); err != nil {
			i.logger.Error("failed to post command link comment", "command_request_id", req.ID, "error", err)
		} else {
			req.FeedbackLinkPosted = true
			if saveErr := i.store.SaveCommandRequest(ctx, req); saveErr != nil {
				i.logger.Error("failed to save link posted flag", "command_request_id", req.ID, "error", saveErr)
			}
		}
	}
}

func isOpencodeBackedCommand(name string) bool {
	switch name {
	case "refine", "apply", "opencode":
		return true
	}
	return false
}

func buildCommandRunURL(baseURL string, commandRequestID int64) string {
	base := strings.TrimSuffix(baseURL, "/")
	return fmt.Sprintf("%s/ui/command-runs/%d", base, commandRequestID)
}

// CommandDedupeKey returns the durable dedupe key for a GitHub command comment.
func CommandDedupeKey(commentNodeID string) string {
	return "github-comment:" + commentNodeID
}
