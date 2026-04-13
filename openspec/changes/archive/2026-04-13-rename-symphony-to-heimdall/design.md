## Context

`Symphony` currently appears in nearly every operator-facing and implementation-facing surface in the repository: the Go module path, import paths, binary naming, branch prefixes, bootstrap file paths, slash commands, environment-variable prefixes, default filesystem paths, database naming, docs, tests, OpenSpec specs, and archived change artifacts. The requested rename is therefore a cross-cutting identity migration rather than a narrow branding edit.

The repository is still early enough that a clean break is preferable to carrying dual names indefinitely. At the same time, the change must remain explicit about the surfaces that define workflow identity so implementation can update them consistently instead of leaving a partially renamed system.

## Goals / Non-Goals

**Goals:**
- Establish `Heimdall` as the single canonical product, repository, binary, module, command, branch, config, and documentation identity.
- Replace all in-repo `Symphony` references, including active and archived OpenSpec artifacts, with `Heimdall` references where they describe the current system.
- Make breaking operator-facing namespace changes explicit, especially for slash commands, branch prefixes, bootstrap paths, dotenv keys, filesystem defaults, monitor-label examples, and runtime database naming.
- Give `/opsx-apply` a deterministic task list that can execute the rename without guessing which categories of references are in or out of scope.

**Non-Goals:**
- Adding alias support that keeps both `Symphony` and `Heimdall` names working indefinitely.
- Changing the service architecture, provider model, auth model, or workflow semantics beyond the naming surfaces required by the rename.
- Introducing new product capabilities unrelated to the rename.

## Decisions

### 1. Treat the rename as a full repository identity migration
The implementation will update all maintained in-repo references that define or describe the product's identity, not just user-facing marketing text.

This includes:
- repository and module naming references
- source-code identifiers and import paths that embed `symphony`
- command namespaces such as `/symphony ...`
- deterministic branch and bootstrap path conventions
- dotenv prefixes and default filesystem/database paths
- docs, tests, durable specs, active change artifacts, and archived change artifacts

**Why:** A partial rename would leave the operator experience and developer workflow inconsistent.

**Alternative considered:** Rename only docs and top-level branding while leaving internal namespaces unchanged. Rejected because it preserves long-term confusion and contradicts the request to change all `Symphony` references.

### 2. Make the breaking namespace transition atomic
Implementation should switch canonical names in one change instead of supporting both `SYMPHONY_*` and `HEIMDALL_*`, both `/symphony` and `/heimdall`, or both `.symphony/` and `.heimdall/` during an extended compatibility period.

**Why:** The project is still early-stage, and a clean break keeps configuration, tests, documentation, and workflow logic simpler.

**Alternative considered:** Temporary dual-read or dual-command compatibility. Rejected because it increases parser, config, and test complexity for little value at the current maturity of the system.

### 3. Rewrite archived OpenSpec artifacts for naming consistency
Archived changes under `openspec/changes/archive/` will be renamed in place where they reference `Symphony` as the product identity.

**Why:** The request is to remove `Symphony` references across the repo, and leaving the archive untouched would preserve a large pocket of stale naming.

**Alternative considered:** Preserve archived artifacts as historical records under the old name. Rejected for this change because repo-wide naming consistency is the priority.

### 4. Use a single canonical mapping for all renamed namespaces
Implementation should apply one explicit mapping set everywhere the old name appears:

| Current | New |
| --- | --- |
| `Symphony` | `Heimdall` |
| `symphony` | `heimdall` |
| `/symphony` | `/heimdall` |
| `SYMPHONY_` | `HEIMDALL_` |
| `.symphony/` | `.heimdall/` |
| `symphony/` branch prefix | `heimdall/` branch prefix |
| `symphony-monitored` examples/defaults | `heimdall-monitored` examples/defaults |
| `/etc/symphony`, `/var/lib/symphony`, `/var/log/symphony` | `/etc/heimdall`, `/var/lib/heimdall`, `/var/log/heimdall` |
| `symphony.db` | `heimdall.db` |

**Why:** A shared mapping reduces accidental partial renames and gives tests and docs a single source of truth.

**Alternative considered:** Renaming each subsystem independently. Rejected because it makes it easier for code, tests, and docs to drift.

### 5. Require pre-deployment cleanup of existing Symphony-managed runtime assets
Because the rename is atomic and intentionally does not preserve legacy aliases, operators should complete, close, or manually reconcile any live `symphony/*` automation branches, `Symphony`-named service units, and `SYMPHONY_*`-based deployments before switching to the renamed build.

**Why:** The renamed service should not need fallback logic for legacy namespaces to remain safe and deterministic.

**Alternative considered:** Keep legacy branch or command recognition after the rename. Rejected because it extends the compatibility surface indefinitely.

## Risks / Trade-offs

- **[Existing operator configs break on upgrade]** → Document the rename as breaking, update docs/examples together, and require migration from `SYMPHONY_*` keys and old filesystem paths before deployment.
- **[Open automation branches or PRs still use old naming]** → Call out pre-deployment cleanup and ensure the implementation updates managed-PR identity logic consistently.
- **[Literal search/replace introduces accidental non-product edits]** → Use targeted review across source, docs, tests, and OpenSpec artifacts instead of blind global replacement.
- **[Rewriting archived changes reduces historical fidelity]** → Accept the trade-off explicitly for repo-wide naming consistency and keep the change limited to naming, not semantic rewrites.

## Migration Plan

1. Update durable OpenSpec specs, proposal artifacts, and tasks so the rename scope is implementation-ready.
2. Rename implementation-facing identifiers and conventions together: module path, binary path, branch/bootstrap naming, command namespace, config prefix, filesystem defaults, label examples, and runtime defaults.
3. Update tests and fixtures to use the new canonical names and examples.
4. Update docs and setup guidance so a fresh operator install uses only `Heimdall` naming.
5. Rewrite remaining in-repo OpenSpec references, including archived artifacts, for consistency.
6. Before deploying the renamed build, migrate operator config and paths and reconcile any still-open `Symphony`-managed runtime artifacts.

Rollback is a standard git revert of the rename change before deployment. After deployment, rollback would also require restoring the previous `SYMPHONY_*`-based configuration and old filesystem/service naming.

## Open Questions

- None at proposal time.
