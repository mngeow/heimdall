## Why

Heimdall can currently reject valid `/heimdall refine` and related PR commands because it resolves the target change from runtime state but validates and executes against the wrong or stale worktree. Real PR-command runs can also fail or retry incorrectly after opencode starts because `opencode run --format json` emits very large newline-delimited JSON events that overflow Heimdall's current event reader, intermediate generic error events can be treated as terminal even when the session later succeeds, blank error payloads can produce useless `refine failed:` logs, and the session identity from the first JSON block is not yet tracked durably for logging and recovery; the recent LES-8 failure also exposed that the intended OpenSpec change name needs a clearer contract: it should be derived from the Linear ticket title with explicit normalization rules so proposal generation, bindings, and later PR commands converge on the same change identity.

## What Changes

- Make agent-driven PR commands use one canonical, prepared pull-request worktree for change validation, OpenSpec inspection, and opencode execution.
- Require Heimdall to materialize or refresh that PR worktree before checking whether the resolved change still exists, so validation reflects the repository state that will actually be executed.
- Tighten PR worktree creation so existing PR branches are recreated from the fetched PR head branch instead of being reset from the repository default branch.
- Scope PR-command runtime lookups to the durable pull-request and repository binding context instead of inferring bindings from branch name alone.
- Ensure the execution adapters used by refine, apply, and generic opencode commands run in the same prepared PR worktree that OpenSpec validation uses.
- Make PR-comment opencode event parsing consume newline-delimited JSON output with very large single-event lines, preserve later event classification, capture the session ID from the first structured event, and avoid failing on local token-length limits.
- Make refine/apply outcome classification wait for the real terminal session outcome instead of treating an early empty generic error event as a final failure, and require non-empty fallback failure summaries when the run really fails without detail.
- Make activation proposal naming explicitly derive the canonical OpenSpec change name from the normalized Linear ticket title, including logic such as converting spaces to hyphens.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `feature-pr-command-workflows`: Change how agent-driven PR commands resolve, validate, and execute against the pull request's real bound worktree and binding context, including resilience to large valid opencode JSON event lines and correct terminal outcome handling after intermediate generic errors.
- `feature-openspec-proposal-pr`: Change deterministic OpenSpec change naming so the canonical change name is derived from the normalized Linear ticket title.
- `service-execution-runtime`: Change PR-command worktree preparation, branch materialization, execution-adapter scoping, and opencode JSON event parsing so OpenSpec and opencode run against the same prepared checkout, valid large NDJSON events do not abort command execution, terminal outcomes are classified correctly, and session IDs from the first event are logged.
- `service-runtime-state`: Change PR-command binding lookup requirements so active bindings are resolved from durable pull-request and repository linkage instead of branch-name-only inference, and persist observed opencode session identities for PR-comment execution.

## Impact

- Affects activation proposal naming in `internal/workflow`, PR-command execution in `internal/workflow`, git worktree management in `internal/repo`, opencode JSON-event parsing and session tracking in `internal/exec`, and runtime-state queries plus persisted execution metadata in `internal/store`.
- Tightens how Heimdall interprets pull request branch state, local mirror content, and active OpenSpec bindings before mutating a proposal branch.
- Reduces false stale-binding rejections, prevents valid large opencode event payloads from aborting PR commands, avoids false failed-job retries after intermediate empty error events, makes failure logs actionable by guaranteeing non-empty summaries, makes OpenSpec change names align with normalized Linear ticket titles, and prevents PR-command runs from mutating or validating against the wrong checkout.
