## MODIFIED Requirements

### Requirement: Agent selection is explicit and policy-controlled
The execution runtime MUST use the repository's configured default spec-writing agent only for activation-triggered proposal generation. For `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, and `/heimdall opencode`, Heimdall MUST require an explicitly selected agent that is allowed for the repository, MUST preserve the raw prompt tail after the first standalone `--`, MUST resolve exactly one target change before execution starts, MUST verify that the resolved change exists in the active worktree before invoking opencode, and `/heimdall opencode` MUST also require an allowlisted command alias for that repository.

#### Scenario: Activation proposal is started
- **WHEN** an activated work item starts the proposal pull request workflow
- **THEN** Heimdall runs the local OpenCode execution by using the repository's configured default spec-writing agent
- **AND** it does not require per-run agent input for that activation path

#### Scenario: User runs refine with an allowed agent
- **WHEN** a pull request comment requests `/heimdall refine --agent claude-sonnet -- Clarify rollback behavior.` and the repository allows `claude-sonnet`
- **THEN** Heimdall runs the refinement execution by using the selected agent `claude-sonnet`
- **AND** it does not fall back to the repository default spec-writing agent for that comment-driven run

#### Scenario: User runs refine with a multiline prompt body
- **WHEN** a pull request comment requests `/heimdall refine --agent claude-sonnet --` on one line and additional prompt text continues on later lines
- **THEN** Heimdall preserves that later text as the prompt tail for the same refine run
- **AND** it passes the preserved prompt body into the refinement execution instead of truncating it at the first newline

#### Scenario: User runs refine with a mixed inline and multiline prompt body
- **WHEN** a pull request comment requests `/heimdall refine --agent claude-sonnet -- Perfect, now add for me:` on one line and additional prompt text continues on later lines
- **THEN** Heimdall preserves both the inline text after `--` and all subsequent lines as the prompt tail for the same refine run
- **AND** it passes the combined prompt body into the refinement execution instead of silently discarding the later lines

#### Scenario: User runs refine with a triple-backtick delimited prompt
- **WHEN** a pull request comment requests `/heimdall refine --agent claude-sonnet --` followed by a prompt wrapped in triple backticks
- **THEN** Heimdall strips the outer triple backticks and surrounding whitespace and passes only the inner content as the prompt tail
- **AND** it passes that inner content into the refinement execution

#### Scenario: Multiline prompt with CRLF line endings is normalized
- **WHEN** a pull request comment body contains CRLF (`\r\n`) line endings and the refine command uses a multiline prompt after a trailing `--`
- **THEN** Heimdall normalizes the prompt tail to use LF-only line endings by stripping `\r` characters during reassembly
- **AND** it passes the normalized prompt body into the refinement execution without embedded carriage returns

#### Scenario: User runs a PR command without an allowed agent
- **WHEN** a pull request comment requests `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, or `/heimdall opencode` with an agent that is not allowed for the repository
- **THEN** Heimdall does not start the requested execution
- **AND** it records and reports that the requested agent is not authorized for that repository

#### Scenario: User runs a PR command without a resolved target change
- **WHEN** a pull request comment requests `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, or `/heimdall opencode` without an explicit change name and Heimdall cannot resolve exactly one active change for that pull request
- **THEN** Heimdall does not start the requested execution with an empty change name
- **AND** it records and reports that the command target must be specified or cannot be resolved

#### Scenario: User runs a PR command against a stale bound change
- **WHEN** a pull request comment requests `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, or `/heimdall opencode`, Heimdall resolves exactly one change name from runtime state, and that change is missing from the worktree it is about to use
- **THEN** Heimdall does not start the requested execution against the missing change
- **AND** it records and reports that the resolved change does not exist in the active worktree
