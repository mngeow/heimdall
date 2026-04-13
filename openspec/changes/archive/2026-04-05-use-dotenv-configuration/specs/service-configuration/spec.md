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
Heimdall MUST define a stable `HEIMDALL_`-prefixed dotenv schema for server, storage, provider polling, routing, and authorization settings. The dotenv schema MUST support multiple repository definitions and explicit routing rules without requiring nested YAML.

#### Scenario: Multiple repositories are configured in dotenv
- **WHEN** the dotenv file declares more than one managed repository and their routing rules
- **THEN** Heimdall loads each repository's settings, allowlists, and routing selectors from the documented dotenv key set
- **AND** its repository resolution behavior remains explicit rather than positional or implicit

### Requirement: Dotenv parsing and validation are strict before readiness
Heimdall MUST validate required keys, value types, durations, list syntax, repository references, and routing consistency before reporting ready.

#### Scenario: Required key is missing or malformed
- **WHEN** the dotenv file omits a required key or contains an invalid value such as an unreadable duration or an incomplete repository definition
- **THEN** Heimdall does not report ready
- **AND** it emits a validation error that identifies the offending key and reason

### Requirement: Dotenv configuration supports secret-bearing settings without storing them in SQLite
Heimdall MUST load secret-bearing settings through the dotenv key set, including file-path-based references for values that are impractical to inline, and it MUST keep those resolved secrets out of SQLite runtime state.

#### Scenario: GitHub App private key is referenced from the dotenv file
- **WHEN** the dotenv file provides the GitHub App private key through a filesystem path setting
- **THEN** Heimdall loads that credential through the dotenv-derived configuration
- **AND** SQLite stores only operational metadata rather than the secret material
