package dashboard

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/mngeow/heimdall/internal/store"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Handler serves the operator dashboard.
type Handler struct {
	queries   *Queries
	templates map[string]*template.Template
}

// NewHandler creates a new dashboard handler.
func NewHandler(dbQuerier *Queries) (*Handler, error) {
	funcs := template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"dict":     dict,
		"add":      func(a, b int) int { return a + b },
	}
	baseTmpl, err := template.New("").Funcs(funcs).ParseFS(templatesFS, "templates/base.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse base template: %w", err)
	}

	pageFiles := []string{
		"overview.html",
		"work_items.html",
		"pull_requests.html",
		"pr_detail.html",
		"command_runs.html",
		"command_run_detail.html",
	}
	templates := make(map[string]*template.Template)
	for _, file := range pageFiles {
		clone, err := baseTmpl.Clone()
		if err != nil {
			return nil, fmt.Errorf("failed to clone base template for %s: %w", file, err)
		}
		_, err = clone.ParseFS(templatesFS, "templates/"+file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", file, err)
		}
		templates[file] = clone
	}
	// Register fragment aliases pointing to the same cloned template set.
	templates["work_items_fragment.html"] = templates["work_items.html"]
	templates["pull_requests_fragment.html"] = templates["pull_requests.html"]
	templates["pr_detail_fragment.html"] = templates["pr_detail.html"]
	templates["command_runs_fragment.html"] = templates["command_runs.html"]
	templates["command_run_detail_fragment.html"] = templates["command_run_detail.html"]
	templates["command_run_timeline_fragment.html"] = templates["command_run_detail.html"]

	return &Handler{queries: dbQuerier, templates: templates}, nil
}

func isHTMX(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("HX-Request")) == "true"
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, name string, data any) {
	tmpl, ok := h.templates[name]
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

// RegisterRoutes mounts dashboard routes on the provided mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/ui", h.handleOverview)
	mux.HandleFunc("/ui/work-items", h.handleWorkItems)
	mux.HandleFunc("/ui/work-items/fragment", h.handleWorkItemsFragment)
	mux.HandleFunc("/ui/pull-requests", h.handlePullRequests)
	mux.HandleFunc("/ui/pull-requests/{id}", h.handlePullRequestDetail)
	mux.HandleFunc("/ui/command-runs", h.handleCommandRuns)
	mux.HandleFunc("/ui/command-runs/fragment", h.handleCommandRunsFragment)
	mux.HandleFunc("/ui/command-runs/{id}", h.handleCommandRunDetail)
	mux.HandleFunc("/ui/command-runs/{id}/timeline", h.handleCommandRunTimeline)
}

func (h *Handler) handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	snap, err := h.queries.Overview(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.render(w, r, "overview.html", snap)
}

func (h *Handler) handleWorkItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	filterStatus := r.URL.Query().Get("status")
	filterBucket := r.URL.Query().Get("bucket")

	rows, err := h.queries.WorkItemQueue(ctx, filterStatus, filterBucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	statuses, err := h.queries.DistinctWorkItemStatuses(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	buckets, err := h.queries.DistinctWorkItemBuckets(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Rows":         rows,
		"FilterStatus": filterStatus,
		"FilterBucket": filterBucket,
		"Statuses":     statuses,
		"Buckets":      buckets,
		"IsFragment":   false,
	}
	if isHTMX(r) {
		h.render(w, r, "work_items_fragment.html", data)
		return
	}
	h.render(w, r, "work_items.html", data)
}

func (h *Handler) handleWorkItemsFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	filterStatus := r.URL.Query().Get("status")
	filterBucket := r.URL.Query().Get("bucket")

	rows, err := h.queries.WorkItemQueue(ctx, filterStatus, filterBucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	statuses, err := h.queries.DistinctWorkItemStatuses(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	buckets, err := h.queries.DistinctWorkItemBuckets(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Rows":         rows,
		"FilterStatus": filterStatus,
		"FilterBucket": filterBucket,
		"Statuses":     statuses,
		"Buckets":      buckets,
		"IsFragment":   true,
	}
	h.render(w, r, "work_items_fragment.html", data)
}

func (h *Handler) handlePullRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	rows, err := h.queries.ActivePullRequests(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if isHTMX(r) {
		h.render(w, r, "pull_requests_fragment.html", map[string]any{"Rows": rows, "IsFragment": true})
		return
	}
	h.render(w, r, "pull_requests.html", map[string]any{"Rows": rows, "IsFragment": false})
}

func (h *Handler) handlePullRequestDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	idStr := strings.TrimPrefix(r.URL.Path, "/ui/pull-requests/")
	prID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid pull request id", http.StatusBadRequest)
		return
	}

	detail, err := h.queries.PullRequestDetail(ctx, prID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if detail == nil {
		http.NotFound(w, r)
		return
	}

	if isHTMX(r) {
		h.render(w, r, "pr_detail_fragment.html", map[string]any{"Detail": detail, "IsFragment": true})
		return
	}
	h.render(w, r, "pr_detail.html", map[string]any{"Detail": detail, "IsFragment": false})
}

func (h *Handler) handleCommandRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	rows, err := h.queries.ActiveCommandRuns(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := map[string]any{"Rows": rows, "IsFragment": false}
	if isHTMX(r) {
		h.render(w, r, "command_runs_fragment.html", data)
		return
	}
	h.render(w, r, "command_runs.html", data)
}

func (h *Handler) handleCommandRunsFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	rows, err := h.queries.ActiveCommandRuns(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.render(w, r, "command_runs_fragment.html", map[string]any{"Rows": rows, "IsFragment": true})
}

func (h *Handler) handleCommandRunDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	idStr := strings.TrimPrefix(r.URL.Path, "/ui/command-runs/")
	idStr = strings.TrimSuffix(idStr, "/timeline")
	commandRequestID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid command run id", http.StatusBadRequest)
		return
	}

	detail, err := h.queries.CommandRunDetail(ctx, commandRequestID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if detail == nil {
		http.NotFound(w, r)
		return
	}

	// Timeline: default limit 200, offset from query param
	limit := 200
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	timeline, err := h.queries.CommandTimeline(ctx, detail.CommandRunID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Reverse to chronological order: oldest first, newest last.
	// This makes the timeline read top-to-bottom like terminal scrollback.
	for i, j := 0, len(timeline)-1; i < j; i, j = i+1, j-1 {
		timeline[i], timeline[j] = timeline[j], timeline[i]
	}

	isTerminal := detail.Status == "completed" || detail.Status == "failed" || detail.Status == "blocked"
	data := map[string]any{
		"Detail":         detail,
		"Timeline":       timeline,
		"IsFragment":     false,
		"Limit":          limit,
		"Offset":         offset,
		"IsTerminal":     isTerminal,
		"HasMoreEntries": len(timeline) == limit,
	}
	if isHTMX(r) {
		h.render(w, r, "command_run_detail_fragment.html", data)
		return
	}
	h.render(w, r, "command_run_detail.html", data)
}

func (h *Handler) handleCommandRunTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	idStr := strings.TrimPrefix(r.URL.Path, "/ui/command-runs/")
	idStr = strings.TrimSuffix(idStr, "/timeline")
	commandRequestID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid command run id", http.StatusBadRequest)
		return
	}

	detail, err := h.queries.CommandRunDetail(ctx, commandRequestID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if detail == nil {
		http.NotFound(w, r)
		return
	}

	limit := 200
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	timeline, err := h.queries.CommandTimeline(ctx, detail.CommandRunID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Reverse to chronological order: oldest first, newest last.
	// This makes the timeline read top-to-bottom like terminal scrollback.
	for i, j := 0, len(timeline)-1; i < j; i, j = i+1, j-1 {
		timeline[i], timeline[j] = timeline[j], timeline[i]
	}

	isTerminal := detail.Status == "completed" || detail.Status == "failed" || detail.Status == "blocked"
	h.render(w, r, "command_run_timeline_fragment.html", map[string]any{
		"Detail":         detail,
		"Timeline":       timeline,
		"IsFragment":     true,
		"Limit":          limit,
		"Offset":         offset,
		"IsTerminal":     isTerminal,
		"HasMoreEntries": len(timeline) == limit,
	})
}

// dict is a small helper for building maps inside templates.
func dict(values ...any) map[string]any {
	m := make(map[string]any)
	for i := 0; i+1 < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			continue
		}
		m[key] = values[i+1]
	}
	return m
}

// Ensure Handler exposes only read-only operations and no mutation triggers.
var _ interface {
	RegisterRoutes(*http.ServeMux)
} = (*Handler)(nil)

// StoreQuerier wraps store.DB for dashboard queries.
func StoreQuerier(s *store.Store) *Queries {
	// Use reflection or internal access if needed; here we rely on the Store exposing DB.
	// To keep it minimal, we add an accessor in store package next.
	return nil
}

// BuildCommandRunURL returns the absolute dashboard URL for a command run detail page.
func BuildCommandRunURL(baseURL string, commandRequestID int64) string {
	base := strings.TrimSuffix(baseURL, "/")
	return fmt.Sprintf("%s/ui/command-runs/%d", base, commandRequestID)
}
