## MODIFIED Requirements

### Requirement: Linear polling uses durable cursors and state snapshots
The Linear provider MUST persist polling cursor state, compare the current issue state with the last stored snapshot before emitting activation events, and in v1 MUST detect those activation events through polling without requiring inbound Linear webhooks.

#### Scenario: Poll cycle sees a recently updated issue
- **WHEN** Heimdall polls Linear for recently updated issues in a configured scope
- **THEN** it compares each issue's current lifecycle bucket to the last stored snapshot
- **AND** it emits a normalized transition event only when the issue newly enters the configured active lifecycle bucket

#### Scenario: No Linear webhook is configured
- **WHEN** Heimdall runs in v1 with Linear polling enabled and no Linear webhook endpoint configured
- **THEN** it continues detecting activation events through polling and stored snapshots
- **AND** the board-triggered workflow path does not depend on inbound Linear webhook delivery
