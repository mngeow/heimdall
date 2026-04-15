# Product Design

## Problem

Harness engineering work often starts as a kanban ticket, but the path from ticket to actionable engineering change still involves a lot of repeated manual work:

- creating a branch
- translating the ticket into a better scoped spec
- scaffolding an OpenSpec change
- committing the initial artifacts
- opening a pull request
- iterating on the spec before implementation starts

Heimdall exists to remove that setup work while keeping the engineer in control of the final plan and implementation.

## Desired User Experience

The happy path for V1 is intentionally small:

1. A Linear issue is moved into the configured active state.
2. Heimdall detects that transition during polling.
3. Heimdall creates a branch in the configured GitHub repository.
4. Heimdall creates or reuses an OpenSpec change from the Linear issue title and description.
5. Heimdall runs the repository's configured default spec-writing agent through local `opencode` to generate the proposal artifacts required by the change's OpenSpec schema.
6. Heimdall commits the generated OpenSpec artifacts and opens a proposal pull request against `main`.
7. The engineer refines the spec from GitHub comments, which Heimdall discovers during polling.
8. When ready, the engineer triggers `/opsx-apply` from the PR with an allowed agent.
9. Heimdall commits the resulting changes back to the same branch.

The user should not need a separate Heimdall UI in V1.

That said, Heimdall does expose a small, read-only private operator dashboard on its existing HTTP server so operators can quickly inspect queued Linear work items, active automation pull requests, and Heimdall-tracked command/activity history without reading raw SQLite or external logs. This dashboard is server-rendered HTML with HTMX for light interactivity and is intentionally not a workflow-control surface.

## Ease Of Use Decisions

To keep V1 easy to operate and easy to adopt, the design makes these choices:

- one service binary, not multiple deployable services
- one project-root `.env` file plus optional file-backed secret references
- SQLite by default, so a single Linux VM is enough to run the system
- polling for Linear to avoid requiring public ingress from Linear
- polling for GitHub PR command intake to avoid requiring public ingress from GitHub
- slash commands in GitHub so the refinement loop stays in the PR
- deterministic branch names and change names so retries reconcile instead of duplicating work
- one open automation PR per issue per repository

## Primary Goals

- Convert kanban movement into a ready-to-review OpenSpec PR with minimal user effort.
- Keep the workflow centered on Linear and GitHub.
- Make auth and deployment simple enough for a single engineer to operate.
- Preserve a clean expansion path for Jira and other board systems later.
- Make all machine-driven actions auditable through commits, PR comments, and structured logs.

## V1 Non-Goals

- multi-tenant hosted SaaS
- a custom browser UI for workflow control
- automatic merge or automatic deployment after PR creation
- deep project-management features inside Heimdall
- real-time Linear or GitHub webhooks in the first release
- arbitrary shell access from GitHub comments

## Default V1 Conventions

These defaults keep the product predictable and easy to reason about:

- Branch name: `heimdall/<issue-key>-<description-or-title-slug>`
- OpenSpec change name: `<issue-key>-<description-or-title-slug>` (lowercased)
- PR title: `[<issue-key>] OpenSpec proposal for <issue title>`
- Initial proposal commit message: `docs(openspec): propose <change-name> via heimdall`
- No-change proposal result: fail the workflow as blocked and log the reason instead of opening an empty PR
- Refinement commit message: `docs(openspec): refine <change-name>`
- Apply commit message: `feat: implement <change-name> via heimdall`

## Activation Proposal Logging

The activation proposal path should be easy to follow from host logs alone.

Operators should expect structured log entries that include:

- workflow run id
- work item key
- repository ref
- branch name when known
- worktree path when known
- change name when known
- workflow step name and outcome
- blocked or failure reason when the proposal flow cannot continue

Logs must stay redacted enough to avoid exposing installation tokens, raw prompt bodies, or secrets.

## Routing Model

V1 should be opinionated about repository routing:

- If only one repository is configured, Heimdall uses it automatically.
- If multiple repositories are configured, routing should be based on explicit config rules.
- The first routing rules should be simple: Linear team, project, or label matches.
- If no routing rule matches, Heimdall should fail clearly and comment on the failure instead of guessing.

## Human Control Points

Automation should start the work, not silently finish it.

The human stays in control at these points:

- reviewing the generated proposal and design in the PR
- refining the spec via PR comments
- choosing when to run `/opsx-apply`
- choosing which approved agent to use for `/opsx-apply`
- deciding whether to archive the change after implementation
