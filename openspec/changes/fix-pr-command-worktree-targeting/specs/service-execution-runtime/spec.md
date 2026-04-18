## ADDED Requirements

### Requirement: PR-comment commands use a canonical prepared worktree
Heimdall MUST derive the worktree path for `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, and `/heimdall opencode` from the managed repository's configured local mirror path and the pull request head branch, MUST materialize or refresh that worktree before OpenSpec target validation, and MUST run OpenSpec inspection and opencode execution against that same prepared directory.

#### Scenario: PR command derives its worktree from the managed repository mirror
- **WHEN** Heimdall begins an agent-driven PR command for a managed pull request
- **THEN** it derives the command worktree path from the repository's configured local mirror path and the pull request head branch
- **AND** it does not switch to a hardcoded temporary root unrelated to that repository's documented worktree layout

#### Scenario: Change validation runs after worktree preparation
- **WHEN** Heimdall is about to validate whether the resolved change still exists for an agent-driven PR command
- **THEN** it has already fetched the repository mirror and prepared or refreshed the command worktree for that pull request
- **AND** the validation result reflects the repository state that the command will actually execute against

#### Scenario: OpenSpec and opencode share the same prepared worktree
- **WHEN** Heimdall validates and then executes an agent-driven PR command
- **THEN** the OpenSpec client and opencode execution path both use the same prepared pull-request worktree directory
- **AND** Heimdall does not validate against one checkout and execute against another

### Requirement: Existing PR branches are materialized from the fetched branch ref
When Heimdall prepares a worktree for an already-existing proposal or pull-request branch, it MUST use the fetched branch ref from the local mirror as the worktree source. Heimdall MUST use the repository default branch only when it is creating a brand-new automation branch that does not yet exist in the mirror.

#### Scenario: Existing PR branch is recreated for a PR command
- **WHEN** Heimdall prepares the local worktree for a pull request whose head branch already exists in the fetched mirror
- **THEN** it materializes that worktree from the fetched head-branch ref
- **AND** it does not reseed that worktree from the repository default branch

#### Scenario: New automation branch still seeds from the default branch
- **WHEN** Heimdall prepares a worktree for a brand-new automation branch that does not yet exist in the mirror
- **THEN** it may seed that new branch from the repository default branch
- **AND** proposal bootstrapping continues to work for new changes

### Requirement: Opencode JSON event parsing tolerates large newline-delimited events
Heimdall MUST consume `opencode run --format json` output as a newline-delimited JSON event stream without relying on reader token limits that reject otherwise valid single-event lines. Heimdall MUST continue classifying permission, input, error, and completion outcomes even when an individual event line contains a very large text or tool payload, MAY ignore non-JSON noise lines, and MUST still process a final valid JSON event line that ends at EOF without a trailing newline.

#### Scenario: Large text event line does not abort parsing
- **WHEN** Heimdall is reading `opencode run --format json` output for a PR-comment refine or apply run and a valid `text` event line contains a payload large enough to exceed traditional scanner token thresholds
- **THEN** Heimdall continues consuming the event stream
- **AND** it does not fail the command only because its local event reader rejected that large line

#### Scenario: Later structured events are still classified after a large event
- **WHEN** a large valid JSON event line is followed by later structured events such as `permission.asked`, `tool_use`, or `step_finish`
- **THEN** Heimdall still observes those later events
- **AND** it classifies the execution outcome from the structured event stream instead of aborting early with a token-length reader error

#### Scenario: Final event at EOF is still processed
- **WHEN** the opencode process exits after writing a final valid JSON event line without a trailing newline
- **THEN** Heimdall still parses that final event
- **AND** it does not drop the terminal outcome solely because the stream ended at EOF
