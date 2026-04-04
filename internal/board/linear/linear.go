package linear

import (
	"context"
	"fmt"
	"time"

	"github.com/mngeow/symphony/internal/store"
)

// Provider implements the Linear board provider adapter
type Provider struct {
	apiToken string
	teamKeys []string
	store    *store.Store
}

// NewProvider creates a new Linear provider
func NewProvider(apiToken string, teamKeys []string, store *store.Store) *Provider {
	return &Provider{
		apiToken: apiToken,
		teamKeys: teamKeys,
		store:    store,
	}
}

// WorkItem represents a normalized work item from Linear
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

// PollResult represents the result of a poll cycle
type PollResult struct {
	WorkItems []WorkItem
	Cursor    string
}

// Poll fetches recently updated issues from Linear
func (p *Provider) Poll(ctx context.Context) (*PollResult, error) {
	// TODO: Implement actual Linear GraphQL API call
	// For now, return empty result to allow compilation
	return &PollResult{
		WorkItems: []WorkItem{},
		Cursor:    time.Now().Format(time.RFC3339),
	}, nil
}

// NormalizeState converts a Linear state to a lifecycle bucket
func (p *Provider) NormalizeState(stateName string) string {
	// TODO: Make this configurable
	activeStates := map[string]bool{
		"In Progress": true,
		"Doing":       true,
	}

	if activeStates[stateName] {
		return "active"
	}
	return "inactive"
}

// ProcessTransitions checks for state transitions and emits events
func (p *Provider) ProcessTransitions(ctx context.Context, items []WorkItem) error {
	for _, item := range items {
		// Get or create work item in store
		existing, err := p.store.GetWorkItemByKey(ctx, "linear", item.Key)
		if err != nil {
			return fmt.Errorf("failed to get work item: %w", err)
		}

		newBucket := p.NormalizeState(item.State)

		// Check for transition to active state
		if existing == nil || p.NormalizeState(existing.StateName) != "active" {
			if newBucket == "active" {
				// Emit transition event
				event := &store.WorkItemEvent{
					Provider:       "linear",
					EventType:      "entered_active_state",
					IdempotencyKey: fmt.Sprintf("linear:%s:entered_active_state:%s", item.Key, item.UpdatedAt.Format(time.RFC3339)),
					OccurredAt:     item.UpdatedAt,
					DetectedAt:     time.Now(),
				}

				if existing != nil {
					event.WorkItemID = existing.ID
				}

				if err := p.store.SaveWorkItemEvent(ctx, event); err != nil {
					return fmt.Errorf("failed to save event: %w", err)
				}
			}
		}

		// Save/update work item
		workItem := &store.WorkItem{
			Provider:           "linear",
			ProviderWorkItemID: item.ID,
			WorkItemKey:        item.Key,
			Title:              item.Title,
			StateName:          item.State,
			LifecycleBucket:    newBucket,
			Team:               item.Team,
		}

		if err := p.store.SaveWorkItem(ctx, workItem); err != nil {
			return fmt.Errorf("failed to save work item: %w", err)
		}
	}

	return nil
}
