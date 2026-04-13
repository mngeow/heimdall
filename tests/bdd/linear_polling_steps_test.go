package bdd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/mngeow/heimdall/internal/board/linear"
	"github.com/mngeow/heimdall/internal/store"
)

var linearPollTime = time.Date(2026, time.April, 5, 10, 0, 0, 0, time.UTC)

const persistedLinearCheckpoint = "2026-04-05T09:55:00Z"

type linearServerResponse struct {
	status  int
	headers map[string]string
	body    string
}

func registerLinearPollingSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a Linear GraphQL server with a valid project-scoped response$`, linearServerWithValidProjectResponse)
	sc.Step(`^a Linear GraphQL server with multiple pages of project issues$`, linearServerWithMultiplePages)
	sc.Step(`^a Linear GraphQL server that rejects the API key$`, linearServerRejectsAPIKey)
	sc.Step(`^a Linear GraphQL server with a rate-limited response$`, linearServerRateLimited)
	sc.Step(`^an existing Linear poll checkpoint$`, existingLinearPollCheckpoint)
	sc.Step(`^an existing inactive snapshot for issue "([^"]*)"$`, existingInactiveSnapshotForIssue)
	sc.Step(`^Heimdall polls Linear through the board provider$`, heimdallPollsLinearThroughBoardProvider)
	sc.Step(`^Heimdall processes the same Linear poll result again$`, heimdallProcessesSameLinearPollResultAgain)
	sc.Step(`^the Linear poll should succeed$`, linearPollShouldSucceed)
	sc.Step(`^the Linear poll should fail with "([^"]*)"$`, linearPollShouldFailWith)
	sc.Step(`^the board provider should scope requests to the configured project name$`, boardProviderShouldScopeRequestsToProject)
	sc.Step(`^the board provider should load the project-scoped issue "([^"]*)"$`, boardProviderShouldLoadIssue)
	sc.Step(`^the board provider should load (\d+) project-scoped issues$`, boardProviderShouldLoadIssueCount)
	sc.Step(`^the Linear poll checkpoint should be persisted$`, linearCheckpointShouldBePersisted)
	sc.Step(`^the Linear poll checkpoint should remain unchanged$`, linearCheckpointShouldRemainUnchanged)
	sc.Step(`^the board provider should emit an entered_active_state event$`, boardProviderShouldEmitEnteredActiveState)
	sc.Step(`^the board provider should not emit a duplicate entered_active_state event$`, boardProviderShouldNotEmitDuplicateEnteredActiveState)
}

func linearServerWithValidProjectResponse(ctx context.Context) error {
	return startLinearServer(getTC(ctx), []linearServerResponse{{
		status: http.StatusOK,
		body:   `{"data":{"issues":{"nodes":[{"id":"linear-issue-1","identifier":"ENG-123","title":"Add rate limiting","description":"More details","updatedAt":"2026-04-05T09:59:30Z","state":{"id":"state-1","name":"In Progress","type":"started"},"team":{"id":"team-1","key":"ENG","name":"Engineering"},"project":{"id":"project-1","name":"Core Platform"},"labels":{"nodes":[{"id":"label-1","name":"backend"}]}}],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}`,
	}})
}

func linearServerWithMultiplePages(ctx context.Context) error {
	return startLinearServer(getTC(ctx), []linearServerResponse{
		{
			status: http.StatusOK,
			body:   `{"data":{"issues":{"nodes":[{"id":"linear-issue-1","identifier":"ENG-123","title":"Add rate limiting","description":"More details","updatedAt":"2026-04-05T09:59:30Z","state":{"id":"state-1","name":"In Progress","type":"started"},"team":{"id":"team-1","key":"ENG","name":"Engineering"},"project":{"id":"project-1","name":"Core Platform"},"labels":{"nodes":[]}}],"pageInfo":{"hasNextPage":true,"endCursor":"cursor-2"}}}}`,
		},
		{
			status: http.StatusOK,
			body:   `{"data":{"issues":{"nodes":[{"id":"linear-issue-2","identifier":"ENG-124","title":"Improve retries","description":"More details","updatedAt":"2026-04-05T09:59:45Z","state":{"id":"state-1","name":"In Progress","type":"started"},"team":{"id":"team-1","key":"ENG","name":"Engineering"},"project":{"id":"project-1","name":"Core Platform"},"labels":{"nodes":[]}}],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}`,
		},
	})
}

func linearServerRejectsAPIKey(ctx context.Context) error {
	return startLinearServer(getTC(ctx), []linearServerResponse{{
		status: http.StatusUnauthorized,
		body:   `{"errors":[{"message":"Unauthorized"}]}`,
	}})
}

func linearServerRateLimited(ctx context.Context) error {
	return startLinearServer(getTC(ctx), []linearServerResponse{{
		status: http.StatusOK,
		headers: map[string]string{
			"X-RateLimit-Requests-Remaining":   "0",
			"X-RateLimit-Requests-Reset":       "1712311800000",
			"X-RateLimit-Complexity-Remaining": "0",
		},
		body: `{"errors":[{"message":"Too many requests","extensions":{"code":"RATELIMITED"}}]}`,
	}})
}

func existingLinearPollCheckpoint(ctx context.Context) error {
	tc := getTC(ctx)
	tc.linearCheckpoint = persistedLinearCheckpoint
	return tc.store.SetProviderCursor(ctx, &store.ProviderCursor{
		Provider:    "linear",
		ScopeKey:    "linear:project:Core Platform",
		CursorValue: persistedLinearCheckpoint,
		CursorKind:  "timestamp",
	})
}

func existingInactiveSnapshotForIssue(ctx context.Context, issueKey string) error {
	tc := getTC(ctx)
	observedAt := linearPollTime.Add(-time.Minute)
	return tc.store.SaveWorkItem(ctx, &store.WorkItem{
		Provider:           "linear",
		ProviderWorkItemID: "existing-" + issueKey,
		WorkItemKey:        issueKey,
		Title:              "Add rate limiting",
		StateName:          "Todo",
		LifecycleBucket:    "inactive",
		Team:               "ENG",
		LastSeenUpdatedAt:  &observedAt,
	})
}

func heimdallPollsLinearThroughBoardProvider(ctx context.Context) error {
	tc := getTC(ctx)
	tc.linearActivated = nil
	tc.linearPollResult, tc.linearPollErr = tcLinearProvider(tc).Poll(ctx)
	if tc.linearPollErr != nil {
		return nil
	}
	tc.linearActivated, tc.linearPollErr = tcLinearProvider(tc).ProcessTransitions(ctx, tc.linearPollResult.WorkItems)
	return nil
}

func heimdallProcessesSameLinearPollResultAgain(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.linearPollResult == nil {
		return fmt.Errorf("expected a prior Linear poll result")
	}
	tc.linearActivated, tc.linearPollErr = tcLinearProvider(tc).ProcessTransitions(ctx, tc.linearPollResult.WorkItems)
	return nil
}

func linearPollShouldSucceed(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.linearPollErr != nil {
		return fmt.Errorf("expected Linear poll to succeed, got %v", tc.linearPollErr)
	}
	if tc.linearPollResult == nil {
		return fmt.Errorf("expected a Linear poll result")
	}
	return nil
}

func linearPollShouldFailWith(ctx context.Context, message string) error {
	tc := getTC(ctx)
	if tc.linearPollErr == nil {
		return fmt.Errorf("expected Linear poll to fail with %q", message)
	}
	if !strings.Contains(strings.ToLower(tc.linearPollErr.Error()), strings.ToLower(message)) {
		return fmt.Errorf("expected Linear poll error to contain %q, got %v", message, tc.linearPollErr)
	}
	return nil
}

func boardProviderShouldScopeRequestsToProject(ctx context.Context) error {
	tc := getTC(ctx)
	if len(tc.linearRequests) == 0 {
		return fmt.Errorf("expected at least one Linear GraphQL request")
	}
	if !strings.Contains(tc.linearRequests[0], `"projectName":"Core Platform"`) {
		return fmt.Errorf("expected request body to scope to Core Platform, got %s", tc.linearRequests[0])
	}
	return nil
}

func boardProviderShouldLoadIssue(ctx context.Context, issueKey string) error {
	tc := getTC(ctx)
	for _, item := range tc.linearPollResult.WorkItems {
		if item.Key == issueKey {
			return nil
		}
	}
	return fmt.Errorf("expected project-scoped issue %q in poll result", issueKey)
}

func boardProviderShouldLoadIssueCount(ctx context.Context, expected int) error {
	tc := getTC(ctx)
	if tc.linearPollResult == nil {
		return fmt.Errorf("expected a Linear poll result")
	}
	if len(tc.linearPollResult.WorkItems) != expected {
		return fmt.Errorf("expected %d Linear work items, got %d", expected, len(tc.linearPollResult.WorkItems))
	}
	return nil
}

func linearCheckpointShouldBePersisted(ctx context.Context) error {
	tc := getTC(ctx)
	cursor, err := tc.store.GetProviderCursor(ctx, "linear", "linear:project:Core Platform")
	if err != nil {
		return err
	}
	if cursor == nil || cursor.CursorValue != linearPollTime.Format(time.RFC3339Nano) {
		return fmt.Errorf("expected persisted Linear checkpoint %q, got %#v", linearPollTime.Format(time.RFC3339Nano), cursor)
	}
	return nil
}

func linearCheckpointShouldRemainUnchanged(ctx context.Context) error {
	tc := getTC(ctx)
	cursor, err := tc.store.GetProviderCursor(ctx, "linear", "linear:project:Core Platform")
	if err != nil {
		return err
	}
	if cursor == nil || cursor.CursorValue != tc.linearCheckpoint {
		return fmt.Errorf("expected Linear checkpoint to remain %q, got %#v", tc.linearCheckpoint, cursor)
	}
	return nil
}

func boardProviderShouldEmitEnteredActiveState(ctx context.Context) error {
	tc := getTC(ctx)
	if len(tc.linearActivated) != 1 {
		return fmt.Errorf("expected one entered_active_state activation, got %d", len(tc.linearActivated))
	}
	return nil
}

func boardProviderShouldNotEmitDuplicateEnteredActiveState(ctx context.Context) error {
	tc := getTC(ctx)
	if len(tc.linearActivated) != 0 {
		return fmt.Errorf("expected no duplicate activation, got %d", len(tc.linearActivated))
	}
	return nil
}

func startLinearServer(tc *testContext, responses []linearServerResponse) error {
	if tc.config == nil {
		return fmt.Errorf("expected Heimdall config before starting Linear server")
	}
	if tc.linearCleanup != nil {
		tc.linearCleanup()
		tc.linearCleanup = nil
	}
	tc.linearRequests = nil
	remaining := append([]linearServerResponse(nil), responses...)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		tc.linearRequests = append(tc.linearRequests, string(body))
		if len(remaining) == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors":[{"message":"unexpected extra request"}]}`))
			return
		}
		response := remaining[0]
		remaining = remaining[1:]
		for key, value := range response.headers {
			w.Header().Set(key, value)
		}
		w.WriteHeader(response.status)
		_, _ = w.Write([]byte(response.body))
	}))
	tc.linearCleanup = server.Close
	tc.linearPollResult = nil
	tc.linearPollErr = nil
	tc.linearActivated = nil
	provider := linear.NewProvider(tc.config.Linear.APIToken, tc.config.Linear.ProjectName, tc.config.Linear.ActiveStates, tc.config.Linear.PollInterval, tc.store,
		linear.WithEndpoint(server.URL),
		linear.WithHTTPClient(server.Client()),
		linear.WithNow(func() time.Time { return linearPollTime }),
		linear.WithPageSize(1),
	)
	tc.linearProvider = provider
	return nil
}

func tcLinearProvider(tc *testContext) *linear.Provider {
	return tc.linearProvider
}
