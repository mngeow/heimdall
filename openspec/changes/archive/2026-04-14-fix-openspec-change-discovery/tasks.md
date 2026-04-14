## 1. OpenSpec Change Discovery Fixes

- [x] 1.1 Update `internal/exec/clients.go` so `OpenSpecClient.ListChanges` parses the real `openspec list --json` object response and returns change names from that payload.
- [x] 1.2 Update activation proposal workflow wiring so OpenSpec list, status, and apply-instruction calls execute in the workflow run's repository worktree instead of Heimdall's process cwd.

## 2. Regression Test Coverage

- [x] 2.1 Add a focused Go unit test for the OpenSpec list-response parser using the observed `{"changes":[...]}` JSON shape.
- [x] 2.2 Add a workflow regression test that proves activation proposal change discovery uses the run worktree when Heimdall's process cwd differs from the target repository worktree.

## 3. Behavior Test Updates

- [x] 3.1 Add or update Gherkin coverage in `tests/features/` for activation proposal change discovery succeeding when OpenSpec list output uses the CLI object response shape.
- [x] 3.2 Update the corresponding Godog step bindings, fake OpenSpec fixtures, and workflow test helpers so the new change-discovery behavior scenario executes against the corrected parser and worktree scope.

## 4. Verification

- [x] 4.1 Run the relevant Go unit, workflow, and BDD test suites and verify the activation proposal path passes with the corrected OpenSpec change-discovery behavior.

## 5. Apply-Instructions Parser Fix

- [x] 5.1 Update `internal/exec/clients.go` so `GetApplyInstructions` parses the machine-readable `openspec instructions apply --json` output without treating progress text as JSON.
- [x] 5.2 Add a focused Go unit test for `GetApplyInstructions` that covers the observed apply-instructions output shape and preserves the readiness check.
- [x] 5.3 Add or update workflow and BDD coverage to verify activation proposal runs keep the readiness-check step and succeed through apply-instruction lookup.
- [x] 5.4 Re-run the relevant Go unit, workflow, and BDD test suites and verify the activation proposal path passes through both change discovery and the readiness check.
