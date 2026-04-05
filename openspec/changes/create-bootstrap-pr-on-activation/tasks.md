## 1. Workflow Contract And Docs

- [ ] 1.1 Update the product and workflow docs to describe the activation-triggered bootstrap PR flow as the initial implementation step, and note that the simple file change is intended to later be replaced by OpenSpec proposal generation.
- [ ] 1.2 Define the bootstrap PR title, PR body, branch naming, simple file-change expectations, and no-change failure behavior from the source issue title and description.
- [ ] 1.3 Define the detailed logging expectations for activation bootstrap runs, including the key workflow fields, step names, and redaction boundaries operators should rely on.

## 2. Activation Bootstrap Execution

- [ ] 2.1 Extend the activation workflow to extract the detected issue title and description and create the bootstrap workflow run for the resolved repository.
- [ ] 2.2 Implement worktree creation from the resolved repository's configured local mirror path before agent execution.
- [ ] 2.3 Implement the local OpenCode bootstrap execution by using the general agent with model `gpt-5.4` and issue-seeded prompt context that produces a small non-empty repository file change.
- [ ] 2.4 Detect whether the bootstrap run produced repository changes and fail clearly when no file changes exist to commit.
- [ ] 2.5 Emit structured logs for activation workflow creation, repository resolution, issue seeding, worktree creation, OpenCode start and completion, and empty-change failures.

## 3. Git And GitHub Mutation Path

- [ ] 3.1 Implement deterministic branch naming based on the activated issue description, with a safe fallback when the description does not yield a usable slug.
- [ ] 3.2 Commit and push the bootstrap file change to the activation branch by using the GitHub App installation credentials.
- [ ] 3.3 Open or reuse the bootstrap pull request with the expected title and body derived from the source issue.
- [ ] 3.4 Reconcile existing repository bindings, branches, and pull requests so retries do not duplicate work.
- [ ] 3.5 Emit structured logs for branch reconciliation, commit and push attempts, pull request create-or-reuse decisions, and terminal workflow outcomes.

## 4. Testing

- [ ] 4.1 Write Gherkin behavior tests for successful activation-triggered bootstrap PR creation with a simple file change, no-change bootstrap failure, duplicate activation reconciliation, and GitHub PR creation or reuse.
- [ ] 4.2 Implement or update the step bindings, fake OpenCode execution fixtures, git fixtures, and GitHub fixtures needed to execute those activation bootstrap scenarios.
- [ ] 4.3 Add focused Go tests for description-derived branch naming, worktree creation from the configured mirror, and empty-change failure handling.
- [ ] 4.4 Add focused tests for the bootstrap logging output so expected workflow identifiers, step transitions, and failure reasons are emitted without leaking secrets or raw prompt bodies.

## 5. Verification

- [ ] 5.1 Run the relevant automated test suite and verify the activation-triggered bootstrap PR scenarios pass.
- [ ] 5.2 Verify from service logs or test captures that the new bootstrap logging is detailed enough for operators to follow workflow progress end to end.
