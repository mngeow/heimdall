## 1. Configuration And Documentation

- [x] 1.1 Add `HEIMDALL_REPO_<ID>_PR_MONITOR_LABEL` to the repository dotenv schema, typed config structs, and startup validation.
- [x] 1.2 Update `docs/setup/github.md`, `docs/authentication.md`, and the repo config examples to document the new label setting and the exact minimum GitHub App permissions.

## 2. GitHub Label Reconciliation And Polling

- [x] 2.1 Extend the GitHub adapter to create or reuse the configured repository monitor label when a repository enables label-scoped PR monitoring.
- [x] 2.2 Update pull request create-or-reuse reconciliation so Heimdall adds the configured monitor label to managed pull requests without replacing unrelated labels.
- [x] 2.3 Narrow GitHub polling to Heimdall-managed pull requests that carry the configured monitor label when that setting is present, while preserving current behavior for repositories that do not configure it.

## 3. Behavior Test Coverage

- [x] 3.1 Write Gherkin `.feature` scenarios for configured monitor-label validation, automatic repository-label creation, automatic PR labeling, and polling that ignores unlabeled pull requests when label-scoped monitoring is enabled.
- [x] 3.2 Update the Go-based BDD step bindings, GitHub fixtures, and polling fixtures so the behavior suite can exercise label creation, additive PR labeling, and label-filtered polling.
- [x] 3.3 Add focused automated tests for config validation, label create-or-reuse behavior, and additive PR label reconciliation.

## 4. Verification

- [x] 4.1 Run the relevant automated tests, including the Gherkin behavior suite and focused automated tests for config and GitHub adapter behavior, and verify they pass.
- [x] 4.2 Verify in a test repository that Heimdall creates the configured repository label, applies it to Heimdall pull requests, and only monitors labeled Heimdall pull requests when the setting is enabled.
