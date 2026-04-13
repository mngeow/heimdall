# Heimdall Design Docs

Heimdall is a Linux-hosted Go service that turns kanban movement into OpenSpec-driven engineering work.

In the initial design, Heimdall:
- polls Linear for issues entering an active state
- creates a git branch for the target repository
- generates an OpenSpec change from the issue title and description
- commits and pushes the generated spec artifacts
- opens a GitHub pull request to `main`
- polls GitHub for new PR comments on Heimdall-managed pull requests so it can refine specs or run `/opsx-apply` with an allowed agent
- commits any resulting changes back to the same branch

## Confirmed V1 Decisions

- Deployment model: single-user service on one Linux host
- Linear trigger model: polling, not Linear webhooks
- GitHub authentication: GitHub App
- GitHub PR command intake: polling, not GitHub webhooks
- Linear authentication: API key from a dedicated Linear service account plus local service config
- OpenCode and OpenSpec execution: local CLI on the host
- PR interaction model: slash commands in GitHub PR comments
- Default data store: SQLite for the simplest installation path

## Important Note

Linear and GitHub PR command intake are both polling-based in V1, so the standard deployment path does not require a public inbound webhook endpoint.

## Document Map

- `product.md`: user experience, goals, non-goals, and V1 conventions
- `architecture.md`: runtime architecture, package layout, data model, and repository strategy
- `workflows.md`: end-to-end flows for issue activation, spec refinement, and apply execution
- `authentication.md`: GitHub App, Linear key, command authorization, and secret handling
- `operations.md`: deployment, configuration, filesystem layout, observability, and recovery
- `logging.md`: logging strategy, log fields, retention, and how to view Heimdall logs
- `extensibility.md`: how V1 stays ready for Jira, other SCMs, and remote execution later
- `setup/README.md`: operator setup order for Linux host, GitHub, and Linear
- `setup/github.md`: exact GitHub App, polling, and repository setup
- `setup/linear.md`: exact Linear account, API key, and state-mapping setup
- `setup/linux-host.md`: Linux host dependencies and server preparation checklist
- `database/README.md`: database design overview
- `database/schema.md`: SQLite schema notes and Mermaid ERD

## Design Principles

- Stay inside the tools the user already lives in: Linear and GitHub
- Prefer opinionated defaults over a large admin surface
- Keep the service auditable and deterministic
- Use explicit abstractions so provider expansion does not require a rewrite
- Preserve human control over implementation while automating the repetitive setup work
