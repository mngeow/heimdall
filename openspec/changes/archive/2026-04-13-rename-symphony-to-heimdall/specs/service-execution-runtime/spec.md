## MODIFIED Requirements

### Requirement: OpenSpec and OpenCode execution runs locally on the host
Heimdall MUST run activation-triggered bootstrap execution through the local `opencode` CLI on the Linux host where Heimdall is running, and it MUST verify that required local executables such as `git`, `openspec`, and `opencode` are available before the service reports readiness.

#### Scenario: Activation bootstrap is requested for an activated work item
- **WHEN** Heimdall begins an activation-triggered bootstrap workflow for a routed work item
- **THEN** it invokes the local `opencode` tooling available on the host by using the general agent and model `gpt-5.4`
- **AND** it performs that execution inside the repository worktree created for the workflow run

#### Scenario: Required executable is missing at startup
- **WHEN** Heimdall starts and one of `git`, `openspec`, or `opencode` is unavailable on the configured executable path
- **THEN** the service does not report ready for workflow execution
- **AND** it records an operator-visible startup failure that identifies the missing executable

### Requirement: OpenSpec CLI JSON output controls workflow decisions
The initial activation-triggered bootstrap pull request flow MUST NOT require OpenSpec CLI JSON output, although refine, apply, and archive flows continue to use it.

#### Scenario: Activation bootstrap is prepared for execution
- **WHEN** Heimdall prepares the activation-triggered bootstrap workflow
- **THEN** it does not block on OpenSpec change discovery, artifact dependencies, or apply readiness
- **AND** it executes the bootstrap workflow from the issue seed data and repository state alone

#### Scenario: Apply workflow is requested from a pull request comment
- **WHEN** Heimdall prepares to run `/opsx-apply` for an OpenSpec change
- **THEN** it reads OpenSpec apply instructions and current status from CLI JSON output
- **AND** it does not guess task readiness or context file selection from filesystem conventions alone

### Requirement: Agent selection is explicit and policy-controlled
The execution runtime MUST use the fixed OpenCode bootstrap profile of the general agent with model `gpt-5.4` for activation-triggered workflows, MUST continue to use the repository's default spec-writing agent for refine operations, and MUST require an explicitly selected allowlisted agent for apply operations.

#### Scenario: Activation bootstrap is started
- **WHEN** an activated work item starts the bootstrap pull request workflow
- **THEN** Heimdall runs the local OpenCode execution by using the general agent and model `gpt-5.4`
- **AND** it does not require the operator to select an agent for that activation path

#### Scenario: User runs apply without an allowed agent
- **WHEN** a pull request comment requests `/opsx-apply` with an agent that is not allowed for the repository
- **THEN** Heimdall does not start the apply execution
- **AND** it records and reports that the requested agent is not authorized for that repository

### Requirement: Execution metadata is auditable
The execution runtime MUST record the command, executor, and version details needed to audit and troubleshoot proposal, refine, apply, archive, and activation-triggered bootstrap steps.

#### Scenario: Heimdall runs an OpenSpec or OpenCode step
- **WHEN** Heimdall executes a workflow step through `openspec`, `opencode`, `git`, or GitHub API-backed repository mutation logic
- **THEN** it records the step outcome and the executor details needed for audit and recovery
- **AND** those records are linked to the workflow run they belong to

### Requirement: Activation bootstrap runs from a worktree created off the configured mirror
Heimdall MUST create the activation bootstrap worktree from the repository mirror configured by the resolved repository's local mirror path before invoking OpenCode.

#### Scenario: Bootstrap worktree is created
- **WHEN** an activated work item starts the bootstrap pull request workflow for the resolved repository
- **THEN** Heimdall uses that repository's configured local mirror path as the git source for the new worktree
- **AND** it runs the bootstrap execution inside that worktree rather than the bare mirror itself

### Requirement: Empty bootstrap executions fail visibly
Heimdall MUST fail the activation bootstrap workflow when the OpenCode execution completes without producing any repository changes to commit.

#### Scenario: OpenCode bootstrap produces no file changes
- **WHEN** the activation-triggered OpenCode run exits without leaving any modified, added, or deleted repository files
- **THEN** Heimdall does not create an empty commit, branch push, or pull request
- **AND** it records the workflow run as failed or blocked with a visible reason
