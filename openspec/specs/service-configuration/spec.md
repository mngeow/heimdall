# Service: Configuration

## ADDED Requirements

### Requirement: Heimdall loads runtime configuration from a single project-root dotenv file
Heimdall MUST read its application configuration from a single dotenv-formatted `.env` file for the standard v1 deployment path, and the canonical file location MUST be the Heimdall project root. Heimdall MUST NOT require `config.yaml` or any other YAML configuration file to start.

#### Scenario: Service starts with a valid project-root dotenv file
- **WHEN** Heimdall starts in a project root where `.env` contains a complete valid configuration
- **THEN** it loads its runtime configuration from that dotenv file
- **AND** it continues startup without reading YAML configuration

#### Scenario: Only legacy YAML configuration is present
- **WHEN** Heimdall starts and the project-root `.env` file is missing but a legacy YAML configuration file is present
- **THEN** it does not report ready
- **AND** it emits an operator-visible error that the service now expects dotenv configuration

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

### Requirement: Repository configuration can declare a GitHub PR monitor label
Heimdall MUST support an optional `HEIMDALL_REPO_<ID>_PR_MONITOR_LABEL` setting for each managed repository. When present, Heimdall MUST treat that value as the GitHub label name used to mark and narrow monitoring for that repository's Heimdall pull requests, and it MUST reject empty or whitespace-only values.

#### Scenario: Repository config declares a PR monitor label
- **WHEN** Heimdall loads a repository block that includes `HEIMDALL_REPO_PLATFORM_PR_MONITOR_LABEL=heimdall-monitored`
- **THEN** it stores `heimdall-monitored` as that repository's GitHub PR monitor label
- **AND** the GitHub adapter uses that label name for PR reconciliation and polling filters in that repository

#### Scenario: Repository config omits a PR monitor label
- **WHEN** Heimdall loads a managed repository block that does not define `HEIMDALL_REPO_PLATFORM_PR_MONITOR_LABEL`
- **THEN** it accepts that repository configuration
- **AND** GitHub PR monitoring for that repository continues without label-based filtering

#### Scenario: Repository config declares an empty PR monitor label
- **WHEN** Heimdall loads a repository block where `HEIMDALL_REPO_PLATFORM_PR_MONITOR_LABEL` is empty or whitespace only
- **THEN** it does not report ready
- **AND** it emits a validation error for that repository setting

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

### Requirement: Dotenv configuration supports secret-bearing settings without storing them in SQLite
Heimdall MUST load secret-bearing settings through the dotenv key set, including file-path-based references for values that are impractical to inline, and it MUST keep those resolved secrets out of SQLite runtime state.

#### Scenario: GitHub App private key is referenced from the dotenv file
- **WHEN** the dotenv file provides the GitHub App private key through a filesystem path setting
- **THEN** Heimdall loads that credential through the dotenv-derived configuration
- **AND** SQLite stores only operational metadata rather than the secret material
