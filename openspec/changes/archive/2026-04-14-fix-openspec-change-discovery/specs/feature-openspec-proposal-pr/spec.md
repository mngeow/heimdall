## ADDED Requirements

### Requirement: Activation proposal discovers the generated change from OpenSpec CLI output
Heimdall MUST compare OpenSpec change listings from the proposal worktree before and after activation proposal generation to determine which change was created by the run, and it MUST use that discovered change name for later apply-instruction lookup and binding persistence.

#### Scenario: Proposal generation creates a new OpenSpec change
- **WHEN** Heimdall finishes an activation proposal generation run in the target repository worktree
- **AND** the before/after `openspec list --json` results differ by one newly created change
- **THEN** Heimdall binds the workflow to that discovered change name
- **AND** it requests apply instructions for that discovered change before continuing proposal publication

### Requirement: Activation proposal keeps the post-generation readiness check
Heimdall MUST continue to request apply instructions for the discovered change after proposal generation completes, and it MUST only treat that step as a read-only readiness check before proposal publication continues.

#### Scenario: Proposal generation reaches the readiness check
- **WHEN** Heimdall has already discovered the newly created OpenSpec change for an activation proposal run
- **THEN** it requests apply instructions for that change as a readiness check
- **AND** it does not interpret that step as implementation or task execution
