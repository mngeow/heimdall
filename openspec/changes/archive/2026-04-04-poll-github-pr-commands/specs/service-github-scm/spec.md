## REMOVED Requirements

### Requirement: GitHub webhooks are verified before processing
**Reason**: GitHub pull request command intake and pull request lifecycle detection now use polling instead of inbound webhook delivery.
**Migration**: Disable GitHub App webhook delivery for the standard Heimdall deployment path, remove webhook-secret configuration, and configure GitHub polling settings instead.

## ADDED Requirements

### Requirement: GitHub pull request activity is polled before processing
Heimdall MUST poll GitHub for new pull request comments and relevant pull request lifecycle changes by using its GitHub App installation credentials, and it MUST not require inbound GitHub webhook delivery for the standard v1 deployment path.

#### Scenario: GitHub poll sees a new command comment on a Heimdall-managed pull request
- **WHEN** Heimdall runs a GitHub poll cycle for a managed repository and observes a new issue comment on a Heimdall-managed pull request within the configured polling window
- **THEN** it makes that comment available to the runtime components responsible for authorization and command parsing
- **AND** the command-intake path does not depend on a public inbound webhook endpoint

#### Scenario: GitHub poll sees a relevant pull request state change
- **WHEN** Heimdall polls a managed repository and detects a relevant lifecycle change on a Heimdall-managed pull request
- **THEN** it makes that pull request change available to the runtime components responsible for pull request reconciliation
- **AND** the reconciliation path does not depend on inbound webhook delivery
