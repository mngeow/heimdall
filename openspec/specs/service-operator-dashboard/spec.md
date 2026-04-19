# Service: Operator Dashboard

## ADDED Requirements

### Requirement: Heimdall serves an embedded private operator dashboard
Heimdall MUST serve a private operator dashboard from its existing HTTP server, and the dashboard MUST be rendered by the same Go application rather than a separately deployed frontend service.

#### Scenario: Operator opens the dashboard root
- **WHEN** an operator requests Heimdall's dashboard root on a running instance
- **THEN** Heimdall returns a server-rendered HTML page from its existing operator HTTP surface
- **AND** the dashboard is available without requiring a separate UI deployment artifact

### Requirement: Dashboard overview summarizes current queue and automation state
Heimdall MUST provide an overview screen that summarizes current queued work and active automation state from durable runtime records.

#### Scenario: Operator opens the overview screen
- **WHEN** an operator requests the dashboard overview
- **THEN** Heimdall shows summary counts for tracked work items, active pull requests, and current workflow or job state
- **AND** the overview links the operator into more detailed work-item and pull-request screens

### Requirement: Dashboard shows tracked work items across all statuses
Heimdall MUST provide a work-item queue screen that lists tracked Linear work items across all current statuses and lifecycle buckets, with enough linked workflow context for an operator to understand queue state.

#### Scenario: Operator reviews the work-item queue
- **WHEN** an operator opens the work-item queue screen
- **THEN** Heimdall lists tracked work items with their key, title, current status, lifecycle bucket, team, and last-seen update time
- **AND** each row shows any linked repository binding or recent workflow status when that context exists

### Requirement: Dashboard shows active pull requests and Heimdall-tracked PR activity
Heimdall MUST provide an active pull-request screen and a pull-request detail view that expose each active Heimdall-managed pull request together with its tracked command and workflow history.

#### Scenario: Operator reviews active pull requests
- **WHEN** an operator opens the active pull-request screen
- **THEN** Heimdall lists active Heimdall-managed pull requests with repository, number, title, state, bound work item, and branch or change identity
- **AND** the screen lets the operator navigate to a detail view for a selected pull request

#### Scenario: Operator opens a pull-request detail view
- **WHEN** an operator requests a specific Heimdall-managed pull request detail view
- **THEN** Heimdall shows the pull request's linked work item, repository binding, and Heimdall-tracked command/activity history
- **AND** the history identifies command requests, authorization outcomes, workflow linkage, and recent audit or execution status for that pull request

### Requirement: Dashboard interactions refresh incrementally without mutation
Heimdall MUST support incremental dashboard filter and refresh interactions through HTML fragment responses, and those interactions MUST remain read-only.

#### Scenario: Operator changes a dashboard filter
- **WHEN** an operator changes a work-item or pull-request dashboard filter or requests a refresh interaction
- **THEN** Heimdall returns updated HTML for the targeted dashboard region without requiring a full SPA client
- **AND** the interaction does not create branches, comments, workflow runs, or other repository mutations

<!-- Merged from opsx change delta -->
## ADDED Requirements

### Requirement: Dashboard lists active opencode command runs
Heimdall MUST provide a private dashboard screen for currently active PR-comment opencode runs, and that screen MUST list enough context to identify each run and navigate to its command detail view.

#### Scenario: Operator opens the active command-runs screen
- **WHEN** an operator requests the dashboard screen for active command runs
- **THEN** Heimdall lists each currently queued, starting, running, or blocked opencode-backed command with repository, pull request, command kind, actor, current status, start time, and session ID when known
- **AND** each row links to a command detail view for that specific command request

### Requirement: Dashboard command detail view shows live-tailed human-readable output with HTMX refresh
Heimdall MUST provide a private command detail page that shows the current state and human-readable output timeline for a selected opencode-backed PR command. While the command is not terminal, the page MUST behave like a live tail of the opencode output: newly available entries MUST appear at the end of the visible timeline in observed stream order without requiring a manual page reload, and the page MUST remain read-only.

#### Scenario: Running command detail view behaves like a live tail
- **WHEN** an operator opens the dashboard detail page for a currently running `/heimdall refine` or `/opsx-apply` command
- **THEN** Heimdall renders the current human-readable output timeline for that command
- **AND** HTMX refresh interactions append newly available output and update status incrementally in stream order until the command reaches a terminal state

#### Scenario: Existing output remains visible while new output is tailed in
- **WHEN** a running command detail page already shows earlier human-readable output entries and Heimdall observes later opencode events for the same command
- **THEN** the dashboard keeps the earlier entries visible on that same page
- **AND** it adds the newer entries after them so the operator experiences the page like a live tail of the output stream

#### Scenario: Queued command detail view is still useful before session start
- **WHEN** an operator opens the command detail page immediately after Heimdall accepted the command but before opencode has emitted its first structured event
- **THEN** Heimdall shows the command as queued or starting on that same page
- **AND** the page later transitions into live-output view without requiring a different URL or mutation action

### Requirement: Dashboard command detail view shows explicit state transitions
Heimdall MUST display the current execution state of a command-linked opencode run as one of `queued`, `starting`, `running`, `blocked`, `completed`, or `failed`, and the UI MUST make state transitions visible to the operator.

#### Scenario: Command transitions from queued to running are visible
- **WHEN** a command detail page is open while the associated command transitions from `queued` to `starting` and then to `running`
- **THEN** the dashboard updates the displayed state label at each transition
- **AND** the live-tail output area becomes active once the state reaches `running`

### Requirement: Dashboard renders terminal state and summary clearly
Heimdall MUST display the terminal outcome of a completed, blocked, or failed command prominently on the command detail page, including the terminal status and a concise human-readable summary.

#### Scenario: Completed command shows success summary
- **WHEN** a command-linked opencode run reaches `completed` state
- **THEN** the dashboard shows a clear success indicator together with the terminal summary
- **AND** the live-tail polling stops or slows down

#### Scenario: Failed command shows error summary
- **WHEN** a command-linked opencode run reaches `failed` state
- **THEN** the dashboard shows a clear failure indicator together with the error summary
- **AND** the live-tail polling stops

#### Scenario: Blocked command shows blocker details
- **WHEN** a command-linked opencode run reaches `blocked` state due to a permission or input request
- **THEN** the dashboard shows a clear blocked indicator together with the blocker details and any provided resolution instruction
- **AND** the live-tail polling slows down or stops until the command resumes

### Requirement: Dashboard supports viewing recent completed command runs
Heimdall MUST allow operators to view recently completed or failed command runs in addition to currently active ones, so the command-runs screen is not limited to only queued, starting, running, or blocked commands.

#### Scenario: Operator filters to see recently completed commands
- **WHEN** an operator requests the command-runs screen with a filter for completed or failed commands
- **THEN** Heimdall lists recently terminal command runs with their final status, completion time, and session ID
- **AND** each row links to the same command detail view that retains the full output timeline and terminal summary

### Requirement: Live tail output is bounded for performance
Heimdall MUST limit the number of display entries returned for a live-tail refresh to a reasonable default so that long-running commands do not degrade dashboard performance.

#### Scenario: Long-running command tail is bounded
- **WHEN** a command has generated more human-readable output entries than the live-tail query limit
- **THEN** the dashboard fragment returns only the most recent entries within the limit
- **AND** earlier entries remain accessible through a separate request or pagination control
