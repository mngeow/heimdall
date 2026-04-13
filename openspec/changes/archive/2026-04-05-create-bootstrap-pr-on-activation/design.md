## Context

Heimdall's current product and durable specs assume that a Linear activation event starts an OpenSpec proposal workflow: create a deterministic branch, scaffold an OpenSpec change, generate proposal artifacts, commit them, and open a pull request. The next implementation step is intentionally narrower. After an issue is observed in an active state, Heimdall should extract the issue title and description, create a worktree from the configured local bare mirror, invoke the local `opencode` CLI with the general agent and model `gpt-5.4`, let that execution make a simple non-empty repository file change, commit and push the resulting branch, and open a pull request.

This is still a cross-cutting workflow change. It affects Linear activation handling, repo worktree management, execution-runtime behavior, git mutation steps, GitHub branch push and PR creation, and the workflow semantics documented in the existing specs. It is not meant to replace the longer-term OpenSpec proposal direction; it is meant to prove the end-to-end activation-to-PR plumbing first, then later swap the simple file change for real OpenSpec proposal generation.

Because this workflow crosses several system boundaries, operators also need much better runtime visibility than the current high-level logging provides. The bootstrap path should log enough step-level detail to show where a run is, what decision it just made, and why it failed, without dumping secrets or raw prompt bodies.

## Goals / Non-Goals

**Goals:**
- Implement a simpler activation-driven bootstrap PR workflow first.
- Seed the bootstrap execution from the triggering issue's title and description.
- Create the worktree from the configured repository mirror path, make a simple non-empty repository file change through local OpenCode execution, commit it, push it, and open or reuse a pull request.
- Keep the flow local, single-host, and GitHub-App-backed.
- Preserve reconcile-before-create behavior so retries do not create duplicate branches or pull requests.
- Keep the bootstrap execution step easy to replace with later OpenSpec proposal generation.

**Non-Goals:**
- Replacing or removing Heimdall's longer-term OpenSpec proposal workflow direction.
- Generating or maintaining an OpenSpec change as part of this initial activation bootstrap change.
- Letting the user choose the activation bootstrap agent or model in v1.
- Building a general-purpose code-generation orchestration system beyond this single bootstrap path.
- Supporting multiple repository mirrors or agent profiles in a single activation run beyond the resolved target repository.

## Decisions

### Decision: Use a fixed OpenCode execution profile for activation bootstrap
Activation-triggered bootstrap runs will use the local `opencode` CLI with the general agent and model `gpt-5.4`.

Rationale:
- matches the requested behavior exactly
- keeps v1 operator configuration simple
- avoids introducing a second policy surface for activation-time agent selection

Alternatives considered:
- Reuse per-repository apply-agent allowlists. Rejected because the request explicitly wants a fixed bootstrap agent and model.
- Attempt full OpenSpec artifact generation immediately. Rejected because this change is intentionally validating the simpler activation-to-PR path first.

### Decision: Create the worktree from the configured bare mirror path
Heimdall will ensure the bare mirror exists at the resolved repository's configured local mirror path and create the activation worktree from that mirror before invoking OpenCode.

Rationale:
- aligns with the repo-manager architecture already documented for Heimdall
- avoids full clone cost on every activation
- keeps the git mutation path deterministic and recoverable

Alternatives considered:
- Clone the repository fresh for each activation. Rejected because it is slower and ignores the existing mirror-based design.
- Run OpenCode directly in the bare mirror. Rejected because agent-driven file changes require a normal worktree.

### Decision: Derive branch identity from issue content, with description-first slugging
The activation workflow will use a deterministic branch name that includes the issue key and a slug derived from the issue description, falling back to the title when the description does not yield a usable slug.

Representative shape:
- `heimdall/<issue-key>-<description-slug>`

Rationale:
- matches the request to base the branch on the ticket description
- preserves deterministic branch reconciliation for retries
- avoids creating branch names that ignore the richer issue description

Alternatives considered:
- Use only the issue title. Rejected because the request explicitly points to the ticket description.
- Use a random branch name. Rejected because it would break reconcile-before-create behavior.

### Decision: Treat the bootstrap change as successful only when the agent produces real repository mutations
If the OpenCode bootstrap run completes without leaving any file changes to commit, Heimdall will treat the workflow as failed or blocked rather than creating an empty branch or PR.

Rationale:
- avoids misleading automation PRs with no meaningful content
- makes the operator-visible outcome align with the actual repository state

Alternatives considered:
- Open an empty PR describing the issue. Rejected because the requested flow explicitly includes a code change.
- Commit no-op formatting or metadata changes automatically. Rejected because it hides agent failure behind artificial output.

### Decision: Keep the bootstrap mutation intentionally replaceable
The activation bootstrap will only require a simple non-empty repository file change and will avoid introducing durable behavior that depends on the absence of an OpenSpec change. The surrounding activation, worktree, git, and PR orchestration should remain reusable when a later change swaps in actual OpenSpec proposal generation.

Rationale:
- matches the request to do something simpler first
- preserves the longer-term OpenSpec-first product direction
- keeps the future replacement localized to the execution step instead of the whole workflow

Alternatives considered:
- Treat the bootstrap PR as a permanent replacement for OpenSpec proposal generation. Rejected because that is not the requested end state.
- Specify a rigid bootstrap file contract now. Rejected because the important part of this change is proving the end-to-end PR path, not locking in a temporary file shape.

### Decision: Seed the PR title and body from the issue title and description plus bootstrap summary
The PR title will be derived from the issue title, while the PR body will include the source issue reference, the issue description, and a short summary of the generated change.

Rationale:
- satisfies the requirement for an appropriate title and description
- keeps the PR understandable without requiring a separate Heimdall UI
- preserves a clear audit trail from Linear issue to GitHub pull request

Alternatives considered:
- Use only the branch name as the PR title. Rejected because it is less readable for reviewers.
- Use only an agent-generated PR body. Rejected because the source issue context should remain explicit.

### Decision: Emit step-level structured logs for activation bootstrap runs
Heimdall will emit detailed structured logs for the activation-triggered bootstrap workflow, including workflow creation, repository resolution, worktree creation, OpenCode invocation and completion, change detection, branch reconciliation, commit and push attempts, pull request creation or reuse, and terminal workflow outcomes.

Expected log context includes:
- workflow run identifier
- work item identifier and issue key
- target repository identifier
- branch name when known
- worktree path when known
- workflow step name and outcome
- retry, blocked, and failure reasons when applicable

The logging boundary will still exclude secrets, installation tokens, and raw prompt bodies.

Rationale:
- gives operators enough detail to understand what Heimdall is doing during long-running activation flows
- makes failures diagnosable without attaching a debugger or reading the database directly
- keeps the future OpenSpec proposal generation swap easier because the same workflow-level logging can stay in place

Alternatives considered:
- Keep only coarse start and end logs. Rejected because they do not show where bootstrap runs stall or fail.
- Log full prompt and credential context for debugging. Rejected because it creates unnecessary secret and privacy risk.

## Risks / Trade-offs

- The temporary bootstrap step diverges from the project's longer-term OpenSpec-first user experience -> Mitigation: document this as an incremental bootstrap path and keep the swap to real proposal generation localized.
- A fixed agent and model can be brittle if repository content or prompting needs change -> Mitigation: isolate the bootstrap prompt construction behind the execution adapter so it can evolve later.
- Simple file changes may produce low-value PRs -> Mitigation: keep the mutation minimal, treat it as an explicit bootstrap step, and surface failures instead of forcing a PR when the result is poor.
- More detailed logging increases log volume -> Mitigation: keep logs structured and step-focused, and avoid dumping large payloads or prompt bodies.
- Description-derived branch slugs can be messy or empty -> Mitigation: sanitize aggressively and fall back to the title-derived slug when needed.
- Retries can still collide with partially created git state -> Mitigation: reconcile existing bindings, branches, and PRs before creating new ones.

## Migration Plan

1. Update the activation workflow to enqueue the new bootstrap PR workflow as the initial implementation of the activation path.
2. Extend the execution runtime to create worktrees from the configured mirror, invoke OpenCode with the fixed profile, and detect whether real repository changes were produced.
3. Add step-level structured logging across activation, execution, git, and pull request reconciliation steps.
4. Update GitHub mutation logic to commit, push, and open or reuse PRs for the bootstrap flow.
5. Update product and workflow docs to describe the bootstrap-first behavior and the later intent to replace the simple file change with OpenSpec proposal generation.
6. Validate with behavior tests that cover success, no-change failure, branch reconciliation, PR creation, and the expected logging signals.

Rollback is straightforward because the workflow boundary remains activation-driven and repository-scoped. If the bootstrap flow is not acceptable, the service can revert to the earlier activation-driven proposal workflow in a subsequent change.

## Open Questions

None.
