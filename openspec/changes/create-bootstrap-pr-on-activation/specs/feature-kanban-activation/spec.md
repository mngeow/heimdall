## MODIFIED Requirements

### Requirement: Active state transitions start bootstrap workflows
Symphony MUST detect when a configured work item transitions into the normalized `active` lifecycle bucket and start a bootstrap pull request workflow for the mapped repository.

#### Scenario: Linear issue enters a configured active state
- **WHEN** a Linear issue that was previously stored outside the `active` lifecycle bucket is observed in a configured active state during polling
- **THEN** Symphony creates a workflow run for the activation-triggered bootstrap pull request flow
- **AND** Symphony associates the workflow run with the normalized work item and the triggering transition event

### Requirement: Repository routing is explicit
Symphony MUST resolve the target repository from explicit routing configuration and MUST fail safely when no routing rule matches a work item.

#### Scenario: No repository mapping matches the work item
- **WHEN** a work item enters the `active` lifecycle bucket and no repository mapping matches its configured team, project, label, or default route
- **THEN** Symphony does not create a branch, worktree, or pull request
- **AND** Symphony records a blocked or failed workflow state that explains the missing route

### Requirement: Repeated activation does not duplicate work
Symphony MUST reconcile existing bindings before creating a new branch, worktree, or pull request for the same work item and repository.

#### Scenario: An active binding already exists
- **WHEN** a work item that already has an active repository binding is observed again in the `active` lifecycle bucket
- **THEN** Symphony reuses the existing branch, worktree, and pull request binding
- **AND** Symphony does not create a second open automation pull request for the same work item and repository
