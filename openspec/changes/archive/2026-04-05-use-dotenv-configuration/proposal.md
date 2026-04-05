## Why

Symphony's current design assumes a YAML configuration file plus separate environment-backed secrets. Moving all application configuration into a single dotenv file simplifies deployment and operator setup by giving the service one canonical configuration input to validate, back up, and load at startup.

## What Changes

- **BREAKING** Replace the documented YAML-based application configuration with a project-root `.env` file as Symphony's canonical runtime config source.
- Define the expected dotenv key format for all runtime settings, including server, storage, polling, repository routing, and authorization allowlists.
- Require startup validation that fails clearly when required dotenv keys are missing, malformed, or internally inconsistent.
- Update operator-facing documentation and setup guidance to reference the project-root `.env` file instead of `config.yaml`.

## Capabilities

### New Capabilities
- `service-configuration`: Defines Symphony's canonical dotenv configuration source, key naming, parsing, and startup validation behavior.

### Modified Capabilities

## Impact

- Affects runtime configuration loading and validation behavior.
- Affects operator setup, deployment, backup, and recovery documentation.
- Affects repository routing, polling, and authorization settings because their values now come from dotenv keys rather than YAML fields.
