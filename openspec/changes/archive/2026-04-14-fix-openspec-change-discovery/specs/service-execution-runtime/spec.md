## MODIFIED Requirements

### Requirement: OpenSpec CLI JSON output controls workflow decisions
Activation-triggered proposal, refine, apply, and archive flows MUST use OpenSpec CLI JSON output for change status, change lists, artifact instructions, and apply readiness, and Heimdall MUST parse the actual response shape returned by the CLI rather than assuming a simplified structure.

#### Scenario: Activation proposal discovers changes through OpenSpec list output
- **WHEN** Heimdall lists changes before or after activation proposal generation
- **THEN** it parses the JSON object returned by `openspec list --json`
- **AND** it uses the named changes from that response to determine which change the workflow should inspect next

#### Scenario: Activation proposal reads apply instructions as a readiness check
- **WHEN** Heimdall requests `openspec instructions apply --change <name> --json` after proposal generation
- **THEN** it parses the machine-readable apply-instructions payload returned by the CLI
- **AND** it keeps the readiness-check step in the workflow instead of skipping it

#### Scenario: Apply workflow is requested from a pull request comment
- **WHEN** Heimdall prepares to run `/opsx-apply` for an OpenSpec change
- **THEN** it reads OpenSpec apply instructions and current status from CLI JSON output
- **AND** it does not guess task readiness or context file selection from filesystem conventions alone

### Requirement: Activation proposal runs from a worktree created off the configured mirror
Heimdall MUST create the activation proposal worktree from the repository mirror configured by the resolved repository's local mirror path before invoking OpenSpec and OpenCode, and it MUST execute activation proposal OpenSpec discovery and apply-instruction commands in that worktree while reconciling stale git worktree registrations that would otherwise block deterministic retry paths.

#### Scenario: Proposal worktree is created
- **WHEN** an activated work item starts the proposal pull request workflow for the resolved repository
- **THEN** Heimdall uses that repository's configured local mirror path as the git source for the new worktree
- **AND** it runs proposal generation, OpenSpec change discovery, and apply-instruction lookup inside that worktree rather than the bare mirror or Heimdall process cwd

#### Scenario: Git still tracks a stale worktree registration from a prior failed run
- **WHEN** Heimdall prepares the deterministic proposal worktree path and branch for an activation retry
- **AND** git still reports that branch or worktree path as registered even though the prior worktree location is missing or otherwise prunable
- **THEN** Heimdall prunes or removes the stale git worktree registration before retrying worktree creation
- **AND** it does not require manual operator cleanup just to recreate the deterministic proposal worktree
