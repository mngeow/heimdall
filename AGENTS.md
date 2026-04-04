# Project: Symphony
A Linux-hosted Go service that turns kanban movement into OpenSpec-driven engineering work by converting board activity into branches, specs, PRs, and agent-driven implementation flows.

Built with: Go application code (planned), OpenSpec `spec-driven` workflow, local OpenCode/OpenSpec CLI execution, Markdown docs in `/docs/`, SQLite for v1, GitHub App auth, Linear polling.

## North Star Preferences
- Keep v1 as a single binary for a single-user Linux host.
- Prefer small, explicit interfaces around external systems such as board providers, SCM providers, execution adapters, and storage.
- Keep provider-specific details inside adapters. The core workflow engine should use normalized concepts like work items, transitions, commands, and runs.
- Treat OpenSpec CLI JSON output as the source of truth for change and artifact state. Do not hardcode workflow assumptions beyond documented paths and CLI responses.
- Optimize for operator simplicity first: SQLite, local CLI execution, minimal config, deterministic naming, and reconcile-before-create behavior.
- Keep GitHub comment commands safe and narrow. PR comments are untrusted input until verified, authorized, and parsed.
- Prefer minimal correct changes over broad scaffolding. Do not add multi-tenant, distributed, or UI-heavy architecture unless the user asks for it.
- Design for future Jira-style expansion through adapter seams, not by leaking Linear-specific semantics into the core.
- Use MermaidJS for any diagrams in docs, specs, or design artifacts. Do not introduce ASCII diagrams when a diagram is needed.

## Documentation & Resources
- Start with `docs/README.md`, then read the relevant files in `docs/` before making design or architecture changes.
- Product and workflow expectations are documented in:
  - `docs/product.md`
  - `docs/architecture.md`
  - `docs/workflows.md`
  - `docs/authentication.md`
  - `docs/operations.md`
  - `docs/extensibility.md`
- OpenSpec project config lives in `openspec/config.yaml`.
- Durable requirements belong in `openspec/specs/`.
- Scoped work belongs in `openspec/changes/<change>/` with `proposal.md`, `design.md`, and `tasks.md`.
- The root `README.md` is currently minimal; prefer `docs/` and `openspec/` as the real source of project context.

## Spec And Docs Maintenance
- If behavior, architecture, auth, or workflow expectations change, update the relevant files in `docs/` in the same change.
- If a change affects long-lived product requirements, capture it in `openspec/specs/<capability>/spec.md`.
- If a change is scoped work that should be proposed and implemented, create or update an OpenSpec change under `openspec/changes/<change>/`.
- Do not create a parallel requirements-tracking system unless the user explicitly asks for one.

## Git Workflow
- Use focused branches. For manual work, prefer prefixes like `feat/`, `fix/`, or `docs/`.
- Symphony's automated branch naming is documented in `docs/product.md`; keep automation naming deterministic.
- Use conventional commit prefixes where practical.
- Never push directly to `main` unless the user explicitly asks for it.
- Keep commits scoped to one concern so generated specs, workflow logic, and unrelated edits do not get mixed together.

## Current Repo Status
- This repo is currently design-first. There is no `go.mod`, no runnable Go service, and no canonical build/test commands yet.
- When scaffolding the application, establish the canonical developer commands in-repo and update this file to reflect them.

## Commits & Communication
- Be concise and direct.
- Check `docs/` and `openspec/` before asking questions the repository already answers.
- Ask when repository routing, auth scope, provider behavior, or workflow semantics are ambiguous.
- When implementing Go code, prefer idiomatic package boundaries, constructor injection, explicit `context.Context` plumbing, and table-driven tests.
