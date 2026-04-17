## ADDED Requirements

### Requirement: Pending opencode permission requests are stored durably
Heimdall MUST persist blocked opencode permission request metadata for PR-comment-driven runs, including the permission request ID, the opencode session identity or resume handle, the originating command request, the pull request binding, and the current resolution status, so later approval commands and restarts can recover the waiting state safely.

#### Scenario: Heimdall restarts after a permission-blocked PR command
- **WHEN** Heimdall restarts after a PR-comment-driven run was blocked on permission request `perm_123`
- **THEN** it can still resolve `/heimdall approve perm_123` against the same persisted pending request
- **AND** it does not require the original in-memory poll cycle or worker state to still exist

#### Scenario: Pending permission request is resolved once
- **WHEN** Heimdall approves or otherwise closes a persisted pending permission request
- **THEN** it updates the stored status for that request ID so it is no longer pending
- **AND** a later duplicate approval command for the same request ID is reported as already resolved or rejected
