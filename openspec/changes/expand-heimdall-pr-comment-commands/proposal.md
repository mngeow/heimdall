## Why

Heimdall's current durable specs only define `/heimdall refine` with the repository default agent and `/opsx-apply` with an allowed agent. That leaves the actual PR-comment implementation underspecified for explicit agent-selected spec edits, apply runs with extra prompt guidance, generic opencode command execution, and the cases where `opencode` blocks on a permission request that should be approved explicitly by request ID instead of hanging or being auto-approved.

This change is needed now because Heimdall is moving from workflow design into real PR-comment execution. The command surface, blocked-state behavior, and follow-up permission approval path need to be concrete before the runtime can safely implement them.

## What Changes

- Expand the Heimdall PR comment command surface to support agent-selected spec refinement, agent-selected apply execution with optional prompt guidance, a narrow generic opencode command path, and explicit approval of a pending opencode permission request by request ID.
- Add a consistent `/heimdall ...` command model for comment-driven execution while keeping `/opsx-apply` as a compatibility alias for the apply workflow.
- Define how Heimdall resolves the target OpenSpec change, validates the selected agent, limits generic opencode execution to an allowlisted command set, and validates that permission approvals stay scoped to the same pull request.
- Define non-interactive execution behavior so clarification requests stay blocked, while permission requests emit stable request IDs that can be approved later through a narrow Heimdall command instead of hanging or auto-approving access.
- Add the execution-policy and runtime-state requirements needed to support allowed agents, allowlisted opencode command aliases, persisted pending permission requests, and deterministic permission handling for PR-comment runs.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `feature-pr-command-workflows`: expand the PR comment command surface and define the user-visible behavior for agent-selected refine, apply, generic opencode commands, and explicit permission-request approvals.
- `service-execution-runtime`: define non-interactive opencode execution rules, explicit agent selection for PR-comment execution, and the blocked-and-resumable behavior for input or permission requests.
- `service-runtime-state`: define the durable state Heimdall keeps for pending opencode permission requests so approval commands and restarts can resume safely.
- `service-configuration`: define the repository configuration needed for allowed PR-comment agents and allowlisted generic opencode commands.

## Impact

- Affected code: PR comment parsing and dispatch, OpenCode execution adapters, runtime-state persistence for pending permission requests, workflow orchestration, PR status comment publishing, and behavior tests.
- Affected systems: local `opencode` CLI execution, repository-scoped agent policy, PR-comment-driven workflow safety, and resumable permission-request handling.
- Operator impact: operators must configure allowed agents and any generic opencode command aliases they want exposed through PR comments, and authorized users can explicitly approve a pending permission request by its reported ID.
- Safety impact: Heimdall will treat clarification requests as blocked states and permission requests as explicitly approved resumable states rather than approving them implicitly.
