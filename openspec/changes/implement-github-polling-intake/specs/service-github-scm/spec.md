## MODIFIED Requirements

### Requirement: GitHub comments and PR state are discovered through polling
Symphony MUST poll GitHub on a configured interval for newly created conversation comments on Symphony-managed pull requests and for pull request state changes needed for lifecycle reconciliation, MUST scope that polling to configured repositories and active Symphony pull request bindings, and v1 MUST not require a public inbound webhook endpoint to discover that data.

#### Scenario: Poll cycle finds a new pull request comment after the saved checkpoint
- **WHEN** Symphony polls GitHub for a configured repository and finds a newly created comment on a Symphony-managed pull request beyond the last saved checkpoint for that pull request
- **THEN** it records that comment as a command candidate for parsing and authorization
- **AND** it advances the saved polling progress so the same comment is not rediscovered as new on the next successful poll cycle

#### Scenario: Poll cycle finds a pull request state change after the saved checkpoint
- **WHEN** Symphony polls GitHub and observes a newer pull request state for a managed pull request than the last saved lifecycle observation
- **THEN** it updates the pull request state available to the runtime components responsible for binding and lifecycle synchronization
- **AND** it does not require inbound webhook delivery to detect the state change

#### Scenario: Polling resumes after a restart
- **WHEN** Symphony restarts with previously saved GitHub polling checkpoints for active Symphony pull requests
- **THEN** it resumes polling from that durable state instead of replaying all historical comments as new command requests
- **AND** it can still discover later comments and later pull request state changes for those pull requests
