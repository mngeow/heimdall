## Why

Symphony already detects when a Linear issue enters an active state, but the current workflow still assumes an OpenSpec-first proposal flow. This change replaces that path with a direct agent-driven bootstrap flow that extracts the issue title and description, creates a worktree from the configured local mirror, makes an arbitrary code change through OpenCode, pushes a branch, and opens a pull request.

## What Changes

- **BREAKING** Replace the activation-driven OpenSpec proposal workflow with an activation-driven bootstrap PR workflow.
- Require Symphony to extract the triggering issue's title and description and pass them to a local OpenCode execution that uses the general agent with model `gpt-5.4`.
- Require Symphony to create a fresh git worktree from the repository mirror configured by `SYMPHONY_REPO_PLATFORM_LOCAL_MIRROR_PATH` before agent execution.
- Require Symphony to derive a new branch name from the detected ticket description, commit the resulting repository changes, push the branch, and open or reuse a pull request with an appropriate title and description.
- Require the workflow to fail clearly when the agent does not produce any file changes or when branch or PR creation cannot complete.

## Capabilities

### New Capabilities

### Modified Capabilities
- `feature-kanban-activation`: activated issues now start a bootstrap pull request workflow instead of the current proposal workflow.
- `feature-openspec-proposal-pr`: the activation-seeded branch and PR flow now produces a direct code-change bootstrap PR rather than OpenSpec proposal artifacts.
- `service-execution-runtime`: activation-triggered execution now runs OpenCode locally with the general agent and `gpt-5.4` inside a git worktree created from the configured bare mirror.
- `service-github-scm`: GitHub branch push and pull request creation requirements now cover the activation-triggered bootstrap PR flow.

## Impact

- Affects Linear activation handling, workflow orchestration, and worktree creation.
- Affects local OpenCode execution and git mutation behavior.
- Affects GitHub branch push and PR creation behavior.
- Affects operator expectations and workflow docs because activation no longer creates an OpenSpec change by default.
