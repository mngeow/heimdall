## 1. Configuration And Runtime Wiring

- [x] 1.1 Replace webhook-specific GitHub config and secret handling in `internal/config` with polling-oriented settings such as poll interval and overlap window.
- [x] 1.2 Update application startup and HTTP wiring so GitHub command intake comes from a polling lane instead of `internal/httpserver/webhook.go`, while preserving any private health and readiness endpoints that still matter.

## 2. GitHub Polling Adapter

- [x] 2.1 Extend `internal/scm/github` to mint installation-authenticated API clients that can list repo-scoped issue comments within a polling window and fetch current state for Symphony-managed pull requests.
- [x] 2.2 Implement GitHub polling logic that filters repo activity down to Symphony-managed pull requests and converts eligible comments and pull request changes into runtime inputs for command handling and reconciliation.

## 3. Durable State And Command Intake

- [x] 3.1 Extend the SQLite-backed runtime state to persist GitHub polling checkpoints per managed scope and to resume safely after restart.
- [x] 3.2 Update command deduplication and authorization flow so overlapping GitHub poll windows, repeated observations, and edited comments do not trigger duplicate mutation workflows.

## 4. Behavior Test Coverage

- [x] 4.1 Write or update Gherkin `.feature` scenarios for polling-based GitHub command discovery, duplicate safety across overlapping polls, and operation without a public GitHub webhook endpoint.
- [x] 4.2 Update the Go-based BDD step bindings, fixtures, and fakes in `tests/bdd` so the behavior suite exercises GitHub polling cycles instead of webhook deliveries.

## 5. Documentation And Verification

- [x] 5.1 Update the remaining docs and sample config surfaces to remove GitHub webhook setup from the standard operator path and document polling-based GitHub configuration and operations.
- [x] 5.2 Run the relevant automated verification, including the Gherkin behavior suite and `go test ./...`, and confirm the polling-based GitHub flow passes before closing the change.
