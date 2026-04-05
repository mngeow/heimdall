# Service: GitHub SCM

## ADDED Requirements

### Requirement: GitHub integration uses a GitHub App
Symphony MUST use a GitHub App for GitHub API and git push operations and MUST rely on short-lived installation tokens for repository mutations.

#### Scenario: Symphony needs to push a bootstrap branch
- **WHEN** Symphony needs to push an activation-triggered bootstrap branch to GitHub
- **THEN** it mints a short-lived installation token from its GitHub App configuration
- **AND** it uses that token for the branch push instead of a long-lived personal access token

### Requirement: GitHub pull request activity is polled before processing
Symphony MUST poll GitHub for new pull request comments and relevant pull request lifecycle changes by using its GitHub App installation credentials, and it MUST not require inbound GitHub webhook delivery for the standard v1 deployment path.

#### Scenario: GitHub poll sees a new command comment on a Symphony-managed pull request
- **WHEN** Symphony runs a GitHub poll cycle for a managed repository and observes a new issue comment on a Symphony-managed pull request within the configured polling window
- **THEN** it makes that comment available to the runtime components responsible for authorization and command parsing
- **AND** the command-intake path does not depend on a public inbound webhook endpoint

#### Scenario: GitHub poll sees a relevant pull request state change
- **WHEN** Symphony polls a managed repository and detects a relevant lifecycle change on a Symphony-managed pull request
- **THEN** it makes that pull request change available to the runtime components responsible for pull request reconciliation
- **AND** the reconciliation path does not depend on inbound webhook delivery

### Requirement: GitHub repository operations support the automation lifecycle
The GitHub SCM service MUST create or reuse branches, open or reuse pull requests, and publish pull request comments for Symphony workflows.

#### Scenario: Symphony publishes a bootstrap pull request
- **WHEN** an activation-triggered bootstrap workflow succeeds for a mapped repository
- **THEN** the GitHub SCM service ensures the bootstrap branch exists in the repository
- **AND** it opens or reuses a pull request against the configured base branch with the bootstrap title and description derived from the source issue

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
