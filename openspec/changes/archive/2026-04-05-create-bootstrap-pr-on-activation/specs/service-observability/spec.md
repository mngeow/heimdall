## MODIFIED Requirements

### Requirement: Heimdall emits structured operational logs
Heimdall MUST emit structured operational logs to stdout and stderr so a Linux service manager such as `systemd` and `journald` can collect them without application-managed log files, and v1 operators MUST be able to inspect those logs through the host journal. For activation-triggered bootstrap workflows, those logs MUST include step-level context that lets operators trace workflow progress and diagnose failures without exposing secrets or raw prompt bodies.

#### Scenario: Operator tails logs during an activation bootstrap workflow
- **WHEN** an operator follows Heimdall logs while an activation-triggered bootstrap pull request workflow is running
- **THEN** the logs identify the workflow run, work item, repository, current workflow step, and step outcome as the run moves through routing, worktree creation, OpenCode execution, git mutation, and pull request reconciliation
- **AND** the logs include blocked, retry, and failure reasons that are specific enough for debugging without exposing secrets, installation tokens, or raw prompt content
