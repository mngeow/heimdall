# Service: Runtime State

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

### Requirement: Pending opencode permission requests are stored durably
Heimdall MUST persist blocked opencode permission request metadata for PR-comment-driven runs only when it has a real machine-readable permission event with non-empty identifiers. The persisted record MUST include the exact permission request ID, the opencode session identity or resume handle, the originating command request, the pull request binding, and the current resolution status, so later approval commands and restarts can recover the waiting state safely.

#### Scenario: Heimdall restarts after a permission-blocked PR command
- **WHEN** Heimdall restarts after a PR-comment-driven run was blocked on permission request `perm_123`
- **THEN** it can still resolve `/heimdall approve perm_123` against the same persisted pending request
- **AND** it does not require the original in-memory poll cycle or worker state to still exist

#### Scenario: Pending permission request is resolved once
- **WHEN** Heimdall approves or otherwise closes a persisted pending permission request
- **THEN** it updates the stored status for that request ID so it is no longer pending
- **AND** a later duplicate approval command for the same request ID is reported as already resolved or rejected

#### Scenario: Blank permission identifiers are never persisted
- **WHEN** Heimdall encounters CLI help text, a generic execution error, or any other failed run without a real permission event carrying non-empty request and session identifiers
- **THEN** it does not create a pending permission-request row for that failed run
- **AND** it does not publish an approval command that points at an empty or synthetic request ID

#### Scenario: Pending permission request keeps its originating command linkage
- **WHEN** Heimdall persists a real blocked permission request for a PR-comment-driven run
- **THEN** the stored pending request remains linked to the originating command request and same session identity
- **AND** a later `/heimdall approve <request-id>` can recover the exact blocked run context without guessing

### Requirement: PR-comment opencode session identities are stored durably
Heimdall MUST persist the exact `sessionID` observed in the first structured opencode event for every PR-comment refine, apply, or generic opencode run. The stored session identity MUST remain linked to the originating command request, pull request, and any later pending permission request or retry metadata so restarts and operator debugging do not rely on logs alone.

#### Scenario: First observed session ID is stored with the command execution state
- **WHEN** Heimdall starts a PR-comment opencode run and the first structured event reports `sessionID` `ses_abc`
- **THEN** Heimdall persists `ses_abc` as part of that run's durable execution state
- **AND** later worker retries or inspection paths can recover the same observed session identity from runtime state

#### Scenario: Pending permission request reuses the same persisted session ID
- **WHEN** a PR-comment run later blocks on permission after Heimdall already observed and stored `sessionID` `ses_abc`
- **THEN** the pending permission request remains linked to the same stored session identity `ses_abc`
- **AND** a later approval command can resume the exact blocked session without guessing or synthesizing a new session handle

#### Scenario: Session identity is never synthesized from missing data
- **WHEN** Heimdall does not observe a real structured opencode event carrying a session identity for a PR-comment run
- **THEN** it does not invent or persist a synthetic `sessionID`
- **AND** it records that the real session identity was unavailable instead of storing guessed runtime state
