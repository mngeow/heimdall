# Service: Execution Runtime

## ADDED Requirements

### Requirement: OpenSpec and OpenCode execution runs locally on the host
Heimdall MUST run activation-triggered OpenSpec proposal generation through the local `openspec` and `opencode` CLIs on the Linux host where Heimdall is running, and it MUST verify that required local executables such as `git`, `openspec`, and `opencode` are available before the service reports readiness.

#### Scenario: Activation proposal is requested for an activated work item
- **WHEN** Heimdall begins an activation-triggered proposal workflow for a routed work item
- **THEN** it invokes the local `openspec` and `opencode` tooling available on the host by using the repository's configured default spec-writing agent
- **AND** it performs that execution inside the repository worktree created for the workflow run

#### Scenario: Required executable is missing at startup
- **WHEN** Heimdall starts and one of `git`, `openspec`, or `opencode` is unavailable on the configured executable path
- **THEN** the service does not report ready for workflow execution
- **AND** it records an operator-visible startup failure that identifies the missing executable

### Requirement: OpenSpec CLI JSON output controls workflow decisions
Activation-triggered proposal, refine, apply, and archive flows MUST use OpenSpec CLI JSON output for change status, artifact instructions, and apply readiness, and Heimdall MUST NOT guess artifact order or readiness from filesystem conventions alone.

#### Scenario: Activation proposal is prepared for execution
- **WHEN** Heimdall prepares the activation-triggered proposal workflow
- **THEN** it creates or reuses the target change and reads OpenSpec CLI status and artifact instructions
- **AND** it uses that CLI output to decide which artifacts must be generated before the proposal branch is committed

#### Scenario: Apply workflow is requested from a pull request comment
- **WHEN** Heimdall prepares to run `/opsx-apply` for an OpenSpec change
- **THEN** it reads OpenSpec apply instructions and current status from CLI JSON output
- **AND** it does not guess task readiness or context file selection from filesystem conventions alone

### Requirement: Agent selection is explicit and policy-controlled
The execution runtime MUST use the repository's configured default spec-writing agent for activation-triggered proposal and refine operations, and MUST require an explicitly selected allowlisted agent for apply operations.

#### Scenario: Activation proposal is started
- **WHEN** an activated work item starts the proposal pull request workflow
- **THEN** Heimdall runs the local OpenCode execution by using the repository's configured default spec-writing agent
- **AND** it does not require per-run agent input for that activation path

#### Scenario: User runs apply without an allowed agent
- **WHEN** a pull request comment requests `/opsx-apply` with an agent that is not allowed for the repository
- **THEN** Heimdall does not start the apply execution
- **AND** it records and reports that the requested agent is not authorized for that repository

### Requirement: Execution metadata is auditable
The execution runtime MUST record the command, executor, and version details needed to audit and troubleshoot proposal, refine, apply, archive, and activation-triggered proposal steps.

#### Scenario: Heimdall runs an OpenSpec or OpenCode step
- **WHEN** Heimdall executes a workflow step through `openspec`, `opencode`, `git`, or GitHub API-backed repository mutation logic
- **THEN** it records the step outcome and the executor details needed for audit and recovery
- **AND** those records are linked to the workflow run they belong to

### Requirement: Activation proposal runs from a worktree created off the configured mirror
Heimdall MUST create the activation proposal worktree from the repository mirror configured by the resolved repository's local mirror path before invoking OpenSpec and OpenCode, and it MUST reconcile stale git worktree registrations that would otherwise block deterministic retry paths.

#### Scenario: Proposal worktree is created
- **WHEN** an activated work item starts the proposal pull request workflow for the resolved repository
- **THEN** Heimdall uses that repository's configured local mirror path as the git source for the new worktree
- **AND** it runs the proposal execution inside that worktree rather than the bare mirror itself

#### Scenario: Git still tracks a stale worktree registration from a prior failed run
- **WHEN** Heimdall prepares the deterministic proposal worktree path and branch for an activation retry
- **AND** git still reports that branch or worktree path as registered even though the prior worktree location is missing or otherwise prunable
- **THEN** Heimdall prunes or removes the stale git worktree registration before retrying worktree creation
- **AND** it does not require manual operator cleanup just to recreate the deterministic proposal worktree

### Requirement: Activation proposal generation fails visibly when no commit-ready artifacts are produced
Heimdall MUST fail the activation proposal workflow when the proposal-generation execution completes without producing repository changes for the target OpenSpec change that can be committed.

#### Scenario: Proposal generation produces no file changes
- **WHEN** the activation-triggered proposal run exits without leaving any modified, added, or deleted repository files for the target change
- **THEN** Heimdall does not create an empty commit, branch push, or pull request
- **AND** it records the workflow run as failed or blocked with a visible reason
