# Service: GitHub SCM

## ADDED Requirements

### Requirement: GitHub integration uses a GitHub App
Symphony MUST use a GitHub App for GitHub API and git push operations and MUST rely on short-lived installation tokens for repository mutations.

#### Scenario: Symphony needs to push a proposal branch
- **WHEN** Symphony needs to push a proposal or apply branch to GitHub
- **THEN** it mints a short-lived installation token from its GitHub App configuration
- **AND** it uses that token for the branch push instead of a long-lived personal access token

### Requirement: GitHub comments and PR state are discovered through polling
Symphony MUST poll GitHub for new comments on Symphony-managed pull requests and for pull request state changes needed for lifecycle reconciliation, and v1 MUST not require a public inbound webhook endpoint to discover that data.

#### Scenario: Poll cycle finds a new pull request comment
- **WHEN** Symphony polls GitHub and finds a new comment on a Symphony-managed pull request
- **THEN** it records that comment as a command candidate for parsing and authorization
- **AND** it does not require inbound webhook delivery for the comment to be seen

#### Scenario: Poll cycle finds a pull request state change
- **WHEN** Symphony polls GitHub and observes a state change on a managed pull request
- **THEN** it makes that pull request state available to the runtime components responsible for binding and lifecycle synchronization
- **AND** it does not require public webhook delivery to detect the change

### Requirement: GitHub repository operations support the automation lifecycle
The GitHub SCM service MUST create or reuse branches, open or reuse pull requests, and publish pull request comments for Symphony workflows.

#### Scenario: Symphony publishes a proposal pull request
- **WHEN** a proposal workflow succeeds for a mapped repository
- **THEN** the GitHub SCM service ensures the proposal branch exists in the repository
- **AND** it opens or reuses a pull request against the configured base branch and publishes the related Symphony status comment

### Requirement: GitHub command actors are authorized before mutation
Symphony MUST authorize pull request command actors based on collaborator rights and repository allowlists before running repository mutation workflows.

#### Scenario: Unauthorized user attempts to run apply
- **WHEN** a GitHub user without the required collaborator rights or allowlist entry comments with `/opsx-apply --agent gpt-5.4`
- **THEN** Symphony rejects the mutation request
- **AND** it does not start an apply workflow for that comment

### Requirement: Symphony mutation commands are limited to Symphony-created pull requests
Symphony MUST accept repository mutation commands only on pull requests that were created or adopted as Symphony automation pull requests.

#### Scenario: Command is posted on a non-Symphony pull request
- **WHEN** an otherwise valid Symphony mutation command is posted on a pull request that is not bound to a Symphony workflow
- **THEN** Symphony does not mutate the repository from that comment
- **AND** it records that the pull request is not eligible for Symphony command execution
