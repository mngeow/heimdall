## MODIFIED Requirements

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
