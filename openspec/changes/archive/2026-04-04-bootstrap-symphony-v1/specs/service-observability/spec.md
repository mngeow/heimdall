## MODIFIED Requirements

### Requirement: Heimdall emits structured operational logs
Heimdall MUST emit structured operational logs to stdout and stderr so a Linux service manager such as `systemd` and `journald` can collect them without application-managed log files, and v1 operators MUST be able to inspect those logs through the host journal.

#### Scenario: Heimdall runs under systemd
- **WHEN** Heimdall is started as a `systemd` service on a Linux host
- **THEN** its operational logs are available through the host's journal collection path
- **AND** operators can inspect current workflow activity without requiring application-specific log rotation logic inside Heimdall

#### Scenario: Operator tails Heimdall logs from the host
- **WHEN** an operator follows the Heimdall service journal on the Linux host
- **THEN** current structured workflow logs are visible through the host journal stream
- **AND** no separate application-managed log file is required for normal debugging
