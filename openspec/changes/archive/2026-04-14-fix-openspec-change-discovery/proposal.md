## Why

Activation proposal runs can fail after proposal generation because Heimdall misreads machine-readable OpenSpec CLI output and can execute OpenSpec commands outside the target repository worktree. This blocks both change discovery and the post-generation readiness check for activated issues, and it makes workflow behavior depend on the service process working directory instead of the mapped repository.

## What Changes

- Fix activation proposal change discovery so Heimdall parses the real `openspec list --json` response shape instead of assuming a raw string array.
- Fix activation proposal readiness checking so Heimdall parses `openspec instructions apply --json` as machine-readable output even when the CLI emits human-oriented progress lines separately.
- Ensure activation proposal OpenSpec CLI calls run in the deterministic proposal worktree for the target repository, not the Heimdall service process cwd.
- Keep the existing post-proposal readiness check in the workflow, but make it robust instead of removing it.
- Add regression coverage for both CLI JSON parsing and worktree-scoped activation proposal execution.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `service-execution-runtime`: activation proposal OpenSpec discovery and readiness checks must consume real CLI JSON output and run against the target repository worktree.
- `feature-openspec-proposal-pr`: activation proposal change discovery and post-generation readiness checks must remain reliable before proposal publication continues.

## Impact

- `internal/exec/clients.go`
- `internal/app/app.go`
- `internal/workflow/proposal.go`
- proposal workflow tests, OpenSpec client parsing tests, and BDD coverage for readiness-check behavior
