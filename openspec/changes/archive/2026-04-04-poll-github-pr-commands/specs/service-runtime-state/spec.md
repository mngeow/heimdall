## MODIFIED Requirements

### Requirement: Symphony stores durable runtime state in SQLite
Symphony MUST persist workflow state, provider cursors, work item snapshots, repository bindings, pull request bindings, GitHub polling checkpoints, command requests, jobs, and audit records in SQLite for v1, and it MUST initialize or migrate the required schema before starting workflow processing.

#### Scenario: Symphony restarts after earlier workflow activity
- **WHEN** Symphony restarts on the same Linux host
- **THEN** it loads durable runtime state from SQLite
- **AND** it can resume Linear polling, GitHub polling, deduplication, reconciliation, and pull request command handling without reconstructing state from logs alone

#### Scenario: Symphony starts with an empty or outdated database
- **WHEN** Symphony opens its SQLite database and the required schema is missing or behind the expected version
- **THEN** it initializes or migrates the schema before starting poller, worker, or GitHub polling processing
- **AND** the resulting database layout matches the runtime state model expected by the service

### Requirement: Idempotency keys prevent duplicate processing
Symphony MUST persist idempotency keys for board transitions, repository mutations, and pull request command requests so repeated GitHub polling windows or retries do not repeat the same state change.

#### Scenario: The same GitHub command comment is observed in overlapping poll windows
- **WHEN** Symphony observes the same pull request command comment identity more than once across GitHub poll cycles
- **THEN** the runtime state marks the later observation as duplicate
- **AND** Symphony does not repeat the repository mutation associated with the original request
