## Why

Symphony's GitHub-side intake needs to be explicitly implementation-ready around polling so pull request comments and pull request state changes are discovered without any webhook dependency. The current design already prefers polling in v1, but the change needs a concrete polling implementation contract for checkpointing, restart safety, and command intake behavior.

## What Changes

- Implement a GitHub poller that periodically discovers new comments and pull request state changes for Symphony-managed pull requests.
- Persist GitHub polling checkpoints and deduplication state in SQLite so restarts do not replay previously seen comments or lose lifecycle reconciliation state.
- Route newly discovered PR comments through the existing authorization and command parsing flow without requiring inbound GitHub webhooks.
- Add operational visibility for GitHub polling progress, lag, and failures.
- Keep v1 scoped to polling only; this change does not add a public webhook receiver.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `service-github-scm`: tighten the GitHub polling requirements so intake is scoped to Symphony-managed pull requests, comments are discovered from poll cycles, and pull request state changes are reconciled without webhooks.
- `service-runtime-state`: require durable storage for GitHub polling checkpoints and comment deduplication state so polling remains restart-safe.

## Impact

- GitHub SCM adapter and any background poller worker
- SQLite runtime state schema and repository or pull request binding records
- PR command intake flow for `/symphony` and `/opsx-apply` comments
- Operational logs, counters, and polling configuration
- Behavior tests covering polled comment discovery, restart safety, and duplicate handling
