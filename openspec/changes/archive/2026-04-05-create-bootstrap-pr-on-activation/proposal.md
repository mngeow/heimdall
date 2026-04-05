## Why

Symphony already detects when a Linear issue enters an active state, but the activation-to-PR path itself is still unproven. Instead of jumping straight to automatic OpenSpec proposal generation, this change introduces a simpler bootstrap flow that extracts the issue title and description, creates a worktree from the configured local mirror, makes a small non-empty repository file change through OpenCode, pushes a branch, and opens a pull request.

This is an incremental implementation step, not a change in direction. The longer-term goal remains replacing that simple file change with automatic OpenSpec proposal generation once the activation, worktree, git, and PR plumbing is proven end to end.

## What Changes

- Introduce an activation-driven bootstrap PR workflow as the first implementation step toward automated OpenSpec proposal PRs.
- Require Symphony to extract the triggering issue's title and description and pass them to a local OpenCode execution that uses the general agent with model `gpt-5.4`.
- Require Symphony to create a fresh git worktree from the repository mirror configured by `SYMPHONY_REPO_PLATFORM_LOCAL_MIRROR_PATH` before agent execution.
- Require Symphony to derive a new branch name from the detected ticket description, commit the resulting simple file change, push the branch, and open or reuse a pull request with an appropriate title and description.
- Require the workflow to fail clearly when the agent does not produce any file changes or when branch or PR creation cannot complete.
- Require Symphony to emit more detailed structured logs for activation bootstrap progress, key workflow decisions, and failure points so operators can see what the service is doing.
- Preserve the longer-term OpenSpec proposal workflow direction by keeping the bootstrap mutation small and replaceable.

## Capabilities

### New Capabilities

### Modified Capabilities
- `feature-kanban-activation`: activated issues now start a bootstrap pull request workflow that validates activation-to-PR plumbing before full OpenSpec proposal generation.
- `feature-openspec-proposal-pr`: the activation-seeded branch and PR flow is initially bootstrapped by a simple file change, with the intent to later replace that step with OpenSpec proposal artifact generation.
- `service-execution-runtime`: activation-triggered execution now runs OpenCode locally with the general agent and `gpt-5.4` inside a git worktree created from the configured bare mirror.
- `service-github-scm`: GitHub branch push and pull request creation requirements now cover the activation-triggered bootstrap PR flow.
- `service-observability`: activation-triggered bootstrap runs now emit detailed structured logs that let operators trace workflow progress and diagnose failures.

## Impact

- Affects Linear activation handling, workflow orchestration, and worktree creation.
- Affects local OpenCode execution and git mutation behavior.
- Affects GitHub branch push and PR creation behavior.
- Affects operator debugging and runtime visibility because bootstrap runs now require more detailed structured logs.
- Affects operator expectations and workflow docs because activation initially opens a bootstrap PR before the later planned OpenSpec proposal generation step.
