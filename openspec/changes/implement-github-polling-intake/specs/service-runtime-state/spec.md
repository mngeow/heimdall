## MODIFIED Requirements

### Requirement: Symphony stores durable runtime state in SQLite
Symphony MUST persist workflow state, provider cursors including GitHub polling checkpoints, work item snapshots, repository bindings, pull request bindings, command requests, jobs, and audit records in SQLite for v1, and it MUST initialize or migrate the required schema before starting workflow processing.

#### Scenario: Symphony restarts after earlier workflow activity
- **WHEN** Symphony restarts on the same Linux host
- **THEN** it loads durable runtime state from SQLite, including GitHub polling checkpoints and prior command-request records
- **AND** it can resume polling, deduplication, reconciliation, and pull request command handling without reconstructing state from logs alone

#### Scenario: Symphony starts with an empty or outdated database
- **WHEN** Symphony opens its SQLite database and the required schema is missing or behind the expected version
- **THEN** it initializes or migrates the schema before starting pollers, workers, or command-processing workflows
- **AND** the resulting database layout includes the state needed for GitHub polling checkpoints and command-request persistence

### Requirement: Idempotency keys prevent duplicate processing
Symphony MUST persist idempotency keys for board transitions, repository mutations, and pull request command requests keyed by stable GitHub comment identity so duplicate poll observations or overlapping polling windows do not repeat the same state change.

#### Scenario: The same GitHub comment appears in overlapping poll windows
- **WHEN** two GitHub poll cycles observe the same pull request comment identity for a Symphony-managed pull request
- **THEN** the runtime state marks the later observation as a duplicate command request
- **AND** Symphony does not repeat the repository mutation associated with the original request

#### Scenario: Initial polling backfill encounters a previously stored command request
- **WHEN** Symphony performs an initial poll for a bound pull request and encounters a comment identity that is already stored in runtime state
- **THEN** it does not enqueue a second command request for that comment
- **AND** it can still advance the checkpoint needed for later incremental polling
