# Service: Runtime State

## ADDED Requirements

### Requirement: Symphony stores durable runtime state in SQLite
Symphony MUST persist workflow state, provider cursors, work item snapshots, repository bindings, pull request bindings, command requests, jobs, and audit records in SQLite for v1, and it MUST initialize or migrate the required schema before starting workflow processing.

#### Scenario: Symphony restarts after earlier workflow activity
- **WHEN** Symphony restarts on the same Linux host
- **THEN** it loads durable runtime state from SQLite
- **AND** it can resume polling, deduplication, reconciliation, and pull request command handling without reconstructing state from logs alone

#### Scenario: Symphony starts with an empty or outdated database
- **WHEN** Symphony opens its SQLite database and the required schema is missing or behind the expected version
- **THEN** it initializes or migrates the schema before starting pollers, workers, or command-processing workflows
- **AND** the resulting database layout matches the runtime state model expected by the service

### Requirement: Runtime state enforces one active binding per work item and repository
Symphony MUST persist repository bindings that enforce a single active automation branch and pull request per work item and repository.

#### Scenario: Existing binding is present for the same work item and repository
- **WHEN** Symphony attempts to start a new proposal workflow for a work item that already has an active binding in the same repository
- **THEN** it reuses the existing binding instead of creating a second active branch and pull request pair
- **AND** the runtime state continues to represent one active automation binding for that work item and repository

### Requirement: Idempotency keys prevent duplicate processing
Symphony MUST persist idempotency keys for board transitions, repository mutations, and pull request command requests so duplicate deliveries do not repeat the same state change.

#### Scenario: The same GitHub comment event is delivered twice
- **WHEN** Symphony receives the same command request identity more than once
- **THEN** the runtime state marks the later delivery as duplicate
- **AND** Symphony does not repeat the repository mutation associated with the original request

### Requirement: Jobs and workflow steps support retries and reconciliation
Symphony MUST persist queued jobs, workflow steps, retry counts, and lock keys so long-running work can be retried and reconciled safely.

#### Scenario: A transient external failure interrupts a workflow step
- **WHEN** a retryable step fails because of a temporary external error such as rate limiting or a network timeout
- **THEN** Symphony records the failed attempt on the workflow step and queued job
- **AND** it schedules a retry without losing the workflow run's prior state

### Requirement: Secrets are excluded from SQLite runtime state
Symphony MUST NOT store GitHub App private keys, installation tokens, Linear API keys, or other raw secret material in SQLite.

#### Scenario: Symphony persists runtime state after authentication
- **WHEN** Symphony stores workflow and integration state in SQLite
- **THEN** it stores references and operational metadata only
- **AND** it excludes raw secret material from the database
