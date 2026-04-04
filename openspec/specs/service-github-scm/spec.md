# Service: GitHub SCM

## ADDED Requirements

### Requirement: GitHub integration uses a GitHub App
Symphony MUST use a GitHub App for GitHub API and git push operations and MUST rely on short-lived installation tokens for repository mutations.

#### Scenario: Symphony needs to push a proposal branch
- **WHEN** Symphony needs to push a proposal or apply branch to GitHub
- **THEN** it mints a short-lived installation token from its GitHub App configuration
- **AND** it uses that token for the branch push instead of a long-lived personal access token

### Requirement: GitHub webhooks are verified before processing
Symphony MUST verify the GitHub webhook signature before parsing or acting on incoming webhook payloads and MUST support at minimum the `issue_comment` event for pull request command handling and the `pull_request` event for pull request lifecycle reconciliation.

#### Scenario: GitHub sends an issue comment webhook
- **WHEN** Symphony receives a GitHub `issue_comment` webhook delivery
- **THEN** it verifies the delivery against the configured webhook secret before command parsing occurs
- **AND** it rejects the request if signature verification fails

#### Scenario: GitHub sends a pull request webhook
- **WHEN** Symphony receives a GitHub `pull_request` webhook delivery for a repository managed by Symphony
- **THEN** it verifies the delivery against the configured webhook secret before reconciliation logic runs
- **AND** it makes that pull request event available to the runtime components responsible for binding and lifecycle synchronization

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
