## 1. Workflow Contract And Docs

- [ ] 1.1 Update the product and workflow docs to describe the activation-triggered bootstrap PR flow instead of the current OpenSpec-first proposal flow.
- [ ] 1.2 Define the bootstrap PR title, PR body, branch naming, and no-change failure behavior from the source issue title and description.

## 2. Activation Bootstrap Execution

- [ ] 2.1 Extend the activation workflow to extract the detected issue title and description and create the bootstrap workflow run for the resolved repository.
- [ ] 2.2 Implement worktree creation from the resolved repository's configured local mirror path before agent execution.
- [ ] 2.3 Implement the local OpenCode bootstrap execution by using the general agent with model `gpt-5.4` and issue-seeded prompt context.
- [ ] 2.4 Detect whether the bootstrap run produced repository changes and fail clearly when no file changes exist to commit.

## 3. Git And GitHub Mutation Path

- [ ] 3.1 Implement deterministic branch naming based on the activated issue description, with a safe fallback when the description does not yield a usable slug.
- [ ] 3.2 Commit and push the bootstrap changes to the activation branch by using the GitHub App installation credentials.
- [ ] 3.3 Open or reuse the bootstrap pull request with the expected title and body derived from the source issue.
- [ ] 3.4 Reconcile existing repository bindings, branches, and pull requests so retries do not duplicate work.

## 4. Testing

- [ ] 4.1 Write Gherkin behavior tests for successful activation-triggered bootstrap PR creation, no-change bootstrap failure, duplicate activation reconciliation, and GitHub PR creation or reuse.
- [ ] 4.2 Implement or update the step bindings, fake OpenCode execution fixtures, git fixtures, and GitHub fixtures needed to execute those activation bootstrap scenarios.
- [ ] 4.3 Add focused Go tests for description-derived branch naming, worktree creation from the configured mirror, and empty-change failure handling.

## 5. Verification

- [ ] 5.1 Run the relevant automated test suite and verify the activation-triggered bootstrap PR scenarios pass.
