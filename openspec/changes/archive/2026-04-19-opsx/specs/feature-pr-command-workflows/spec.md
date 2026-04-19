## ADDED Requirements

### Requirement: Accepted PR commands receive immediate visible acknowledgment
Heimdall MUST give users immediate visible feedback when it accepts a new supported pull-request command. For every accepted non-duplicate command request on a Heimdall-managed pull request, Heimdall MUST add the GitHub `eyes` reaction to the triggering comment. For accepted commands that start or resume opencode execution, Heimdall MUST also post one pull-request reply comment that links to the Heimdall UI page for that command's live output.

#### Scenario: Accepted refine command gets reaction and live-output link
- **WHEN** an authorized user posts `/heimdall refine --agent gpt-5.4 -- Clarify rollback behavior.` on a Heimdall-managed pull request and a GitHub poll cycle accepts that comment as a new command request
- **THEN** Heimdall adds the GitHub `eyes` reaction to that exact comment
- **AND** Heimdall posts one pull-request reply comment with a Heimdall UI link for that command's live-output view

#### Scenario: Accepted status command gets recognition without a misleading run link
- **WHEN** an authorized user posts `/heimdall status` on a Heimdall-managed pull request and Heimdall accepts that comment as a new command request
- **THEN** Heimdall adds the GitHub `eyes` reaction to that exact comment
- **AND** it does not post a live-opencode output link that implies opencode execution for the status command

#### Scenario: Rejected or duplicate command does not get accepted-command feedback
- **WHEN** a pull-request comment is unsupported, unauthorized, not on a Heimdall-managed pull request, or is later observed again as a duplicate of an already accepted command request
- **THEN** Heimdall does not add accepted-command feedback that implies the comment was newly queued for execution
- **AND** it does not post a new live-output link comment for that rejected or duplicate observation

### Requirement: Accepted-command feedback is idempotent
Heimdall MUST ensure that accepted-command feedback (reactions and link comments) is posted at most once per command request, even if the GitHub poll cycle observes the same comment again or if a transient GitHub API failure triggers a retry.

#### Scenario: Duplicate poll observation does not create duplicate feedback
- **WHEN** a GitHub poll cycle observes the same pull-request comment a second time after Heimdall has already accepted it and posted the initial feedback
- **THEN** Heimdall does not post a second `eyes` reaction on that comment
- **AND** it does not post a second live-output link comment for the same command request

#### Scenario: Transient GitHub failure does not create duplicate link comment on retry
- **WHEN** Heimdall posts the initial `eyes` reaction successfully but the link comment fails due to a transient GitHub API error, and Heimdall retries the feedback step
- **THEN** the retry posts only the missing link comment
- **AND** it does not post a second `eyes` reaction or a second link comment

### Requirement: Partial acknowledgment failure does not block command execution
Heimdall MUST queue the command for worker execution even if the GitHub acknowledgment step fails partially. A failure to post the reaction or link comment MUST NOT prevent the PR-command worker from dequeuing and executing the accepted command.

#### Scenario: Link comment fails but command still runs
- **WHEN** Heimdall accepts an opencode-backed command and successfully adds the `eyes` reaction, but the link comment fails due to a transient GitHub API error
- **THEN** the command request is still queued for worker execution
- **AND** the PR-command worker eventually dequeues and runs the command

#### Scenario: Reaction fails but command still runs
- **WHEN** Heimdall accepts a command but the `eyes` reaction fails due to a transient GitHub API error
- **THEN** the command request is still queued for worker execution
- **AND** the link comment is still attempted and the worker proceeds normally if the link comment succeeds
