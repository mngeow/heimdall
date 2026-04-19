## 1. Parser Core Fix

- [x] 1.1 Update `Parse()` in `internal/slashcmd/handler.go` to capture subsequent lines when an inline ` -- ` separator is present. After extracting the inline prompt tail, append any lines after the command line separated by `\n`.
- [x] 1.2 Add `stripCR` helper in `internal/slashcmd/handler.go` to remove `\r` from each line during multiline reassembly.
- [x] 1.3 Add `stripBacktickBlock` helper in `internal/slashcmd/handler.go` that trims outer triple-backtick wrappers (``` with optional surrounding whitespace and newlines) from a fully assembled prompt tail.
- [x] 1.4 Wire `stripCR` and `stripBacktickBlock` into the prompt tail assembly path so they run after inline and multiline text is combined.

## 2. Unit Tests

- [x] 2.1 Add `handler_test.go` case: mixed inline + multiline prompt (`/heimdall refine --agent X -- intro:\n1. a\n2. b`) asserts `PromptTail == "intro:\n1. a\n2. b"`.
- [x] 2.2 Add `handler_test.go` case: triple-backtick wrapped prompt asserts backticks are stripped and inner content is preserved.
- [x] 2.3 Add `handler_test.go` case: CRLF input (`\r\n` line endings) asserts `\r` is absent from the resulting `PromptTail`.
- [x] 2.4 Add `handler_test.go` case: trailing ` -- ` with no subsequent lines asserts only inline text is captured (existing behavior preserved).
- [x] 2.5 Run `go test ./internal/slashcmd/...` and confirm all existing and new cases pass.

## 3. BDD Tests and Fixtures

- [x] 3.1 Implement `heimdallShouldUpdateArtifactsWithFullPrompt` in `tests/bdd/steps_test.go` to assert that the recorded `CommandRequest.PromptTail` matches the full multiline comment body (not a truncated version).
- [x] 3.2 Add a new BDD step `theUserCommentsAMultilineRefineWithInlineSeparator` that posts a comment matching the mixed inline+multiline format.
- [x] 3.3 Add a new BDD step `heimdallShouldPreserveTheFullPromptTail` that verifies the prompt stored in the command request contains both the inline and subsequent lines.
- [x] 3.4 Add a BDD scenario in `tests/features/pr_commands.feature` for the mixed inline+multiline prompt case.
- [x] 3.5 Add a BDD scenario in `tests/features/pr_commands.feature` for the triple-backtick delimited prompt case.
- [x] 3.6 Run the BDD test suite (`go test ./tests/bdd/...`) and confirm all scenarios pass.

## 4. Durable Spec Updates

- [x] 4.1 Apply the `feature-pr-command-workflows` delta spec from `openspec/changes/fix-pr-prompt-multiline-parsing/specs/feature-pr-command-workflows/spec.md` into `openspec/specs/feature-pr-command-workflows/spec.md`.
- [x] 4.2 Apply the `service-execution-runtime` delta spec from `openspec/changes/fix-pr-prompt-multiline-parsing/specs/service-execution-runtime/spec.md` into `openspec/specs/service-execution-runtime/spec.md`.

## 5. Verification

- [x] 5.1 Run `go test ./...` and confirm all unit and BDD tests pass.
- [x] 5.2 Review `openspec status --change fix-pr-prompt-multiline-parsing` to ensure all artifacts are marked complete.
