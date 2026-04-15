## Why

Heimdall already persists rich operational state for work items, repository bindings, pull requests, command requests, workflow runs, jobs, and audit events, but operators still have to reconstruct current system state from logs, GitHub, and direct database inspection. A small private dashboard inside the existing Go service would make it much easier to see queued Linear work, active automation pull requests, and the PR command/activity history that explains what Heimdall has done.

## What Changes

- Add a private read-only operator dashboard served from Heimdall's existing HTTP server instead of creating a separate frontend application.
- Add a work-item queue view that shows Linear-backed work items across all statuses and lifecycle buckets, with repository-binding and latest workflow context visible from the same screen.
- Add an active pull request view and drill-down pages that show PR identity, linked work item and branch/change data, Heimdall-tracked comment command history, workflow runs, and recent audit activity.
- Add server-rendered HTML pages with HTMX-driven filtering, refresh, and detail loading so operators can inspect state without a separate SPA stack.
- Update operator documentation and verification coverage for the new embedded dashboard surface.

## Capabilities

### New Capabilities
- `service-operator-dashboard`: private embedded operator UI for inspecting queued work items, active pull requests, and related Heimdall workflow activity.

### Modified Capabilities
- `service-observability`: operator-visible observability now includes a private dashboard for current runtime state in addition to health, readiness, logs, and audit visibility.

## Impact

- Affects Heimdall's HTTP server, operator-facing routes, and server-rendered HTML templates.
- Affects SQLite-backed read models and query paths for work items, repo bindings, pull requests, command requests, workflow runs, jobs, and audit events.
- Adds an HTMX dependency and dashboard-specific handler/template coverage within the same Go binary.
- Changes operator documentation because Heimdall will now expose a private embedded dashboard, not just health/readiness endpoints and host logs.
