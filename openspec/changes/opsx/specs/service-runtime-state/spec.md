## ADDED Requirements

### Requirement: Command-linked live run state is queryable from SQLite
Heimdall MUST persist command-linked live execution state for PR-comment opencode runs so the dashboard can resolve a command detail page from the moment the command is accepted through its terminal outcome. That live state MUST include the originating command-request linkage, current run status, canonical session ID when known, and ordered append-only human-readable output entries without requiring operators to reconstruct the run from host logs.

#### Scenario: Accepted command is visible before the first structured event arrives
- **WHEN** Heimdall accepts `/heimdall apply --agent gpt-5.4` and posts a dashboard link before the worker has observed any opencode output
- **THEN** the linked command view can still load by using persisted command-request state
- **AND** it shows queued or starting status instead of failing because no session ID exists yet

#### Scenario: Live output entries append to the same command-linked run
- **WHEN** later structured opencode events arrive for that accepted command and the first event reports `sessionID` `ses_abc`
- **THEN** Heimdall associates `ses_abc` and later output entries with the same command-linked timeline
- **AND** the dashboard can query those ordered entries without rereading host logs or external process output

#### Scenario: Terminal summary remains available on the same command detail page
- **WHEN** a command-linked opencode run reaches completed, blocked, or failed state
- **THEN** Heimdall persists the terminal status and summary on the same command-linked record
- **AND** the GitHub-linked dashboard page continues to resolve for later inspection of that command outcome

#### Scenario: Persisted output order supports live tail rendering
- **WHEN** Heimdall persists additional human-readable output entries for an already visible command-linked run
- **THEN** it stores those entries after the earlier entries for that same run instead of rewriting the history out of order
- **AND** the dashboard can query them in append order for a live-tail view of the output

### Requirement: Command run state transitions are persisted durably
Heimdall MUST persist state transitions for command-linked opencode runs so the dashboard can reflect the current and historical execution state accurately.

#### Scenario: Run state transitions from queued to starting to running
- **WHEN** a command-linked run progresses from `queued` to `starting` and then to `running`
- **THEN** each state change is persisted on the command-linked record
- **AND** the dashboard reflects the latest state without ambiguity

#### Scenario: Terminal state is persisted with summary
- **WHEN** a command-linked run reaches `completed`, `failed`, or `blocked`
- **THEN** Heimdall persists the terminal state together with a concise human-readable summary
- **AND** the timestamp of the terminal transition is recorded
