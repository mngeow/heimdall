# Authentication And Authorization

## GitHub

V1 should use a GitHub App, not a personal access token.

Reasons:

- short-lived installation tokens are safer than long-lived PATs
- repo access can be granted per installation
- permissions are easier to reason about and audit
- the GitHub App model keeps repository access scoped without requiring a user PAT

## Recommended GitHub App Permissions

Start with the smallest practical set:

- Metadata: read
- Contents: read and write
- Pull requests: read and write
- Issues: read and write

Possible later additions:

- Checks: read and write, if Symphony starts publishing richer execution status

## GitHub Polling Intake

Symphony should poll GitHub for at least:

- new pull request comments on Symphony-managed pull requests
- pull request state changes for reconciliation and future branch state handling

The GitHub poller must:

- persist enough checkpoint state to avoid reprocessing already-seen comments
- limit the queried scope to Symphony-managed pull requests or explicitly configured repositories
- avoid logging full raw API payloads that may contain sensitive content

GitHub App webhooks can remain disabled in v1 because command intake and PR-state reconciliation are polling-driven.

## How GitHub Auth Is Used

GitHub App auth is needed in two places:

- GitHub API calls for creating PRs, posting comments, and inspecting repo state
- local git push operations back to the branch

For git push, Symphony should mint an installation token on demand and use HTTPS remotes such as:

```text
https://x-access-token:<token>@github.com/<owner>/<repo>.git
```

The token should be held in memory only for the duration of the operation.

## Linear

V1 should use a Linear API key generated from a dedicated Linear service account and stored as a secret on the Linux host.

Why this is acceptable for V1:

- the deployment is single-user and single-host
- Linear is polled, so OAuth-style redirect handling is not required
- setup is much simpler than building a multi-user auth flow first

The Linear adapter should:

- limit queries to configured teams, projects, or labels
- store only the fields needed for workflow execution
- avoid treating the Linear token as a general-purpose admin credential

## Command Authorization

GitHub comments are an input surface that can cause repo mutations, so comment authorization must be strict.

Recommended policy:

- only accept commands on PRs opened by Symphony
- only accept commands from repo collaborators with write or admin access
- support an explicit per-repo allowlist in config
- require the selected agent to be present in a per-repo allowlist

Example policy:

- `mngeow` may run `/symphony refine`
- `mngeow` may run `/opsx-apply --agent gpt-5.4`
- `random-external-user` may not trigger any mutation workflow

## Secret Handling

Secrets should be injected through environment variables or file paths outside git.

Recommended secret set:

- `SYMPHONY_GITHUB_APP_ID`
- `SYMPHONY_GITHUB_INSTALLATION_ID` if a single installation is used
- `SYMPHONY_GITHUB_PRIVATE_KEY_FILE`
- `SYMPHONY_LINEAR_API_TOKEN`

The service should never:

- write secrets into the SQLite database
- echo secrets in logs
- commit secrets into repo files

## Audit Expectations

Each mutation should leave an audit trail that answers:

- who requested it
- which issue or PR it targeted
- which agent was used
- which commit was created
- whether the action succeeded, failed, or partially completed

This audit record belongs in SQLite and should also be visible through PR comments where practical.
