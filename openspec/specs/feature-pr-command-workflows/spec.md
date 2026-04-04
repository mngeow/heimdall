# Feature: Pull Request Command Workflows

## ADDED Requirements

### Requirement: Symphony accepts a narrow set of slash commands on automation pull requests
Symphony MUST accept only the documented slash command surface on pull requests that it created and MUST ignore unsupported mutation commands.

#### Scenario: Supported command is posted on a Symphony pull request
- **WHEN** an authorized user comments with `/symphony status`, `/symphony refine`, `/opsx-apply`, or `/opsx-archive` on a Symphony-created pull request
- **THEN** Symphony parses the command and enqueues the matching workflow action
- **AND** Symphony links the command request to the target pull request and repository binding

#### Scenario: Unsupported mutation command is posted
- **WHEN** a pull request comment contains an unsupported Symphony mutation command
- **THEN** Symphony does not run repository mutation logic for that comment
- **AND** Symphony records that the command was ignored or rejected

### Requirement: Refinement updates OpenSpec artifacts without applying implementation tasks
Symphony MUST treat `/symphony refine` as an artifact-only operation that updates the relevant OpenSpec files and does not run implementation apply steps.

#### Scenario: User refines an open proposal
- **WHEN** an authorized user comments `/symphony refine Clarify rollback behavior and add non-goals.` on an active Symphony pull request
- **THEN** Symphony updates the relevant OpenSpec proposal artifacts for that change
- **AND** Symphony does not run implementation task execution as part of the refine command

### Requirement: Apply uses an allowed agent and commits results to the same branch
Symphony MUST run `/opsx-apply` only with an agent allowed for the target repository and MUST commit the resulting task and code changes back to the same proposal branch.

#### Scenario: Authorized apply command selects an allowed agent
- **WHEN** an authorized user comments `/opsx-apply --agent gpt-5.4` on a Symphony pull request whose repository allows `gpt-5.4`
- **THEN** Symphony runs the apply workflow with that selected agent
- **AND** Symphony commits and pushes the resulting task updates and implementation changes to the same branch

### Requirement: Duplicate or edited comments are safe
Symphony MUST deduplicate command execution by comment identity and MUST ignore comment edits in v1.

#### Scenario: GitHub redelivers the same comment event
- **WHEN** GitHub redelivers a previously processed comment event for the same pull request comment
- **THEN** Symphony records the duplicate request without re-running the underlying mutation workflow
- **AND** the repository state remains unchanged by the duplicate delivery

#### Scenario: Existing command comment is edited
- **WHEN** a user edits a previously created command comment on a Symphony pull request
- **THEN** Symphony does not treat the edit as a new command request
- **AND** no new mutation workflow is started from the edit event
