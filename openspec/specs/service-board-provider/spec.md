# Service: Board Provider

## ADDED Requirements

### Requirement: Board providers emit normalized work item models
The board integration layer MUST normalize provider-specific issue data into provider-neutral work item and work item event models for the core workflow engine, and the Linear adapter MUST source that normalized model from the Linear GraphQL issue fields required for polling.

#### Scenario: Linear issue is loaded for workflow execution
- **WHEN** the Linear adapter fetches an issue from the Linear GraphQL API
- **THEN** it maps GraphQL issue fields such as identifier, title, description, updatedAt, state, team, project, and labels into the normalized work item model
- **AND** the core workflow engine does not need to depend on raw Linear field names or GraphQL response shapes to act on the issue

### Requirement: Linear polling uses durable cursors and state snapshots
The Linear provider MUST poll the official Linear GraphQL `issues` connection, page through results with Relay-style cursors, filter to the explicitly configured Linear project scope, order and filter by issue update time, persist the last successful polling checkpoint, compare the current issue state with the last stored snapshot before emitting activation events, and in v1 MUST detect those activation events through polling without requiring inbound Linear webhooks.

#### Scenario: Poll cycle sees a recently updated issue
- **WHEN** Heimdall polls Linear for issues updated since the previous successful checkpoint in the configured Linear project scope
- **THEN** it compares each issue's current lifecycle bucket to the last stored snapshot
- **AND** it emits a normalized transition event only when the issue newly enters the configured active lifecycle bucket

#### Scenario: Poll cycle spans multiple result pages
- **WHEN** a Linear poll request returns more than one page of matching issues
- **THEN** Heimdall continues polling with the returned page cursor until all pages are consumed
- **AND** it advances the durable checkpoint only after the full poll cycle succeeds

#### Scenario: No Linear webhook is configured
- **WHEN** Heimdall runs in v1 with Linear polling enabled and no Linear webhook endpoint configured
- **THEN** it continues detecting activation events through polling and stored snapshots
- **AND** the board-triggered workflow path does not depend on inbound Linear webhook delivery

### Requirement: Board-provider authentication is secret-backed and scoped
The board-provider service MUST use secret-backed credentials and MUST limit polling to explicitly configured board scopes. For Linear v1, the provider MUST authenticate each poll request with a static API key using the `Authorization: <API_KEY>` header and MUST limit polling to the configured project-name scope.

#### Scenario: Heimdall starts with Linear integration enabled
- **WHEN** the Linear board provider is initialized
- **THEN** it reads its API key from a secret-backed source instead of repository files
- **AND** it uses that API key against `https://api.linear.app/graphql` while limiting polls to the configured project-name scope

#### Scenario: Linear API key is invalid
- **WHEN** a Linear poll request receives an authentication failure from the Linear API
- **THEN** Heimdall records the poll cycle as failed
- **AND** it does not advance the durable poll checkpoint

## ADDED Requirements

### Requirement: Linear GraphQL poll cycles handle API errors and rate limits safely
Heimdall MUST treat Linear GraphQL errors, authentication failures, and rate limits as unsuccessful poll cycles, MUST inspect relevant rate-limit metadata when present, and MUST NOT advance the durable checkpoint after a failed or partial cycle.

#### Scenario: Linear returns a rate-limited response
- **WHEN** the Linear API rejects a poll cycle because the static API key has exceeded a request or complexity limit
- **THEN** Heimdall treats the poll cycle as retryable
- **AND** it leaves the last successful checkpoint unchanged

#### Scenario: GraphQL response includes errors
- **WHEN** the Linear API returns a GraphQL response that includes an `errors` array for the poll query
- **THEN** Heimdall treats the poll cycle as failed instead of trusting partial data
- **AND** it does not create new activation events from that partial response

### Requirement: Provider-specific semantics remain outside the core workflow engine
The board-provider service MUST keep provider-specific state names and transition semantics inside provider adapters so additional board systems can be added without rewriting the core workflow engine.

#### Scenario: A future provider defines a different active state name
- **WHEN** a provider uses a state name other than `In Progress` to represent active work
- **THEN** the provider adapter maps that provider-specific state into the normalized `active` lifecycle bucket
- **AND** the core workflow engine continues to consume normalized transitions without provider-specific branching logic
