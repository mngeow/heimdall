## ADDED Requirements

### Requirement: Pull-request binding lookup stays tied to durable PR linkage
Heimdall MUST resolve active bindings for PR-comment execution from the persisted pull-request record and same-repository context. When `pull_requests.repo_binding_id` points to an active binding, Heimdall MUST use that exact binding. If a compatible legacy row lacks that direct linkage, Heimdall MUST fall back only to active bindings in the same repository and head branch, and it MUST NOT consider bindings from another repository solely because the branch name matches.

#### Scenario: Pull request uses its persisted repo binding link
- **WHEN** a Heimdall-managed pull request record has a `repo_binding_id` that points to an active binding
- **THEN** Heimdall resolves PR-command target changes from that exact binding
- **AND** it does not ignore the stored pull-request linkage in favor of a broader branch-name search

#### Scenario: Legacy pull request row falls back to the same repository and branch
- **WHEN** a Heimdall-managed pull request record lacks a usable direct binding link but exactly one active binding in the same repository matches the pull request head branch
- **THEN** Heimdall may use that same-repository binding as the PR-command context
- **AND** it does not require a manual repair before the command can continue

#### Scenario: Same branch name in another repository is ignored during fallback
- **WHEN** Heimdall falls back to repository-and-branch matching for PR-command binding resolution and another repository has an active binding with the same branch name
- **THEN** Heimdall ignores the binding from the other repository
- **AND** it resolves only bindings that belong to the pull request's own repository context

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
