package dashboard

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mngeow/heimdall/internal/store"
)

func setupTestHandler(t *testing.T) (*Handler, *store.Store, func()) {
	ctx := t.Context()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("store.Migrate() error = %v", err)
	}
	q := NewQueries(s.DB())
	h, err := NewHandler(q)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return h, s, func() { s.Close() }
}

func TestHandlerOverview(t *testing.T) {
	ctx := t.Context()
	h, s, cleanup := setupTestHandler(t)
	defer cleanup()

	if err := s.SaveWorkItem(ctx, &store.WorkItem{
		Provider:           "linear",
		ProviderWorkItemID: "linear-1",
		WorkItemKey:        "ENG-1",
		Title:              "Test",
		StateName:          "In Progress",
		LifecycleBucket:    "active",
	}); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/ui")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("expected text/html, got %q", resp.Header.Get("Content-Type"))
	}
	if !strings.Contains(string(body), "Total Work Items") {
		t.Fatalf("expected overview content, got %q", string(body))
	}
}

func TestHandlerWorkItemsFilter(t *testing.T) {
	ctx := t.Context()
	h, s, cleanup := setupTestHandler(t)
	defer cleanup()

	for _, wi := range []*store.WorkItem{
		{Provider: "linear", ProviderWorkItemID: "l1", WorkItemKey: "ENG-1", Title: "A", StateName: "Todo", LifecycleBucket: "inactive"},
		{Provider: "linear", ProviderWorkItemID: "l2", WorkItemKey: "ENG-2", Title: "B", StateName: "In Progress", LifecycleBucket: "active"},
	} {
		if err := s.SaveWorkItem(ctx, wi); err != nil {
			t.Fatal(err)
		}
	}

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/ui/work-items?status=Todo")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "ENG-1") {
		t.Fatalf("expected ENG-1 in filtered results, got %q", string(body))
	}
	if strings.Contains(string(body), "ENG-2") {
		t.Fatalf("did not expect ENG-2 in Todo-filtered results")
	}
}

func TestHandlerWorkItemsHTMXFragment(t *testing.T) {
	ctx := t.Context()
	h, s, cleanup := setupTestHandler(t)
	defer cleanup()

	if err := s.SaveWorkItem(ctx, &store.WorkItem{
		Provider: "linear", ProviderWorkItemID: "l1", WorkItemKey: "ENG-1", Title: "A", StateName: "In Progress", LifecycleBucket: "active",
	}); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/ui/work-items/fragment?bucket=active", nil)
	req.Header.Set("HX-Request", "true")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if strings.Contains(string(body), "<html") {
		t.Fatalf("expected HTML fragment, got full page")
	}
	if !strings.Contains(string(body), "ENG-1") {
		t.Fatalf("expected ENG-1 in fragment, got %q", string(body))
	}
}

func TestHandlerPullRequestDetail(t *testing.T) {
	ctx := t.Context()
	h, s, cleanup := setupTestHandler(t)
	defer cleanup()

	repo := &store.Repository{Provider: "github", RepoRef: "github.com/test/repo", Owner: "test", Name: "repo", DefaultBranch: "main", LocalMirrorPath: "/tmp/repo.git", IsActive: true}
	if err := s.SaveRepository(ctx, repo); err != nil {
		t.Fatal(err)
	}
	wi := &store.WorkItem{Provider: "linear", ProviderWorkItemID: "l1", WorkItemKey: "ENG-1", Title: "Test", StateName: "In Progress", LifecycleBucket: "active"}
	if err := s.SaveWorkItem(ctx, wi); err != nil {
		t.Fatal(err)
	}
	binding := &store.RepoBinding{WorkItemID: wi.ID, RepositoryID: repo.ID, BranchName: "heimdall/ENG-1-test", ChangeName: "ENG-1-test", BindingStatus: "active"}
	if err := s.SaveRepoBinding(ctx, binding); err != nil {
		t.Fatal(err)
	}
	pr := &store.PullRequest{RepositoryID: repo.ID, RepoBindingID: &binding.ID, Number: 42, Title: "PR", Provider: "github", HeadBranch: "heimdall/ENG-1-test", BaseBranch: "main", State: "open", URL: "https://github.com/test/repo/pull/42"}
	if err := s.SavePullRequest(ctx, pr); err != nil {
		t.Fatal(err)
	}
	cmd := &store.CommandRequest{PullRequestID: pr.ID, CommentNodeID: "c1", CommandName: "/heimdall status", ActorLogin: "alice", AuthorizationStatus: "allowed", DedupeKey: "d1", Status: "processed"}
	if err := s.SaveCommandRequest(ctx, cmd); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/ui/pull-requests/1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "Heimdall-Tracked Command / Activity History") {
		t.Fatalf("expected Heimdall-tracked label, got %q", string(body))
	}
	if !strings.Contains(string(body), "github.com/test/repo/pull/42") {
		t.Fatalf("expected GitHub PR link, got %q", string(body))
	}
	if !strings.Contains(string(body), "ENG-1") {
		t.Fatalf("expected linked work item, got %q", string(body))
	}
	forbidden := []string{"linear-token", "PRIVATE KEY", "x-access-token"}
	for _, f := range forbidden {
		if strings.Contains(string(body), f) {
			t.Fatalf("page contains forbidden string %q", f)
		}
	}
}

func TestHandlerPullRequestDetailNotFound(t *testing.T) {
	h, _, cleanup := setupTestHandler(t)
	defer cleanup()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/ui/pull-requests/9999")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestHandlerReadOnlyNoMutationEndpoints(t *testing.T) {
	h, _, cleanup := setupTestHandler(t)
	defer cleanup()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		req, _ := http.NewRequest(method, srv.URL+"/ui/work-items", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405 for %s, got %d", method, resp.StatusCode)
		}
	}
}
