## Context

Symphony already relies on GitHub App polling for pull request comments and lifecycle changes, but it currently identifies eligible pull requests only through internal Symphony bindings. That keeps the monitored set implicit, makes it harder for operators to see which pull requests Symphony should be watching, and gives the GitHub adapter no repo-native marker it can use to narrow polling safely.

This change introduces an optional per-repository GitHub label that marks monitored Symphony pull requests. The design needs to cover configuration, label reconciliation, pull request create-or-reuse behavior, polling filters, and the operator-facing GitHub App permission guidance. It also needs to stay inside the current v1 constraints: one binary, GitHub App auth, polling instead of webhooks, per-repo dotenv settings, and reconcile-before-create behavior.

## Goals / Non-Goals

**Goals:**
- Add an optional per-repository dotenv key that names the GitHub label Symphony should use for monitored pull requests.
- Ensure the configured repository label exists without requiring the operator to pre-create it manually.
- Automatically add that label to Symphony-created or Symphony-reused pull requests.
- Narrow GitHub polling to Symphony-managed pull requests that carry the configured label when the setting is present.
- Document the exact minimum GitHub App permissions needed for Symphony's current PR-only GitHub flow.

**Non-Goals:**
- Labeling arbitrary non-Symphony pull requests.
- Treating the GitHub label as the only source of truth for whether a pull request belongs to Symphony.
- Adding user-configurable label color or description settings in v1.
- Requiring every repository to opt into label-scoped monitoring immediately.

## Decisions

### Decision: Use an optional per-repository `SYMPHONY_REPO_<ID>_PR_MONITOR_LABEL` setting

Symphony will add a new optional dotenv key under each repository block, for example `SYMPHONY_REPO_PLATFORM_PR_MONITOR_LABEL=symphony-monitored`.

Rationale:
- keeps the feature repo-scoped, explicit, and aligned with the existing `SYMPHONY_REPO_<ID>_*` naming scheme
- lets operators enable label-based monitoring only where they want it
- preserves current behavior for repositories that do not configure the setting

Alternatives considered:
- A single global GitHub label setting. Rejected because repository blocks are already the unit of routing and authorization, and repo-specific names are more explicit.
- A hardcoded label name. Rejected because operators may already have naming conventions they want to follow.

### Decision: Create missing labels, but reuse existing labels without rewriting them

When a repository configures a PR monitor label, Symphony will ensure that a repository label with that name exists. If the label is missing, Symphony will create it with a deterministic default description and color. If the label already exists, Symphony will reuse it as-is instead of overwriting user-managed metadata.

Rationale:
- removes manual setup work for the operator
- keeps label reconciliation idempotent
- avoids clobbering an existing repository label that already uses the configured name

Alternatives considered:
- Require operators to create the label manually. Rejected because the request explicitly wants Symphony to create it automatically.
- Force the label's description and color to match Symphony defaults on every reconcile. Rejected because it would overwrite operator-managed presentation for little product value.

### Decision: Add the monitor label during PR create-or-reuse reconciliation

Symphony will add the configured label to a pull request after the PR exists, and it will do so both for newly created pull requests and for existing Symphony pull requests that are being reused or reconciled. The adapter should use additive label application so unrelated labels are preserved.

Rationale:
- backfills existing open Symphony pull requests after rollout instead of only handling new ones
- preserves operator-added labels and avoids destructive label replacement
- keeps the label attached to the same PR identity Symphony already manages

Alternatives considered:
- Label only newly created pull requests. Rejected because already-open Symphony pull requests would remain outside the label-based monitoring scope until manually fixed.
- Replace the full label set on the pull request. Rejected because it could remove unrelated workflow labels that operators rely on.

### Decision: Treat the monitor label as an additional polling filter, not the sole ownership signal

When a repository configures a PR monitor label, the GitHub poller will only treat pull request activity as eligible if the pull request is both Symphony-managed and currently carries the configured label.

Rationale:
- prevents a manually added label on a non-Symphony pull request from expanding Symphony's mutation surface
- keeps the label useful as a repo-native filter without replacing internal Symphony bindings
- matches the request to tag the PRs Symphony should monitor for events

Alternatives considered:
- Use the label alone to determine eligibility. Rejected because it would let an external label mutation broaden the monitored set.
- Keep polling behavior unchanged and use the label only as decoration. Rejected because the request explicitly ties the label to the PRs Symphony monitors.

### Decision: Document the exact minimum GitHub App permissions for the current PR-only flow

The setup docs will state the minimum repository permissions as:
- `Metadata: read`
- `Contents: read and write`
- `Pull requests: read and write`

The docs will also explain that Symphony's current v1 flow does not require `Issues` permission because GitHub's GitHub App permissions matrix allows pull-request comment and label operations under `Pull requests` for PR-backed flows, while collaborator-permission reads remain covered by `Metadata`.

Rationale:
- keeps the GitHub App setup principle aligned with least privilege
- reflects the exact PR-only operations Symphony performs today
- makes the new label creation and label-assignment behavior auditable and easier to approve

Alternatives considered:
- Keep recommending `Issues: read and write` for simplicity. Rejected because the request explicitly asks for the exact permissions needed.
- Remove `Metadata` and rely on broader permissions elsewhere. Rejected because collaborator-permission reads and repository identity checks already map cleanly to `Metadata: read`.

## Risks / Trade-offs

- [An operator removes the monitor label from an open Symphony pull request] -> Mitigation: re-apply the label during PR reconciliation and document that unlabeled PRs are ignored when label-scoped monitoring is enabled.
- [A repository already has a label with the configured name but different semantics] -> Mitigation: document that the configured name should be reserved for Symphony monitoring and reuse existing labels without silently rewriting their presentation.
- [Opt-in config means some repositories still use the older unlabeled monitoring path] -> Mitigation: keep the setting explicit and document the behavior clearly so operators can roll out per repository.
- [Label-based filtering could be mistaken for the only ownership signal] -> Mitigation: require both a Symphony-managed binding and the label when the filter is enabled.

## Migration Plan

1. Add `SYMPHONY_REPO_<ID>_PR_MONITOR_LABEL` to the dotenv schema and validate it when present.
2. Extend the GitHub adapter to create or reuse the configured repository label.
3. Update pull request create-or-reuse reconciliation so Symphony adds the monitor label without replacing other labels.
4. Narrow GitHub polling to labeled Symphony pull requests only when the repository config includes the new setting.
5. Update the GitHub setup and authentication docs to describe the new label setting and the exact GitHub App permissions.
6. Verify against behavior tests and a test repository that the label is created, applied to Symphony pull requests, and used to narrow polling.

Rollback is operational: remove `SYMPHONY_REPO_<ID>_PR_MONITOR_LABEL` from the repository config and redeploy. The GitHub label can remain in the repository; Symphony simply stops requiring it for monitoring.

## Open Questions

None.
