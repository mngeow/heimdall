## Context

The activation proposal workflow now relies on `openspec list --json` to discover the change created by proposal generation before it requests apply instructions and persists the repository binding. In the current implementation, the OpenSpec adapter assumes that `openspec list --json` returns a raw `[]string`, but the CLI actually returns an object containing a `changes` array. That mismatch causes proposal workflows to fail before artifact discovery can complete.

The workflow also performs a post-generation readiness check with `openspec instructions apply --change <name> --json`. That step is intentionally read-only: Heimdall uses it to confirm the generated change is implementation-ready before it commits and publishes the proposal branch. The current adapter reads combined stdout/stderr and tries to parse the merged stream as JSON, which fails when the CLI writes human-oriented progress lines alongside the JSON payload.

There is also a scoping problem in the activation path: the proposal runner passes the correct repository worktree to `opencode`, but the OpenSpec client used by the workflow is constructed without the run's worktree path. That means OpenSpec commands can execute in the Heimdall service process cwd instead of the target repository worktree, which makes discovery depend on where Heimdall was started rather than on the mapped repository.

The existing workflow and fake-client tests did not catch these failures because they stubbed list responses as simple string slices and did not assert that OpenSpec discovery executed inside the run worktree.

## Goals / Non-Goals

**Goals:**
- Parse the real `openspec list --json` response shape so activation proposal change discovery succeeds.
- Parse machine-readable `openspec instructions apply --json` output without breaking when the CLI emits non-JSON progress text on a separate stream.
- Ensure activation proposal OpenSpec commands execute in the run's repository worktree.
- Keep the existing readiness check in the proposal workflow.
- Add regression coverage for list/apply-instructions parsing and workflow worktree scoping.
- Keep the current activation proposal flow intact apart from these correctness fixes.

**Non-Goals:**
- Changing the proposal prompt, proposal agent selection, or branch naming behavior.
- Redesigning how activation proposal generation chooses or discovers change names beyond the current before/after list comparison.
- Removing the post-proposal readiness check from the activation workflow.
- Changing refine, apply, or archive workflow semantics outside the shared OpenSpec client behavior they already depend on.

## Decisions

### Decision: Parse typed `openspec list --json` responses instead of assuming a raw string slice
The OpenSpec execution adapter will model the CLI response as a typed object with a `changes` collection and extract change names from that structure.

Rationale:
- matches the real CLI contract observed at runtime
- keeps JSON parsing explicit and easy to validate in tests
- avoids fragile `map[string]any` parsing and ad hoc casting

Alternatives considered:
- Parse the response as `map[string]any`. Rejected because it weakens type safety and makes parser bugs easier to miss.
- Keep assuming `[]string` and special-case failures. Rejected because it does not match the CLI's real output.

### Decision: Parse apply-instructions JSON from machine-readable output while preserving the readiness check
The OpenSpec execution adapter will continue to call `openspec instructions apply --change <name> --json` during activation proposal execution, but it will parse the JSON payload from the machine-readable command output instead of treating mixed human/log output as JSON.

Rationale:
- preserves the current readiness check that guards proposal publication
- matches the observed CLI behavior where progress text can appear alongside the JSON payload
- lets Heimdall continue using OpenSpec as the source of truth for implementation readiness

Alternatives considered:
- Remove the readiness-check step entirely. Rejected because the workflow should still verify that proposal generation produced an apply-ready change.
- Keep using `CombinedOutput()` and try to unmarshal the merged stream. Rejected because progress text makes that stream non-JSON.

### Decision: Scope OpenSpec execution to the workflow run worktree
The activation workflow will construct or otherwise resolve OpenSpec execution against the workflow run's worktree path before it calls list, status, or apply-instruction commands.

Rationale:
- keeps OpenSpec discovery aligned with the same repository state used by `opencode`
- prevents activation proposal success or failure from depending on Heimdall's process cwd
- preserves the repo-adapter boundary by keeping filesystem scope inside the execution client

Alternatives considered:
- Reuse a singleton OpenSpec client with an empty or mutable working directory. Rejected because it is error-prone and unsafe for concurrent workflow runs.
- Rely on the process cwd being set to the correct repository. Rejected because Heimdall manages multiple repositories and should not depend on operator launch location.

### Decision: Add regression tests at both parser and workflow levels
The change will add focused unit tests for the `openspec list --json` and `openspec instructions apply --json` response handling and a workflow-level regression test that proves change discovery/readiness checks use the run worktree rather than Heimdall's cwd.

Rationale:
- catches the exact failure seen in runtime logs
- covers both the adapter contract and the orchestration wiring
- prevents fake-client tests from masking CLI-shape regressions in the future

Alternatives considered:
- Add only workflow tests. Rejected because parser-shape regressions would be harder to isolate.
- Add only parser tests. Rejected because worktree scoping is a separate wiring bug.

## Risks / Trade-offs

- OpenSpec CLI output could evolve again in the future -> Mitigation: keep the parsers typed, small, and covered by direct unit tests.
- Per-run OpenSpec client scoping adds a small amount of wiring churn -> Mitigation: keep the interface narrow and local to the activation workflow.
- Workflow tests that assert worktree scope can become coupled to client construction details -> Mitigation: assert observable command scope or explicit worktree path propagation rather than private implementation details.

## Migration Plan

1. Update the OpenSpec adapter to parse the real `openspec list --json` response object and the machine-readable `openspec instructions apply --json` response.
2. Update activation proposal workflow wiring so OpenSpec commands use the run worktree.
3. Add unit and workflow regression coverage for parser shape and worktree scoping.
4. Run the Go test suite and verify the activation proposal path no longer fails on change discovery or the readiness check.

Rollback is straightforward: revert the adapter and workflow wiring changes together.

## Open Questions

None.
