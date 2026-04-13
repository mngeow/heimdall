## Why

Heimdall currently has no GitHub-native marker for which automation pull requests it should monitor, so operators cannot see or control that monitoring scope from GitHub itself. Adding a configured pull request monitor label makes the monitored set explicit, lets Heimdall narrow polling safely, and now requires the GitHub App setup docs to state the exact permissions needed for PR-only creation, labeling, comment polling, and collaborator authorization.

## What Changes

- Add an optional per-repository dotenv setting `HEIMDALL_REPO_<ID>_PR_MONITOR_LABEL` that names the GitHub label Heimdall uses to mark monitored automation pull requests.
- Require Heimdall to create or reuse that repository label automatically when the setting is configured.
- Require Heimdall to add the monitor label automatically when it opens or reuses a Heimdall pull request, and to use that label as an additional GitHub polling filter when the setting is configured.
- Update GitHub operator docs to describe the exact minimum GitHub App permissions needed for branch pushes, pull request creation, collaborator-permission reads, PR comment polling, and PR labeling.

## Capabilities

### New Capabilities

### Modified Capabilities
- `service-configuration`: add an optional per-repository dotenv key for the GitHub pull request monitor label and validate it at startup.
- `service-github-scm`: create or reuse the configured monitor label, apply it to Heimdall-managed pull requests, and narrow GitHub polling to labeled Heimdall pull requests when configured.

## Impact

- Affects runtime configuration parsing and validation for per-repository GitHub settings.
- Affects GitHub label reconciliation, pull request create-or-reuse logic, and GitHub polling scope.
- Affects operator setup documentation because the GitHub App permissions and label-related `.env` settings change.
