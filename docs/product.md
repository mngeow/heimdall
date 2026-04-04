# Product Design

## Problem

Harness engineering work often starts as a kanban ticket, but the path from ticket to actionable engineering change still involves a lot of repeated manual work:

- creating a branch
- translating the ticket into a better scoped spec
- scaffolding an OpenSpec change
- committing the initial artifacts
- opening a pull request
- iterating on the spec before implementation starts

Symphony exists to remove that setup work while keeping the engineer in control of the final plan and implementation.

## Desired User Experience

The happy path for V1 is intentionally small:

1. A Linear issue is moved into the configured active state.
2. Symphony detects that transition during polling.
3. Symphony creates a branch in the configured GitHub repository.
4. Symphony generates a new OpenSpec change from the Linear issue title and description.
5. Symphony commits the generated artifacts and opens a pull request against `main`.
6. The engineer refines the spec from GitHub comments.
7. When ready, the engineer triggers `/opsx-apply` from the PR with an allowed agent.
8. Symphony commits the resulting changes back to the same branch.

The user should not need a separate Symphony UI in V1.

## Ease Of Use Decisions

To keep V1 easy to operate and easy to adopt, the design makes these choices:

- one service binary, not multiple deployable services
- one local config file plus environment-backed secrets
- SQLite by default, so a single Linux VM is enough to run the system
- polling for Linear to avoid requiring public ingress from Linear
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
- deep project-management features inside Symphony
- real-time Linear webhooks in the first release
- arbitrary shell access from GitHub comments

## Default V1 Conventions

These defaults keep the product predictable and easy to reason about:

- Branch name: `symphony/<issue-key>-<slug>`
- OpenSpec change name: `<issue-key>-<slug>`
- PR title: `[<issue-key>] OpenSpec proposal for <issue title>`
- Initial commit message: `docs(openspec): propose <change-name>`
- Refinement commit message: `docs(openspec): refine <change-name>`
- Apply commit message: `feat: implement <change-name> via symphony`

## Routing Model

V1 should be opinionated about repository routing:

- If only one repository is configured, Symphony uses it automatically.
- If multiple repositories are configured, routing should be based on explicit config rules.
- The first routing rules should be simple: Linear team, project, or label matches.
- If no routing rule matches, Symphony should fail clearly and comment on the failure instead of guessing.

## Human Control Points

Automation should start the work, not silently finish it.

The human stays in control at these points:

- reviewing the generated proposal and design in the PR
- refining the spec via PR comments
- choosing when to run `/opsx-apply`
- choosing which approved agent to use for `/opsx-apply`
- deciding whether to archive the change after implementation
