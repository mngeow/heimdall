## REMOVED Requirements

### Requirement: Symphony stores durable runtime state in SQLite
**Reason**: The service and storage identity is being renamed from Symphony to Heimdall.
**Migration**: Rename runtime-state references and documented default storage naming to Heimdall while preserving the same SQLite-backed durability model.

## ADDED Requirements

### Requirement: Heimdall stores durable runtime state in SQLite
Heimdall MUST persist workflow state, provider cursors, work item snapshots, repository bindings, pull request bindings, GitHub polling checkpoints, command requests, jobs, and audit records in SQLite for v1, and it MUST initialize or migrate the required schema before starting workflow processing.

#### Scenario: Heimdall restarts after earlier workflow activity
- **WHEN** Heimdall restarts on the same Linux host
- **THEN** it loads durable runtime state from SQLite
- **AND** it can resume Linear polling, GitHub polling, deduplication, reconciliation, and pull request command handling without reconstructing state from logs alone

#### Scenario: Heimdall starts with an empty or outdated database
- **WHEN** Heimdall opens its SQLite database and the required schema is missing or behind the expected version
- **THEN** it initializes or migrates the schema before starting poller, worker, or GitHub polling processing
- **AND** the resulting database layout matches the runtime state model expected by the service

## MODIFIED Requirements

### Requirement: Runtime state enforces one active binding per work item and repository
Heimdall MUST persist repository bindings that enforce a single active automation branch and pull request per work item and repository.

#### Scenario: Existing binding is present for the same work item and repository
- **WHEN** Heimdall attempts to start a new proposal workflow for a work item that already has an active binding in the same repository
- **THEN** it reuses the existing binding instead of creating a second active branch and pull request pair
- **AND** the runtime state continues to represent one active automation binding for that work item and repository

### Requirement: Idempotency keys prevent duplicate processing
Heimdall MUST persist idempotency keys for board transitions, repository mutations, and pull request command requests so repeated GitHub polling windows or retries do not repeat the same state change.

#### Scenario: The same GitHub command comment is observed in overlapping poll windows
- **WHEN** Heimdall observes the same pull request command comment identity more than once across GitHub poll cycles
- **THEN** the runtime state marks the later observation as duplicate
- **AND** Heimdall does not repeat the repository mutation associated with the original request

### Requirement: Jobs and workflow steps support retries and reconciliation
Heimdall MUST persist queued jobs, workflow steps, retry counts, and lock keys so long-running work can be retried and reconciled safely.

#### Scenario: A transient external failure interrupts a workflow step
- **WHEN** a retryable step fails because of a temporary external error such as rate limiting or a network timeout
- **THEN** Heimdall records the failed attempt on the workflow step and queued job
- **AND** it schedules a retry without losing the workflow run's prior state

### Requirement: Secrets are excluded from SQLite runtime state
Heimdall MUST NOT store GitHub App private keys, webhook secrets, installation tokens, or Linear API keys in SQLite.

#### Scenario: Heimdall persists runtime state after authentication
- **WHEN** Heimdall stores workflow and integration state in SQLite
- **THEN** it stores references and operational metadata only
- **AND** it excludes raw secret material from the database
