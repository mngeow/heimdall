## Why

Heimdall now has product docs, architecture docs, setup guidance, database design, logging guidance, and durable OpenSpec specifications, but it still does not have a runnable Go application. The next step is to turn that design-first repository into a first working end-to-end service slice so the documented Linear-to-GitHub workflow can be exercised on a real Linux host.

This change is needed now because the current repository cannot yet validate its own integration model, runtime boundaries, or operator workflow. A single broad bootstrap change is the fastest way to establish the first executable service, developer commands, and behavior-test baseline that later changes can iterate on safely.

## What Changes

- Scaffold the initial Go application layout, configuration model, startup path, and runtime wiring for a single-host Heimdall service.
- Implement the first working Linear poller, GitHub App integration, local repository/worktree management, and OpenSpec proposal workflow.
- Implement the first PR comment command path for status, refine, and apply request intake, including authorization and deduplication boundaries.
- Add SQLite-backed runtime state, job orchestration primitives, observability hooks, and operator-facing health/logging behavior.
- Add executable Gherkin behavior tests and the Go-side test harness needed to verify the bootstrap workflows and keep them passing.
- Establish the canonical local development, test, and verification commands for the repo.

## Capabilities

### New Capabilities
- None. This change implements the existing Heimdall capabilities already captured under `openspec/specs/`.

### Modified Capabilities
- `service-board-provider`: clarify that v1 activation detection is polling-based and does not depend on inbound Linear webhooks.
- `service-github-scm`: clarify the minimum GitHub webhook event intake required for PR command handling and reconciliation.
- `service-execution-runtime`: clarify startup validation of required local executables before the service is considered ready.
- `service-runtime-state`: clarify SQLite schema initialization and migration behavior at service startup.
- `service-observability`: clarify journald-first Linux log access for operators.

## Impact

- Affected code: new Go module, initial `cmd/` and `internal/` packages, config loading, runtime state store, Linear adapter, GitHub adapter, local execution wrappers, and test harness.
- Affected systems: Linear API, GitHub App API and webhooks, local git repositories/worktrees, local OpenSpec/OpenCode CLIs, and SQLite.
- Operator impact: establishes the first runnable service and the first concrete install/test workflow for a Linux host.
- Dependency impact: introduces Go module dependencies, a Go-compatible Gherkin test runner, and the initial database schema/migrations.
