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
