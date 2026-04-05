## MODIFIED Requirements

### Requirement: Activation bootstrap is seeded from work item content
Symphony MUST extract the activated work item's title and description and MUST use that content as the seed for the activation-triggered bootstrap pull request flow.

#### Scenario: Symphony prepares a bootstrap run from an activated issue
- **WHEN** Symphony handles an activated work item that does not yet have an active automation binding in the target repository
- **THEN** it extracts the work item title and description for the bootstrap prompt context
- **AND** it passes that context into the local OpenCode execution that will create the initial repository change

### Requirement: Initial activation bootstrap does not require OpenSpec change creation
Symphony MUST NOT depend on OpenSpec change creation or OpenSpec artifact generation to complete the initial activation-triggered bootstrap pull request flow.

#### Scenario: Activation bootstrap is prepared for execution
- **WHEN** Symphony prepares the activation-triggered bootstrap workflow for a routed work item
- **THEN** it does not require an OpenSpec change to exist before creating the branch and worktree
- **AND** it proceeds by invoking the local OpenCode bootstrap execution directly from the issue seed data

### Requirement: Branches are deterministic
Symphony MUST use deterministic branch naming for activation bootstrap so retries reconcile existing work instead of creating duplicate work branches.

#### Scenario: Symphony creates a branch name for an activated work item
- **WHEN** Symphony prepares a bootstrap branch for work item `ENG-123` whose description yields slug `add-rate-limiting`
- **THEN** it names the branch `symphony/ENG-123-add-rate-limiting`
- **AND** it reuses that same branch identity on later retries for the same work item and repository

### Requirement: Bootstrap changes are committed, pushed, and opened as a pull request
Symphony MUST commit the activation-triggered bootstrap changes to the branch, push the branch to GitHub, and open or reuse a pull request against the configured base branch.

#### Scenario: Symphony completes bootstrap scaffolding for a work item
- **WHEN** Symphony finishes the activation-triggered OpenCode bootstrap execution for a routed work item and repository changes exist
- **THEN** it commits and pushes the bootstrap branch
- **AND** it opens or reuses a pull request targeting the configured base branch

### Requirement: Bootstrap pull requests preserve source issue context
Symphony MUST publish a pull request title and description that reflect the source issue and the generated bootstrap change.

#### Scenario: Bootstrap pull request is opened
- **WHEN** Symphony creates or reuses the bootstrap pull request for a work item
- **THEN** the pull request title references the issue title
- **AND** the pull request description includes the issue description and a short summary of the generated bootstrap change
