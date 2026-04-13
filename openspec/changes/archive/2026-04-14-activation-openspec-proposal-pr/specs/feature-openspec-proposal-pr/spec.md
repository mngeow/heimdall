## REMOVED Requirements

### Requirement: Activation bootstrap is seeded from work item content
**Reason**: Activation now seeds durable OpenSpec proposal generation instead of a temporary bootstrap file mutation.
**Migration**: Use `Activation proposal is seeded from work item content`.

### Requirement: Initial activation bootstrap does not require OpenSpec change creation
**Reason**: Activation now creates or reuses a real OpenSpec change before proposal artifacts are generated.
**Migration**: Use `Activation proposal creates or reuses an OpenSpec change before artifact generation`.

### Requirement: Bootstrap changes are committed, pushed, and opened as a pull request
**Reason**: Activation now commits generated OpenSpec proposal artifacts instead of bootstrap-only repository changes.
**Migration**: Use `Activation proposal artifacts are committed, pushed, and opened as a pull request`.

### Requirement: Bootstrap pull requests preserve source issue context
**Reason**: Activation now publishes proposal-focused pull requests rather than bootstrap pull requests.
**Migration**: Use `Activation proposal pull requests preserve source issue context`.

## MODIFIED Requirements

### Requirement: Branches are deterministic
Heimdall MUST use deterministic branch naming for activation-triggered OpenSpec proposal generation so retries reconcile existing work instead of creating duplicate work branches.

#### Scenario: Heimdall creates a branch name for an activated work item
- **WHEN** Heimdall prepares a proposal branch for work item `ENG-123` whose description yields slug `add-rate-limiting`
- **THEN** it names the branch `heimdall/ENG-123-add-rate-limiting`
- **AND** it reuses that same branch identity on later retries for the same work item and repository

## ADDED Requirements

### Requirement: Activation proposal is seeded from work item content
Heimdall MUST extract the activated work item's title and description and MUST use that content as the seed for activation-triggered OpenSpec proposal generation.

#### Scenario: Heimdall prepares proposal generation from an activated issue
- **WHEN** Heimdall handles an activated work item that does not yet have an active automation binding in the target repository
- **THEN** it extracts the work item title and description for proposal-generation context
- **AND** it passes that context and the deterministic change name into the local OpenCode execution that will create or update the OpenSpec change artifacts

### Requirement: Activation proposal creates or reuses an OpenSpec change before artifact generation
Heimdall MUST create or reuse a deterministic OpenSpec change in the repository worktree before generating activation proposal artifacts, and MUST use OpenSpec CLI status and instruction output to determine which artifacts are required for implementation readiness.

#### Scenario: Heimdall prepares proposal generation for an activated issue
- **WHEN** Heimdall prepares the activation-triggered proposal workflow for work item `ENG-123`
- **THEN** it creates or reuses the OpenSpec change `eng-123-add-rate-limiting`
- **AND** it reads OpenSpec CLI status and artifact instructions before generating the change artifacts

### Requirement: Activation proposal uses deterministic change names
Heimdall MUST derive a deterministic OpenSpec change name from the activated work item's key and normalized slug so retries and later pull request commands target the same change.

#### Scenario: Heimdall creates a change name for an activated work item
- **WHEN** Heimdall prepares a change name for work item `ENG-123` whose title yields slug `add-rate-limiting`
- **THEN** it names the change `eng-123-add-rate-limiting`
- **AND** it reuses that same change identity on later retries for the same work item and repository

### Requirement: Activation proposal artifacts are committed, pushed, and opened as a pull request
Heimdall MUST commit the activation-triggered OpenSpec proposal artifacts to the branch, push the branch to GitHub, and open or reuse a pull request against the configured base branch.

#### Scenario: Heimdall completes proposal generation for a work item
- **WHEN** Heimdall finishes the activation-triggered OpenSpec proposal execution for a routed work item and repository changes exist
- **THEN** it commits and pushes the proposal branch
- **AND** it opens or reuses a pull request targeting the configured base branch

### Requirement: Activation proposal retries recover incomplete deterministic workspaces
Heimdall MUST treat the deterministic activation proposal branch and worktree as recoverable workflow state even when a prior activation run failed before the proposal binding became active.

#### Scenario: A retry continues from a partially prepared proposal workspace
- **WHEN** a previous activation attempt already created or registered the deterministic proposal branch or worktree for a work item
- **AND** that earlier attempt failed before the proposal branch was committed, pushed, or bound as active
- **THEN** Heimdall reuses or repairs that deterministic workspace on retry instead of treating it as a brand-new branch/worktree request
- **AND** later proposal commit, push, and pull request steps still target the same deterministic proposal branch identity

### Requirement: Activation proposal pull requests preserve source issue context
Heimdall MUST publish a pull request title and description that reflect the source issue and the generated OpenSpec change.

#### Scenario: Activation proposal pull request is opened
- **WHEN** Heimdall creates or reuses the activation proposal pull request for a work item
- **THEN** the pull request title references the issue title and indicates that the branch contains an OpenSpec proposal
- **AND** the pull request description includes the issue description, the generated change name, and a short summary of the generated OpenSpec artifacts
