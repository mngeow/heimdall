## ADDED Requirements

### Requirement: Operator dashboard output excludes secrets and raw sensitive payloads
Heimdall MUST treat the operator dashboard as part of its observability surface and MUST expose operational metadata only. Dashboard pages MUST exclude secrets, installation tokens, provider credentials, raw prompt bodies, and raw unparsed GitHub comment payloads.

#### Scenario: Operator opens a dashboard page with workflow activity
- **WHEN** an operator requests a dashboard page that shows work items, pull requests, command requests, workflow runs, jobs, or audit events
- **THEN** Heimdall renders identifiers, statuses, timestamps, summaries, and other operational metadata needed for troubleshooting
- **AND** the rendered page excludes secrets and raw sensitive payload bodies
