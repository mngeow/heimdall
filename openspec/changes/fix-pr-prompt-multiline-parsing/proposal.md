## Why

The PR comment parser silently discards multiline prompt content when a user places the `--` separator on the same line as the command. For example, a comment like `/heimdall refine --agent X -- Perfect, now add for me:\n1. first point\n2. second point` results in only `Perfect, now add for me:` being preserved; the numbered list is lost. This makes the refine command effectively a no-op because the actual instructions never reach opencode. Additionally, users want an unambiguous way to delimit complex multiline prompts (e.g., inside triple backticks) so that markdown formatting, bullet points, and even literal `--` strings inside the prompt are handled correctly.

## What Changes

- Fix the parser in `internal/slashcmd/handler.go` so that when a command line contains an inline ` -- ` separator, any subsequent lines in the same comment are **appended** to the prompt tail instead of being discarded.
- Add support for **triple-backtick delimiters** around the prompt tail. When the prompt text (inline or multiline) is wrapped in ```, the parser strips the outer backticks and uses only the inner content as the raw prompt.
- Strip `\r` carriage returns from each line during multiline reassembly to avoid polluting the prompt with CRLF artifacts from GitHub's API.
- Add focused unit tests and BDD assertions covering:
  - Mixed inline + multiline prompt capture.
  - Triple-backtick delimiter stripping.
  - CRLF line ending normalization.
- Update the no-op BDD step `heimdallShouldUpdateArtifactsWithFullPrompt` in `tests/bdd/steps_test.go` to perform real verification.
- Update durable specs for `feature-pr-command-workflows` and `service-execution-runtime` to codify the combined inline+multiline preservation and backtick-delimiter behavior.

## Capabilities

### New Capabilities
- *(none — this is a fix and refinement of existing parsing behavior)*

### Modified Capabilities
- `feature-pr-command-workflows`: Expand the multiline prompt preservation requirement to cover the case where the command line contains an inline ` -- ` separator **and** additional prompt lines follow. Add a scenario for triple-backtick prompt delimiters.
- `service-execution-runtime`: Expand the raw prompt tail preservation requirement to include triple-backtick stripping and CRLF normalization.

## Impact

- `internal/slashcmd/handler.go` — parser logic for prompt tail extraction.
- `internal/slashcmd/handler_test.go` — new unit test cases.
- `tests/bdd/steps_test.go` — real assertion implementation for multiline prompt scenarios.
- `openspec/specs/feature-pr-command-workflows/spec.md` — updated requirements and scenarios.
- `openspec/specs/service-execution-runtime/spec.md` — updated requirements and scenarios.
- No breaking changes to existing valid single-line commands or trailing-` -- ` multiline commands.
