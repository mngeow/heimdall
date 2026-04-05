## MODIFIED Requirements

### Requirement: GitHub pull request activity is polled before processing
Symphony MUST poll GitHub for new pull request comments and relevant pull request lifecycle changes by using its GitHub App installation credentials, and it MUST not require inbound GitHub webhook delivery for the standard v1 deployment path. When a managed repository configures a PR monitor label, Symphony MUST narrow its GitHub polling scope to Symphony-managed pull requests that currently carry that label.

#### Scenario: GitHub poll sees a new command comment on a labeled Symphony-managed pull request
- **WHEN** Symphony runs a GitHub poll cycle for a managed repository, the repository config includes a PR monitor label, and a new issue comment appears on a Symphony-managed pull request that currently carries that label within the configured polling window
- **THEN** it makes that comment available to the runtime components responsible for authorization and command parsing
- **AND** the command-intake path does not depend on a public inbound webhook endpoint

#### Scenario: GitHub poll ignores an unlabeled Symphony pull request when label-scoped monitoring is configured
- **WHEN** Symphony polls a managed repository configured with PR monitor label `symphony-monitored` and observes comment or lifecycle activity on a Symphony-managed pull request that does not currently carry that label
- **THEN** it does not treat that activity as eligible monitoring input for command handling or pull request reconciliation
- **AND** it leaves recovery to the path that re-applies or backfills the configured label

#### Scenario: GitHub poll sees a relevant pull request state change on a labeled Symphony-managed pull request
- **WHEN** Symphony polls a managed repository configured with a PR monitor label and detects a relevant lifecycle change on a Symphony-managed pull request that currently carries that label
- **THEN** it makes that pull request change available to the runtime components responsible for pull request reconciliation
- **AND** the reconciliation path does not depend on inbound webhook delivery

## ADDED Requirements

### Requirement: Configured monitor labels are reconciled onto Symphony pull requests
When a managed repository configures a GitHub PR monitor label, Symphony MUST ensure that repository label exists and MUST add it to Symphony-created or adopted pull requests without removing unrelated labels.

#### Scenario: Configured monitor label is missing from the repository
- **WHEN** Symphony needs to publish or reconcile a Symphony pull request for a repository configured with PR monitor label `symphony-monitored` and the repository does not yet have a label with that name
- **THEN** it creates the repository label before or during PR reconciliation
- **AND** it applies that label to the target Symphony pull request

#### Scenario: Existing Symphony pull request is missing the configured monitor label
- **WHEN** Symphony reuses or reconciles an existing Symphony-managed pull request for a repository whose configured monitor label already exists in GitHub but is not yet attached to that pull request
- **THEN** it adds the configured label to that pull request
- **AND** it preserves any unrelated labels that were already present
