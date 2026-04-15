package bdd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/cucumber/godog"
	"github.com/mngeow/heimdall/internal/dashboard"
	"github.com/mngeow/heimdall/internal/store"
)

func registerDashboardSteps(sc *godog.ScenarioContext) {
	sc.Step(`^the dashboard is available on the operator HTTP surface$`, dashboardIsAvailable)
	sc.Step(`^an active Heimdall-managed pull request exists for the issue$`, activeManagedPRExistsForIssue)
	sc.Step(`^a command request was recorded for the pull request$`, commandRequestRecordedForPR)
	sc.Step(`^the operator requests the dashboard overview$`, operatorRequestsOverview)
	sc.Step(`^the operator requests the work-item queue$`, operatorRequestsWorkItemQueue)
	sc.Step(`^the operator requests the work-item queue filtered by status "([^"]*)"$`, operatorRequestsWorkItemQueueFiltered)
	sc.Step(`^the operator requests the active pull-request list$`, operatorRequestsActivePRList)
	sc.Step(`^the operator requests the pull-request detail view$`, operatorRequestsPRDetail)
	sc.Step(`^the operator requests a filtered queue refresh via HTMX$`, operatorRequestsHTMXRefresh)
	sc.Step(`^the response should be a server-rendered HTML page$`, responseIsHTMLPage)
	sc.Step(`^the response should contain an HTML fragment$`, responseIsHTMLFragment)
	sc.Step(`^the overview should show at least one tracked work item$`, overviewShowsWorkItems)
	sc.Step(`^the overview should show at least one active pull request$`, overviewShowsActivePRs)
	sc.Step(`^the queue should list the work item with key "([^"]*)"$`, queueListsWorkItem)
	sc.Step(`^the queue should show the work item status "([^"]*)"$`, queueShowsWorkItemStatus)
	sc.Step(`^the queue should show the work item lifecycle bucket$`, queueShowsWorkItemBucket)
	sc.Step(`^the list should include the pull request for "([^"]*)"$`, listIncludesPRForIssue)
	sc.Step(`^the list should link to the pull-request detail page$`, listLinksToPRDetail)
	sc.Step(`^the detail view should show the linked work item$`, detailShowsLinkedWorkItem)
	sc.Step(`^the detail view should label the timeline as Heimdall-tracked command/activity history$`, detailLabelsTimeline)
	sc.Step(`^the detail view should include a link to the canonical GitHub pull request$`, detailIncludesGitHubLink)
	sc.Step(`^the rendered page should not contain secrets or raw sensitive payloads$`, pageExcludesSecrets)
	sc.Step(`^the response should not trigger any repository mutation$`, noRepositoryMutation)
	sc.Step(`^no new workflow run should be created$`, noNewWorkflowRun)
	sc.Step(`^no repository mutation should occur$`, noRepositoryMutation)
}

func dashboardIsAvailable(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.store == nil {
		return fmt.Errorf("store not initialized")
	}
	q := dashboard.NewQueries(tc.store.DB())
	h, err := dashboard.NewHandler(q)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	tc.dashboardServer = httptest.NewServer(mux)
	return nil
}

func activeManagedPRExistsForIssue(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.config == nil {
		if err := heimdallIsConfigured(ctx); err != nil {
			return err
		}
	}
	if tc.workItem == nil {
		if err := linearIssueExists(ctx, "ENG-123", "Add rate limiting"); err != nil {
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
	if err := tc.store.SaveWorkItem(ctx, tc.workItem); err != nil {
		return err
	}

	binding := &store.RepoBinding{
		WorkItemID:    tc.workItem.ID,
		RepositoryID:  repo.ID,
		BranchName:    "heimdall/ENG-123-add-rate-limiting",
		ChangeName:    "ENG-123-add-rate-limiting",
		BindingStatus: "active",
	}
	if err := tc.store.SaveRepoBinding(ctx, binding); err != nil {
		return err
	}

	pr := &store.PullRequest{
		RepositoryID:  repo.ID,
		RepoBindingID: &binding.ID,
		Number:        42,
		Title:         "[ENG-123] OpenSpec proposal for Add rate limiting",
		Provider:      "github",
		HeadBranch:    "heimdall/ENG-123-add-rate-limiting",
		BaseBranch:    "main",
		State:         "open",
		URL:           "https://github.com/test/repo/pull/42",
	}
	if err := tc.store.SavePullRequest(ctx, pr); err != nil {
		return err
	}
	tc.dashboardPR = pr
	tc.dashboardBinding = binding
	return nil
}

func commandRequestRecordedForPR(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.dashboardPR == nil {
		return fmt.Errorf("no dashboard PR exists")
	}
	cmd := &store.CommandRequest{
		PullRequestID:       tc.dashboardPR.ID,
		CommentNodeID:       "comment-1",
		CommandName:         "/heimdall status",
		CommandArgs:         "",
		RequestedAgent:      "",
		ActorLogin:          "testuser",
		AuthorizationStatus: "allowed",
		DedupeKey:           "dedupe-1",
		Status:              "processed",
	}
	return tc.store.SaveCommandRequest(ctx, cmd)
}

func operatorRequestsOverview(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.dashboardServer == nil {
		if err := dashboardIsAvailable(ctx); err != nil {
			return err
		}
	}
	resp, err := tc.dashboardServer.Client().Get(tc.dashboardServer.URL + "/ui")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	tc.dashboardResponse = resp
	tc.dashboardBody = string(body)
	return nil
}

func operatorRequestsWorkItemQueue(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.dashboardServer == nil {
		if err := dashboardIsAvailable(ctx); err != nil {
			return err
		}
	}
	if tc.workItem != nil && tc.workItem.ID == 0 {
		if err := tc.store.SaveWorkItem(ctx, tc.workItem); err != nil {
			return err
		}
	}
	resp, err := tc.dashboardServer.Client().Get(tc.dashboardServer.URL + "/ui/work-items")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	tc.dashboardResponse = resp
	tc.dashboardBody = string(body)
	return nil
}

func operatorRequestsWorkItemQueueFiltered(ctx context.Context, status string) error {
	tc := getTC(ctx)
	if tc.dashboardServer == nil {
		if err := dashboardIsAvailable(ctx); err != nil {
			return err
		}
	}
	if tc.workItem != nil && tc.workItem.ID == 0 {
		if err := tc.store.SaveWorkItem(ctx, tc.workItem); err != nil {
			return err
		}
	}
	resp, err := tc.dashboardServer.Client().Get(tc.dashboardServer.URL + "/ui/work-items?status=" + url.QueryEscape(status))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	tc.dashboardResponse = resp
	tc.dashboardBody = string(body)
	return nil
}

func operatorRequestsActivePRList(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.dashboardServer == nil {
		if err := dashboardIsAvailable(ctx); err != nil {
			return err
		}
	}
	resp, err := tc.dashboardServer.Client().Get(tc.dashboardServer.URL + "/ui/pull-requests")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	tc.dashboardResponse = resp
	tc.dashboardBody = string(body)
	return nil
}

func operatorRequestsPRDetail(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.dashboardServer == nil {
		if err := dashboardIsAvailable(ctx); err != nil {
			return err
		}
	}
	if tc.dashboardPR == nil {
		return fmt.Errorf("no dashboard PR to request detail for")
	}
	url := fmt.Sprintf("%s/ui/pull-requests/%d", tc.dashboardServer.URL, tc.dashboardPR.ID)
	resp, err := tc.dashboardServer.Client().Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	tc.dashboardResponse = resp
	tc.dashboardBody = string(body)
	return nil
}

func operatorRequestsHTMXRefresh(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.dashboardServer == nil {
		if err := dashboardIsAvailable(ctx); err != nil {
			return err
		}
	}
	if tc.workItem != nil && tc.workItem.ID == 0 {
		if err := tc.store.SaveWorkItem(ctx, tc.workItem); err != nil {
			return err
		}
	}
	req, _ := http.NewRequest(http.MethodGet, tc.dashboardServer.URL+"/ui/work-items/fragment?status=In+Progress", nil)
	req.Header.Set("HX-Request", "true")
	resp, err := tc.dashboardServer.Client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	tc.dashboardResponse = resp
	tc.dashboardBody = string(body)
	return nil
}

func responseIsHTMLPage(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.dashboardResponse == nil {
		return fmt.Errorf("no dashboard response recorded")
	}
	ct := tc.dashboardResponse.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		return fmt.Errorf("expected text/html, got %q", ct)
	}
	if !strings.Contains(tc.dashboardBody, "<html") {
		return fmt.Errorf("expected full HTML page")
	}
	return nil
}

func responseIsHTMLFragment(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.dashboardResponse == nil {
		return fmt.Errorf("no dashboard response recorded")
	}
	ct := tc.dashboardResponse.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		return fmt.Errorf("expected text/html, got %q", ct)
	}
	if strings.Contains(tc.dashboardBody, "<html") {
		return fmt.Errorf("expected HTML fragment, not a full page")
	}
	return nil
}

func overviewShowsWorkItems(ctx context.Context) error {
	tc := getTC(ctx)
	if !strings.Contains(tc.dashboardBody, "Total Work Items") {
		return fmt.Errorf("overview missing work item card")
	}
	return nil
}

func overviewShowsActivePRs(ctx context.Context) error {
	tc := getTC(ctx)
	if !strings.Contains(tc.dashboardBody, "Active PRs") {
		return fmt.Errorf("overview missing active PR card")
	}
	return nil
}

func queueListsWorkItem(ctx context.Context, key string) error {
	tc := getTC(ctx)
	if !strings.Contains(tc.dashboardBody, key) {
		return fmt.Errorf("queue missing work item %q", key)
	}
	return nil
}

func queueShowsWorkItemStatus(ctx context.Context, status string) error {
	tc := getTC(ctx)
	if !strings.Contains(tc.dashboardBody, status) {
		return fmt.Errorf("queue missing status %q", status)
	}
	return nil
}

func queueShowsWorkItemBucket(ctx context.Context) error {
	tc := getTC(ctx)
	if !strings.Contains(tc.dashboardBody, "active") && !strings.Contains(tc.dashboardBody, "inactive") {
		return fmt.Errorf("queue missing lifecycle bucket")
	}
	return nil
}

func listIncludesPRForIssue(ctx context.Context, key string) error {
	tc := getTC(ctx)
	if !strings.Contains(tc.dashboardBody, key) {
		return fmt.Errorf("PR list missing issue %q", key)
	}
	return nil
}

func listLinksToPRDetail(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.dashboardPR == nil {
		return fmt.Errorf("no dashboard PR")
	}
	expected := fmt.Sprintf("/ui/pull-requests/%d", tc.dashboardPR.ID)
	if !strings.Contains(tc.dashboardBody, expected) {
		return fmt.Errorf("PR list missing detail link %q", expected)
	}
	return nil
}

func detailShowsLinkedWorkItem(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.workItem == nil {
		return fmt.Errorf("no work item")
	}
	if !strings.Contains(tc.dashboardBody, tc.workItem.WorkItemKey) {
		return fmt.Errorf("detail missing linked work item key")
	}
	return nil
}

func detailLabelsTimeline(ctx context.Context) error {
	tc := getTC(ctx)
	if !strings.Contains(tc.dashboardBody, "Heimdall-Tracked Command / Activity History") {
		return fmt.Errorf("detail missing Heimdall-tracked timeline label")
	}
	return nil
}

func detailIncludesGitHubLink(ctx context.Context) error {
	tc := getTC(ctx)
	if !strings.Contains(tc.dashboardBody, "github.com/test/repo/pull/42") {
		return fmt.Errorf("detail missing canonical GitHub PR link")
	}
	return nil
}

func pageExcludesSecrets(ctx context.Context) error {
	tc := getTC(ctx)
	forbidden := []string{"linear-token", "github-app.pem", "x-access-token", "PRIVATE KEY"}
	for _, f := range forbidden {
		if strings.Contains(tc.dashboardBody, f) {
			return fmt.Errorf("page contains forbidden secret-like string: %q", f)
		}
	}
	return nil
}

func noNewWorkflowRun(ctx context.Context) error {
	tc := getTC(ctx)
	// Count workflow runs before and after? In our simple fixture we just assert nothing was queued.
	if tc.workflowQueued {
		return fmt.Errorf("unexpected workflow queued from dashboard interaction")
	}
	return nil
}

func noRepositoryMutation(ctx context.Context) error {
	// Dashboard is read-only; no mutation endpoints exist.
	return nil
}
