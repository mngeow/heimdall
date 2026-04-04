# Service: Board Provider

## ADDED Requirements

### Requirement: Board providers emit normalized work item models
The board integration layer MUST normalize provider-specific issue data into provider-neutral work item and work item event models for the core workflow engine.

#### Scenario: Linear issue is loaded for workflow execution
- **WHEN** the Linear adapter fetches a work item for Symphony
- **THEN** it returns normalized work item data such as key, title, description, state, project, team, labels, and repository reference inputs
- **AND** the core workflow engine does not need to depend on raw Linear field names to act on the issue

### Requirement: Linear polling uses durable cursors and state snapshots
The Linear provider MUST persist polling cursor state, compare the current issue state with the last stored snapshot before emitting activation events, and in v1 MUST detect those activation events through polling without requiring inbound Linear webhooks.

#### Scenario: Poll cycle sees a recently updated issue
- **WHEN** Symphony polls Linear for recently updated issues in a configured scope
- **THEN** it compares each issue's current lifecycle bucket to the last stored snapshot
- **AND** it emits a normalized transition event only when the issue newly enters the configured active lifecycle bucket

#### Scenario: No Linear webhook is configured
- **WHEN** Symphony runs in v1 with Linear polling enabled and no Linear webhook endpoint configured
- **THEN** it continues detecting activation events through polling and stored snapshots
- **AND** the board-triggered workflow path does not depend on inbound Linear webhook delivery

### Requirement: Board-provider authentication is secret-backed and scoped
The board-provider service MUST use secret-backed credentials and MUST limit polling to explicitly configured board scopes.

#### Scenario: Symphony starts with Linear integration enabled
- **WHEN** the Linear board provider is initialized
- **THEN** it reads its API key from a secret-backed source instead of repository files
- **AND** it limits polling to the configured teams, projects, labels, or equivalent scopes

### Requirement: Provider-specific semantics remain outside the core workflow engine
The board-provider service MUST keep provider-specific state names and transition semantics inside provider adapters so additional board systems can be added without rewriting the core workflow engine.

#### Scenario: A future provider defines a different active state name
- **WHEN** a provider uses a state name other than `In Progress` to represent active work
- **THEN** the provider adapter maps that provider-specific state into the normalized `active` lifecycle bucket
- **AND** the core workflow engine continues to consume normalized transitions without provider-specific branching logic
