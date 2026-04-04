# Service: Observability

## ADDED Requirements

### Requirement: Symphony emits structured operational logs
Symphony MUST emit structured operational logs to stdout and stderr so a Linux service manager such as `systemd` and `journald` can collect them without application-managed log files, and v1 operators MUST be able to inspect those logs through the host journal.

#### Scenario: Symphony runs under systemd
- **WHEN** Symphony is started as a `systemd` service on a Linux host
- **THEN** its operational logs are available through the host's journal collection path
- **AND** operators can inspect current workflow activity without requiring application-specific log rotation logic inside Symphony

#### Scenario: Operator tails Symphony logs from the host
- **WHEN** an operator follows the Symphony service journal on the Linux host
- **THEN** current structured workflow logs are visible through the host journal stream
- **AND** no separate application-managed log file is required for normal debugging

### Requirement: Symphony exposes health and readiness signals
Symphony MUST expose separate health and readiness endpoints for operator and platform checks.

#### Scenario: Operator checks service state after deployment
- **WHEN** the operator requests Symphony's health and readiness endpoints
- **THEN** the service reports process liveness separately from dependency readiness
- **AND** a dependency failure can make readiness fail without making basic liveness fail

### Requirement: Symphony records an audit trail for mutation workflows
Symphony MUST record audit events that identify who requested a mutation, which workflow ran, which agent was used, and what outcome occurred.

#### Scenario: Authorized apply workflow completes
- **WHEN** Symphony finishes an apply workflow that was triggered by an authorized pull request comment
- **THEN** it records an audit event that includes the requesting actor, workflow type, selected agent, and resulting outcome
- **AND** the audit trail remains available independently of transient log retention settings

### Requirement: Sensitive data is excluded from logs and audit output
Symphony MUST redact or exclude secrets and raw sensitive GitHub or Linear API payload material from logs and audit records.

#### Scenario: Authentication or GitHub polling fails
- **WHEN** Symphony records a failure involving GitHub or Linear authentication material or a GitHub polling failure
- **THEN** the resulting logs and audit records exclude raw secrets and unredacted API payload bodies
- **AND** they retain only the operational details required for troubleshooting
