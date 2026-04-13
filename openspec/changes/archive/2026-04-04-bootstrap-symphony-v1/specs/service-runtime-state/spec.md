## MODIFIED Requirements

### Requirement: Heimdall stores durable runtime state in SQLite
Heimdall MUST persist workflow state, provider cursors, work item snapshots, repository bindings, pull request bindings, command requests, jobs, and audit records in SQLite for v1, and it MUST initialize or migrate the required schema before starting workflow processing.

#### Scenario: Heimdall restarts after earlier workflow activity
- **WHEN** Heimdall restarts on the same Linux host
- **THEN** it loads durable runtime state from SQLite
- **AND** it can resume polling, deduplication, reconciliation, and pull request command handling without reconstructing state from logs alone

#### Scenario: Heimdall starts with an empty or outdated database
- **WHEN** Heimdall opens its SQLite database and the required schema is missing or behind the expected version
- **THEN** it initializes or migrates the schema before starting poller, worker, or webhook-driven workflow processing
- **AND** the resulting database layout matches the runtime state model expected by the service
