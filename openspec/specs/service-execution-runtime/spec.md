# Service: Execution Runtime

## ADDED Requirements

### Requirement: OpenSpec and OpenCode execution runs locally on the host
Symphony MUST run OpenSpec and OpenCode workflows through local CLI execution on the Linux host where Symphony is running, and it MUST verify that required local executables such as `git`, `openspec`, and `opencode` are available before the service reports readiness.

#### Scenario: Proposal generation is requested for an activated work item
- **WHEN** Symphony begins a proposal workflow for a routed work item
- **THEN** it invokes the local `openspec` and `opencode` tooling available on the host
- **AND** it performs generation inside the repository worktree for that workflow run

#### Scenario: Required executable is missing at startup
- **WHEN** Symphony starts and one of `git`, `openspec`, or `opencode` is unavailable on the configured executable path
- **THEN** the service does not report ready for workflow execution
- **AND** it records an operator-visible startup failure that identifies the missing executable

### Requirement: OpenSpec CLI JSON output controls workflow decisions
The execution runtime MUST use OpenSpec CLI JSON output as the source of truth for change discovery, artifact dependencies, apply readiness, and archive behavior.

#### Scenario: Apply workflow is requested from a pull request comment
- **WHEN** Symphony prepares to run `/opsx-apply` for an OpenSpec change
- **THEN** it reads OpenSpec apply instructions and current status from CLI JSON output
- **AND** it does not guess task readiness or context file selection from filesystem conventions alone

### Requirement: Agent selection is explicit and policy-controlled
The execution runtime MUST use the repository's default spec-writing agent for refine operations and MUST require an explicitly selected allowlisted agent for apply operations.

#### Scenario: User runs apply without an allowed agent
- **WHEN** a pull request comment requests `/opsx-apply` with an agent that is not allowed for the repository
- **THEN** Symphony does not start the apply execution
- **AND** it records and reports that the requested agent is not authorized for that repository

### Requirement: Execution metadata is auditable
The execution runtime MUST record the command, executor, and version details needed to audit and troubleshoot proposal, refine, apply, and archive steps.

#### Scenario: Symphony runs an OpenSpec or OpenCode step
- **WHEN** Symphony executes a workflow step through `openspec`, `opencode`, `git`, or GitHub API-backed repository mutation logic
- **THEN** it records the step outcome and the executor details needed for audit and recovery
- **AND** those records are linked to the workflow run they belong to
