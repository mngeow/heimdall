## Context

Heimdall's current activation path is still modeled as a bootstrap PR flow: detect a Linear issue entering the active lifecycle bucket, create a worktree and branch, generate a small repository mutation, commit it, push it, and open a pull request. The requested product flow replaces that temporary step with the intended durable behavior: create or reuse an OpenSpec change from the activated issue, generate the change artifacts by running local OpenCode with a configurable agent, commit and push the result, and open or reuse a proposal PR.

This is a cross-cutting workflow change. It affects activation orchestration, repo worktree management, OpenSpec/OpenCode execution, git mutation behavior, GitHub PR publishing, and configuration validation. It must remain aligned with Heimdall's single-host, local-CLI, GitHub-App-backed architecture and continue to rely on OpenSpec CLI JSON output as the source of truth for change state.

## Goals / Non-Goals

**Goals:**
- Replace the activation bootstrap mutation with activation-triggered OpenSpec proposal generation.
- Create deterministic branch and change identities from the activated work item so retries reconcile instead of duplicating work.
- Use a configurable repository-level spec-writing agent for activation proposal generation.
- Commit and push the generated OpenSpec artifacts, then open or reuse a proposal PR that carries the configured GitHub monitor label when present.
- Keep activation, refine, and apply responsibilities clearly separated: activation proposes, refine edits artifacts, apply implements tasks.

**Non-Goals:**
- Running `/opsx-apply` automatically after proposal generation.
- Introducing a new label taxonomy beyond the existing configured GitHub PR monitor label.
- Allowing per-issue agent selection from Linear transitions or ad hoc user input.
- Changing the polling-based Linear or GitHub integration model.
- Introducing non-OpenSpec activation outputs such as bootstrap placeholder files.

## Decisions

### Decision: Activation uses the repository default spec-writing agent
Activation-triggered proposal generation and `/heimdall refine` will both use a required per-repository default spec-writing agent configured in dotenv.

Chosen setting:
- `HEIMDALL_REPO_<ID>_DEFAULT_SPEC_WRITING_AGENT`

Rationale:
- gives operators the requested control over which agent authors proposal artifacts
- keeps one consistent agent policy for spec-authoring flows
- avoids hard-coding the current bootstrap-only `general` + `gpt-5.4` profile into the long-term proposal path

Alternatives considered:
- Keep the fixed bootstrap profile. Rejected because it no longer matches the desired activation flow.
- Add a second activation-only agent setting. Rejected because it creates an unnecessary split between activation proposal and refine authoring behavior.
- Let Linear or PR input choose the proposal agent at runtime. Rejected because activation should stay deterministic and operator-controlled.

### Decision: Heimdall creates or reuses the change before invoking the authoring agent
The activation workflow will derive a deterministic change name, create or reuse that change through the local `openspec` CLI, inspect `openspec status --change <name> --json`, and then run `opencode` with the configured agent to generate the required artifacts.

Representative naming:
- branch: `heimdall/ENG-123-add-rate-limiting`
- change: `eng-123-add-rate-limiting`

Rationale:
- keeps OpenSpec CLI JSON output as the source of truth for artifact order and readiness
- makes retries and later PR commands resolve the same change name predictably
- prevents the authoring agent from inventing filesystem layout or change names outside OpenSpec conventions

Alternatives considered:
- Let the agent create the change name and directory structure itself. Rejected because it weakens determinism and bypasses OpenSpec workflow state.
- Run `opencode` before any OpenSpec CLI calls. Rejected because Heimdall would have to guess artifact order and readiness.

### Decision: Activation proposal generation targets implementation-ready proposal artifacts
The activation path will not stop at scaffolding an empty change directory. It will generate the proposal artifacts required by the change's OpenSpec schema before the branch is committed.

For the current `spec-driven` schema, that means activation proposal generation is successful only when the change reaches a state where the apply-required artifact set is complete.

Rationale:
- matches the intended activation-to-review experience of opening a meaningful OpenSpec PR instead of a placeholder
- makes the resulting PR immediately usable for spec review and later `/opsx-apply`
- keeps the activation flow aligned with the repository's existing OpenSpec-first design direction

Alternatives considered:
- Commit only the scaffolded change directory. Rejected because it recreates a bootstrap-style placeholder step.
- Generate only `proposal.md` and leave later artifacts for manual follow-up. Rejected because the repository already expects an implementation-ready OpenSpec change before apply.

### Decision: Worktree, branch, change, and PR reconciliation remain idempotent
Activation retries will first reconcile existing bindings. If a workflow binding already exists for the work item and repository, Heimdall will reuse the existing worktree, branch, change name, and pull request instead of creating another proposal run.

Rationale:
- preserves the current reconcile-before-create operating model
- avoids duplicate change directories and competing pull requests for the same Linear issue
- keeps recovery and repeated polling safe

Alternatives considered:
- Always rerun proposal generation on every active-state observation. Rejected because it would create duplicate work and noisy PR churn.
- Deduplicate only at the pull request layer. Rejected because branch and change naming would still collide earlier in the flow.

### Decision: Proposal PR publishing reuses the existing monitor-label mechanism
The activation proposal PR title and body will reflect the source issue and generated change, while GitHub reconciliation will continue to apply the repository's configured PR monitor label when one is set.

Representative PR title:
- `[ENG-123] OpenSpec proposal for Add rate limiting`

Representative PR body sections:
- source issue reference and description
- generated OpenSpec change name
- short summary of the proposal artifacts created

Rationale:
- satisfies the requested PR title behavior without inventing a second labeling scheme
- preserves compatibility with the existing label-scoped PR polling design
- keeps the PR understandable from GitHub alone

Alternatives considered:
- Add new proposal-specific labels automatically. Rejected because the user explicitly chose to keep only the existing configured monitor label behavior.
- Use only the branch or change name as the PR title. Rejected because reviewers need the source issue context to remain obvious.

## Risks / Trade-offs

- Proposal quality now depends on the configured spec-writing agent and prompt quality -> Mitigation: keep prompt construction and CLI orchestration inside the execution adapter so it can evolve without changing the workflow contract.
- Requiring a repository default spec-writing agent adds operator configuration surface -> Mitigation: keep it to one explicit dotenv key per repository and update docs/examples in the same implementation change.
- Deterministic change names can become stale if a ticket title changes after activation -> Mitigation: derive the name once for the first successful activation binding and reuse it on retries.
- Generating full proposal artifacts during activation increases runtime compared with the old bootstrap file flow -> Mitigation: keep the workflow scoped to artifact generation only and continue using local worktrees and reconcile-before-create behavior.
- OpenSpec readiness checks introduce more CLI calls -> Mitigation: rely on JSON status and instructions responses rather than adding custom filesystem inference or extra orchestration layers.

## Migration Plan

1. Update the activation workflow contract, configuration docs, and durable specs from bootstrap PR creation to activation-triggered OpenSpec proposal generation.
2. Add repository configuration loading and validation for the default spec-writing agent.
3. Extend the activation executor to derive deterministic change names, create or reuse the change through `openspec`, inspect CLI JSON status, and invoke `opencode` with the configured agent.
4. Update git and GitHub publishing so generated OpenSpec artifacts are committed, pushed, and opened as proposal PRs with the existing monitor label behavior.
5. Add and run behavior tests that cover proposal generation success, no-change failure, repeated activation reconciliation, configuration validation, and PR publishing.

Rollback is straightforward because the change stays within the existing activation workflow boundary. Reverting the implementation can restore the earlier bootstrap path without changing the surrounding polling, worktree, or PR ownership model.

## Open Questions

None.
