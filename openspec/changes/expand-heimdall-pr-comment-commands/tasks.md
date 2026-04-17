## 1. Command Grammar And Configuration

- [ ] 1.1 Extend `internal/config/config.go` and `internal/config/config_test.go` so repository config keeps `DEFAULT_SPEC_WRITING_AGENT` activation-only, validates `ALLOWED_AGENTS`, and parses optional `/heimdall opencode` alias settings and permission profiles.
- [ ] 1.2 Replace the current slash-command parsing in `internal/slashcmd/handler.go` and `internal/slashcmd/handler_test.go` with the shared grammar for `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, `/heimdall opencode`, and `/heimdall approve`, including `--agent`, optional `change-name`, alias parsing, prompt-tail capture after `--`, and request-ID parsing for explicit approvals.
- [ ] 1.3 Update `internal/slashcmd/intake.go` plus the command-request persistence model in `internal/store/workflows.go` so queued PR-command jobs store the parsed command kind, selected agent, raw prompt tail, alias metadata, and approval request IDs needed by later execution.

## 2. PR Command Execution Pipeline

- [ ] 2.1 Implement PR-command execution requests and single-change resolution in the slash-command/workflow path so refine, apply, and generic opencode runs reject ambiguous pull requests instead of guessing the target change, while `/heimdall approve` resolves exactly one persisted pending permission request ID on the same pull request.
- [ ] 2.2 Extend `internal/exec/clients.go` with non-interactive session-aware refine/apply/generic-opencode execution methods that apply fixed permission profiles, classify blocked `needs_input` and `needs_permission` outcomes, and expose permission reply/resume operations for explicit approvals.
- [ ] 2.3 Add the queued worker handling that executes `pr_command_refine`, `pr_command_apply`, `pr_command_opencode`, and `pr_command_approve` jobs, persists pending permission-request/session state, resumes blocked runs after approval, updates command/job state, and keeps Heimdall responsible for commit/push behavior around successful runs.
- [ ] 2.4 Update the GitHub feedback path so successful, rejected, ambiguous, blocked, and resumed PR-command outcomes post the expected pull-request comments, including permission request IDs, exact approval commands, and sanitized blocker summaries.

## 3. Behavior And Unit Test Coverage

- [ ] 3.1 Rewrite `tests/features/pr_commands.feature` to cover `/heimdall refine` with explicit agents, `/heimdall apply`, `/opsx-apply` compatibility, `/heimdall opencode` aliases, ambiguous change targeting, blocked clarification results, blocked permission results, and explicit `/heimdall approve <request-id>` recovery.
- [ ] 3.2 Implement or update the Go BDD step bindings and fixtures in `tests/bdd` so the PR-command feature scenarios can execute against the new parser, queue, persisted pending-permission state, config, and GitHub-feedback behavior.
- [ ] 3.3 Add focused unit tests for config validation, command parsing, authorization, execution-profile selection, pending permission-request persistence and resolution, and blocked-result classification in `internal/config`, `internal/slashcmd`, `internal/store`, and `internal/exec`.

## 4. Verification

- [ ] 4.1 Run `go test ./...` and confirm the new PR-command grammar, blocked-run handling, and behavior scenarios pass before closing the change.
