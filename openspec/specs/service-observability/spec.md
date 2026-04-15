# Service: Observability

## ADDED Requirements

### Requirement: Heimdall emits structured operational logs
Heimdall MUST emit structured operational logs to stdout and stderr so a Linux service manager such as `systemd` and `journald` can collect them without application-managed log files, and v1 operators MUST be able to inspect those logs through the host journal. For activation-triggered bootstrap workflows, those logs MUST include step-level context that lets operators trace workflow progress and diagnose failures without exposing secrets or raw prompt bodies.

#### Scenario: Heimdall runs under systemd
- **WHEN** Heimdall is started as a `systemd` service on a Linux host
- **THEN** its operational logs are available through the host's journal collection path
- **AND** operators can inspect current workflow activity without requiring application-specific log rotation logic inside Heimdall

#### Scenario: Operator tails Heimdall logs from the host
- **WHEN** an operator follows the Heimdall service journal on the Linux host
- **THEN** current structured workflow logs are visible through the host journal stream
- **AND** no separate application-managed log file is required for normal debugging

#### Scenario: Operator tails logs during an activation bootstrap workflow
- **WHEN** an operator follows Heimdall logs while an activation-triggered bootstrap pull request workflow is running
- **THEN** the logs identify the workflow run, work item, repository, current workflow step, and step outcome as the run moves through routing, worktree creation, OpenCode execution, git mutation, and pull request reconciliation
- **AND** the logs include blocked, retry, and failure reasons that are specific enough for debugging without exposing secrets, installation tokens, or raw prompt content

### Requirement: Heimdall exposes health and readiness signals
Heimdall MUST expose separate health and readiness endpoints for operator and platform checks.

#### Scenario: Operator checks service state after deployment
- **WHEN** the operator requests Heimdall's health and readiness endpoints
- **THEN** the service reports process liveness separately from dependency readiness
- **AND** a dependency failure can make readiness fail without making basic liveness fail

### Requirement: Heimdall records an audit trail for mutation workflows
Heimdall MUST record audit events that identify who requested a mutation, which workflow ran, which agent was used, and what outcome occurred.

#### Scenario: Authorized apply workflow completes
- **WHEN** Heimdall finishes an apply workflow that was triggered by an authorized pull request comment
- **THEN** it records an audit event that includes the requesting actor, workflow type, selected agent, and resulting outcome
- **AND** the audit trail remains available independently of transient log retention settings

### Requirement: Sensitive data is excluded from logs and audit output
Heimdall MUST redact or exclude secrets and raw sensitive webhook material from logs and audit records.

#### Scenario: Authentication or webhook verification fails
- **WHEN** Heimdall records a failure involving GitHub or Linear authentication material
- **THEN** the resulting logs and audit records exclude raw secrets and unredacted webhook bodies
- **AND** they retain only the operational details required for troubleshooting

### Requirement: Operator dashboard output excludes secrets and raw sensitive payloads
Heimdall MUST treat the operator dashboard as part of its observability surface and MUST expose operational metadata only. Dashboard pages MUST exclude secrets, installation tokens, provider credentials, raw prompt bodies, and raw unparsed GitHub comment payloads.

#### Scenario: Operator opens a dashboard page with workflow activity
- **WHEN** an operator requests a dashboard page that shows work items, pull requests, command requests, workflow runs, jobs, or audit events
- **THEN** Heimdall renders identifiers, statuses, timestamps, summaries, and other operational metadata needed for troubleshooting
- **AND** the rendered page excludes secrets and raw sensitive payload bodies
