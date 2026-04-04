## 1. Runtime State And Schema

- [ ] 1.1 Add SQLite schema and state-access changes for GitHub polling checkpoints, pull request lifecycle observations, and duplicate-safe comment identities.
- [ ] 1.2 Extend repository and pull request binding persistence so GitHub polling can resume after restart and support initial backfill for existing Symphony-managed pull requests.

## 2. GitHub Poller

- [ ] 2.1 Implement a GitHub polling worker that loads active Symphony pull request bindings, groups them by configured repository and installation, and polls on `github.poll_interval`.
- [ ] 2.2 Fetch newly created pull request conversation comments and pull request state changes, then persist normalized command candidates and lifecycle observations before advancing checkpoints.
- [ ] 2.3 Add initial backfill, retry, and rate-limit handling so first-run polling and transient GitHub failures do not lose command intake.

## 3. Workflow Integration And Observability

- [ ] 3.1 Wire polled GitHub comment candidates into the existing authorization, parsing, and command execution pipeline without introducing webhook dependencies.
- [ ] 3.2 Wire polled pull request state observations into lifecycle reconciliation so active bindings stay synchronized and terminal pull requests stop generating unnecessary polling work.
- [ ] 3.3 Add structured logs and counters for poll cycles, discovered comments, discovered state changes, duplicate suppression, lag, and failures.

## 4. Behavior Test Coverage

- [ ] 4.1 Add or update Gherkin feature scenarios for new comment discovery by polling, restart-safe checkpoint resume, and duplicate suppression across overlapping poll windows.
- [ ] 4.2 Update the Go step bindings, fixtures, and test runner integration needed to simulate GitHub poll cycles, saved checkpoints, and restart behavior for those scenarios.

## 5. Verification

- [ ] 5.1 Update the relevant operations, workflow, and setup docs if the implemented polling behavior or operator expectations change from the current design docs.
- [ ] 5.2 Run the relevant automated tests for the Gherkin behavior coverage and any GitHub SCM or runtime state tests, then fix any failures before marking the change complete.
