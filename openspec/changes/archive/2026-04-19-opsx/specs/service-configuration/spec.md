## ADDED Requirements

### Requirement: Dotenv configuration declares a public operator URL for GitHub links
Heimdall MUST support `HEIMDALL_SERVER_PUBLIC_URL` as the absolute base URL for operator endpoints exposed outside the process, and it MUST use that setting when building dashboard links published into GitHub pull-request reply comments.

#### Scenario: Valid public operator URL is configured
- **WHEN** Heimdall loads dotenv configuration that includes `HEIMDALL_SERVER_PUBLIC_URL=http://127.0.0.1:8080`
- **THEN** it stores that absolute URL as the base for operator dashboard links
- **AND** GitHub reply comments that reference live command output are built from that configured base URL

#### Scenario: Public operator URL is missing and PR-command workflows are active
- **WHEN** Heimdall starts with at least one managed repository configured for PR-comment mutation commands and without a valid absolute `HEIMDALL_SERVER_PUBLIC_URL`
- **THEN** it does not report ready
- **AND** it emits a validation error that the operator URL must be configured as an absolute URL

#### Scenario: Public operator URL is missing and no PR-command workflows are active
- **WHEN** Heimdall starts with no managed repository configured for PR-comment mutation commands and without a valid absolute `HEIMDALL_SERVER_PUBLIC_URL`
- **THEN** it may report ready
- **AND** it emits a warning that dashboard link comments are unavailable
