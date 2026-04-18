## Why

Heimdall's current durable specs only define `/heimdall refine` with the repository default agent and `/opsx-apply` with an allowed agent. That leaves the actual PR-comment implementation underspecified for explicit agent-selected spec edits, apply runs with extra prompt guidance, generic opencode command execution, and the cases where `opencode` blocks on a permission request that should be approved explicitly by request ID instead of hanging or being auto-approved.

Implementation work on this change also exposed two rounds of correctness gaps that the earlier artifacts did not pin down tightly enough. The first round covered multiline prompt text after a standalone `--`, missing single-change resolution, placeholder refine success paths, and accepted queued PR commands that could remain silently queued without a started worker and visible terminal outcomes. Runtime debugging then exposed adapter-level gaps: Heimdall can invoke `opencode run` with unsupported prompt flags, classify CLI help or usage output as permission blockers, persist blank permission or session identifiers, execute against stale change bindings that no longer exist in the worktree, leave successful jobs stuck in `running`, and report approval success without using the supported permission-reply API or observing the resumed session's real outcome. Those behaviors are user-visible contract bugs, so this change needs to define the execution-adapter, state-transition, and approval guarantees before implementation can be considered complete.

This change is needed now because Heimdall is moving from workflow design into real PR-comment execution. The command surface, blocked-state behavior, and follow-up permission approval path need to be concrete before the runtime can safely implement them.

## What Changes

- Expand the Heimdall PR comment command surface to support agent-selected spec refinement, agent-selected apply execution with optional prompt guidance, a narrow generic opencode command path, and explicit approval of a pending opencode permission request by request ID.
- Add a consistent `/heimdall ...` command model for comment-driven execution while keeping `/opsx-apply` as a compatibility alias for the apply workflow.
- Define how Heimdall resolves the target OpenSpec change, validates the selected agent, limits generic opencode execution to an allowlisted command set, and validates that permission approvals stay scoped to the same pull request.
- Require agent-driven commands to preserve the raw multiline prompt body that follows the first standalone `--`, including the case where the command line ends with `--` and the prompt continues on later lines.
- Require refine, apply, and generic opencode commands to resolve exactly one target change before execution starts and reject missing or ambiguous targets instead of running with an empty change name.
- Require refine and apply execution to use the supported `opencode run` contract with positional messages and machine-readable JSON events instead of unsupported prompt flags and heuristic stderr parsing.
- Require Heimdall to detect permission blockers only from explicit machine-readable permission events, to persist the exact request and session identifiers from those events, and to treat generic CLI help or usage output as normal execution failures rather than approval prompts.
- Require agent-driven commands to verify that the resolved OpenSpec change still exists in the current worktree before execution begins so stale bindings are rejected instead of passed into opencode.
- Require `/heimdall refine` to execute a real refine run by using the resolved change and preserved prompt tail, and to report success only after that execution actually completes.
- Require Heimdall to start a long-running PR-command worker as part of normal service runtime so accepted PR-comment commands do not remain silently queued.
- Require every queued PR-comment command in the supported surface, including `/heimdall status`, to execute through that worker path by using durable command-request, pull-request, and repository identifiers.
- Require queued PR-command execution to publish visible terminal outcomes and command/job states so duplicates in later poll windows do not mask a stuck, failed, or falsely still-running original command.
- Require the PR-command executor entry points for status, refine, apply, generic opencode, and approval to perform their real workflow responsibilities instead of returning placeholder success comments or state-only stub outcomes.
- Define non-interactive execution behavior so clarification requests stay blocked, while permission requests emit stable request IDs that can be approved later through a narrow Heimdall command instead of hanging or auto-approving access.
- Require successful queued jobs to transition to `completed` and release the pull-request lock so later same-PR commands can run.
- Require `/heimdall approve` to use the supported opencode permission-reply API and to report the resumed session's real terminal outcome instead of an optimistic state-only acknowledgment.
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

- Affected code: PR comment parsing and dispatch, multiline prompt capture, PR-command worker startup and lifecycle, durable queued-command loading, `PRCommandExecutor` status/refine/apply/opencode/approve flows, OpenCode execution adapters, JSON event parsing, permission-reply and resumed-session handling, change-existence validation, queue completion transitions, runtime-state persistence for pending permission requests, workflow orchestration, PR status comment publishing, and behavior tests.
- Affected systems: local `opencode` CLI and SDK execution, repository-scoped agent policy, PR-comment-driven workflow safety, and resumable permission-request handling.
- Operator impact: operators must configure allowed agents and any generic opencode command aliases they want exposed through PR comments, and authorized users can explicitly approve a pending permission request by its reported ID.
- Safety impact: Heimdall will treat clarification requests as blocked states and permission requests as explicitly approved resumable states rather than approving them implicitly.
