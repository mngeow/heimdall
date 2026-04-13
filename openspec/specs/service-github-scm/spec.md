# Service: GitHub SCM

## ADDED Requirements

### Requirement: GitHub integration uses a GitHub App
Heimdall MUST use a GitHub App for GitHub API and git push operations and MUST rely on short-lived installation tokens for repository mutations.

#### Scenario: Heimdall needs to push a bootstrap branch
- **WHEN** Heimdall needs to push an activation-triggered bootstrap branch to GitHub
- **THEN** it mints a short-lived installation token from its GitHub App configuration
- **AND** it uses that token for the branch push instead of a long-lived personal access token

### Requirement: GitHub pull request activity is polled before processing
Heimdall MUST poll GitHub for new pull request comments and relevant pull request lifecycle changes by using its GitHub App installation credentials, and it MUST not require inbound GitHub webhook delivery for the standard v1 deployment path. When a managed repository configures a PR monitor label, Heimdall MUST narrow its GitHub polling scope to Heimdall-managed pull requests that currently carry that label.

#### Scenario: GitHub poll sees a new command comment on a labeled Heimdall-managed pull request
- **WHEN** Heimdall runs a GitHub poll cycle for a managed repository, the repository config includes a PR monitor label, and a new issue comment appears on a Heimdall-managed pull request that currently carries that label within the configured polling window
- **THEN** it makes that comment available to the runtime components responsible for authorization and command parsing
- **AND** the command-intake path does not depend on a public inbound webhook endpoint

#### Scenario: GitHub poll ignores an unlabeled Heimdall pull request when label-scoped monitoring is configured
- **WHEN** Heimdall polls a managed repository configured with PR monitor label `heimdall-monitored` and observes comment or lifecycle activity on a Heimdall-managed pull request that does not currently carry that label
- **THEN** it does not treat that activity as eligible monitoring input for command handling or pull request reconciliation
- **AND** it leaves recovery to the path that re-applies or backfills the configured label

#### Scenario: GitHub poll sees a relevant pull request state change on a labeled Heimdall-managed pull request
- **WHEN** Heimdall polls a managed repository configured with a PR monitor label and detects a relevant lifecycle change on a Heimdall-managed pull request that currently carries that label
- **THEN** it makes that pull request change available to the runtime components responsible for pull request reconciliation
- **AND** the reconciliation path does not depend on inbound webhook delivery

### Requirement: GitHub repository operations support the automation lifecycle
The GitHub SCM service MUST create or reuse branches, open or reuse pull requests, and publish pull request comments for Heimdall workflows.

#### Scenario: Heimdall publishes a bootstrap pull request
- **WHEN** an activation-triggered bootstrap workflow succeeds for a mapped repository
- **THEN** the GitHub SCM service ensures the bootstrap branch exists in the repository
- **AND** it opens or reuses a pull request against the configured base branch with the bootstrap title and description derived from the source issue

### Requirement: Configured monitor labels are reconciled onto Heimdall pull requests
When a managed repository configures a GitHub PR monitor label, Heimdall MUST ensure that repository label exists and MUST add it to Heimdall-created or adopted pull requests without removing unrelated labels.

#### Scenario: Configured monitor label is missing from the repository
- **WHEN** Heimdall needs to publish or reconcile a Heimdall pull request for a repository configured with PR monitor label `heimdall-monitored` and the repository does not yet have a label with that name
- **THEN** it creates the repository label before or during PR reconciliation
- **AND** it applies that label to the target Heimdall pull request

#### Scenario: Existing Heimdall pull request is missing the configured monitor label
- **WHEN** Heimdall reuses or reconciles an existing Heimdall-managed pull request for a repository whose configured monitor label already exists in GitHub but is not yet attached to that pull request
- **THEN** it adds the configured label to that pull request
- **AND** it preserves any unrelated labels that were already present

### Requirement: GitHub command actors are authorized before mutation
Heimdall MUST authorize pull request command actors based on collaborator rights and repository allowlists before running repository mutation workflows.

#### Scenario: Unauthorized user attempts to run apply
- **WHEN** a GitHub user without the required collaborator rights or allowlist entry comments with `/opsx-apply --agent gpt-5.4`
- **THEN** Heimdall rejects the mutation request
- **AND** it does not start an apply workflow for that comment

### Requirement: Heimdall mutation commands are limited to Heimdall-created pull requests
Heimdall MUST accept repository mutation commands only on pull requests that were created or adopted as Heimdall automation pull requests.

#### Scenario: Command is posted on a non-Heimdall pull request
- **WHEN** an otherwise valid Heimdall mutation command is posted on a pull request that is not bound to a Heimdall workflow
- **THEN** Heimdall does not mutate the repository from that comment
- **AND** it records that the pull request is not eligible for Heimdall command execution
