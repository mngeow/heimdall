## MODIFIED Requirements

### Requirement: Activation proposal creates or reuses an OpenSpec change before artifact generation
Heimdall MUST create or reuse a deterministic OpenSpec change in the repository worktree before generating activation proposal artifacts, and MUST use the normalized Linear ticket title as the basis for that change name while continuing to use OpenSpec CLI status and instruction output to determine which artifacts are required for implementation readiness.

#### Scenario: Heimdall prepares proposal generation for an activated issue
- **WHEN** Heimdall prepares the activation-triggered proposal workflow for work item `ENG-123` whose title normalizes to `add-rate-limiting`
- **THEN** it creates or reuses the OpenSpec change `add-rate-limiting`
- **AND** it reads OpenSpec CLI status and artifact instructions before generating the change artifacts

### Requirement: Activation proposal uses deterministic change names
Heimdall MUST derive a deterministic OpenSpec change name from the activated work item's title by normalizing it into a kebab-case slug. The normalization MUST lowercase letters, convert spaces to hyphens, collapse repeated whitespace or separator runs into a single hyphen, trim leading and trailing hyphens, and strip unsupported punctuation so retries and later pull request commands target the same change.

#### Scenario: Heimdall creates a change name from a spaced Linear title
- **WHEN** Heimdall prepares a change name for work item `ENG-123` with title `Add   Rate Limiting`
- **THEN** it names the change `add-rate-limiting`
- **AND** it reuses that same change identity on later retries for the same work item and repository

#### Scenario: Heimdall strips punctuation while normalizing the title
- **WHEN** Heimdall prepares a change name for work item `ENG-123` with title `Feature: add rate limiting, please`
- **THEN** it names the change `feature-add-rate-limiting-please`
- **AND** it does not preserve spaces or punctuation in the canonical OpenSpec change name
