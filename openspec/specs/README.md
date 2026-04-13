# Heimdall Durable Specs

These OpenSpec capabilities are intentionally split into feature-facing behavior and service-facing behavior.

## Feature Specs

- `feature-kanban-activation`: starting Heimdall workflows from kanban state changes
- `feature-openspec-proposal-pr`: generating OpenSpec changes, commits, and pull requests
- `feature-pr-command-workflows`: refining specs and applying work from pull request comments

## Service Specs

- `service-board-provider`: polling, normalization, and provider abstraction
- `service-github-scm`: GitHub App auth, webhooks, repository actions, and command authorization
- `service-execution-runtime`: local OpenSpec and OpenCode execution behavior
- `service-runtime-state`: SQLite-backed state, idempotency, jobs, and bindings
- `service-observability`: logging, auditability, and health behavior
- `service-behavior-testing`: executable Gherkin behavior tests for critical workflows
