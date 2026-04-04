## 1. Project Bootstrap

- [ ] 1.1 Initialize the Go module, repository package layout, and `cmd/symphony/main.go` entrypoint.
- [ ] 1.2 Implement config loading for file-backed settings and environment-backed secrets.
- [ ] 1.3 Implement startup validation for required local executables: `git`, `openspec`, and `opencode`.
- [ ] 1.4 Add structured logger bootstrap plus `/healthz` and `/readyz` HTTP endpoints.

## 2. SQLite Runtime State

- [ ] 2.1 Add SQLite schema creation or migration for the runtime-state tables defined in the database design.
- [ ] 2.2 Implement store operations for provider cursors, work items, transition events, and repository bindings.
- [ ] 2.3 Implement store operations for pull requests, command requests, workflow runs and steps, jobs, and audit events.
- [ ] 2.4 Implement the worker queue with issue-scoped and repo-scoped lock keys plus retry scheduling.

## 3. Linear Activation Pipeline

- [ ] 3.1 Implement the Linear adapter with configured scope polling and normalized work item mapping.
- [ ] 3.2 Persist last-seen snapshots and cursors so `entered_active_state` detection is idempotent.
- [ ] 3.3 Implement repository routing resolution and enqueue `propose` workflow runs for matched work items.

## 4. GitHub And Repository Management

- [ ] 4.1 Implement GitHub App authentication, installation token minting, and webhook signature verification.
- [ ] 4.2 Implement webhook ingestion for `issue_comment` and `pull_request` events.
- [ ] 4.3 Implement local bare-mirror synchronization and per-run worktree management.
- [ ] 4.4 Implement deterministic branch and change naming for routed work items.
- [ ] 4.5 Implement GitHub repository operations for branch push, pull request reconcile or create, and PR comment publishing.

## 5. OpenSpec Proposal Workflow

- [ ] 5.1 Implement local `openspec` and `opencode` wrappers that use JSON output and record executor metadata.
- [ ] 5.2 Implement proposal workflow orchestration state transitions and workflow-step recording.
- [ ] 5.3 Implement OpenSpec change generation, artifact creation, and commit creation inside the proposal worktree.
- [ ] 5.4 Implement proposal branch push, pull request creation or reconciliation, and proposal status comments.

## 6. Pull Request Command Workflows

- [ ] 6.1 Implement parsing for `/symphony status`, `/symphony refine`, and `/opsx-apply` commands.
- [ ] 6.2 Implement command authorization, deduplication, and queue handoff using persisted command requests.
- [ ] 6.3 Implement `/symphony status` responses for Symphony-managed pull requests.
- [ ] 6.4 Implement `/symphony refine` execution using the repository default spec-writing agent.
- [ ] 6.5 Implement `/opsx-apply` execution with allowlisted agent selection and PR feedback comments.

## 7. Gherkin Behavior Tests

- [ ] 7.1 Write Gherkin `.feature` files for proposal creation from an activated work item.
- [ ] 7.2 Write Gherkin `.feature` files for pull request refine and apply command handling.
- [ ] 7.3 Write Gherkin `.feature` files for unauthorized command rejection and duplicate comment safety.
- [ ] 7.4 Bind the Gherkin features into the Go test suite with `godog`, step definitions, fixtures, and external-system fakes.
- [ ] 7.5 Add supporting unit and integration tests for stores, routing, webhook verification, and CLI wrapper behavior.

## 8. Documentation And Verification

- [ ] 8.1 Update repo docs with the canonical developer commands established by the bootstrap implementation.
- [ ] 8.2 Verify the Gherkin behavior tests and the relevant automated Go tests pass for the bootstrap change.
