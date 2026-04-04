## MODIFIED Requirements

### Requirement: GitHub webhooks are verified before processing
Symphony MUST verify the GitHub webhook signature before parsing or acting on incoming webhook payloads and MUST support at minimum the `issue_comment` event for pull request command handling and the `pull_request` event for pull request lifecycle reconciliation.

#### Scenario: GitHub sends an issue comment webhook
- **WHEN** Symphony receives a GitHub `issue_comment` webhook delivery
- **THEN** it verifies the delivery against the configured webhook secret before command parsing occurs
- **AND** it rejects the request if signature verification fails

#### Scenario: GitHub sends a pull request webhook
- **WHEN** Symphony receives a GitHub `pull_request` webhook delivery for a repository managed by Symphony
- **THEN** it verifies the delivery against the configured webhook secret before reconciliation logic runs
- **AND** it makes that pull request event available to the runtime components responsible for binding and lifecycle synchronization
