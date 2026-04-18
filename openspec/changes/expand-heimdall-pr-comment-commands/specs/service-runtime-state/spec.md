## ADDED Requirements

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
