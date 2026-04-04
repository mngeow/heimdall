# GitHub Setup

## Purpose

Symphony uses a GitHub App for repository access, PR creation, PR comments, and webhook delivery.

V1 should not use a personal access token.

## What You Need Before Starting

- the GitHub organization or user account that owns the repository
- the repository or repositories Symphony should manage
- a public HTTPS URL for Symphony, such as `https://symphony.example.com/webhooks/github`
- a place to store the GitHub App private key outside git

## Step 1: Create The GitHub App

Create a GitHub App dedicated to Symphony.

Recommended values:

- App name: `Symphony`
- Homepage URL: your project or operator docs URL
- Webhook URL: `https://<your-domain>/webhooks/github`
- Webhook secret: a strong random secret stored outside git

## Step 2: Configure Repository Permissions

Use the smallest permissions that still allow Symphony to function:

- Metadata: read-only
- Contents: read and write
- Pull requests: read and write
- Issues: read and write

Do not add broader permissions until there is a concrete need.

## Step 3: Subscribe To Webhook Events

Enable these events:

- `issue_comment`
- `pull_request`

Important detail:

- PR comments are delivered through `issue_comment`, not a separate PR-comment event.

## Step 4: Generate The Private Key

After the app is created:

1. Generate a private key.
2. Store it in a root-readable path outside the repository, for example `/etc/symphony/github-app.pem`.
3. Record the GitHub App ID.

## Step 5: Install The App On Target Repositories

Install the app on every repository Symphony should manage.

If you only want Symphony to manage a subset of repositories, install it only on that subset.

After installation, record the installation ID if your deployment uses a single installation.

## Step 6: Confirm Branch Rules Will Not Block Symphony

Symphony creates and pushes branches like `symphony/<issue-key>-<slug>`.

Check that:

- the app can push new branches
- branch protection rules do not accidentally match and block `symphony/*`
- PRs can target `main`

## Step 7: Wire The App Into Symphony Config And Secrets

Set these secrets and config values:

```bash
SYMPHONY_GITHUB_APP_ID=<app-id>
SYMPHONY_GITHUB_INSTALLATION_ID=<installation-id>
SYMPHONY_GITHUB_PRIVATE_KEY_FILE=/etc/symphony/github-app.pem
SYMPHONY_GITHUB_WEBHOOK_SECRET=<webhook-secret>
```

Config values should include:

- public webhook path, usually `/webhooks/github`
- base branch, usually `main`
- repo routing rules
- allowed GitHub users
- allowed apply agents

## Step 8: Verify Webhook Delivery

Once Symphony is reachable over HTTPS:

1. Open the GitHub App webhook delivery page.
2. Send a test delivery.
3. Confirm Symphony accepts the request and returns a success status.
4. After deployment, use a real PR comment to verify slash-command intake.

## Minimal Per-Repo Checklist

For each managed repository, verify:

- the app is installed
- `main` is the intended base branch
- the repo accepts PRs from `symphony/*` branches
- the repo appears in `/etc/symphony/config.yaml`
- `allowed_users` contains the operators who may run commands
- `allowed_agents` contains the agent names permitted for `/opsx-apply`

## Easy-To-Miss GitHub Details

- Without `issue_comment`, PR slash commands will never arrive.
- Without a public HTTPS webhook endpoint, PR comment workflows will not work.
- Installing the app on the organization is not enough if it is not granted access to the target repositories.
- The GitHub App private key and webhook secret are both critical recovery assets.
- Symphony should only process mutation commands on PRs that it created.
