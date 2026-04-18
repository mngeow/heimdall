## Context

PR-command execution currently uses a different worktree model than the activation proposal workflow. Proposal runs already derive a deterministic worktree path from the repository's configured local mirror path, but PR-command execution hardcodes `/tmp/heimdall-worktrees/...`, validates change existence before refreshing that worktree, resolves bindings by branch name alone, and leaves opencode execution on a separately scoped or empty working directory. Together, those gaps can make Heimdall reject a valid bound change as stale or run against the wrong checkout.

Separate from the worktree mismatch, Heimdall currently parses `opencode run --format json` stdout with a `bufio.Scanner`-style token reader. Real `text` events can carry a large answer payload on a single newline-delimited JSON line, so refine and apply runs can abort with `bufio.Scanner: token too long` before Heimdall ever sees the later permission, error, or completion events that determine the real outcome.

The current durable proposal spec also describes deterministic change names as `<linear-key>-<slug>`, while the desired contract for this change is that the OpenSpec change name comes from the Linear ticket title after explicit normalization such as converting spaces to hyphens. That naming rule needs to be made explicit so proposal generation and later PR-command targeting agree on the same canonical change identity.

The design must preserve the existing single-host, local-CLI execution model and should avoid introducing new runtime configuration when the repository's mirror path and pull request branch already provide a deterministic worktree identity.

## Goals / Non-Goals

**Goals:**
- Make PR-command execution derive one canonical worktree path from the managed repository and PR head branch.
- Ensure worktree preparation happens before change-existence validation so validation reflects the directory that execution will actually mutate.
- Ensure OpenSpec validation, opencode execution, and commit/push steps all use the same prepared PR worktree.
- Prevent PR-command binding resolution from crossing repository boundaries when branch names happen to match.
- Make PR-comment opencode event parsing robust to very large newline-delimited JSON event lines while preserving current blocker and error classification behavior.
- Make activation proposal derive the canonical OpenSpec change name from the normalized Linear ticket title.
- Preserve proposal workflow behavior for brand-new branches while fixing PR-command behavior for already-existing proposal branches.

**Non-Goals:**
- Adding a new operator-configurable PR worktree root outside the existing local-mirror/worktree model.
- Redesigning the broader workflow-run data model or introducing distributed worktree coordination.
- Changing deterministic proposal branch naming; this change only updates the OpenSpec change-name contract.
- Changing slash-command syntax, agent authorization policy, or permission-approval behavior.
- Redesigning the opencode CLI protocol away from newline-delimited JSON events.

## Decisions

### 1. PR-command worktrees use the same deterministic path strategy as proposal workflows
PR-command execution will derive its worktree path from `workflow.GenerateWorktreePath(repository.LocalMirrorPath, pr.HeadBranch)` instead of hardcoding `/tmp/heimdall-worktrees/...`.

Rationale:
- The repository already has a documented and tested deterministic worktree path strategy.
- Mirror-adjacent worktrees match the documented `/var/lib/heimdall/` layout and reduce split-brain debugging between proposal and PR-command flows.
- No new configuration is required because the mirror path already exists in repository config and runtime state.

Alternatives considered:
- **Keep the `/tmp` path**: rejected because it diverges from the documented filesystem model and created the observed LES-8 mismatch.
- **Persist a separate PR-command worktree path in SQLite**: rejected because the path is already deterministic from mirror path and branch name, so extra persisted state would duplicate derivable data.

### 2. PR-command execution prepares the worktree once, then reuses that path for validation and execution
The executor will derive the canonical worktree path, ensure the mirror is current, reconcile stale worktree registrations, materialize the worktree, and only then run OpenSpec change validation. The same prepared path will be reused for OpenSpec inspection, opencode execution, change detection, commit, and push.

Rationale:
- Validating before preparation lets stale directories or missing git metadata masquerade as real repository state.
- A single prepared path gives operators one place to inspect when debugging a PR command.
- Reusing the same path across validation and execution prevents OpenSpec and opencode from disagreeing about which change exists.

Alternatives considered:
- **Validate against the current on-disk path before refreshing it**: rejected because it preserves the current false stale-binding behavior.
- **Validate against the bare mirror instead of a worktree**: rejected because commands mutate a checkout, not the bare mirror.

### 3. Existing PR branches are materialized from the fetched branch ref, not reseeded from the default branch
Repo worktree creation will prefer the fetched branch ref for `pr.HeadBranch` when that branch already exists in the mirror. Seeding from `repository.DefaultBranch` remains the fallback only for brand-new automation branches that do not yet exist in the mirror, such as the initial proposal workflow branch.

Rationale:
- PR-command runs target an existing proposal branch and must reflect the current remote branch contents.
- Reseeding an existing PR branch from `main` can erase the very OpenSpec change Heimdall is attempting to refine or apply.

Alternatives considered:
- **Always seed from the default branch**: rejected because it recreates the observed false-missing-change failure mode.
- **Add separate proposal and PR-command repo-manager methods immediately**: deferred unless implementation reveals the current helper cannot cleanly support both source-ref cases.

### 4. Binding lookup is anchored to durable pull-request linkage, with repository-scoped fallback for compatibility
PR-command target resolution will use the pull request's persisted `repo_binding_id` when it points to an active binding. If a legacy or repaired row lacks that direct linkage, Heimdall may fall back to active bindings in the same repository and head branch, but it must never consider bindings from another repository solely because the branch name matches.

Rationale:
- `pull_requests.repo_binding_id` already exists specifically to represent the durable relationship between a managed PR and its active binding.
- Repository-scoped fallback avoids forcing an immediate data migration for older or partial rows.

Alternatives considered:
- **Branch-name-only joins**: rejected because cross-repository branch collisions can resolve the wrong change.
- **Require a database migration before any fix ships**: rejected because the existing schema already contains enough linkage to fix current PRs safely.

### 5. Worktree scope must be explicit at the execution boundary
PR-command orchestration will thread the prepared worktree path explicitly into the OpenSpec and OpenCode execution paths used by refine, apply, and generic opencode. Concrete adapters may implement that as per-run client construction or explicit worktree setters, but they must not depend on an empty default cwd or a different path than validation used.

Rationale:
- The current OpenCode adapter can be constructed with an empty directory, which makes PR-command execution vulnerable to drift from the validated worktree.
- Explicit worktree scope is easier to test and aligns with the project's preference for small, explicit interfaces.

Alternatives considered:
- **Leave OpenCode on an implicit process cwd**: rejected because it makes PR-command behavior environment-dependent.
- **Rely on shared mutable client state without command-scoped tests**: rejected because it hides the critical invariant that validation and execution must share one worktree.

### 6. Activation proposal change names use the normalized Linear ticket title
The canonical OpenSpec change name will be derived from the Linear ticket title after normalization to kebab-case. At minimum, normalization will lowercase letters, convert spaces to hyphens, collapse repeated separator runs, trim leading and trailing hyphens, and strip unsupported punctuation. The Linear ticket key may continue to appear in deterministic branch naming, but it is no longer part of the canonical OpenSpec change name.

Rationale:
- The requested contract is that the OpenSpec change identity comes from the Linear ticket title itself.
- Explicit normalization rules prevent spaces and punctuation from producing inconsistent or invalid change names.
- Using the same normalized title-based identity across retries and later PR commands reduces drift between proposal generation and PR-command targeting.

Alternatives considered:
- **Keep `<linear-key>-<slug>` as the OpenSpec change name**: rejected because the desired behavior is title-derived naming rather than key-prefixed naming.
- **Use the raw title without normalization**: rejected because spaces and punctuation make the resulting change names inconsistent and harder to use safely in tooling.

### 7. Opencode JSON events are parsed as an NDJSON stream without scanner token limits
Heimdall will treat `opencode run --format json` stdout as newline-delimited JSON and parse it with a reader that can consume complete lines larger than the default `bufio.Scanner` token cap. The parser will unmarshal one complete line at a time, ignore non-JSON noise lines, preserve the final valid line even if it ends at EOF without a trailing newline, and keep only the minimal execution state needed to classify permission, input, error, and terminal outcomes.

Rationale:
- Real `text` events can place a full answer on one JSON line, so the existing scanner-based reader fails before Heimdall can observe the real later events.
- NDJSON preserves event boundaries cleanly, so a line-oriented reader matches the CLI contract without buffering the entire stream.
- Preserving the final EOF-terminated line avoids a brittle requirement that the opencode process always flush a trailing newline before exit.

Alternatives considered:
- **Increase the scanner buffer size**: rejected because it still hardcodes a maximum line size and turns a real protocol-level stream into a guessed threshold.
- **Use a raw `json.Decoder` over the entire stdout stream**: rejected because the current CLI path can emit non-JSON help or log noise, and line-scoped recovery is simpler and safer for that mixed-output model.

## Risks / Trade-offs

- **[Risk] Mirror-adjacent worktree paths may differ from ad hoc operator checkouts** → **Mitigation:** make the deterministic PR-command path part of the documented and tested behavior so operators know which checkout Heimdall uses.
- **[Risk] Legacy pull-request rows without `repo_binding_id` could still fail if repository-scoped fallback is not implemented carefully** → **Mitigation:** require exact-binding lookup first, then same-repository fallback, with explicit ambiguity rejection tests.
- **[Risk] Branch-ref detection depends on the mirror being freshly fetched** → **Mitigation:** keep `EnsureBareMirror` as a prerequisite for every PR-command worktree preparation.
- **[Risk] Recreating worktrees more deterministically may surface stale-registration issues more often** → **Mitigation:** reuse and extend the existing stale-worktree reconciliation behavior and add regression tests around retries.
- **[Risk] Title-derived change names can collide when different Linear tickets share the same normalized title** → **Mitigation:** keep OpenSpec change discovery as the runtime source of truth for binding persistence and cover the normalized title contract explicitly in proposal-generation tests.
- **[Risk] A single opencode event line can still be very large in memory** → **Mitigation:** read one NDJSON event line at a time, retain only minimal classification state, and add regression coverage for large-event runs rather than buffering the full stream history.

## Migration Plan

1. Update PR-command orchestration to derive the canonical worktree path from the repository mirror path and PR head branch.
2. Change worktree preparation to run before change validation and to return the prepared path for downstream steps.
3. Update repo-manager worktree materialization to prefer an existing fetched branch ref before falling back to the default branch.
4. Update binding lookup to prefer `pull_requests.repo_binding_id` with repository-scoped fallback for legacy rows.
5. Update activation proposal naming so the intended OpenSpec change name is derived from the normalized Linear ticket title.
6. Update OpenSpec and OpenCode execution boundaries so both are scoped to the same prepared worktree.
7. Replace the scanner-limited opencode event reader with an NDJSON parser that tolerates very large single-event lines and EOF-terminated final events.
8. Roll out with focused regression tests for false stale-binding rejection, branch-name collisions across repositories, worktree recreation of an existing PR branch, title-to-kebab-case change-name generation, and large opencode event streams.

Rollback is code-only: revert the executor, repo-manager, and store-query changes. No new runtime schema or external API migration is required.

## Open Questions

- None blocking. The current schema, CLI contract, and documented filesystem model are sufficient for this fix set.
