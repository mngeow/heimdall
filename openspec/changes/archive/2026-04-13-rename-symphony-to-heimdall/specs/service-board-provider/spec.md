## MODIFIED Requirements

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
