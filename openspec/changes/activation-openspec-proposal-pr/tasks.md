## 1. Workflow Contract And Documentation

- [ ] 1.1 Update `docs/README.md`, `docs/product.md`, `docs/workflows.md`, and `docs/architecture.md` to replace the activation bootstrap path with activation-triggered OpenSpec proposal generation.
- [ ] 1.2 Update `docs/operations.md`, `docs/setup/github.md`, `README.md`, and the supported env template files to document `HEIMDALL_REPO_<ID>_DEFAULT_SPEC_WRITING_AGENT`, proposal PR titles, deterministic change naming, and existing monitor-label behavior.

## 2. Configuration And Activation Proposal Execution

- [ ] 2.1 Extend repository configuration loading and readiness validation to require `HEIMDALL_REPO_<ID>_DEFAULT_SPEC_WRITING_AGENT` and surface clear operator errors when it is missing or blank.
- [ ] 2.2 Update the activation workflow to derive deterministic branch and change names, create or reuse the OpenSpec change through the local `openspec` CLI, and use CLI JSON status/instructions as the source of truth for required artifacts.
- [ ] 2.3 Implement activation proposal generation through local `opencode` by using the repository's default spec-writing agent plus the Linear issue title and description as prompt context until the apply-required artifacts are complete.
- [ ] 2.4 Detect and fail activation proposal runs that do not leave commit-ready repository changes, and record audit metadata for the `openspec`, `opencode`, `git`, and GitHub publishing steps.
- [ ] 2.5 Reconcile existing workflow bindings so repeated activation reuses the same worktree, branch, change, and pull request instead of creating duplicate proposal runs.

## 3. Git And GitHub Publishing

- [ ] 3.1 Ensure activation proposal worktrees are created from the configured repository mirror and reused safely across retries.
- [ ] 3.2 Update the git mutation path to commit and push generated OpenSpec artifacts to the deterministic activation branch by using GitHub App installation credentials.
- [ ] 3.3 Update pull request create-or-reuse logic to publish the proposal-focused title and body from the source issue and generated change, while preserving or applying the configured GitHub PR monitor label.
- [ ] 3.4 Persist and reconcile the repository, branch, change, and pull request bindings needed for later `/heimdall refine` and `/opsx-apply` commands.

## 4. Behavior And Unit Test Coverage

- [ ] 4.1 Write Gherkin `.feature` scenarios for successful activation proposal generation, missing default spec-writing agent configuration, no-change proposal failure, duplicate activation reconciliation, and proposal PR creation or reuse with the configured monitor label.
- [ ] 4.2 Implement or update the Godog step bindings, fake OpenSpec/OpenCode fixtures, git fixtures, GitHub fixtures, and config fixtures needed to execute the new activation proposal scenarios.
- [ ] 4.3 Add focused Go tests for deterministic change naming, CLI-status-driven artifact selection, proposal PR title/body generation, and monitor-label reconciliation on activation-created pull requests.

## 5. Verification

- [ ] 5.1 Run the relevant automated test suite and verify the activation proposal behavior scenarios pass.
- [ ] 5.2 Verify from test captures or runtime logs that activation proposal runs record the change name, selected default spec-writing agent, branch, and PR outcome without leaking secrets or raw prompt bodies.
