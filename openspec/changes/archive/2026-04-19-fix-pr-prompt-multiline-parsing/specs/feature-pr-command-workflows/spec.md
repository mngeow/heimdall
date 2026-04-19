## MODIFIED Requirements

### Requirement: Refinement updates OpenSpec artifacts without applying implementation tasks
Heimdall MUST treat `/heimdall refine` as an artifact-only operation that targets a single OpenSpec change, requires an explicitly selected allowed agent, preserves the raw prompt tail after the first standalone `--`, and does not run implementation apply steps.

#### Scenario: User refines an open proposal with a selected agent
- **WHEN** an authorized user comments `/heimdall refine --agent gpt-5.4 -- Clarify rollback behavior and add non-goals.` on an active Heimdall pull request that resolves to one active change
- **THEN** Heimdall runs refinement for that change with agent `gpt-5.4`
- **AND** it updates the relevant OpenSpec proposal artifacts for that change
- **AND** it does not run implementation task execution as part of the refine command

#### Scenario: Multiline refine prompt is preserved after a trailing separator
- **WHEN** an authorized user comments `/heimdall refine --agent gpt-5.4 --` on one line of a Heimdall pull request comment and the following lines contain additional prompt text such as numbered implementation asks
- **THEN** Heimdall preserves those later lines as the raw prompt tail for the same refine command
- **AND** it passes that prompt tail into refinement execution instead of silently discarding the multiline body

#### Scenario: Mixed inline and multiline refine prompt is fully preserved
- **WHEN** an authorized user comments `/heimdall refine --agent gpt-5.4 -- Perfect, now add for me:` on one line and the following lines contain additional prompt text such as numbered or bulleted implementation asks
- **THEN** Heimdall preserves both the inline text after `--` and all subsequent lines as the raw prompt tail for the same refine command
- **AND** it passes the combined prompt tail into refinement execution instead of silently truncating at the command line

#### Scenario: Refine reports success only after real execution succeeds
- **WHEN** an authorized user comments `/heimdall refine --agent gpt-5.4 -- Clarify rollback behavior.` on a Heimdall pull request that resolves to one active change
- **THEN** Heimdall runs the real refine execution for that change before posting a success comment
- **AND** it does not publish a successful refine outcome when no refine execution was attempted

## ADDED Requirements

### Requirement: PR comment prompts MAY be delimited with triple backticks
Heimdall MUST support triple-backtick (```) delimiters around the prompt tail of an agent-driven PR command. When the prompt tail is wrapped in a complete triple-backtick block, Heimdall MUST strip the outer backticks and surrounding whitespace and use only the inner content as the raw prompt passed to execution.

#### Scenario: Refine prompt is wrapped in triple backticks
- **WHEN** an authorized user comments `/heimdall refine --agent gpt-5.4 --` followed by a prompt wrapped in triple backticks that contains markdown lists and literal `--` strings
- **THEN** Heimdall strips the outer triple backticks and passes only the inner content as the raw prompt tail
- **AND** the inner content preserves its newlines, bullet points, and literal `--` sequences intact

#### Scenario: Inline prompt with trailing triple backticks is preserved
- **WHEN** an authorized user comments `/heimdall refine --agent gpt-5.4 -- ```\nAdd error handling\n``` ` on one line or across multiple lines
- **THEN** Heimdall strips the outer triple backticks and passes only `Add error handling` as the raw prompt tail
