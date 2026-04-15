## 1. Dashboard foundation

- [x] 1.1 Extend the existing HTTP server with dashboard route wiring for `/ui`, `/ui/work-items`, `/ui/pull-requests`, and pull-request detail endpoints.
- [x] 1.2 Add shared server-rendered layout/template plumbing and HTMX fragment handlers so the dashboard stays inside the same Go binary without a separate frontend app.
- [x] 1.3 Keep the dashboard read-only by ensuring the new handlers expose inspection views only and do not trigger repository mutation workflows.

## 2. SQLite-backed dashboard views

- [ ] 2.1 Implement dashboard read-query services for overview counts sourced from work items, pull requests, workflow runs, and jobs.
- [ ] 2.2 Implement the work-item queue screen and filter fragments using `work_items`, `repo_bindings`, and recent `workflow_runs` to show all tracked statuses and lifecycle buckets.
- [ ] 2.3 Implement the active pull-request list and detail view using `pull_requests`, `repo_bindings`, `work_items`, `command_requests`, `workflow_runs`, `workflow_steps`, `jobs`, and `audit_events`.
- [x] 2.4 Label the PR detail timeline as Heimdall-tracked command/activity history, add the canonical GitHub PR link, and ensure rendered fields exclude secrets and raw sensitive payloads.

## 3. Documentation updates

- [x] 3.1 Update `docs/product.md`, `docs/architecture.md`, and `docs/operations.md` to describe the embedded private operator dashboard, its HTMX/server-rendered approach, and its private deployment expectations.
- [x] 3.2 Update any setup or operator-facing docs that need dashboard route or access guidance once the concrete HTTP surface is implemented.

## 4. Verification

- [x] 4.1 Add Gherkin feature coverage for the dashboard overview, work-item queue, active pull-request listing, PR detail activity timeline, and read-only filter/refresh behavior.
- [x] 4.2 Implement or update Go step bindings, fixtures, and test-runner integration needed to execute the new dashboard behavior scenarios.
- [x] 4.3 Add focused handler/query tests for dashboard rendering, joins, filtering, and sensitive-data exclusion.
- [x] 4.4 Run the relevant automated tests and verify the dashboard change passes before marking the work complete.
