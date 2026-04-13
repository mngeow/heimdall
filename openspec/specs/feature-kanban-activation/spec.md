# Feature: Kanban Activation

## ADDED Requirements

### Requirement: Active state transitions start bootstrap workflows
Heimdall MUST detect when a configured work item transitions into the normalized `active` lifecycle bucket and start an activation-triggered OpenSpec proposal pull request workflow for the mapped repository.

#### Scenario: Linear issue enters a configured active state
- **WHEN** a Linear issue that was previously stored outside the `active` lifecycle bucket is observed in a configured active state during polling
- **THEN** Heimdall creates a workflow run for the activation-triggered OpenSpec proposal pull request flow
- **AND** Heimdall associates the workflow run with the normalized work item and the triggering transition event

### Requirement: Repository routing is explicit
Heimdall MUST resolve the target repository from explicit routing configuration and MUST fail safely when no routing rule matches a work item.

#### Scenario: No repository mapping matches the work item
- **WHEN** a work item enters the `active` lifecycle bucket and no repository mapping matches its configured team, project, label, or default route
- **THEN** Heimdall does not create a branch, worktree, or pull request
- **AND** Heimdall records a blocked or failed workflow state that explains the missing route

### Requirement: Repeated activation does not duplicate work
Heimdall MUST reconcile existing bindings and prior activation workflow state before creating a new branch, worktree, OpenSpec change, or pull request for the same work item and repository.

#### Scenario: An active binding already exists
- **WHEN** a work item that already has an active repository binding is observed again in the `active` lifecycle bucket
- **THEN** Heimdall reuses the existing branch, worktree, OpenSpec change, and pull request binding
- **AND** Heimdall does not create a second open automation pull request or a second active change for the same work item and repository

#### Scenario: A prior activation run failed before an active binding was saved
- **WHEN** the same work item is observed again in the `active` lifecycle bucket
- **AND** a previous activation attempt for that repository already created or registered the deterministic branch or worktree but did not finish far enough to save an active binding
- **THEN** Heimdall reconciles and reuses or repairs that deterministic proposal workspace on retry instead of starting a conflicting second setup
- **AND** it does not fail the retry solely because the deterministic branch or worktree was already registered by git
