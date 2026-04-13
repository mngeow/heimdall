# GitHub Setup

## Purpose

Heimdall should use a GitHub App for repository access, pull request creation, pull request comments, and polling-based pull request command intake.

V1 should not use a personal access token.

V1 should also not require a public GitHub webhook endpoint. Under this setup model, Heimdall only needs outbound HTTPS access to the GitHub API.

This document reflects the polling-based GitHub model requested for Heimdall. The broader repo-wide design follow-up is tracked in `openspec/changes/poll-github-pr-commands/`.

## Polling Model At A Glance

Instead of asking GitHub to push events into Heimdall, Heimdall should wake up on a fixed interval, read new pull request activity through the GitHub API, and then decide whether any supported command was posted on a Heimdall-managed pull request.

```mermaid
flowchart LR
    Timer["GitHub poll interval"] --> PRs["List Heimdall-managed PRs"]
    PRs --> Comments["Read new issue comments"]
    Comments --> Dedupe["Dedupe by comment ID"]
    Dedupe --> Auth["Authorize actor and parse command"]
    Auth --> Queue["Queue Heimdall workflow"]
```

Important consequences:

- no public DNS name is required for GitHub event intake
- no inbound firewall rule is required for GitHub event intake
- no GitHub webhook secret is required for the normal v1 setup path
- command latency is now bounded by the polling interval instead of arriving in real time

## What You Need Before Starting

- the GitHub organization or user account that owns the repository
- the repository or repositories Heimdall should manage
- permission to create and install a GitHub App on those repositories
- outbound HTTPS from the Heimdall host to `github.com` and `api.github.com`
- a place to store the GitHub App private key outside git
- optionally, the `gh` CLI if you want an easy way to inspect installation details from the terminal

You do not need:

- a public HTTPS URL for Heimdall
- a reverse proxy just for GitHub event delivery
- a webhook secret

## Step 1: Create Or Reconfigure The GitHub App

Create a GitHub App dedicated to Heimdall.

If you already created one for webhook delivery, you can usually keep the same app and edit its settings so it no longer expects webhook traffic.

Recommended values when creating or editing the app:

- App name: `Heimdall`
- Description: optional, but it is helpful to note that this app is used for polling-based Heimdall automation
- Homepage URL: your project or operator docs URL
- Callback URL: leave blank unless you later add a user-auth flow
- Request user authorization during installation: off
- Device flow: off
- Setup URL: blank unless you have your own operator onboarding page
- Active: disabled or unchecked
- Webhook URL: leave blank
- Webhook secret: leave blank

Precise GitHub UI detail:

- the critical setting is the GitHub App "Active" checkbox in the webhook section
- when "Active" is disabled, GitHub does not send webhook deliveries for the app
- once webhook delivery is disabled, the webhook URL and webhook secret are not needed for Heimdall's normal v1 flow

If you are converting an existing app from webhook mode to polling mode:

1. Open the GitHub App settings page.
2. Find the webhook section.
3. Disable the "Active" setting.
4. Remove any old webhook URL if GitHub still shows one.
5. Remove any stored webhook secret from your host secret store.
6. Save the app settings.

## Step 2: Configure Repository Permissions

Use the smallest permissions that still allow Heimdall to function.

Recommended repository permissions:

- Metadata: read-only
- Contents: read and write
- Pull requests: read and write

Why these are needed:

- Metadata lets Heimdall inspect repository identity, installation scope, and collaborator permission levels for command authorization.
- Contents lets Heimdall push proposal and apply branches.
- Pull requests lets Heimdall create, read, and update pull requests, poll and publish PR comments, create the repository label used for monitored PRs, and add that label to pull requests.
- Issues is not required for Heimdall's current PR-only flow. GitHub exposes pull request comments and labels through issues-style endpoints, but GitHub's GitHub App permissions matrix allows those pull-request operations under `Pull requests` permission. Add `Issues` only if Heimdall later needs to act on standalone GitHub issues.

Do not add broader permissions until there is a concrete need.

## Step 3: Do Not Subscribe To Webhook Events

Under the polling model, GitHub does not need to deliver `issue_comment` or `pull_request` events to Heimdall.

That means:

- leave webhook delivery disabled
- do not configure webhook event subscriptions for the normal v1 path
- do not provision a public `/webhooks/github` endpoint just for GitHub command handling

Operator note:

- pull request comments are still represented as issue comments in GitHub's data model
- that detail matters for API polling logic, but not for GitHub App webhook setup, because webhooks are intentionally not used here

## Step 4: Generate The Private Key

After the app is created:

1. Generate a private key from the GitHub App settings page.
2. Store it in a root-readable path outside the repository, for example `/etc/heimdall/github-app.pem`.
3. Set restrictive file permissions such as `0400` or `0600`.
4. Record the GitHub App ID.

Example:

```bash
install -m 0400 /path/to/downloaded/private-key.pem /etc/heimdall/github-app.pem
```

Treat the private key as a critical recovery asset. Anyone with the private key and app metadata can mint installation tokens.

## Step 5: Install The App On Target Repositories

Install the app on every repository Heimdall should manage.

If you only want Heimdall to manage a subset of repositories, install it only on that subset.

After installation, record the installation ID if your deployment uses a single installation.

Precise ways to find the installation ID:

1. GitHub UI path:
   open the installation configuration page for the repository or organization and inspect the installation details page.
2. `gh` CLI path:

```bash
gh api "/repos/<owner>/<repo>/installation" --jq '.id'
```

That command is useful when you are authenticated in `gh` as a user who can inspect the target repository installation.

## Step 6: Choose Polling Settings Carefully

Polling removes the public ingress requirement, but it introduces timing and rate-limit choices that must be explicit.

Recommended starting values for v1:

- poll interval: `30s`
- overlap or lookback window: `2m`
- pull request scope: only repositories Heimdall manages, and preferably only open Heimdall-managed pull requests

How to choose these values:

- A `30s` poll interval keeps command latency low enough for normal operator use without creating unnecessary API pressure.
- A `2m` overlap or lookback window gives Heimdall room to survive clock skew, temporary GitHub API lag, process restarts, or a missed poll cycle.
- Restricting the scope to Heimdall-managed pull requests keeps the poller predictable and avoids scanning unrelated repository traffic.

Operational rule:

- the lookback window should be larger than the poll interval
- Heimdall should dedupe by stable comment identity, not by timestamp alone
- overlapping windows are expected and safe only if command deduplication is durable

## Step 7: Wire The App Into Heimdall `.env` And Secrets

Set these entries in the project-root `.env` file or the process environment:

```bash
HEIMDALL_GITHUB_APP_ID=<app-id>
HEIMDALL_GITHUB_INSTALLATION_ID=<installation-id>
HEIMDALL_GITHUB_PRIVATE_KEY_FILE=/etc/heimdall/github-app.pem
```

Do not set `HEIMDALL_GITHUB_WEBHOOK_SECRET` for the polling-based setup path.

Your Heimdall environment-variable schema should capture, at minimum, these GitHub-side semantics:

- base branch, usually `main`
- GitHub polling interval
- GitHub overlap or lookback window
- repo routing rules
- allowed GitHub users
- allowed apply agents

One reasonable polling-oriented `.env` shape is:

```dotenv
HEIMDALL_GITHUB_BASE_BRANCH=main
HEIMDALL_GITHUB_POLL_INTERVAL=30s
HEIMDALL_GITHUB_LOOKBACK_WINDOW=2m
```

If you want GitHub polling in repository `PLATFORM` limited to explicitly labeled Heimdall pull requests, add the optional per-repository setting:

```dotenv
HEIMDALL_REPO_PLATFORM_PR_MONITOR_LABEL=heimdall-monitored
```

When this setting is present, Heimdall should:

- create the repository label automatically if it does not already exist
- reuse the existing repository label if it already exists
- add that label automatically to Heimdall-created or reconciled pull requests for that repository
- ignore unlabeled pull requests in that repository for comment and lifecycle polling

If your exact config schema differs, preserve the same meaning even if the final field names change.

## Step 8: Know What Heimdall Should Poll

For GitHub command intake, Heimdall should perform a narrow polling loop rather than a broad repository crawl.

Recommended cycle behavior:

1. Load the last successful GitHub polling checkpoint from SQLite.
2. Compute a safe read window such as `last_successful_poll - lookback_window`.
3. Enumerate only the repositories managed by Heimdall.
4. Enumerate only the open pull requests Heimdall already manages, and when a repository configures `HEIMDALL_REPO_<ID>_PR_MONITOR_LABEL`, restrict that set further to pull requests that currently carry the configured label.
5. Read issue comments that are new within the polling window.
6. Ignore comments on non-Heimdall pull requests, and ignore unlabeled pull requests when label-scoped monitoring is configured for that repository.
7. Ignore comment edits in v1.
8. Dedupe every candidate command by stable comment ID or node ID before starting work.
9. Authorize the commenter before running any mutation workflow.
10. Persist the new successful checkpoint only after the cycle finishes cleanly.

This matters because polling correctness depends more on durable checkpoints and dedupe than on the exact timestamp returned by any single API call.

## Step 9: Confirm Branch Rules Will Not Block Heimdall

Heimdall creates and pushes branches like `heimdall/<issue-key>-<slug>`.

Check that:

- the app can push new branches
- branch protection rules do not accidentally match and block `heimdall/*`
- pull requests can target `main`

## Step 10: Verify Polling End To End

After Heimdall is running:

1. Confirm the host has outbound HTTPS access to GitHub.
2. Confirm there is no dependency on public inbound GitHub webhook traffic.
3. Create or reuse a Heimdall-managed pull request.
4. If `HEIMDALL_REPO_<ID>_PR_MONITOR_LABEL` is configured, confirm the repository label exists and the pull request carries it.
5. Add a test comment such as `/heimdall status` from an allowed GitHub user.
6. Wait one or two poll intervals.
7. Confirm Heimdall detects the comment and posts its response or audit-visible result.
8. Add a second command such as `/heimdall refine Clarify rollback behavior.` and verify it is also detected after polling.
9. Confirm the same comment is not executed twice if the polling windows overlap.

If end-to-end verification fails, check these first:

- the GitHub App is installed on the correct repository
- the GitHub App private key path is correct and readable
- the installation ID is correct
- the polling interval and lookback window are configured
- the configured PR monitor label exists and is attached to the Heimdall pull request if label-scoped monitoring is enabled
- the repository is actually managed by Heimdall routing
- the commenting user is in the allowed-user set
- the test pull request is a Heimdall-managed pull request, not an unrelated PR

## Minimal Per-Repo Checklist

For each managed repository, verify:

- the app is installed
- webhook delivery is disabled for the app
- `main` is the intended base branch
- the repo accepts pull requests from `heimdall/*` branches
- the repo appears in `HEIMDALL_REPOS`
- `HEIMDALL_REPO_<ID>_PR_MONITOR_LABEL` is set if you want label-scoped PR monitoring for that repository
- `HEIMDALL_REPO_<ID>_ALLOWED_USERS` contains the operators who may run commands
- `HEIMDALL_REPO_<ID>_ALLOWED_AGENTS` contains the agent names permitted for `/opsx-apply`
- the GitHub polling interval and lookback semantics are configured for that deployment

## Easy-To-Miss GitHub Details

- No public GitHub webhook endpoint is required in this setup model.
- PR comments still live under GitHub's issue-comment model even when you do not use webhooks.
- GitHub's PR comment and PR label APIs use issues-style endpoints, but GitHub App `Pull requests` permission is sufficient for Heimdall's current PR-only flow.
- Polling introduces intentional delay; the user-visible delay is usually one poll interval plus processing time.
- The overlap or lookback window must be larger than zero, or restart and clock-skew gaps can drop commands.
- Dedupe by comment identity, not only by `created_at`, or the same command can run twice.
- If label-scoped monitoring is enabled for a repository, removing the monitor label from a Heimdall pull request can cause polling to ignore that pull request until reconciliation adds the label back.
- Installing the app on the organization is not enough if it is not granted access to the target repositories.
- The GitHub App private key remains a critical recovery asset even though the webhook secret is gone.
- Heimdall should only process mutation commands on pull requests that it created or explicitly adopted.
