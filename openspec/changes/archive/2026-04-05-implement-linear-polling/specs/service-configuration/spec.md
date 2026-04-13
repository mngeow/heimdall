## MODIFIED Requirements

### Requirement: Dotenv configuration preserves explicit structured runtime settings
Heimdall MUST define a stable `HEIMDALL_`-prefixed dotenv schema for server, storage, provider polling, routing, and authorization settings. The dotenv schema MUST support multiple repository definitions and explicit routing rules without requiring nested YAML. For Linear v1 polling, the dotenv schema MUST include `HEIMDALL_LINEAR_PROJECT_NAME` as the configured project name that scopes board polling.

#### Scenario: Multiple repositories are configured in dotenv
- **WHEN** the dotenv file declares more than one managed repository and their routing rules
- **THEN** Heimdall loads each repository's settings, allowlists, and routing selectors from the documented dotenv key set
- **AND** its repository resolution behavior remains explicit rather than positional or implicit

#### Scenario: Linear project-scoped polling is configured
- **WHEN** Heimdall starts with Linear polling enabled
- **THEN** it reads `HEIMDALL_LINEAR_PROJECT_NAME` from the dotenv configuration
- **AND** it uses that configured project name as the v1 scope for Linear polling

### Requirement: Dotenv parsing and validation are strict before readiness
Heimdall MUST validate required keys, value types, durations, list syntax, repository references, and routing consistency before reporting ready. When Linear polling is enabled, validation MUST fail if `HEIMDALL_LINEAR_PROJECT_NAME` is missing or empty.

#### Scenario: Required key is missing or malformed
- **WHEN** the dotenv file omits a required key or contains an invalid value such as an unreadable duration or an incomplete repository definition
- **THEN** Heimdall does not report ready
- **AND** it emits a validation error that identifies the offending key and reason

#### Scenario: Linear project name is missing
- **WHEN** the dotenv file enables Linear polling but does not provide `HEIMDALL_LINEAR_PROJECT_NAME`
- **THEN** Heimdall does not report ready
- **AND** it emits a validation error for the missing Linear project configuration
