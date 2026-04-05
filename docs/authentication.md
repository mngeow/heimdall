# Authentication And Authorization

## GitHub

V1 should use a GitHub App, not a personal access token.

Reasons:

- short-lived installation tokens are safer than long-lived PATs
- repo access can be granted per installation
- permissions are easier to reason about and audit
- installation-scoped API access fits Symphony's polling-based GitHub model

## Recommended GitHub App Permissions

Start with the smallest practical set:

- Metadata: read
- Contents: read and write
- Pull requests: read and write
- Issues: read and write

Possible later additions:

- Checks: read and write, if Symphony starts publishing richer execution status

## GitHub Poll Intake

Symphony should poll for at least:

- pull request comments through GitHub's issue-comment model for slash commands on pull requests
- pull request state for reconciliation and future branch state handling

The GitHub polling path must:

- use GitHub App installation authentication for API reads
- filter candidate comments down to Symphony-managed pull requests before mutation logic runs
- dedupe by stable comment identity rather than timestamp alone
- avoid logging full raw comment or API payload bodies that may contain sensitive content

## How GitHub Auth Is Used

GitHub App auth is needed in two places:

- GitHub API calls for polling comments, inspecting PR state, creating PRs, posting comments, and inspecting repo state
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

- limit queries to the configured Linear project in v1
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
