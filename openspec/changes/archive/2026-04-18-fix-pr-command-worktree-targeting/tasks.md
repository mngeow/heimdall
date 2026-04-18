## 1. Canonical PR worktree preparation

- [x] 1.1 Replace the hardcoded PR-command worktree path logic in `internal/workflow/pr_commands.go` with the shared deterministic worktree-path strategy based on `GenerateWorktreePath(repository.LocalMirrorPath, pr.HeadBranch)` and return that prepared path for downstream use.
- [x] 1.2 Update `internal/repo/manager.go` so worktree creation prefers the fetched branch ref for an already-existing PR/proposal branch and falls back to the repository default branch only when bootstrapping a brand-new automation branch.

## 2. PR-command target resolution and execution scoping

- [x] 2.1 Update PR-command binding resolution in `internal/store/workflows.go` and related workflow code to use `pull_requests.repo_binding_id` first, then same-repository branch fallback for legacy rows, while rejecting cross-repository branch-name collisions.
- [x] 2.2 Reorder refine/apply/opencode orchestration in `internal/workflow/pr_commands.go` so worktree preparation happens before change-existence validation and scope OpenSpec validation, opencode execution, change detection, commit, and push to the same prepared worktree.
- [x] 2.3 Update the OpenSpec/OpenCode execution boundary in `internal/exec` and app wiring so PR-command execution never runs with an empty or mismatched worktree directory.

## 3. Deterministic activation change naming

- [x] 3.1 Update activation proposal change-name generation in `internal/workflow` and related helper or prompt-building code so the canonical OpenSpec change name is derived from the normalized Linear ticket title by converting spaces to hyphens and applying the documented slug-cleaning rules.
- [x] 3.2 Extend `tests/features/proposal_creation.feature`, its step bindings, and focused proposal workflow tests to cover title-derived change names, including space-to-hyphen normalization, repeated-separator collapse, and persistence of the discovered title-derived change name.
- [x] 4.1 Extend `tests/features/pr_commands.feature` with scenarios covering canonical PR-command worktree derivation, validation after worktree refresh, cross-repository branch-name collisions, and existing PR-branch recreation from the fetched branch ref.
- [x] 4.2 Update the BDD step bindings and focused unit tests in `internal/workflow`, `internal/repo`, `internal/store`, and `internal/exec` to cover deterministic worktree selection, durable binding lookup, shared OpenSpec/opencode worktree scoping, and existing-branch materialization.

## 5. Documentation and verification

- [x] 5.1 Update `docs/workflows.md`, `docs/architecture.md`, and `docs/operations.md` so the documented PR-command flow, worktree layout, and activation proposal change-name derivation match the canonical mirror-adjacent behavior and title-to-kebab-case naming rule.
- [x] 5.2 Run `go test ./...` and confirm the new PR-command worktree, binding-resolution, and title-derived change-name regressions pass before closing the change.

## 6. Large opencode event stream parsing

- [x] 6.1 Update `internal/exec/clients.go` so `parseOpencodeEvents` consumes newline-delimited `opencode run --format json` output with a reader that supports very large single-event lines, skips non-JSON noise safely, and still processes a final valid event at EOF without requiring a trailing newline.
- [x] 6.2 Add focused `internal/exec/clients_test.go` coverage for valid JSON event lines larger than the old scanner limit, later permission or terminal-event detection after a large text event, and final-event parsing when the stream ends without a newline.
- [x] 6.3 Extend PR-command behavior tests in `tests/features/pr_commands.feature`, BDD step bindings, and/or focused workflow tests so refine/apply runs are not failed solely because opencode emitted a large valid JSON event line.
- [x] 6.4 Update PR-command outcome handling in `internal/exec` and `internal/workflow/pr_commands.go` so intermediate generic `tool_use` error events do not override a later successful terminal outcome, and true failures always surface a non-empty fallback summary instead of blank `refine failed:` or `apply failed:` messages.
- [x] 6.5 Capture the `sessionID` from the first structured opencode event for every PR-comment run, persist it in runtime state linked to the originating command request and any pending permission request, and include that session identity in execution logs.
- [x] 6.6 Add focused tests in `internal/exec`, `internal/workflow`, `internal/store`, and/or BDD coverage for intermediate empty error events followed by success, non-empty fallback failure summaries, session-ID persistence, and session-ID log correlation.

## 7. Verification

- [x] 7.1 Run `go test ./internal/exec ./internal/workflow ./tests/bdd && go test ./...` and confirm the large-event parser regressions and existing PR-command flows pass.

## 8. Documentation

- [x] 8.1 Update `docs/workflows.md`, `docs/operations.md`, and any relevant logging or troubleshooting docs so operators know Heimdall records opencode session IDs, follows terminal session outcomes instead of intermediate generic errors, and never reports blank refine/apply failure messages.
