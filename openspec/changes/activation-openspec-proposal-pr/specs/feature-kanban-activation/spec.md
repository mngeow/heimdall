## MODIFIED Requirements

### Requirement: Active state transitions start bootstrap workflows
Heimdall MUST detect when a configured work item transitions into the normalized `active` lifecycle bucket and start an activation-triggered OpenSpec proposal pull request workflow for the mapped repository.

#### Scenario: Linear issue enters a configured active state
- **WHEN** a Linear issue that was previously stored outside the `active` lifecycle bucket is observed in a configured active state during polling
- **THEN** Heimdall creates a workflow run for the activation-triggered OpenSpec proposal pull request flow
- **AND** Heimdall associates the workflow run with the normalized work item and the triggering transition event

### Requirement: Repeated activation does not duplicate work
Heimdall MUST reconcile existing bindings before creating a new branch, worktree, OpenSpec change, or pull request for the same work item and repository.

#### Scenario: An active binding already exists
- **WHEN** a work item that already has an active repository binding is observed again in the `active` lifecycle bucket
- **THEN** Heimdall reuses the existing branch, worktree, OpenSpec change, and pull request binding
- **AND** Heimdall does not create a second open automation pull request or a second active change for the same work item and repository
