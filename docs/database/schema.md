# SQLite Schema

## Design Intent

The database should capture durable workflow state, not become a second configuration system.

Configuration belongs in the project-root `.env` file or equivalent environment variables, and multiline secrets should be referenced through external secret files where practical.

SQLite stores:

- poll cursors
- latest work-item snapshots
- normalized transition events
- workflow runs and steps
- repo bindings and pull requests
- slash-command requests
- async jobs and audit events

## Mermaid ERD

```mermaid
erDiagram
    REPOSITORIES ||--o{ REPO_BINDINGS : has
    REPOSITORIES ||--o{ WORKFLOW_RUNS : has
    REPOSITORIES ||--o{ PULL_REQUESTS : has

    WORK_ITEMS ||--o{ WORK_ITEM_EVENTS : emits
    WORK_ITEMS ||--o{ REPO_BINDINGS : binds
    WORK_ITEMS ||--o{ WORKFLOW_RUNS : drives

    WORKFLOW_RUNS ||--o{ WORKFLOW_STEPS : contains
    WORKFLOW_RUNS ||--o{ JOBS : schedules
    WORKFLOW_RUNS ||--o{ AUDIT_EVENTS : records

    PULL_REQUESTS ||--o{ COMMAND_REQUESTS : receives
    REPO_BINDINGS ||--o| PULL_REQUESTS : tracks
    COMMAND_REQUESTS ||--o| WORKFLOW_RUNS : triggers
    COMMAND_REQUESTS ||--o{ JOBS : schedules
    COMMAND_REQUESTS ||--o{ AUDIT_EVENTS : records

    PROVIDER_CURSORS {
        integer id PK
        text provider
        text scope_key UK
        text cursor_value
        text cursor_kind
        datetime last_polled_at
    }

    REPOSITORIES {
        integer id PK
        text provider
        text repo_ref UK
        text owner
        text name
        text default_branch
        text branch_prefix
        text local_mirror_path
        boolean is_active
    }

    WORK_ITEMS {
        integer id PK
        text provider
        text provider_work_item_id UK
        text work_item_key UK
        text title
        text state_name
        text lifecycle_bucket
        text team_key
        datetime last_seen_updated_at
    }

    WORK_ITEM_EVENTS {
        integer id PK
        integer work_item_id FK
        text provider
        text provider_event_id
        text event_type
        text event_version
        text idempotency_key UK
        datetime occurred_at
        datetime detected_at
    }

    REPO_BINDINGS {
        integer id PK
        integer work_item_id FK
        integer repository_id FK
        text branch_name UK
        text change_name UK
        text binding_status
        text last_head_sha
        datetime created_at
        datetime updated_at
    }

    PULL_REQUESTS {
        integer id PK
        integer repository_id FK
        integer repo_binding_id FK
        text provider
        text provider_pr_node_id UK
        integer number
        text title
        text base_branch
        text head_branch
        text state
        text url
    }

    COMMAND_REQUESTS {
        integer id PK
        integer pull_request_id FK
        text comment_node_id UK
        text command_name
        text command_args
        text requested_agent
        text actor_login
        text authorization_status
        text dedupe_key UK
        integer workflow_run_id FK
        text status
    }

    WORKFLOW_RUNS {
        integer id PK
        integer work_item_id FK
        integer repository_id FK
        integer trigger_event_id FK
        text run_type
        text status
        text change_name
        text branch_name
        text worktree_path
        text requested_by_type
        text requested_by_login
        integer attempt_count
    }

    WORKFLOW_STEPS {
        integer id PK
        integer workflow_run_id FK
        text step_name
        integer step_order
        text status
        text executor
        text command_line
        text tool_version
        integer attempt_count
    }

    JOBS {
        integer id PK
        integer workflow_run_id FK
        integer command_request_id FK
        text job_type
        text lock_key
        text status
        integer priority
        datetime run_after
        integer attempt_count
        integer max_attempts
    }

    AUDIT_EVENTS {
        integer id PK
        integer workflow_run_id FK
        integer command_request_id FK
        text event_type
        text severity
        text actor_type
        text actor_login
        text agent_name
        text commit_sha
        text summary
        datetime occurred_at
    }
```

## Table Roles

### `provider_cursors`

Stores the current polling position for each provider scope, such as a configured Linear project or a GitHub repository poll scope.

### `repositories`

Stores the repositories Heimdall manages and the local bare-mirror paths it uses for worktree creation.

### `work_items`

Stores the latest normalized snapshot of each tracked board item.

This is the current state table, not the full event history.

### `work_item_events`

Stores normalized transition events such as `entered_active_state` and provides the main idempotency boundary for polling.

### `repo_bindings`

Represents the durable one-issue-to-one-repo automation binding.

This is the record that ties together the work item, branch name, change name, and current lifecycle status.

### `pull_requests`

Stores GitHub PR identity and state so Heimdall can reconcile polled comment activity and PR lifecycle changes.

### `command_requests`

Stores PR comment commands, their dedupe keys, authorization results, selected agent, and downstream workflow linkage.

### `workflow_runs`

Stores top-level runs for `propose`, `refine`, `apply`, `archive`, and reconciliation work.

### `workflow_steps`

Stores step-level execution details inside a workflow run, including which executor ran, which command line was used, and how many attempts occurred.

### `jobs`

Stores queued async work with retry scheduling and lock keys.

Recommended lock-key shapes:

- `issue:<provider>:<work-item-key>`
- `repo:<repo-ref>`

### `audit_events`

Stores append-only audit records that answer who requested a change, which agent ran, which commit was created, and whether the action succeeded.

## Important Constraints

- `work_item_events.idempotency_key` must be unique
- `command_requests.dedupe_key` must be unique
- `command_requests.comment_node_id` must be unique
- `repo_bindings(work_item_id, repository_id)` should be unique
- `pull_requests(repository_id, number)` should be unique

## What Does Not Belong In SQLite

- GitHub App private keys
- installation tokens
- Linear API keys
- static repo routing rules
- allowed GitHub users and agents

Those belong in the service configuration and secret store.
