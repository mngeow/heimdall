## ADDED Requirements

### Requirement: Pull-request binding lookup stays tied to durable PR linkage
Heimdall MUST resolve active bindings for PR-comment execution from the persisted pull-request record and same-repository context. When `pull_requests.repo_binding_id` points to an active binding, Heimdall MUST use that exact binding. If a compatible legacy row lacks that direct linkage, Heimdall MUST fall back only to active bindings in the same repository and head branch, and it MUST NOT consider bindings from another repository solely because the branch name matches.

#### Scenario: Pull request uses its persisted repo binding link
- **WHEN** a Heimdall-managed pull request record has a `repo_binding_id` that points to an active binding
- **THEN** Heimdall resolves PR-command target changes from that exact binding
- **AND** it does not ignore the stored pull-request linkage in favor of a broader branch-name search

#### Scenario: Legacy pull request row falls back to the same repository and branch
- **WHEN** a Heimdall-managed pull request record lacks a usable direct binding link but exactly one active binding in the same repository matches the pull request head branch
- **THEN** Heimdall may use that same-repository binding as the PR-command context
- **AND** it does not require a manual repair before the command can continue

#### Scenario: Same branch name in another repository is ignored during fallback
- **WHEN** Heimdall falls back to repository-and-branch matching for PR-command binding resolution and another repository has an active binding with the same branch name
- **THEN** Heimdall ignores the binding from the other repository
- **AND** it resolves only bindings that belong to the pull request's own repository context
