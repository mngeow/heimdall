# Feature: OpenSpec Proposal Pull Request

## ADDED Requirements

### Requirement: Proposal generation is seeded from work item content
Symphony MUST create or reuse an OpenSpec change for an activated work item and MUST generate the artifacts required for implementation readiness from the work item title and description.

#### Scenario: Symphony creates a new change from an activated issue
- **WHEN** Symphony handles an activated work item that does not yet have an OpenSpec change in the target repository
- **THEN** Symphony creates a new OpenSpec change for that work item
- **AND** Symphony generates the required artifacts using the work item title and description as the proposal seed

### Requirement: Proposal generation follows OpenSpec workflow state
Symphony MUST use OpenSpec CLI JSON output as the source of truth for artifact status, dependency order, and apply readiness.

#### Scenario: Artifact generation order is determined for a change
- **WHEN** Symphony prepares to generate or update proposal artifacts for a change
- **THEN** it reads OpenSpec status and instructions from CLI JSON output
- **AND** it generates artifacts in the dependency order returned by OpenSpec instead of inferring the order from filesystem conventions alone

### Requirement: Branches and change names are deterministic
Symphony MUST use deterministic branch and change naming so retries reconcile existing work instead of creating duplicate work branches.

#### Scenario: Symphony creates branch and change names for a work item
- **WHEN** Symphony prepares a proposal branch for work item `ENG-123` with slug `add-rate-limit`
- **THEN** it names the branch `symphony/ENG-123-add-rate-limit`
- **AND** it names the OpenSpec change `ENG-123-add-rate-limit`

### Requirement: Proposal artifacts are committed, pushed, and opened as a pull request
Symphony MUST commit generated proposal artifacts to the proposal branch, push the branch to GitHub, and open or reuse a pull request against the configured base branch.

#### Scenario: Symphony completes proposal scaffolding for a work item
- **WHEN** Symphony finishes generating the required proposal artifacts for a routed work item
- **THEN** it commits and pushes the proposal branch
- **AND** it opens or reuses a pull request targeting the configured base branch

### Requirement: Proposal pull requests advertise next actions
Symphony MUST publish a pull request comment that identifies the generated change and the supported next commands after proposal creation.

#### Scenario: Proposal pull request is opened
- **WHEN** Symphony creates or reuses the proposal pull request for a work item
- **THEN** it comments with the change name and current proposal status
- **AND** it lists the supported next actions for refinement and apply workflows
