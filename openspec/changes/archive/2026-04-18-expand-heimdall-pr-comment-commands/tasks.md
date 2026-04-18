## 1. Command Grammar And Configuration

- [x] 1.1 Extend `internal/config/config.go` and `internal/config/config_test.go` so repository config keeps `DEFAULT_SPEC_WRITING_AGENT` activation-only, validates `ALLOWED_AGENTS`, and parses optional `/heimdall opencode` alias settings and permission profiles.
- [x] 1.2 Replace the current slash-command parsing in `internal/slashcmd/handler.go` and `internal/slashcmd/handler_test.go` with the shared grammar for `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, `/heimdall opencode`, and `/heimdall approve`, including `--agent`, optional `change-name`, alias parsing, prompt-tail capture after `--`, and request-ID parsing for explicit approvals.
- [x] 1.3 Update `internal/slashcmd/intake.go` plus the command-request persistence model in `internal/store/workflows.go` so queued PR-command jobs store the parsed command kind, selected agent, raw prompt tail, alias metadata, and approval request IDs needed by later execution.
- [x] 1.4 Update the PR-comment parser and parser tests so agent-driven commands preserve multiline prompt text after the first standalone `--`, including the case where the command line ends with `--` and the prompt continues on later lines.

## 2. PR Command Execution Pipeline

- [x] 2.1 Implement PR-command execution requests and single-change resolution in the slash-command/workflow path so refine, apply, and generic opencode runs reject ambiguous pull requests instead of guessing the target change, while `/heimdall approve` resolves exactly one persisted pending permission request ID on the same pull request.
- [x] 2.2 Extend `internal/exec/clients.go` with non-interactive session-aware refine/apply/generic-opencode execution methods that apply fixed permission profiles, classify blocked `needs_input` and `needs_permission` outcomes, and expose permission reply/resume operations for explicit approvals.
- [x] 2.3 Start a long-running PR-command worker during normal app startup so queued PR-command jobs are executed alongside the existing Linear and GitHub polling loops.
- [x] 2.4 Add the queued worker handling that executes `pr_command_status`, `pr_command_refine`, `pr_command_apply`, `pr_command_opencode`, and `pr_command_approve` jobs, loads command requests, pull requests, and repositories by durable stored IDs, persists pending permission-request/session state, resumes blocked runs after approval, and keeps Heimdall responsible for commit/push behavior around successful runs.
- [x] 2.5 Update the GitHub feedback and command/job state path so successful, rejected, ambiguous, blocked, resumed, and terminal missing-state outcomes post the expected pull-request comments and do not leave accepted commands silently queued.
- [x] 2.6 Tighten worker-side execution so refine, apply, and generic opencode commands resolve exactly one target change immediately before execution and reject missing or empty targets instead of running with a blank change name.
- [x] 2.7 Replace the placeholder refine-success path with real non-interactive refine execution that consumes the preserved prompt tail, reports truthful success/no-change/blocked/failure outcomes, and only comments success after the refine run actually completes.
- [x] 2.8 Implement `ExecuteStatus` in `internal/workflow/pr_commands.go` so it loads real pull-request-bound change state and comments an accurate status summary instead of a placeholder response.
- [x] 2.9 Implement `ExecuteRefine` in `internal/workflow/pr_commands.go` as the real refine orchestration path, including change resolution, prompt-tail use, execution, commit/push handling, and truthful PR feedback.
- [x] 2.10 Implement `ExecuteApply` in `internal/workflow/pr_commands.go` as the real apply orchestration path, including change resolution, execution, commit/push handling, and truthful PR feedback.
- [x] 2.11 Implement `ExecuteOpencode` in `internal/workflow/pr_commands.go` as the real allowlisted alias execution path, including change resolution, configured command execution, and truthful PR feedback.
- [x] 2.12 Implement `ExecuteApprove` in `internal/workflow/pr_commands.go` so it performs the real one-time permission reply and blocked-session resume path before reporting success.
- [x] 2.13 Update `internal/exec/clients.go` so refine and apply use the supported `opencode run` contract with positional messages, request `--format json`, and classify outcomes from structured event streams instead of keyword matching formatted output.
- [x] 2.14 Tighten blocked-result detection so only real `permission.asked` events create pending permission requests, extract non-empty request/session IDs, and treat CLI help or usage output as normal execution failures rather than approval blockers.
- [x] 2.15 Validate that the resolved change still exists in the current worktree before starting refine, apply, or generic opencode execution, and reject stale bindings when the change is missing.
- [x] 2.16 Update the PR-command worker and queue lifecycle so successful commands mark both the command request and the queue job as completed, releasing the pull-request lock for later same-PR commands.
- [x] 2.17 Implement `/heimdall approve` through the supported opencode permission-reply API and observe the resumed session until a real terminal outcome is available for PR feedback.
- [x] 2.18 Persist only valid pending permission requests with non-empty request/session IDs and originating command-request linkage, and fail loudly instead of publishing empty approval commands.

## 3. Behavior And Unit Test Coverage

- [x] 3.1 Rewrite `tests/features/pr_commands.feature` to cover `/heimdall refine` with explicit agents, `/heimdall apply`, `/opsx-apply` compatibility, `/heimdall opencode` aliases, ambiguous change targeting, blocked clarification results, blocked permission results, and explicit `/heimdall approve <request-id>` recovery.
- [x] 3.2 Implement or update the Go BDD step bindings and fixtures in `tests/bdd` so the PR-command feature scenarios can execute against the new parser, queue, persisted pending-permission state, config, and GitHub-feedback behavior.
- [x] 3.3 Add focused unit tests for config validation, command parsing, authorization, execution-profile selection, pending permission-request persistence and resolution, blocked-result classification, worker startup, status replies, durable-ID lookups, dispatch coverage, and command/job state transitions in `internal/config`, `internal/slashcmd`, `internal/store`, `internal/exec`, and `internal/workflow`.
- [x] 3.4 Update `docs/architecture.md` and `docs/workflows.md` so the runtime worker responsibility and queued PR-command execution path match the implemented behavior.
- [x] 3.5 Extend `tests/features/pr_commands.feature` with multiline refine-prompt scenarios and missing-target-change rejection scenarios that match real GitHub comment formatting.
- [x] 3.6 Update the BDD step bindings plus focused unit tests in `internal/slashcmd` and `internal/workflow` to cover multiline prompt preservation, worker-side change resolution, and truthful refine outcomes.
- [x] 3.7 Add focused unit and/or behavior coverage that proves `ExecuteStatus`, `ExecuteRefine`, `ExecuteApply`, `ExecuteOpencode`, and `ExecuteApprove` each perform real work and do not pass through placeholder success branches.
- [x] 3.8 Extend unit and behavior coverage for supported positional-message invocation, JSON event parsing, false-permission regressions from CLI help output, stale change rejection, non-empty pending-permission persistence, and successful queue-job completion.
- [x] 3.9 Add approval-resume coverage that proves Heimdall replies once to the exact request ID through the supported API, observes the resumed session outcome, and reports that real outcome back to the pull request.

## 4. Verification

- [x] 4.1 Run `go test ./...` and confirm the new PR-command grammar, blocked-run handling, and behavior scenarios pass before closing the change.
- [x] 4.2 Run `go test ./...` again and confirm multiline refine prompts, omitted change-name resolution, and real refine execution outcomes all pass with the updated behavior.
- [x] 4.3 Run `go test ./...` again plus a focused real-opencode/manual verification that CLI help output is treated as execution failure, real permission IDs are surfaced exactly, later approval resumes successfully, and successful jobs do not remain `running`.
