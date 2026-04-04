## Why

Symphony currently assumes a public GitHub webhook endpoint for pull request comment workflows. That makes the single-host deployment model harder to operate for users who do not want to expose public ingress, even though the rest of v1 already prefers polling and operator simplicity.

This change is needed now to align GitHub command intake with that operator model: Symphony should poll GitHub for new pull request comments and pull request state changes instead of requiring inbound webhook delivery.

## What Changes

- Replace webhook-based GitHub pull request command intake with GitHub App polling against configured repositories.
- Remove the v1 requirement for a public GitHub webhook URL and webhook secret when using the standard Symphony deployment path.
- Add durable GitHub polling checkpoints, deduplication rules, and reconciliation behavior so restarts and overlapping poll windows do not re-run the same command.
- Update operator-facing setup and configuration guidance to explain GitHub polling intervals, lookback windows, installation setup, and verification steps.
- Preserve the narrow command surface and authorization rules: only supported commands on Symphony-managed pull requests from authorized actors may trigger mutation workflows.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `feature-pr-command-workflows`: change pull request command intake from webhook delivery to polling-based detection while preserving authorization and duplicate-safety behavior.
- `service-github-scm`: change GitHub integration from inbound webhook handling to GitHub App polling for pull request comments and related pull request state.
- `service-runtime-state`: persist GitHub polling checkpoints and related deduplication state so polling remains restart-safe and idempotent.

## Impact

- Affected code: GitHub config loading, GitHub adapter polling logic, command intake routing, runtime-state persistence, and behavior tests.
- Affected systems: GitHub App configuration, GitHub REST API usage and rate limits, SQLite runtime state, and operator deployment networking.
- Operator impact: removes the need to expose a public webhook endpoint for GitHub, but requires explicit polling interval and lookback configuration.
- Dependency impact: no new external service is required, but GitHub API polling volume and backoff behavior must be managed carefully.
