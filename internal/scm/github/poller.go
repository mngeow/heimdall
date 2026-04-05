package github

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	gh "github.com/google/go-github/v57/github"
	"github.com/mngeow/symphony/internal/store"
)

type pollAPI interface {
	ListIssueCommentsSince(ctx context.Context, owner, repo string, since time.Time) ([]*gh.IssueComment, error)
	GetPullRequest(ctx context.Context, owner, repo string, number int) (*gh.PullRequest, error)
}

// Poller discovers GitHub activity for Symphony-managed pull requests.
type Poller struct {
	api            pollAPI
	store          *store.Store
	lookbackWindow time.Duration
	now            func() time.Time
}

// PollResult contains runtime inputs discovered during a polling cycle.
type PollResult struct {
	Commands    []DiscoveredCommand
	Reconciled  []store.PullRequest
	Checkpoints map[string]time.Time
}

// DiscoveredCommand is a candidate command comment from a managed pull request.
type DiscoveredCommand struct {
	RepoRef       string
	PullRequest   *store.PullRequest
	CommentNodeID string
	ActorLogin    string
	Body          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewPoller creates a new GitHub poller.
func NewPoller(api pollAPI, store *store.Store, lookbackWindow time.Duration) *Poller {
	return &Poller{
		api:            api,
		store:          store,
		lookbackWindow: lookbackWindow,
		now:            time.Now,
	}
}

// Poll reads GitHub comments and pull request state for managed repositories.
func (p *Poller) Poll(ctx context.Context) (*PollResult, error) {
	repositories, err := p.store.ListActiveRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	result := &PollResult{Checkpoints: make(map[string]time.Time, len(repositories))}
	for _, repository := range repositories {
		repositoryResult, checkpoint, err := p.pollRepository(ctx, repository)
		if err != nil {
			return nil, err
		}
		result.Commands = append(result.Commands, repositoryResult.Commands...)
		result.Reconciled = append(result.Reconciled, repositoryResult.Reconciled...)
		result.Checkpoints[repository.RepoRef] = checkpoint
	}

	return result, nil
}

func (p *Poller) pollRepository(ctx context.Context, repository store.Repository) (*PollResult, time.Time, error) {
	owner, repoName, err := ParseRepoRef(repository.RepoRef)
	if err != nil {
		return nil, time.Time{}, err
	}

	checkpointTime := p.now().UTC()
	since := checkpointTime.Add(-p.lookbackWindow)
	lastCheckpoint, err := p.store.GetGitHubPollCheckpoint(ctx, repository.RepoRef)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to load github poll checkpoint for %s: %w", repository.RepoRef, err)
	}
	if lastCheckpoint != nil {
		since = lastCheckpoint.Add(-p.lookbackWindow)
	}

	managedPRs, err := p.store.ListManagedPullRequests(ctx, repository.ID)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to list managed pull requests for %s: %w", repository.RepoRef, err)
	}

	result := &PollResult{}
	if len(managedPRs) == 0 {
		if err := p.store.SetGitHubPollCheckpoint(ctx, repository.RepoRef, checkpointTime); err != nil {
			return nil, time.Time{}, fmt.Errorf("failed to update github poll checkpoint for %s: %w", repository.RepoRef, err)
		}
		return result, checkpointTime, nil
	}

	managedByNumber := make(map[int]*store.PullRequest, len(managedPRs))
	eligibleNumbers := make(map[int]struct{}, len(managedPRs))
	for i := range managedPRs {
		pr := managedPRs[i]
		managedByNumber[pr.Number] = &pr
	}

	for _, managedPR := range managedPRs {
		remotePR, err := p.api.GetPullRequest(ctx, owner, repoName, managedPR.Number)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("failed to reconcile pull request %s#%d: %w", repository.RepoRef, managedPR.Number, err)
		}
		if repository.PRMonitorLabel != "" && !pullRequestHasLabel(remotePR, repository.PRMonitorLabel) {
			continue
		}
		eligibleNumbers[managedPR.Number] = struct{}{}

		updatedPR := managedPR
		updatedPR.ProviderPRNodeID = remotePR.GetNodeID()
		updatedPR.Title = remotePR.GetTitle()
		updatedPR.BaseBranch = remotePR.GetBase().GetRef()
		updatedPR.HeadBranch = remotePR.GetHead().GetRef()
		updatedPR.State = pullRequestState(remotePR)
		updatedPR.URL = remotePR.GetHTMLURL()

		if err := p.store.SavePullRequest(ctx, &updatedPR); err != nil {
			return nil, time.Time{}, fmt.Errorf("failed to save reconciled pull request %s#%d: %w", repository.RepoRef, managedPR.Number, err)
		}

		if managedPR.ProviderPRNodeID != updatedPR.ProviderPRNodeID || managedPR.Title != updatedPR.Title || managedPR.BaseBranch != updatedPR.BaseBranch || managedPR.HeadBranch != updatedPR.HeadBranch || managedPR.State != updatedPR.State || managedPR.URL != updatedPR.URL {
			result.Reconciled = append(result.Reconciled, updatedPR)
		}
	}

	if len(eligibleNumbers) > 0 {
		comments, err := p.api.ListIssueCommentsSince(ctx, owner, repoName, since)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("failed to poll issue comments for %s: %w", repository.RepoRef, err)
		}

		for _, comment := range comments {
			issueNumber, ok := pullRequestNumberFromComment(comment)
			if !ok {
				continue
			}
			if _, ok := eligibleNumbers[issueNumber]; !ok {
				continue
			}

			managedPR := managedByNumber[issueNumber]
			if managedPR == nil {
				continue
			}

			result.Commands = append(result.Commands, DiscoveredCommand{
				RepoRef:       repository.RepoRef,
				PullRequest:   managedPR,
				CommentNodeID: commentIdentity(comment),
				ActorLogin:    comment.GetUser().GetLogin(),
				Body:          comment.GetBody(),
				CreatedAt:     comment.GetCreatedAt().Time,
				UpdatedAt:     comment.GetUpdatedAt().Time,
			})
		}
	}

	if err := p.store.SetGitHubPollCheckpoint(ctx, repository.RepoRef, checkpointTime); err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to update github poll checkpoint for %s: %w", repository.RepoRef, err)
	}

	return result, checkpointTime, nil
}

func commentIdentity(comment *gh.IssueComment) string {
	if nodeID := comment.GetNodeID(); nodeID != "" {
		return nodeID
	}
	return fmt.Sprintf("issue-comment:%d", comment.GetID())
}

func pullRequestNumberFromComment(comment *gh.IssueComment) (int, bool) {
	issueURL := strings.TrimSpace(comment.GetIssueURL())
	if issueURL == "" {
		return 0, false
	}

	parts := strings.Split(strings.TrimRight(issueURL, "/"), "/")
	if len(parts) == 0 {
		return 0, false
	}

	issueNumber, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, false
	}

	return issueNumber, true
}

func pullRequestState(pr *gh.PullRequest) string {
	if pr.GetMerged() {
		return "merged"
	}
	return pr.GetState()
}

func pullRequestHasLabel(pr *gh.PullRequest, label string) bool {
	if strings.TrimSpace(label) == "" {
		return true
	}
	for _, existing := range pr.Labels {
		if strings.EqualFold(existing.GetName(), label) {
			return true
		}
	}
	return false
}
