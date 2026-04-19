## ADDED Requirements

### Requirement: GitHub command feedback publishes acknowledgment reactions and dashboard links
The GitHub SCM service MUST support accepted-command feedback by adding the GitHub `eyes` reaction to the triggering pull-request comment, and for opencode-backed commands it MUST publish a pull-request reply comment containing an absolute Heimdall dashboard URL for the command detail view.

#### Scenario: GitHub feedback is published for an accepted opencode-backed command
- **WHEN** Heimdall accepts `/heimdall apply --agent gpt-5.4` on a Heimdall-managed pull request
- **THEN** the GitHub SCM service adds the `eyes` reaction to the triggering comment
- **AND** it posts one reply comment whose link resolves to the absolute Heimdall UI URL for that accepted command request

#### Scenario: Accepted non-opencode command is acknowledged without a live-output link
- **WHEN** Heimdall accepts `/heimdall status` on a Heimdall-managed pull request
- **THEN** the GitHub SCM service adds the `eyes` reaction to the triggering comment
- **AND** it does not publish a live-output dashboard link comment for that status request

### Requirement: GitHub feedback operations are idempotent at the command-request level
The GitHub SCM service MUST de-duplicate reaction and reply-comment attempts for the same command request so that retries or duplicate poll observations do not produce multiple identical reactions or link comments on the same pull-request comment.

#### Scenario: Duplicate reaction attempt is suppressed
- **WHEN** the GitHub SCM service receives a second request to add the `eyes` reaction for a command request that already has that reaction recorded as posted
- **THEN** it does not call the GitHub reactions API again
- **AND** it returns success without creating a duplicate reaction

#### Scenario: Duplicate link comment attempt is suppressed
- **WHEN** the GitHub SCM service receives a second request to post a live-output link comment for a command request that already has that link comment recorded as posted
- **THEN** it does not call the GitHub issue comments API again
- **AND** it returns success without creating a duplicate comment
