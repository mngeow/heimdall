# Feature: Kanban Activation

## ADDED Requirements

### Requirement: Active state transitions start bootstrap workflows
Heimdall MUST detect when a configured work item transitions into the normalized `active` lifecycle bucket and start a bootstrap pull request workflow for the mapped repository.

#### Scenario: Linear issue enters a configured active state
- **WHEN** a Linear issue that was previously stored outside the `active` lifecycle bucket is observed in a configured active state during polling
- **THEN** Heimdall creates a workflow run for the activation-triggered bootstrap pull request flow
- **AND** Heimdall associates the workflow run with the normalized work item and the triggering transition event

### Requirement: Repository routing is explicit
Heimdall MUST resolve the target repository from explicit routing configuration and MUST fail safely when no routing rule matches a work item.

#### Scenario: No repository mapping matches the work item
- **WHEN** a work item enters the `active` lifecycle bucket and no repository mapping matches its configured team, project, label, or default route
- **THEN** Heimdall does not create a branch, worktree, or pull request
- **AND** Heimdall records a blocked or failed workflow state that explains the missing route

### Requirement: Repeated activation does not duplicate work
Heimdall MUST reconcile existing bindings before creating a new branch, worktree, or pull request for the same work item and repository.

#### Scenario: An active binding already exists
- **WHEN** a work item that already has an active repository binding is observed again in the `active` lifecycle bucket
- **THEN** Heimdall reuses the existing branch, worktree, and pull request binding
- **AND** Heimdall does not create a second open automation pull request for the same work item and repository
