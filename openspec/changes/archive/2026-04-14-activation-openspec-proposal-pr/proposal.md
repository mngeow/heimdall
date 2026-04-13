## Why

The activation flow still opens a temporary bootstrap PR instead of creating the OpenSpec change artifacts that engineers actually need to review and refine. Heimdall should turn an issue entering the active state directly into an OpenSpec proposal PR so the Linear-to-PR path matches the intended product workflow.

## What Changes

- Replace the activation bootstrap file flow with an activation proposal flow that creates a worktree and deterministic branch, derives a change from the Linear issue title and description, and runs local `opencode` to generate an OpenSpec change.
- **BREAKING** Require a per-repository default spec-writing agent setting so operators can choose which OpenCode agent generates activation proposals and refine edits.
- Require the activation flow to commit and push the generated OpenSpec artifacts, then open or reuse a proposal PR whose title reflects the issue and whose configured GitHub monitor label is applied during reconciliation.
- Preserve the existing idempotent activation behavior so retries reuse the same branch, worktree, change, and PR instead of duplicating proposal work.
- Require retry-safe git worktree reconciliation so stale registered worktrees and partially failed earlier runs do not force manual operator cleanup before activation can succeed again.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `feature-kanban-activation`: activation should start the durable proposal workflow instead of a temporary bootstrap PR flow.
- `feature-openspec-proposal-pr`: activation should create an OpenSpec change from the work item seed data, commit the generated artifacts, and publish a proposal-focused pull request.
- `service-github-scm`: activation-triggered repository publishing should open or reuse the proposal PR shape and ensure the configured monitor label is applied during reconciliation.
- `service-execution-runtime`: activation should run a configurable proposal-generation agent through the local CLI workflow instead of the fixed bootstrap profile.
- `service-configuration`: runtime configuration should declare and validate the activation proposal agent setting used for activation-triggered proposal generation.

## Impact

- Workflow engine sequencing for activation-triggered runs
- Local git/worktree and OpenSpec/OpenCode execution adapters
- Runtime configuration parsing and validation
- Operator configuration and setup documentation
- GitHub pull request publishing and reconciliation
- Product and workflow documentation for the activation path
