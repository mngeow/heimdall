## Context

Heimdall's current design assumes a YAML configuration file at `/etc/heimdall/config.yaml` plus separate environment-backed secrets. This change replaces that split model with a single project-root `.env` configuration source while preserving the existing single-host, single-binary deployment model and the current routing, polling, and authorization semantics.

The main design challenge is expressing structured configuration such as repository lists, routing rules, allowlists, and durations in a flat dotenv format without making the operator experience ambiguous or brittle.

## Goals / Non-Goals

**Goals:**
- Establish one canonical runtime configuration file in dotenv format.
- Preserve all existing runtime configuration semantics, including multi-repository routing and per-repo authorization settings.
- Define a stable, documented dotenv key scheme that operators can author and validate.
- Fail startup clearly when dotenv configuration is missing, malformed, or inconsistent.

**Non-Goals:**
- Supporting YAML and dotenv as equal long-term configuration sources.
- Moving configuration into SQLite or an external configuration service.
- Changing repository routing, polling, or authorization policy semantics beyond their dotenv representation.
- Reworking provider integrations or workflow behavior outside configuration loading.

## Decisions

### Decision: Use a single canonical project-root `.env` file
Heimdall will treat `.env` in the project root as the standard v1 configuration file.

Rationale:
- keeps a single operator-facing source of truth in a predictable repo-local location
- replaces the current split between YAML structure and environment-backed secrets
- aligns with the user's request to move application configuration to dotenv

Alternatives considered:
- Keep YAML and add optional dotenv overrides. Rejected because dual-source precedence increases ambiguity and migration risk.
- Read process environment only. Rejected because the change explicitly wants a file-backed configuration source that can be checked, validated, and backed up as one artifact.

### Decision: Use stable `HEIMDALL_`-prefixed keys with named repo blocks
Scalar settings will use stable `HEIMDALL_` keys. Repeated structures will use a top-level identifier list and named per-repo namespaces rather than nested YAML or positional indexes.

Representative shape:
- `HEIMDALL_SERVER_LISTEN_ADDRESS`
- `HEIMDALL_STORAGE_DSN`
- `HEIMDALL_LINEAR_ACTIVE_STATES`
- `HEIMDALL_REPOS=platform,docs`
- `HEIMDALL_REPO_PLATFORM_REMOTE`
- `HEIMDALL_REPO_PLATFORM_ALLOWED_USERS`

Rationale:
- keeps the key set grepable and explicit
- avoids fragile positional numbering for repository definitions
- preserves support for multiple repositories and explicit routing rules

Alternatives considered:
- JSON blobs inside dotenv values. Rejected because they make editing and validation harder for operators.
- Numeric indexes for repo objects. Rejected because renumbering becomes error-prone as repositories are added or removed.

### Decision: Keep secret-bearing settings in the dotenv schema, with file-path support for multiline values
All runtime configuration settings will be represented in the dotenv schema. For values such as GitHub App private keys, the preferred form will be a path-setting key that points to a host-local secret file.

Rationale:
- keeps the dotenv file as the canonical configuration source
- avoids awkward multiline secret embedding in dotenv syntax
- preserves the current expectation that sensitive files stay outside git and outside SQLite

Alternatives considered:
- Require all secrets inline in dotenv values. Rejected because PEM-style credentials are cumbersome and easy to corrupt.
- Keep secrets exclusively in separate process environment variables. Rejected because it would reintroduce a second configuration source.

### Decision: Validate the full dotenv configuration before readiness
Startup validation will parse and validate the entire project-root `.env` configuration before the service reports ready. Validation must cover required keys, durations, list syntax, referenced repo IDs, routing consistency, and unsupported legacy YAML usage.

Rationale:
- prevents partial startup with hidden misconfiguration
- gives operators one clear feedback point during deployment
- fits the repo's preference for deterministic, reconcile-safe behavior

Alternatives considered:
- Lazy validation when each subsystem first uses its config. Rejected because it hides deployment errors until runtime.

## Risks / Trade-offs

- Flat dotenv keys are more verbose than YAML nesting -> Mitigation: use stable prefixes, named repo blocks, and clear operator examples.
- A project-root `.env` file ties configuration discovery to the deployment working directory -> Mitigation: document the required working directory and make startup errors explicit when the file is missing.
- This is a breaking operational change for existing deployments -> Mitigation: document the migration path clearly and fail with explicit startup errors when only YAML is present.
- A single dotenv file may encourage operators to place too many secrets in one file -> Mitigation: prefer file-path keys for multiline secrets and document restrictive file permissions.

## Migration Plan

1. Document the canonical project-root `.env` path and key scheme in the relevant docs and setup guides.
2. Define the YAML-to-dotenv field mapping, including repeated repo settings and routing rules.
3. Update startup behavior so Heimdall reads and validates dotenv configuration from the project root and does not silently fall back to YAML.
4. Roll out by placing `.env` at the project root, restarting Heimdall, and confirming readiness.
5. Roll back by restoring the prior release and its `config.yaml`-based configuration if needed.

## Open Questions

None.
