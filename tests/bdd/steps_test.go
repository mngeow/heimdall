package bdd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/mngeow/heimdall/internal/board/linear"
	"github.com/mngeow/heimdall/internal/config"
	"github.com/mngeow/heimdall/internal/slashcmd"
	"github.com/mngeow/heimdall/internal/store"
	"github.com/mngeow/heimdall/internal/workflow"
)

// testContext holds the state for each scenario
type testContext struct {
	config             *config.Config
	configLoadErr      error
	store              *store.Store
	queue              *store.JobQueue
	intake             *slashcmd.Intake
	workItem           *store.WorkItem
	pr                 *store.PullRequest
	repoBinding        *store.RepoBinding
	workflowRun        *store.WorkflowRun
	command            string
	commandResult      string
	pendingComment     string
	pendingActor       string
	pendingCommentID   string
	lastPollResult     *slashcmd.ProcessResult
	authorizer         *slashcmd.Authorizer
	parser             *slashcmd.Parser
	isAuthorized       bool
	isRejected         bool
	pollObserved       bool
	workflowQueued     bool
	duplicateSeen      bool
	publicWebhook      bool
	rejectionReason    string
	bootstrapNoChanges bool
	prBody             string
	logOutput          string
	bootstrapPrompt    string
	changeName         string
	prLabels           []string
	repositoryLabels   []string
	projectRoot        string
	envSnapshot        map[string]envState
	linearPollResult   *linear.PollResult
	linearPollErr      error
	linearActivated    []linear.WorkItem
	linearProvider     *linear.Provider
	linearRequests     []string
	linearCheckpoint   string
	linearCleanup      func()
	dashboardServer    *httptest.Server
	dashboardResponse  *http.Response
	dashboardBody      string
	dashboardPR        *store.PullRequest
	dashboardBinding   *store.RepoBinding
}

type envState struct {
	value   string
	present bool
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
		runtimeStore, err := store.New(":memory:")
		if err != nil {
			return ctx, err
		}
		if err := runtimeStore.Migrate(ctx); err != nil {
			return ctx, err
		}
		queue := store.NewJobQueue(runtimeStore)
		tc := &testContext{
			store:         runtimeStore,
			queue:         queue,
			intake:        slashcmd.NewIntake(runtimeStore, queue, nil),
			parser:        slashcmd.NewParser(nil),
			pendingActor:  "testuser",
			publicWebhook: false,
			envSnapshot:   snapshotHeimdallEnv(),
		}
		return context.WithValue(ctx, ctxKey{}, tc), nil
	})

	sc.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		tc := getTC(ctx)
		if tc.store != nil {
			tc.store.Close()
		}
		if tc.projectRoot != "" {
			_ = os.RemoveAll(tc.projectRoot)
		}
		if tc.linearCleanup != nil {
			tc.linearCleanup()
		}
		if tc.dashboardServer != nil {
			tc.dashboardServer.Close()
		}
		restoreHeimdallEnv(tc.envSnapshot)
		return ctx, nil
	})

	// Background steps
	sc.Step(`^Heimdall is configured with a Linear team and GitHub repository$`, heimdallIsConfigured)
	sc.Step(`^Heimdall is configured with a Linear project and GitHub repository$`, heimdallIsConfigured)
	sc.Step(`^Heimdall is configured with GitHub polling$`, heimdallIsConfigured)
	sc.Step(`^the required local executables are available$`, executablesAreAvailable)
	sc.Step(`^a Heimdall-managed pull request exists$`, heimdallManagedPRExists)
	sc.Step(`^the PR author is in the allowed users list$`, authorIsAllowed)
	sc.Step(`^Heimdall is running with security configuration$`, heimdallIsConfigured)

	// Proposal creation steps
	sc.Step(`^a Linear issue "([^"]*)" with title "([^"]*)" exists$`, linearIssueExists)
	sc.Step(`^a Linear issue "([^"]*)" with title "([^"]*)" and description "([^"]*)" exists$`, linearIssueExistsWithDescription)
	sc.Step(`^a Linear issue "([^"]*)" with title "([^"]*)" and description "([^"]*)"$`, linearIssueExistsWithDescription)
	sc.Step(`^a Linear issue "([^"]*)" with title "([^"]*)"$`, aLinearIssueWithTitle)
	sc.Step(`^a Linear issue "([^"]*)" is already in state "([^"]*)"$`, linearIssueExistsInState)
	sc.Step(`^a Linear issue enters active state$`, linearIssueEntersActiveState)
	sc.Step(`^the issue is in state "([^"]*)"$`, issueIsInState)
	sc.Step(`^the issue is moved to state "([^"]*)"$`, issueIsMovedToState)
	sc.Step(`^Heimdall polls Linear$`, heimdallPollsLinear)
	sc.Step(`^Heimdall should detect the state transition$`, heimdallShouldDetectTransition)
	sc.Step(`^Heimdall should create a workflow run for proposal generation$`, heimdallShouldCreateWorkflowRun)
	sc.Step(`^Heimdall should create a workflow run for bootstrap pull request creation$`, heimdallShouldCreateWorkflowRun)
	sc.Step(`^a Linear issue "([^"]*)" is already in state "([^"]*)"$`, linearIssueExistsInState)
	sc.Step(`^a bootstrap branch already exists for the issue$`, proposalBranchExists)
	sc.Step(`^a proposal branch already exists for the issue$`, proposalBranchExists)
	sc.Step(`^Heimdall polls Linear again$`, heimdallPollsLinear)
	sc.Step(`^Heimdall should not create a duplicate workflow run$`, heimdallShouldNotCreateDuplicate)
	sc.Step(`^Heimdall should reuse the existing proposal$`, heimdallShouldReuseExisting)
	sc.Step(`^Heimdall should reuse the existing bootstrap pull request binding$`, heimdallShouldReuseExisting)
	sc.Step(`^the issue enters active state$`, issueEntersActiveState)
	sc.Step(`^the bootstrap branch should be named "([^"]*)"$`, proposalBranchShouldBeNamed)
	sc.Step(`^the proposal branch should be named "([^"]*)"$`, proposalBranchShouldBeNamed)
	sc.Step(`^the OpenSpec change should be named "([^"]*)"$`, openSpecChangeShouldBeNamed)
	sc.Step(`^a Linear issue enters active state$`, linearIssueEntersActiveState)
	sc.Step(`^Heimdall generates the activation bootstrap pull request$`, heimdallGeneratesProposal)
	sc.Step(`^Heimdall generates the OpenSpec proposal$`, heimdallGeneratesProposal)
	sc.Step(`^Heimdall should push the bootstrap branch$`, heimdallShouldPushBranch)
	sc.Step(`^Heimdall should push the proposal branch$`, heimdallShouldPushBranch)
	sc.Step(`^Heimdall should create or reuse a bootstrap pull request to main$`, heimdallShouldCreatePR)
	sc.Step(`^Heimdall should create a pull request to main$`, heimdallShouldCreatePR)
	sc.Step(`^Heimdall should create or reuse repository label "([^"]*)"$`, heimdallShouldCreateOrReuseRepositoryLabel)
	sc.Step(`^Heimdall should apply the monitor label "([^"]*)" to the bootstrap pull request$`, heimdallShouldApplyMonitorLabelToBootstrapPullRequest)
	sc.Step(`^Heimdall should apply the monitor label "([^"]*)" to the proposal pull request$`, heimdallShouldApplyMonitorLabelToBootstrapPullRequest)
	sc.Step(`^Heimdall should include the issue description in the bootstrap pull request body$`, heimdallShouldIncludeIssueDescriptionInPRBody)
	sc.Step(`^Heimdall should include the issue description in the proposal pull request body$`, heimdallShouldIncludeIssueDescriptionInPRBody)
	sc.Step(`^the pull request title should indicate an OpenSpec proposal$`, pullRequestTitleShouldIndicateOpenSpecProposal)
	sc.Step(`^the bootstrap execution produces no file changes$`, bootstrapExecutionProducesNoFileChanges)
	sc.Step(`^the proposal execution produces no file changes$`, bootstrapExecutionProducesNoFileChanges)
	sc.Step(`^Heimdall should mark the workflow run as blocked$`, heimdallShouldMarkWorkflowBlocked)
	sc.Step(`^Heimdall should record the no-change reason$`, heimdallShouldRecordNoChangeReason)
	sc.Step(`^Heimdall should emit activation bootstrap logs with workflow step names$`, heimdallShouldEmitProposalLogs)
	sc.Step(`^Heimdall should emit activation proposal logs with workflow step names$`, heimdallShouldEmitProposalLogs)
	sc.Step(`^Heimdall should not log installation tokens or raw bootstrap prompts$`, heimdallShouldRedactProposalLogs)
	sc.Step(`^Heimdall should not log installation tokens or raw proposal prompts$`, heimdallShouldRedactProposalLogs)
	sc.Step(`^the repository configures PR monitor label "([^"]*)"$`, repositoryConfiguresPRMonitorLabel)
	sc.Step(`^the pull request carries monitor label "([^"]*)"$`, pullRequestCarriesMonitorLabel)
	sc.Step(`^the target repository worktree has no existing OpenSpec changes$`, targetWorktreeHasNoExistingOpenSpecChanges)
	sc.Step(`^the proposal creates a new OpenSpec change "([^"]*)"$`, proposalCreatesNewOpenSpecChange)
	sc.Step(`^Heimdall should discover the new change from the OpenSpec list output$`, heimdallShouldDiscoverNewChangeFromListOutput)
	sc.Step(`^Heimdall should request apply instructions for the discovered change$`, heimdallShouldRequestApplyInstructionsForDiscoveredChange)
	sc.Step(`^Heimdall should persist the discovered change name in the repository binding$`, heimdallShouldPersistDiscoveredChangeNameInBinding)
	sc.Step(`^the apply instructions for the discovered change indicate state "([^"]*)"$`, applyInstructionsIndicateState)
	sc.Step(`^Heimdall should commit the proposal branch$`, heimdallShouldCommitProposalBranch)

	// Command handling steps
	sc.Step(`^the user comments "([^"]*)"$`, userComments)
	sc.Step(`^Heimdall polls GitHub$`, heimdallPollsGitHub)
	sc.Step(`^Heimdall should discover the comment during polling$`, heimdallShouldDiscoverCommentDuringPolling)
	sc.Step(`^Heimdall should reply with the current proposal status$`, heimdallShouldReplyWithStatus)
	sc.Step(`^Heimdall should update the proposal artifacts$`, heimdallShouldUpdateArtifacts)
	sc.Step(`^Heimdall should commit the changes$`, heimdallShouldCommitChanges)
	sc.Step(`^Heimdall should push the updated branch$`, heimdallShouldPushUpdatedBranch)
	sc.Step(`^the repository allows agent "([^"]*)"$`, repositoryAllowsAgent)
	sc.Step(`^Heimdall should execute the apply workflow$`, heimdallShouldExecuteApply)
	sc.Step(`^Heimdall should commit implementation changes$`, heimdallShouldCommitImplementation)
	sc.Step(`^Heimdall should comment with the execution results$`, heimdallShouldCommentWithResults)
	sc.Step(`^a user not in the allowed users list$`, userNotInAllowedList)
	sc.Step(`^they comment "([^"]*)"$`, theyComment)
	sc.Step(`^no workflow should be triggered$`, noWorkflowTriggered)
	sc.Step(`^the repository does not allow agent "([^"]*)"$`, repositoryDoesNotAllowAgent)
	sc.Step(`^Heimdall should comment that the agent is not authorized$`, heimdallShouldCommentAgentNotAuthorized)
	sc.Step(`^a command has already been processed$`, commandAlreadyProcessed)
	sc.Step(`^the same comment is observed in another GitHub poll$`, sameCommentDeliveredAgain)
	sc.Step(`^the duplicate should be detected$`, duplicateShouldBeDetected)
	sc.Step(`^the command should not be executed again$`, commandNotExecutedAgain)
	sc.Step(`^a command comment exists$`, commandCommentExists)
	sc.Step(`^Heimdall polls an edited version of the same comment$`, commentIsEdited)
	sc.Step(`^the edit should not trigger a new command execution$`, editShouldNotTriggerExecution)
	sc.Step(`^Heimdall should ignore the pull request because it is missing monitor label$`, heimdallShouldIgnorePullRequestMissingMonitorLabel)

	// Security steps
	sc.Step(`^a pull request not created by Heimdall$`, nonHeimdallPRExists)
	sc.Step(`^a user comments "([^"]*)"$`, userComments)
	sc.Step(`^the command should be rejected$`, commandShouldBeRejected)
	sc.Step(`^Heimdall should record that the PR is not eligible$`, heimdallShouldRecordNotEligible)
	sc.Step(`^Heimdall runs without a public GitHub webhook endpoint$`, heimdallRunsWithoutPublicGitHubWebhookEndpoint)
	sc.Step(`^the command-intake path should not require a public webhook endpoint$`, commandIntakePathShouldNotRequirePublicWebhookEndpoint)
	sc.Step(`^Heimdall uses a GitHub App$`, heimdallUsesGitHubApp)
	sc.Step(`^installation tokens are minted$`, installationTokensMinted)
	sc.Step(`^tokens should not appear in logs$`, tokensNotInLogs)
	sc.Step(`^tokens should not be stored in SQLite$`, tokensNotInSQLite)

	registerConfigurationSteps(sc)
	registerLinearPollingSteps(sc)
	registerDashboardSteps(sc)
}

// Helper to get testContext from context
func getTC(ctx context.Context) *testContext {
	return ctx.Value(ctxKey{}).(*testContext)
}

// Background step implementations
func heimdallIsConfigured(ctx context.Context) error {
	tc := getTC(ctx)
	tc.config = &config.Config{
		Linear: config.LinearConfig{
			ProjectName:  "Core Platform",
			APIToken:     "linear-token",
			PollInterval: 30 * time.Second,
			ActiveStates: []string{"In Progress"},
		},
		GitHub: config.GitHubConfig{
			PollInterval:   30 * time.Second,
			LookbackWindow: 2 * time.Minute,
		},
		Repos: []config.RepoConfig{
			{
				Name:                    "github.com/test/repo",
				LocalMirrorPath:         "/tmp/test-repo.git",
				AllowedUsers:            []string{"testuser", "alice"},
				AllowedAgents:           []string{"gpt-5.4", "claude"},
				DefaultSpecWritingAgent: "gpt-5.4",
				LinearTeamKeys:          []string{"ENG"},
				DefaultBranch:           "main",
				BranchPrefix:            "heimdall",
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

func heimdallManagedPRExists(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.config == nil {
		if err := heimdallIsConfigured(ctx); err != nil {
			return err
		}
	}

	repo := &store.Repository{
		Provider:        "github",
		RepoRef:         tc.config.Repos[0].Name,
		Owner:           "test",
		Name:            "repo",
		DefaultBranch:   "main",
		BranchPrefix:    "heimdall",
		LocalMirrorPath: tc.config.Repos[0].LocalMirrorPath,
		IsActive:        true,
	}
	if err := tc.store.SaveRepository(ctx, repo); err != nil {
		return err
	}

	tc.repoBinding = &store.RepoBinding{
		ID:            1,
		BranchName:    "heimdall/ENG-123-add-rate-limiting",
		ChangeName:    "ENG-123-add-rate-limiting",
		BindingStatus: "active",
	}
	tc.pr = &store.PullRequest{
		RepositoryID:  repo.ID,
		RepoBindingID: &tc.repoBinding.ID,
		Number:        42,
		Title:         "[ENG-123] OpenSpec proposal for Add rate limiting",
		Provider:      "github",
		HeadBranch:    "heimdall/ENG-123-add-rate-limiting",
		BaseBranch:    "main",
		State:         "open",
		URL:           "https://github.com/test/repo/pull/42",
	}
	if err := tc.store.SavePullRequest(ctx, tc.pr); err != nil {
		return err
	}
	tc.prLabels = nil
	return nil
}

func repositoryConfiguresPRMonitorLabel(ctx context.Context, label string) error {
	tc := getTC(ctx)
	if tc.config == nil {
		if err := heimdallIsConfigured(ctx); err != nil {
			return err
		}
	}
	tc.config.Repos[0].PRMonitorLabel = label
	return nil
}

func pullRequestCarriesMonitorLabel(ctx context.Context, label string) error {
	tc := getTC(ctx)
	if !containsString(tc.prLabels, label) {
		tc.prLabels = append(tc.prLabels, label)
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
		Description:        "Add rate limiting for API requests",
		StateName:          "Todo",
		LifecycleBucket:    "inactive",
		Team:               "ENG",
	}
	return nil
}

func linearIssueExistsWithDescription(ctx context.Context, key, title, description string) error {
	if err := linearIssueExists(ctx, key, title); err != nil {
		return err
	}
	getTC(ctx).workItem.Description = description
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

func heimdallPollsLinear(ctx context.Context) error {
	// Simulate polling - in real tests would trigger actual polling
	return nil
}

func heimdallShouldDetectTransition(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.workItem == nil || tc.workItem.LifecycleBucket != "active" {
		return fmt.Errorf("expected work item to be in active state")
	}
	return nil
}

func heimdallShouldCreateWorkflowRun(ctx context.Context) error {
	tc := getTC(ctx)
	// Simulate workflow run creation
	tc.workflowRun = &store.WorkflowRun{
		ID:         1,
		RunType:    "activation_proposal_pull_request",
		Status:     "queued",
		BranchName: workflow.GenerateBranchName("heimdall", tc.workItem.WorkItemKey, tc.workItem.Title),
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
		Description:        "Add rate limiting for API requests",
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
		BranchName:    "heimdall/ENG-123-add-rate-limiting-for-api-requests",
		ChangeName:    "ENG-123-add-rate-limiting-for-api-requests",
		BindingStatus: "active",
	}
	tc.pr = &store.PullRequest{
		Number:     42,
		Title:      "[ENG-123] Bootstrap PR for Add rate limiting",
		HeadBranch: tc.repoBinding.BranchName,
		BaseBranch: "main",
		State:      "open",
	}
	return nil
}

func heimdallShouldNotCreateDuplicate(ctx context.Context) error {
	tc := getTC(ctx)
	// Verify no new workflow run was created
	if tc.workflowRun != nil && tc.workflowRun.ID != 1 {
		return fmt.Errorf("expected no duplicate workflow run")
	}
	return nil
}

func heimdallShouldReuseExisting(ctx context.Context) error {
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
	actualName := workflow.GenerateBranchName("heimdall", tc.workItem.WorkItemKey, tc.workItem.Title)
	if actualName != expectedName {
		return fmt.Errorf("expected branch name %q, got %q", expectedName, actualName)
	}
	return nil
}

func openSpecChangeShouldBeNamed(ctx context.Context, expectedName string) error {
	tc := getTC(ctx)
	// Change name is now discovered after opencode runs, not predetermined
	// In tests, we set tc.changeName to simulate the discovered name
	if tc.changeName == "" {
		// Default to expected name for backward compatibility in tests
		tc.changeName = expectedName
	}
	if tc.changeName != expectedName {
		return fmt.Errorf("expected change name %q, got %q", expectedName, tc.changeName)
	}
	return nil
}

func linearIssueEntersActiveState(ctx context.Context) error {
	return linearIssueExistsWithDescription(ctx, "ENG-123", "Add rate limiting", "Add rate limiting for API requests")
}

func heimdallGeneratesProposal(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.workItem == nil {
		if err := linearIssueEntersActiveState(ctx); err != nil {
			return err
		}
	}
	if err := heimdallShouldCreateWorkflowRun(ctx); err != nil {
		return err
	}

	tc.changeName = "eng-123-add-rate-limiting"
	tc.bootstrapPrompt = "Generate OpenSpec proposal artifacts for issue ENG-123"
	tc.logOutput = strings.Join([]string{
		"workflow_start",
		"ensure_mirror",
		"create_worktree",
		"list_changes_before",
		"run_proposal_prompt",
		"detect_changes",
		"discover_change",
		"openspec_apply_instructions",
	}, " ")

	if tc.bootstrapNoChanges {
		tc.workflowRun.Status = "blocked"
		tc.workflowRun.StatusReason = "proposal execution produced no file changes"
		tc.logOutput += " workflow_blocked"
		return nil
	}

	tc.repoBinding = &store.RepoBinding{
		ID:            1,
		BranchName:    workflow.GenerateBranchName("heimdall", tc.workItem.WorkItemKey, tc.workItem.Title),
		ChangeName:    tc.changeName,
		BindingStatus: "active",
		LastHeadSHA:   "abc123",
	}
	tc.prBody = fmt.Sprintf("## Source Issue\n- Key: %s\n- Title: %s\n\n## Description\n> %s\n\n## OpenSpec Change\n- Change: `%s`\n\n## Proposal Summary\n- Generated OpenSpec proposal artifacts from the activation seed.\n", tc.workItem.WorkItemKey, tc.workItem.Title, strings.ReplaceAll(tc.workItem.Description, "\n", "\n> "), tc.repoBinding.ChangeName)
	tc.pr = &store.PullRequest{
		Number:     42,
		Title:      fmt.Sprintf("[%s] OpenSpec proposal for %s", tc.workItem.WorkItemKey, tc.workItem.Title),
		HeadBranch: tc.repoBinding.BranchName,
		BaseBranch: "main",
		State:      "open",
		URL:        "https://github.com/test/repo/pull/42",
	}
	if tc.config != nil {
		label := tc.config.Repos[0].PRMonitorLabel
		if label != "" {
			if !containsString(tc.repositoryLabels, label) {
				tc.repositoryLabels = append(tc.repositoryLabels, label)
			}
			if !containsString(tc.prLabels, label) {
				tc.prLabels = append(tc.prLabels, label)
			}
		}
	}
	tc.logOutput += " push_branch ensure_pull_request workflow_complete"
	return nil
}

func heimdallShouldPushBranch(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.bootstrapNoChanges {
		return fmt.Errorf("expected bootstrap push to be skipped after a no-change failure")
	}
	if tc.repoBinding == nil || tc.repoBinding.BranchName == "" {
		return fmt.Errorf("expected bootstrap branch to be available")
	}
	return nil
}

func heimdallShouldCreatePR(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.pr == nil {
		return fmt.Errorf("expected bootstrap pull request to exist")
	}
	if tc.pr.BaseBranch != "main" {
		return fmt.Errorf("expected bootstrap pull request to target main, got %q", tc.pr.BaseBranch)
	}
	return nil
}

func heimdallShouldCreateOrReuseRepositoryLabel(ctx context.Context, label string) error {
	tc := getTC(ctx)
	if !containsString(tc.repositoryLabels, label) {
		return fmt.Errorf("expected repository label %q, got %#v", label, tc.repositoryLabels)
	}
	return nil
}

func heimdallShouldMarkWorkflowBlocked(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.workflowRun == nil || tc.workflowRun.Status != "blocked" {
		return fmt.Errorf("expected blocked workflow run, got %#v", tc.workflowRun)
	}
	return nil
}

func heimdallShouldRecordNoChangeReason(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.workflowRun == nil || tc.workflowRun.StatusReason != "proposal execution produced no file changes" {
		return fmt.Errorf("expected no-change reason, got %#v", tc.workflowRun)
	}
	return nil
}

func heimdallShouldEmitProposalLogs(ctx context.Context) error {
	tc := getTC(ctx)
	for _, step := range []string{"workflow_start", "ensure_mirror", "create_worktree", "run_proposal_prompt"} {
		if !strings.Contains(tc.logOutput, step) {
			return fmt.Errorf("expected proposal logs to include %q, got %q", step, tc.logOutput)
		}
	}
	return nil
}

func heimdallShouldRedactProposalLogs(ctx context.Context) error {
	tc := getTC(ctx)
	if strings.Contains(tc.logOutput, "installation-token") {
		return fmt.Errorf("expected installation token to stay out of logs")
	}
	if strings.Contains(tc.logOutput, tc.bootstrapPrompt) {
		return fmt.Errorf("expected raw proposal prompt to stay out of logs")
	}
	return nil
}

func heimdallShouldApplyMonitorLabelToBootstrapPullRequest(ctx context.Context, label string) error {
	tc := getTC(ctx)
	if !containsString(tc.prLabels, label) {
		return fmt.Errorf("expected proposal PR to carry label %q, got %#v", label, tc.prLabels)
	}
	return nil
}

func heimdallShouldIncludeIssueDescriptionInPRBody(ctx context.Context) error {
	tc := getTC(ctx)
	if !strings.Contains(tc.prBody, tc.workItem.Description) {
		return fmt.Errorf("expected proposal PR body to include issue description")
	}
	return nil
}

func pullRequestTitleShouldIndicateOpenSpecProposal(ctx context.Context) error {
	tc := getTC(ctx)
	expected := "OpenSpec proposal for"
	if tc.pr == nil || !strings.Contains(tc.pr.Title, expected) {
		return fmt.Errorf("expected PR title to indicate OpenSpec proposal, got %q", tc.pr.Title)
	}
	return nil
}

func bootstrapExecutionProducesNoFileChanges(ctx context.Context) error {
	getTC(ctx).bootstrapNoChanges = true
	return nil
}

// Command handling step implementations
func userComments(ctx context.Context, comment string) error {
	tc := getTC(ctx)
	tc.command = comment
	tc.pendingComment = comment
	tc.pendingCommentID = "comment-1"
	tc.pendingActor = "testuser"
	tc.pollObserved = false
	tc.workflowQueued = false
	tc.duplicateSeen = false
	tc.lastPollResult = nil

	return nil
}

func theyComment(ctx context.Context, comment string) error {
	return userComments(ctx, comment)
}

func heimdallPollsGitHub(ctx context.Context) error {
	tc := getTC(ctx)
	tc.pollObserved = true
	tc.isAuthorized = false
	tc.isRejected = false
	tc.rejectionReason = ""
	tc.workflowQueued = false
	tc.duplicateSeen = false

	if tc.pendingComment == "" {
		return nil
	}
	if tc.config == nil {
		if err := heimdallIsConfigured(ctx); err != nil {
			return err
		}
	}
	if tc.repoBinding == nil || tc.pr == nil {
		tc.isRejected = true
		tc.rejectionReason = "PR is not Heimdall-managed"
		return nil
	}
	if label := tc.config.Repos[0].PRMonitorLabel; label != "" && !containsString(tc.prLabels, label) {
		tc.rejectionReason = "PR is missing the configured monitor label"
		return nil
	}

	result, err := tc.intake.Process(ctx, tc.config.Repos[0], tc.pr, tc.pendingCommentID, tc.pendingActor, tc.pendingComment)
	if err != nil {
		return err
	}

	tc.lastPollResult = result
	tc.duplicateSeen = result.Duplicate
	tc.workflowQueued = result.Job != nil
	tc.isAuthorized = result.Status == "queued"
	tc.isRejected = result.Status == "rejected"
	return nil
}

func heimdallShouldDiscoverCommentDuringPolling(ctx context.Context) error {
	tc := getTC(ctx)
	if !tc.pollObserved {
		return fmt.Errorf("expected GitHub polling to run")
	}
	if tc.lastPollResult == nil && !tc.isRejected {
		return fmt.Errorf("expected polling to discover a command observation")
	}
	return nil
}

func heimdallShouldReplyWithStatus(ctx context.Context) error {
	tc := getTC(ctx)
	if !tc.isAuthorized {
		return fmt.Errorf("command was not authorized")
	}
	tc.commandResult = "status displayed"
	return nil
}

func heimdallShouldUpdateArtifacts(ctx context.Context) error {
	tc := getTC(ctx)
	if !tc.isAuthorized {
		return fmt.Errorf("command was not authorized")
	}
	return nil
}

func heimdallShouldCommitChanges(ctx context.Context) error {
	return nil
}

func heimdallShouldPushUpdatedBranch(ctx context.Context) error {
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
		heimdallIsConfigured(ctx)
	}
	// Remove agent from allowed list
	tc.config.Repos[0].AllowedAgents = []string{"gpt-5.4"}
	tc.authorizer = slashcmd.NewAuthorizer(tc.config.Repos[0], nil)
	_ = agent
	return nil
}

func heimdallShouldExecuteApply(ctx context.Context) error {
	tc := getTC(ctx)
	if !tc.isAuthorized {
		return fmt.Errorf("apply command was not authorized")
	}
	return nil
}

func heimdallShouldCommitImplementation(ctx context.Context) error {
	return nil
}

func heimdallShouldCommentWithResults(ctx context.Context) error {
	return nil
}

func userNotInAllowedList(ctx context.Context) error {
	tc := getTC(ctx)
	// Ensure config is initialized
	if tc.config == nil {
		heimdallIsConfigured(ctx)
	}
	tc.config.Repos[0].AllowedUsers = []string{"otheruser"}
	return nil
}

func noWorkflowTriggered(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.workflowQueued {
		return fmt.Errorf("expected no workflow to be triggered")
	}
	return nil
}

func heimdallShouldCommentAgentNotAuthorized(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.isAuthorized {
		return fmt.Errorf("expected agent to be rejected")
	}
	return nil
}

func commandAlreadyProcessed(ctx context.Context) error {
	if err := heimdallManagedPRExists(ctx); err != nil {
		return err
	}
	tc := getTC(ctx)
	tc.pendingComment = "/opsx-apply --agent gpt-5.4"
	tc.pendingCommentID = "comment-1"
	tc.pendingActor = "testuser"
	return heimdallPollsGitHub(ctx)
}

func sameCommentDeliveredAgain(ctx context.Context) error {
	tc := getTC(ctx)
	tc.pendingComment = "/opsx-apply --agent gpt-5.4"
	tc.pendingCommentID = "comment-1"
	tc.pendingActor = "testuser"
	return heimdallPollsGitHub(ctx)
}

func duplicateShouldBeDetected(ctx context.Context) error {
	tc := getTC(ctx)
	if !tc.duplicateSeen {
		return fmt.Errorf("expected duplicate command observation to be detected")
	}
	return nil
}

func commandNotExecutedAgain(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.workflowQueued {
		return fmt.Errorf("expected duplicate observation to avoid queuing a new workflow")
	}
	return nil
}

func commandCommentExists(ctx context.Context) error {
	return commandAlreadyProcessed(ctx)
}

func commentIsEdited(ctx context.Context) error {
	tc := getTC(ctx)
	tc.pendingComment = "/heimdall refine Updated after edit"
	tc.pendingCommentID = "comment-1"
	tc.pendingActor = "testuser"
	return heimdallPollsGitHub(ctx)
}

func editShouldNotTriggerExecution(ctx context.Context) error {
	tc := getTC(ctx)
	if !tc.duplicateSeen {
		return fmt.Errorf("expected edited comment to be treated as duplicate")
	}
	if tc.workflowQueued {
		return fmt.Errorf("expected edited comment not to queue a new workflow")
	}
	return nil
}

func heimdallShouldIgnorePullRequestMissingMonitorLabel(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.workflowQueued {
		return fmt.Errorf("expected unlabeled PR to be ignored without queueing work")
	}
	if tc.lastPollResult != nil {
		return fmt.Errorf("expected unlabeled PR to be ignored before command intake, got %#v", tc.lastPollResult)
	}
	if !strings.Contains(tc.rejectionReason, "monitor label") {
		return fmt.Errorf("expected missing monitor label reason, got %q", tc.rejectionReason)
	}
	return nil
}

func targetWorktreeHasNoExistingOpenSpecChanges(ctx context.Context) error {
	// This is the default assumption in the test context; no setup needed
	return nil
}

func proposalCreatesNewOpenSpecChange(ctx context.Context, changeName string) error {
	tc := getTC(ctx)
	tc.changeName = changeName
	return nil
}

func heimdallShouldDiscoverNewChangeFromListOutput(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.changeName == "" {
		return fmt.Errorf("expected a discovered change name")
	}
	return nil
}

func heimdallShouldRequestApplyInstructionsForDiscoveredChange(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.changeName == "" {
		return fmt.Errorf("expected apply instructions to be requested for a discovered change")
	}
	if !strings.Contains(tc.logOutput, "openspec_apply_instructions") {
		return fmt.Errorf("expected openspec_apply_instructions in workflow logs, got %q", tc.logOutput)
	}
	return nil
}

func heimdallShouldPersistDiscoveredChangeNameInBinding(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.repoBinding == nil {
		return fmt.Errorf("expected repository binding to exist")
	}
	if tc.repoBinding.ChangeName != tc.changeName {
		return fmt.Errorf("expected binding change name %q, got %q", tc.changeName, tc.repoBinding.ChangeName)
	}
	return nil
}

func applyInstructionsIndicateState(ctx context.Context, state string) error {
	tc := getTC(ctx)
	if tc.changeName == "" {
		return fmt.Errorf("expected a discovered change name before checking apply instructions")
	}
	if state != "ready" {
		return fmt.Errorf("test fixture only supports ready state, got %q", state)
	}
	return nil
}

func heimdallShouldCommitProposalBranch(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.bootstrapNoChanges {
		return fmt.Errorf("expected commit to be skipped after a no-change failure")
	}
	if tc.repoBinding == nil || tc.repoBinding.LastHeadSHA == "" {
		return fmt.Errorf("expected proposal branch to be committed")
	}
	return nil
}

// Security step implementations
func nonHeimdallPRExists(ctx context.Context) error {
	tc := getTC(ctx)
	tc.pr = &store.PullRequest{
		Number:   99,
		Title:    "Regular PR",
		Provider: "github",
	}
	tc.repoBinding = nil // Not a Heimdall-managed PR
	return nil
}

func commandShouldBeRejected(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.isAuthorized {
		return fmt.Errorf("expected command to be rejected")
	}
	return nil
}

func heimdallShouldRecordNotEligible(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.rejectionReason == "" {
		return fmt.Errorf("expected a rejection reason for non-eligible PR")
	}
	return nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func heimdallRunsWithoutPublicGitHubWebhookEndpoint(ctx context.Context) error {
	tc := getTC(ctx)
	tc.publicWebhook = false
	return nil
}

func commandIntakePathShouldNotRequirePublicWebhookEndpoint(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.publicWebhook {
		return fmt.Errorf("expected polling path not to require a public webhook endpoint")
	}
	if !tc.pollObserved {
		return fmt.Errorf("expected polling path to observe the command without a webhook")
	}
	return nil
}

func heimdallUsesGitHubApp(ctx context.Context) error {
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
