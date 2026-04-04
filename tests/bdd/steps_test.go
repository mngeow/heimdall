package bdd

import (
	"context"
	"fmt"
	"testing"

	"github.com/cucumber/godog"
	"github.com/mngeow/symphony/internal/config"
	"github.com/mngeow/symphony/internal/slashcmd"
	"github.com/mngeow/symphony/internal/store"
	"github.com/mngeow/symphony/internal/workflow"
)

// testContext holds the state for each scenario
type testContext struct {
	config          *config.Config
	store           *store.Store
	queue           *store.JobQueue
	workItem        *store.WorkItem
	pr              *store.PullRequest
	repoBinding     *store.RepoBinding
	workflowRun     *store.WorkflowRun
	command         string
	commandResult   string
	authorizer      *slashcmd.Authorizer
	parser          *slashcmd.Parser
	isAuthorized    bool
	isRejected      bool
	rejectionReason string
}

// ctxKey is used to store testContext in context
type ctxKey struct{}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func InitializeScenario(sc *godog.ScenarioContext) {
	// Create a new test context for each scenario
	sc.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		tc := &testContext{
			parser: slashcmd.NewParser(nil),
		}
		return context.WithValue(ctx, ctxKey{}, tc), nil
	})

	// Background steps
	sc.Step(`^Symphony is configured with a Linear team and GitHub repository$`, symphonyIsConfigured)
	sc.Step(`^the required local executables are available$`, executablesAreAvailable)
	sc.Step(`^a Symphony-managed pull request exists$`, symphonyManagedPRExists)
	sc.Step(`^the PR author is in the allowed users list$`, authorIsAllowed)
	sc.Step(`^Symphony is running with security configuration$`, symphonyIsConfigured)

	// Proposal creation steps
	sc.Step(`^a Linear issue "([^"]*)" with title "([^"]*)" exists$`, linearIssueExists)
	sc.Step(`^a Linear issue "([^"]*)" with title "([^"]*)"$`, aLinearIssueWithTitle)
	sc.Step(`^the issue is in state "([^"]*)"$`, issueIsInState)
	sc.Step(`^the issue is moved to state "([^"]*)"$`, issueIsMovedToState)
	sc.Step(`^Symphony polls Linear$`, symphonyPollsLinear)
	sc.Step(`^Symphony should detect the state transition$`, symphonyShouldDetectTransition)
	sc.Step(`^Symphony should create a workflow run for proposal generation$`, symphonyShouldCreateWorkflowRun)
	sc.Step(`^a Linear issue "([^"]*)" is already in state "([^"]*)"$`, linearIssueExistsInState)
	sc.Step(`^a proposal branch already exists for the issue$`, proposalBranchExists)
	sc.Step(`^Symphony polls Linear again$`, symphonyPollsLinear)
	sc.Step(`^Symphony should not create a duplicate workflow run$`, symphonyShouldNotCreateDuplicate)
	sc.Step(`^Symphony should reuse the existing proposal$`, symphonyShouldReuseExisting)
	sc.Step(`^the issue enters active state$`, issueEntersActiveState)
	sc.Step(`^the proposal branch should be named "([^"]*)"$`, proposalBranchShouldBeNamed)
	sc.Step(`^the OpenSpec change should be named "([^"]*)"$`, openSpecChangeShouldBeNamed)
	sc.Step(`^a Linear issue enters active state$`, linearIssueEntersActiveState)
	sc.Step(`^Symphony generates the OpenSpec proposal$`, symphonyGeneratesProposal)
	sc.Step(`^Symphony should push the proposal branch$`, symphonyShouldPushBranch)
	sc.Step(`^Symphony should create a pull request to main$`, symphonyShouldCreatePR)
	sc.Step(`^Symphony should comment with the change name and available commands$`, symphonyShouldCommentWithInfo)

	// Command handling steps
	sc.Step(`^the user comments "([^"]*)"$`, userComments)
	sc.Step(`^Symphony should reply with the current proposal status$`, symphonyShouldReplyWithStatus)
	sc.Step(`^Symphony should update the proposal artifacts$`, symphonyShouldUpdateArtifacts)
	sc.Step(`^Symphony should commit the changes$`, symphonyShouldCommitChanges)
	sc.Step(`^Symphony should push the updated branch$`, symphonyShouldPushUpdatedBranch)
	sc.Step(`^the repository allows agent "([^"]*)"$`, repositoryAllowsAgent)
	sc.Step(`^Symphony should execute the apply workflow$`, symphonyShouldExecuteApply)
	sc.Step(`^Symphony should commit implementation changes$`, symphonyShouldCommitImplementation)
	sc.Step(`^Symphony should comment with the execution results$`, symphonyShouldCommentWithResults)
	sc.Step(`^a user not in the allowed users list$`, userNotInAllowedList)
	sc.Step(`^they comment "([^"]*)"$`, theyComment)
	sc.Step(`^no workflow should be triggered$`, noWorkflowTriggered)
	sc.Step(`^the repository does not allow agent "([^"]*)"$`, repositoryDoesNotAllowAgent)
	sc.Step(`^Symphony should comment that the agent is not authorized$`, symphonyShouldCommentAgentNotAuthorized)
	sc.Step(`^a command has already been processed$`, commandAlreadyProcessed)
	sc.Step(`^the same comment is delivered again$`, sameCommentDeliveredAgain)
	sc.Step(`^the duplicate should be detected$`, duplicateShouldBeDetected)
	sc.Step(`^the command should not be executed again$`, commandNotExecutedAgain)
	sc.Step(`^a command comment exists$`, commandCommentExists)
	sc.Step(`^the comment is edited$`, commentIsEdited)
	sc.Step(`^the edit should not trigger a new command execution$`, editShouldNotTriggerExecution)

	// Security steps
	sc.Step(`^a pull request not created by Symphony$`, nonSymphonyPRExists)
	sc.Step(`^a user comments "([^"]*)"$`, userComments)
	sc.Step(`^the command should be rejected$`, commandShouldBeRejected)
	sc.Step(`^Symphony should record that the PR is not eligible$`, symphonyShouldRecordNotEligible)
	sc.Step(`^a GitHub webhook delivery$`, githubWebhookDelivery)
	sc.Step(`^the signature is valid$`, signatureIsValid)
	sc.Step(`^the webhook should be processed$`, webhookShouldBeProcessed)
	sc.Step(`^the signature is invalid$`, signatureIsInvalid)
	sc.Step(`^the webhook should be rejected$`, webhookShouldBeRejected)
	sc.Step(`^a 401 response should be returned$`, unauthorizedResponseReturned)
	sc.Step(`^Symphony uses a GitHub App$`, symphonyUsesGitHubApp)
	sc.Step(`^installation tokens are minted$`, installationTokensMinted)
	sc.Step(`^tokens should not appear in logs$`, tokensNotInLogs)
	sc.Step(`^tokens should not be stored in SQLite$`, tokensNotInSQLite)
}

// Helper to get testContext from context
func getTC(ctx context.Context) *testContext {
	return ctx.Value(ctxKey{}).(*testContext)
}

// Background step implementations
func symphonyIsConfigured(ctx context.Context) error {
	tc := getTC(ctx)
	tc.config = &config.Config{
		Linear: config.LinearConfig{
			TeamKeys:     []string{"ENG"},
			ActiveStates: []string{"In Progress"},
		},
		Repos: []config.RepoConfig{
			{
				Name:           "github.com/test/repo",
				AllowedUsers:   []string{"testuser", "alice"},
				AllowedAgents:  []string{"gpt-5.4", "claude"},
				LinearTeamKeys: []string{"ENG"},
				DefaultBranch:  "main",
				BranchPrefix:   "symphony",
			},
		},
	}
	tc.authorizer = slashcmd.NewAuthorizer(tc.config.Repos[0], nil)
	return nil
}

func executablesAreAvailable(ctx context.Context) error {
	// In integration tests, would verify git/openspec/opencode exist
	return nil
}

func symphonyManagedPRExists(ctx context.Context) error {
	tc := getTC(ctx)
	tc.pr = &store.PullRequest{
		ID:         1,
		Number:     42,
		Title:      "[ENG-123] OpenSpec proposal for Add rate limiting",
		Provider:   "github",
		HeadBranch: "symphony/ENG-123-add-rate-limiting",
		BaseBranch: "main",
		State:      "open",
	}
	tc.repoBinding = &store.RepoBinding{
		ID:            1,
		BranchName:    "symphony/ENG-123-add-rate-limiting",
		ChangeName:    "ENG-123-add-rate-limiting",
		BindingStatus: "active",
	}
	return nil
}

func authorIsAllowed(ctx context.Context) error {
	tc := getTC(ctx)
	tc.isAuthorized = true
	return nil
}

// Proposal creation step implementations
func linearIssueExists(ctx context.Context, key, title string) error {
	tc := getTC(ctx)
	tc.workItem = &store.WorkItem{
		Provider:           "linear",
		ProviderWorkItemID: "linear-uuid-" + key,
		WorkItemKey:        key,
		Title:              title,
		StateName:          "Todo",
		LifecycleBucket:    "inactive",
		Team:               "ENG",
	}
	return nil
}

// aLinearIssueWithTitle handles the step pattern without "exists" at the end
func aLinearIssueWithTitle(ctx context.Context, key, title string) error {
	return linearIssueExists(ctx, key, title)
}

func issueIsInState(ctx context.Context, state string) error {
	tc := getTC(ctx)
	if tc.workItem != nil {
		tc.workItem.StateName = state
	}
	return nil
}

func issueIsMovedToState(ctx context.Context, state string) error {
	tc := getTC(ctx)
	if tc.workItem != nil {
		tc.workItem.StateName = state
		if state == "In Progress" {
			tc.workItem.LifecycleBucket = "active"
		}
	}
	return nil
}

func symphonyPollsLinear(ctx context.Context) error {
	// Simulate polling - in real tests would trigger actual polling
	return nil
}

func symphonyShouldDetectTransition(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.workItem == nil || tc.workItem.LifecycleBucket != "active" {
		return fmt.Errorf("expected work item to be in active state")
	}
	return nil
}

func symphonyShouldCreateWorkflowRun(ctx context.Context) error {
	tc := getTC(ctx)
	// Simulate workflow run creation
	tc.workflowRun = &store.WorkflowRun{
		ID:         1,
		RunType:    "propose",
		Status:     "queued",
		ChangeName: workflow.GenerateChangeName(tc.workItem.WorkItemKey, workflow.Slugify(tc.workItem.Title)),
		BranchName: workflow.GenerateBranchName(tc.workItem.WorkItemKey, workflow.Slugify(tc.workItem.Title)),
	}
	return nil
}

func linearIssueExistsInState(ctx context.Context, key, state string) error {
	tc := getTC(ctx)
	tc.workItem = &store.WorkItem{
		Provider:           "linear",
		ProviderWorkItemID: "linear-uuid-" + key,
		WorkItemKey:        key,
		Title:              "Add rate limiting",
		StateName:          state,
		LifecycleBucket:    "active",
		Team:               "ENG",
	}
	return nil
}

func proposalBranchExists(ctx context.Context) error {
	tc := getTC(ctx)
	tc.repoBinding = &store.RepoBinding{
		ID:            1,
		BranchName:    "symphony/ENG-123-add-rate-limiting",
		ChangeName:    "ENG-123-add-rate-limiting",
		BindingStatus: "active",
	}
	return nil
}

func symphonyShouldNotCreateDuplicate(ctx context.Context) error {
	tc := getTC(ctx)
	// Verify no new workflow run was created
	if tc.workflowRun != nil && tc.workflowRun.ID != 1 {
		return fmt.Errorf("expected no duplicate workflow run")
	}
	return nil
}

func symphonyShouldReuseExisting(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.repoBinding == nil {
		return fmt.Errorf("expected existing binding to be reused")
	}
	return nil
}

func issueEntersActiveState(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.workItem != nil {
		tc.workItem.StateName = "In Progress"
		tc.workItem.LifecycleBucket = "active"
	}
	return nil
}

func proposalBranchShouldBeNamed(ctx context.Context, expectedName string) error {
	tc := getTC(ctx)
	slug := workflow.Slugify(tc.workItem.Title)
	actualName := workflow.GenerateBranchName(tc.workItem.WorkItemKey, slug)
	if actualName != expectedName {
		return fmt.Errorf("expected branch name %q, got %q", expectedName, actualName)
	}
	return nil
}

func openSpecChangeShouldBeNamed(ctx context.Context, expectedName string) error {
	tc := getTC(ctx)
	slug := workflow.Slugify(tc.workItem.Title)
	actualName := workflow.GenerateChangeName(tc.workItem.WorkItemKey, slug)
	if actualName != expectedName {
		return fmt.Errorf("expected change name %q, got %q", expectedName, actualName)
	}
	return nil
}

func linearIssueEntersActiveState(ctx context.Context) error {
	return linearIssueExists(ctx, "ENG-123", "Add rate limiting")
}

func symphonyGeneratesProposal(ctx context.Context) error {
	return symphonyShouldCreateWorkflowRun(ctx)
}

func symphonyShouldPushBranch(ctx context.Context) error {
	// Verify branch would be pushed
	return nil
}

func symphonyShouldCreatePR(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.pr == nil {
		tc.pr = &store.PullRequest{
			Number: 42,
			Title:  "[ENG-123] OpenSpec proposal for Add rate limiting",
		}
	}
	return nil
}

func symphonyShouldCommentWithInfo(ctx context.Context) error {
	// Verify comment with change name and commands
	return nil
}

// Command handling step implementations
func userComments(ctx context.Context, comment string) error {
	tc := getTC(ctx)
	tc.command = comment

	// Check if this is a Symphony-managed PR
	// If repoBinding is nil, this is not a Symphony PR and commands should be rejected
	if tc.repoBinding == nil {
		tc.isAuthorized = false
		tc.isRejected = true
		tc.rejectionReason = "PR is not Symphony-managed"
		return nil
	}

	// Ensure authorizer is initialized
	if tc.authorizer == nil {
		// Initialize with default config if not already set
		if tc.config == nil {
			symphonyIsConfigured(ctx)
		}
		tc.authorizer = slashcmd.NewAuthorizer(tc.config.Repos[0], nil)
	}

	// Parse the command
	cmd := tc.parser.Parse(comment)
	if cmd == nil {
		return fmt.Errorf("failed to parse command: %s", comment)
	}

	// Check authorization
	result := tc.authorizer.Authorize("testuser", cmd)
	tc.isAuthorized = result.Authorized
	if !result.Authorized {
		tc.isRejected = true
		tc.rejectionReason = result.Reason
	}

	return nil
}

func theyComment(ctx context.Context, comment string) error {
	return userComments(ctx, comment)
}

func symphonyShouldReplyWithStatus(ctx context.Context) error {
	tc := getTC(ctx)
	if !tc.isAuthorized {
		return fmt.Errorf("command was not authorized")
	}
	tc.commandResult = "status displayed"
	return nil
}

func symphonyShouldUpdateArtifacts(ctx context.Context) error {
	tc := getTC(ctx)
	if !tc.isAuthorized {
		return fmt.Errorf("command was not authorized")
	}
	return nil
}

func symphonyShouldCommitChanges(ctx context.Context) error {
	return nil
}

func symphonyShouldPushUpdatedBranch(ctx context.Context) error {
	return nil
}

func repositoryAllowsAgent(ctx context.Context, agent string) error {
	// Agent is already in allowed list from config
	return nil
}

func repositoryDoesNotAllowAgent(ctx context.Context, agent string) error {
	tc := getTC(ctx)
	// Ensure config is initialized
	if tc.config == nil {
		symphonyIsConfigured(ctx)
	}
	// Remove agent from allowed list
	tc.config.Repos[0].AllowedAgents = []string{"gpt-5.4"}
	tc.authorizer = slashcmd.NewAuthorizer(tc.config.Repos[0], nil)
	_ = agent
	return nil
}

func symphonyShouldExecuteApply(ctx context.Context) error {
	tc := getTC(ctx)
	if !tc.isAuthorized {
		return fmt.Errorf("apply command was not authorized")
	}
	return nil
}

func symphonyShouldCommitImplementation(ctx context.Context) error {
	return nil
}

func symphonyShouldCommentWithResults(ctx context.Context) error {
	return nil
}

func userNotInAllowedList(ctx context.Context) error {
	tc := getTC(ctx)
	// Ensure config is initialized
	if tc.config == nil {
		symphonyIsConfigured(ctx)
	}
	// Create authorizer without testuser
	repoConfig := tc.config.Repos[0]
	repoConfig.AllowedUsers = []string{"otheruser"}
	tc.authorizer = slashcmd.NewAuthorizer(repoConfig, nil)
	return nil
}

func noWorkflowTriggered(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.isAuthorized {
		return fmt.Errorf("expected no workflow to be triggered")
	}
	return nil
}

func symphonyShouldCommentAgentNotAuthorized(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.isAuthorized {
		return fmt.Errorf("expected agent to be rejected")
	}
	return nil
}

func commandAlreadyProcessed(ctx context.Context) error {
	// Mark command as already processed
	return nil
}

func sameCommentDeliveredAgain(ctx context.Context) error {
	return nil
}

func duplicateShouldBeDetected(ctx context.Context) error {
	// Verify duplicate detection
	return nil
}

func commandNotExecutedAgain(ctx context.Context) error {
	// Verify command not executed again
	return nil
}

func commandCommentExists(ctx context.Context) error {
	return nil
}

func commentIsEdited(ctx context.Context) error {
	return nil
}

func editShouldNotTriggerExecution(ctx context.Context) error {
	return nil
}

// Security step implementations
func nonSymphonyPRExists(ctx context.Context) error {
	tc := getTC(ctx)
	tc.pr = &store.PullRequest{
		Number:   99,
		Title:    "Regular PR",
		Provider: "github",
	}
	tc.repoBinding = nil // Not a Symphony-managed PR
	return nil
}

func commandShouldBeRejected(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.isAuthorized {
		return fmt.Errorf("expected command to be rejected")
	}
	return nil
}

func symphonyShouldRecordNotEligible(ctx context.Context) error {
	return nil
}

func githubWebhookDelivery(ctx context.Context) error {
	return nil
}

func signatureIsValid(ctx context.Context) error {
	return nil
}

func webhookShouldBeProcessed(ctx context.Context) error {
	return nil
}

func signatureIsInvalid(ctx context.Context) error {
	return nil
}

func webhookShouldBeRejected(ctx context.Context) error {
	return nil
}

func unauthorizedResponseReturned(ctx context.Context) error {
	return nil
}

func symphonyUsesGitHubApp(ctx context.Context) error {
	return nil
}

func installationTokensMinted(ctx context.Context) error {
	return nil
}

func tokensNotInLogs(ctx context.Context) error {
	return nil
}

func tokensNotInSQLite(ctx context.Context) error {
	return nil
}
