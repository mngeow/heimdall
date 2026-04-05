package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/mngeow/symphony/internal/store"
)

const (
	defaultGraphQLEndpoint = "https://api.linear.app/graphql"
	defaultPageSize        = 50
	defaultOverlapWindow   = 30 * time.Second
	providerName           = "linear"
	pollCursorKind         = "timestamp"
)

const pollIssuesQuery = `query PollIssues($first: Int!, $after: String, $updatedSince: DateTimeOrDuration!, $projectName: String!) {
  issues(
    first: $first
    after: $after
    orderBy: updatedAt
    filter: {
      updatedAt: { gte: $updatedSince }
      project: { name: { eq: $projectName } }
    }
  ) {
    nodes {
      id
      identifier
      title
      description
      updatedAt
      state {
        id
        name
        type
      }
      team {
        id
        key
        name
      }
      project {
        id
        name
      }
      labels {
        nodes {
          id
          name
        }
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}`

// Provider implements the Linear board provider adapter.
type Provider struct {
	apiToken      string
	projectName   string
	activeStates  map[string]struct{}
	store         *store.Store
	client        *http.Client
	endpoint      string
	pageSize      int
	overlapWindow time.Duration
	now           func() time.Time
}

// Option configures the Linear provider.
type Option func(*Provider)

// WithHTTPClient overrides the HTTP client used for GraphQL requests.
func WithHTTPClient(client *http.Client) Option {
	return func(p *Provider) {
		if client != nil {
			p.client = client
		}
	}
}

// WithEndpoint overrides the default Linear GraphQL endpoint.
func WithEndpoint(endpoint string) Option {
	return func(p *Provider) {
		if strings.TrimSpace(endpoint) != "" {
			p.endpoint = strings.TrimSpace(endpoint)
		}
	}
}

// WithPageSize overrides the default GraphQL page size.
func WithPageSize(pageSize int) Option {
	return func(p *Provider) {
		if pageSize > 0 {
			p.pageSize = pageSize
		}
	}
}

// WithNow overrides the provider clock.
func WithNow(now func() time.Time) Option {
	return func(p *Provider) {
		if now != nil {
			p.now = now
		}
	}
}

// WithOverlapWindow overrides the checkpoint overlap window.
func WithOverlapWindow(overlapWindow time.Duration) Option {
	return func(p *Provider) {
		if overlapWindow > 0 {
			p.overlapWindow = overlapWindow
		}
	}
}

// NewProvider creates a new Linear provider.
func NewProvider(apiToken, projectName string, activeStates []string, overlapWindow time.Duration, store *store.Store, options ...Option) *Provider {
	provider := &Provider{
		apiToken:      strings.TrimSpace(apiToken),
		projectName:   strings.TrimSpace(projectName),
		activeStates:  buildActiveStateSet(activeStates),
		store:         store,
		client:        &http.Client{Timeout: 15 * time.Second},
		endpoint:      defaultGraphQLEndpoint,
		pageSize:      defaultPageSize,
		overlapWindow: defaultOverlapWindow,
		now:           time.Now,
	}

	if overlapWindow > 0 {
		provider.overlapWindow = overlapWindow
	}

	for _, option := range options {
		option(provider)
	}

	return provider
}

// WorkItem represents a normalized work item from Linear.
type WorkItem struct {
	ID            string
	Key           string
	Title         string
	Description   string
	State         string
	Project       string
	Team          string
	Labels        []string
	RepositoryRef string
	UpdatedAt     time.Time
}

// PollResult represents the result of a poll cycle.
type PollResult struct {
	WorkItems []WorkItem
	Cursor    string
}

type graphQLRequest struct {
	Query     string              `json:"query"`
	Variables pollIssuesVariables `json:"variables"`
}

type pollIssuesVariables struct {
	First        int     `json:"first"`
	After        *string `json:"after,omitempty"`
	UpdatedSince string  `json:"updatedSince"`
	ProjectName  string  `json:"projectName"`
}

type graphQLResponse struct {
	Data   pollIssuesData `json:"data"`
	Errors []graphQLError `json:"errors"`
}

type graphQLError struct {
	Message    string              `json:"message"`
	Extensions graphQLErrorDetails `json:"extensions"`
}

type graphQLErrorDetails struct {
	Code string `json:"code"`
}

type pollIssuesData struct {
	Issues issueConnection `json:"issues"`
}

type issueConnection struct {
	Nodes    []issueNode `json:"nodes"`
	PageInfo pageInfo    `json:"pageInfo"`
}

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type issueNode struct {
	ID          string          `json:"id"`
	Identifier  string          `json:"identifier"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	UpdatedAt   string          `json:"updatedAt"`
	State       issueState      `json:"state"`
	Team        issueTeam       `json:"team"`
	Project     issueProject    `json:"project"`
	Labels      labelConnection `json:"labels"`
}

type issueState struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type issueTeam struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type issueProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type labelConnection struct {
	Nodes []issueLabel `json:"nodes"`
}

type issueLabel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type pollPageResult struct {
	WorkItems []WorkItem
	PageInfo  pageInfo
}

// Poll fetches recently updated issues from Linear.
func (p *Provider) Poll(ctx context.Context) (*PollResult, error) {
	pollStartedAt := p.now().UTC()
	updatedSince, err := p.loadUpdatedSince(ctx, pollStartedAt)
	if err != nil {
		return nil, err
	}

	items := make([]WorkItem, 0)
	var after *string
	for {
		page, err := p.pollPage(ctx, updatedSince, after)
		if err != nil {
			return nil, err
		}
		items = append(items, page.WorkItems...)
		if !page.PageInfo.HasNextPage || page.PageInfo.EndCursor == "" {
			break
		}
		nextCursor := page.PageInfo.EndCursor
		after = &nextCursor
	}

	checkpoint := pollStartedAt.Format(time.RFC3339Nano)
	if err := p.store.SetProviderCursor(ctx, &store.ProviderCursor{
		Provider:    providerName,
		ScopeKey:    p.scopeKey(),
		CursorValue: checkpoint,
		CursorKind:  pollCursorKind,
	}); err != nil {
		return nil, fmt.Errorf("failed to persist linear poll checkpoint: %w", err)
	}

	return &PollResult{WorkItems: items, Cursor: checkpoint}, nil
}

// NormalizeState converts a Linear state to a lifecycle bucket.
func (p *Provider) NormalizeState(stateName string) string {
	if _, ok := p.activeStates[normalizeStateName(stateName)]; ok {
		return "active"
	}
	return "inactive"
}

// ProcessTransitions checks for state transitions and emits events.
func (p *Provider) ProcessTransitions(ctx context.Context, items []WorkItem) ([]WorkItem, error) {
	activated := make([]WorkItem, 0)
	for _, item := range items {
		existing, err := p.store.GetWorkItemByKey(ctx, providerName, item.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to get work item %s: %w", item.Key, err)
		}

		previousBucket := ""
		if existing != nil {
			previousBucket = existing.LifecycleBucket
		}

		newBucket := p.NormalizeState(item.State)
		observedAt := item.UpdatedAt.UTC()
		workItem := &store.WorkItem{
			Provider:           providerName,
			ProviderWorkItemID: item.ID,
			WorkItemKey:        item.Key,
			Title:              item.Title,
			Description:        item.Description,
			StateName:          item.State,
			LifecycleBucket:    newBucket,
			Project:            item.Project,
			Team:               item.Team,
			Labels:             item.Labels,
			LastSeenUpdatedAt:  &observedAt,
		}

		if err := p.store.SaveWorkItem(ctx, workItem); err != nil {
			return nil, fmt.Errorf("failed to save work item %s: %w", item.Key, err)
		}

		if previousBucket == "active" || newBucket != "active" {
			continue
		}

		event := &store.WorkItemEvent{
			WorkItemID:      workItem.ID,
			Provider:        providerName,
			ProviderEventID: fmt.Sprintf("%s:%s", item.ID, observedAt.Format(time.RFC3339Nano)),
			EventType:       "entered_active_state",
			IdempotencyKey:  fmt.Sprintf("linear:%s:entered_active_state:%s", item.ID, observedAt.Format(time.RFC3339Nano)),
			OccurredAt:      observedAt,
			DetectedAt:      p.now().UTC(),
		}

		if err := p.store.SaveWorkItemEvent(ctx, event); err != nil {
			return nil, fmt.Errorf("failed to save work item event for %s: %w", item.Key, err)
		}

		activated = append(activated, item)
	}

	return activated, nil
}

func (p *Provider) loadUpdatedSince(ctx context.Context, pollStartedAt time.Time) (time.Time, error) {
	updatedSince := pollStartedAt.Add(-p.overlapWindow)
	checkpoint, err := p.store.GetProviderCursor(ctx, providerName, p.scopeKey())
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to load linear poll checkpoint: %w", err)
	}
	if checkpoint == nil || checkpoint.CursorValue == "" {
		return updatedSince, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, checkpoint.CursorValue)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse linear poll checkpoint %q: %w", checkpoint.CursorValue, err)
	}

	return parsed.Add(-p.overlapWindow), nil
}

func (p *Provider) pollPage(ctx context.Context, updatedSince time.Time, after *string) (*pollPageResult, error) {
	requestBody, err := json.Marshal(graphQLRequest{
		Query: pollIssuesQuery,
		Variables: pollIssuesVariables{
			First:        p.pageSize,
			After:        after,
			UpdatedSince: updatedSince.UTC().Format(time.RFC3339Nano),
			ProjectName:  p.projectName,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal linear poll request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create linear poll request: %w", err)
	}
	req.Header.Set("Authorization", p.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Linear GraphQL API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Linear GraphQL response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("linear authentication failed with status %d%s", resp.StatusCode, formatRateLimitMetadata(resp.Header))
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("linear poll request failed with status %d%s: %s", resp.StatusCode, formatRateLimitMetadata(resp.Header), strings.TrimSpace(string(body)))
	}

	var payload graphQLResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode Linear GraphQL response: %w", err)
	}

	if len(payload.Errors) > 0 {
		return nil, formatGraphQLErrors(payload.Errors, resp.Header)
	}

	workItems := make([]WorkItem, 0, len(payload.Data.Issues.Nodes))
	for _, node := range payload.Data.Issues.Nodes {
		updatedAt, err := time.Parse(time.RFC3339, node.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Linear issue %s updatedAt %q: %w", node.Identifier, node.UpdatedAt, err)
		}

		labels := make([]string, 0, len(node.Labels.Nodes))
		for _, label := range node.Labels.Nodes {
			name := strings.TrimSpace(label.Name)
			if name == "" {
				continue
			}
			labels = append(labels, name)
		}
		sort.Strings(labels)

		workItems = append(workItems, WorkItem{
			ID:          node.ID,
			Key:         strings.TrimSpace(node.Identifier),
			Title:       strings.TrimSpace(node.Title),
			Description: node.Description,
			State:       strings.TrimSpace(node.State.Name),
			Project:     strings.TrimSpace(node.Project.Name),
			Team:        strings.TrimSpace(node.Team.Key),
			Labels:      labels,
			UpdatedAt:   updatedAt.UTC(),
		})
	}

	return &pollPageResult{
		WorkItems: workItems,
		PageInfo:  payload.Data.Issues.PageInfo,
	}, nil
}

func (p *Provider) scopeKey() string {
	return "linear:project:" + p.projectName
}

func buildActiveStateSet(activeStates []string) map[string]struct{} {
	values := make(map[string]struct{}, len(activeStates))
	for _, state := range activeStates {
		normalized := normalizeStateName(state)
		if normalized == "" {
			continue
		}
		values[normalized] = struct{}{}
	}
	return values
}

func normalizeStateName(stateName string) string {
	return strings.ToLower(strings.TrimSpace(stateName))
}

func formatGraphQLErrors(errors []graphQLError, headers http.Header) error {
	messages := make([]string, 0, len(errors))
	rateLimited := false
	for _, graphQLError := range errors {
		message := strings.TrimSpace(graphQLError.Message)
		if graphQLError.Extensions.Code != "" {
			message = fmt.Sprintf("%s (%s)", message, graphQLError.Extensions.Code)
		}
		messages = append(messages, message)
		if strings.EqualFold(graphQLError.Extensions.Code, "RATELIMITED") {
			rateLimited = true
		}
	}

	prefix := "linear GraphQL query failed"
	if rateLimited {
		prefix = "linear poll rate limited"
	}

	return fmt.Errorf("%s%s: %s", prefix, formatRateLimitMetadata(headers), strings.Join(messages, "; "))
}

func formatRateLimitMetadata(headers http.Header) string {
	parts := make([]string, 0, 7)
	for _, header := range []string{
		"X-RateLimit-Requests-Limit",
		"X-RateLimit-Requests-Remaining",
		"X-RateLimit-Requests-Reset",
		"X-RateLimit-Complexity-Limit",
		"X-RateLimit-Complexity-Remaining",
		"X-RateLimit-Complexity-Reset",
		"X-RateLimit-Endpoint-Name",
	} {
		value := strings.TrimSpace(headers.Get(header))
		if value == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", header, value))
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}
