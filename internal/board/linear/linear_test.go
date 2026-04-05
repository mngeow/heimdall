package linear

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mngeow/symphony/internal/store"
)

var testPollTime = time.Date(2026, time.April, 5, 10, 0, 0, 0, time.UTC)

func TestPollProjectScopePaginationAndCheckpoint(t *testing.T) {
	ctx := context.Background()
	runtimeStore := testLinearStore(t)

	requestBodies := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requestBodies = append(requestBodies, string(body))
		if len(requestBodies) == 1 {
			_, _ = w.Write([]byte(`{"data":{"issues":{"nodes":[{"id":"linear-issue-1","identifier":"ENG-123","title":"Add rate limiting","description":"More details","updatedAt":"2026-04-05T09:59:30Z","state":{"id":"state-1","name":"In Progress","type":"started"},"team":{"id":"team-1","key":"ENG","name":"Engineering"},"project":{"id":"project-1","name":"Core Platform"},"labels":{"nodes":[]}}],"pageInfo":{"hasNextPage":true,"endCursor":"cursor-2"}}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"issues":{"nodes":[{"id":"linear-issue-2","identifier":"ENG-124","title":"Improve retries","description":"More details","updatedAt":"2026-04-05T09:59:45Z","state":{"id":"state-1","name":"In Progress","type":"started"},"team":{"id":"team-1","key":"ENG","name":"Engineering"},"project":{"id":"project-1","name":"Core Platform"},"labels":{"nodes":[]}}],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}`))
	}))
	defer server.Close()

	provider := NewProvider("linear-token", "Core Platform", []string{"In Progress"}, 30*time.Second, runtimeStore,
		WithEndpoint(server.URL),
		WithHTTPClient(server.Client()),
		WithPageSize(1),
		WithNow(func() time.Time { return testPollTime }),
	)

	result, err := provider.Poll(ctx)
	if err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	if len(result.WorkItems) != 2 {
		t.Fatalf("expected 2 work items, got %d", len(result.WorkItems))
	}
	if len(requestBodies) != 2 {
		t.Fatalf("expected 2 GraphQL requests, got %d", len(requestBodies))
	}
	if !strings.Contains(requestBodies[0], `"projectName":"Core Platform"`) {
		t.Fatalf("expected project-scoped request body, got %s", requestBodies[0])
	}

	cursor, err := runtimeStore.GetProviderCursor(ctx, providerName, "linear:project:Core Platform")
	if err != nil {
		t.Fatalf("GetProviderCursor() error = %v", err)
	}
	if cursor == nil || cursor.CursorValue != testPollTime.Format(time.RFC3339Nano) {
		t.Fatalf("expected checkpoint %q, got %#v", testPollTime.Format(time.RFC3339Nano), cursor)
	}
}

func TestPollRateLimitDoesNotAdvanceCheckpoint(t *testing.T) {
	ctx := context.Background()
	runtimeStore := testLinearStore(t)
	if err := runtimeStore.SetProviderCursor(ctx, &store.ProviderCursor{
		Provider:    providerName,
		ScopeKey:    "linear:project:Core Platform",
		CursorValue: "2026-04-05T09:55:00Z",
		CursorKind:  pollCursorKind,
	}); err != nil {
		t.Fatalf("SetProviderCursor() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Requests-Remaining", "0")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errors":[{"message":"Too many requests","extensions":{"code":"RATELIMITED"}}]}`))
	}))
	defer server.Close()

	provider := NewProvider("linear-token", "Core Platform", []string{"In Progress"}, 30*time.Second, runtimeStore,
		WithEndpoint(server.URL),
		WithHTTPClient(server.Client()),
		WithNow(func() time.Time { return testPollTime }),
	)

	_, err := provider.Poll(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "rate limited") {
		t.Fatalf("expected rate limit error, got %v", err)
	}

	cursor, err := runtimeStore.GetProviderCursor(ctx, providerName, "linear:project:Core Platform")
	if err != nil {
		t.Fatalf("GetProviderCursor() error = %v", err)
	}
	if cursor == nil || cursor.CursorValue != "2026-04-05T09:55:00Z" {
		t.Fatalf("expected unchanged checkpoint, got %#v", cursor)
	}
}

func TestProcessTransitionsSuppressesDuplicateActivation(t *testing.T) {
	ctx := context.Background()
	runtimeStore := testLinearStore(t)
	observedAt := testPollTime.Add(-time.Minute)
	if err := runtimeStore.SaveWorkItem(ctx, &store.WorkItem{
		Provider:           providerName,
		ProviderWorkItemID: "linear-issue-1",
		WorkItemKey:        "ENG-123",
		Title:              "Add rate limiting",
		StateName:          "Todo",
		LifecycleBucket:    "inactive",
		Team:               "ENG",
		LastSeenUpdatedAt:  &observedAt,
	}); err != nil {
		t.Fatalf("SaveWorkItem() error = %v", err)
	}

	provider := NewProvider("linear-token", "Core Platform", []string{"In Progress"}, 30*time.Second, runtimeStore,
		WithNow(func() time.Time { return testPollTime }),
	)

	items := []WorkItem{{
		ID:          "linear-issue-1",
		Key:         "ENG-123",
		Title:       "Add rate limiting",
		Description: "More details",
		State:       "In Progress",
		Project:     "Core Platform",
		Team:        "ENG",
		UpdatedAt:   testPollTime,
	}}

	activated, err := provider.ProcessTransitions(ctx, items)
	if err != nil {
		t.Fatalf("ProcessTransitions() error = %v", err)
	}
	if len(activated) != 1 {
		t.Fatalf("expected one activation, got %d", len(activated))
	}

	repeated, err := provider.ProcessTransitions(ctx, items)
	if err != nil {
		t.Fatalf("ProcessTransitions() repeat error = %v", err)
	}
	if len(repeated) != 0 {
		t.Fatalf("expected duplicate activation to be suppressed, got %d", len(repeated))
	}

	snapshot, err := runtimeStore.GetWorkItemByKey(ctx, providerName, "ENG-123")
	if err != nil {
		t.Fatalf("GetWorkItemByKey() error = %v", err)
	}
	if snapshot == nil || snapshot.LifecycleBucket != "active" {
		t.Fatalf("expected active snapshot, got %#v", snapshot)
	}
}

func testLinearStore(t *testing.T) *store.Store {
	t.Helper()
	runtimeStore, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	if err := runtimeStore.Migrate(context.Background()); err != nil {
		t.Fatalf("store.Migrate() error = %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeStore.Close()
	})
	return runtimeStore
}
