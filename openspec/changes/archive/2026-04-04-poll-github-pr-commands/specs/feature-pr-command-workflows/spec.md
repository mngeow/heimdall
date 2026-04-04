## MODIFIED Requirements

### Requirement: Symphony accepts a narrow set of slash commands on automation pull requests
Symphony MUST detect only the documented slash command surface by polling GitHub comments on pull requests that it created and MUST ignore unsupported mutation commands.

#### Scenario: Supported command is discovered during polling on a Symphony pull request
- **WHEN** an authorized user posts `/symphony status`, `/symphony refine`, `/opsx-apply`, or `/opsx-archive` on a Symphony-created pull request and a GitHub poll cycle observes that new comment
- **THEN** Symphony parses the command and enqueues the matching workflow action
- **AND** Symphony links the command request to the target pull request and repository binding

#### Scenario: Unsupported mutation command is discovered during polling
- **WHEN** a GitHub poll cycle observes a pull request comment that contains an unsupported Symphony mutation command
- **THEN** Symphony does not run repository mutation logic for that comment
- **AND** Symphony records that the command was ignored or rejected

### Requirement: Duplicate or edited comments are safe
Symphony MUST deduplicate command execution by comment identity, MUST ignore comment edits in v1, and MUST remain safe when overlapping GitHub poll windows observe the same comment more than once.

#### Scenario: Overlapping GitHub polls observe the same command comment twice
- **WHEN** two GitHub poll cycles observe the same previously posted command comment for the same pull request
- **THEN** Symphony records the later observation as duplicate without re-running the underlying mutation workflow
- **AND** the repository state remains unchanged by the duplicate observation

#### Scenario: Existing command comment is edited after initial detection
- **WHEN** a user edits a previously created command comment on a Symphony pull request after Symphony already observed the original comment
- **THEN** Symphony does not treat the edit as a new command request
- **AND** no new mutation workflow is started from the edited comment
