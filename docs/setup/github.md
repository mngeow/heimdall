# GitHub Setup

## Purpose

Symphony uses a GitHub App for repository access, PR creation, PR comments, and polling GitHub for new pull request comments and pull request state changes.

V1 should not use a personal access token.

## What You Need Before Starting

- the GitHub organization or user account that owns the repository
- an owner or admin account that can create and install GitHub Apps for that owner
- the repository or repositories Symphony should manage
- a place to store the GitHub App private key outside git

Important:

- a public Symphony URL is not required in v1
- GitHub App webhooks can remain disabled in v1 because Symphony polls GitHub instead

## Step 1: Create The GitHub App

Create a GitHub App dedicated to Symphony.

Navigate to the correct GitHub settings page:

- organization-owned repositories: `Organization settings -> Developer settings -> GitHub Apps -> New GitHub App`
- user-owned repositories: `Settings -> Developer settings -> GitHub Apps -> New GitHub App`

If GitHub Apps are managed centrally by your organization, create the app under that organization rather than under a personal account.

Recommended values:

- App name: `Symphony` or another globally unique name such as `Symphony Acme Prod`
- Homepage URL: your project or operator docs URL
- Webhooks: disabled for v1

Other creation-form guidance:

- Description: optional, but useful if multiple operators will see the app in GitHub
- Callback URL: not required in v1 because Symphony does not use a browser-driven OAuth flow
- Setup URL: not required in v1
- User authorization or OAuth-related options: leave at the default unless you intentionally add a user-to-server auth flow later

When GitHub shows the webhook section, disable it by clearing `Active`. In v1, you should not need to enter a webhook URL or webhook secret.

After you create the app, GitHub opens the app settings page. Record the numeric `App ID`. Symphony needs the `App ID`, not the app slug and not the OAuth `Client ID`.

If you later want a lower-latency ingress path, you can add a webhook relay or tunnel as a separate deployment choice, but the core v1 design does not require it.

## Step 2: Configure Repository Permissions

Use the smallest permissions that still allow Symphony to function:

- Metadata: read-only
- Contents: read and write
- Pull requests: read and write
- Issues: read and write

Additional guidance:

- Account permissions: none required in v1
- Webhook event subscriptions: none required in v1 because Symphony polls GitHub instead of consuming webhook deliveries
- Repository selection: prefer only the repositories Symphony should actually manage

Do not add broader permissions until there is a concrete need.

## Step 3: Configure Polling Instead Of Webhooks

No GitHub webhook subscriptions are required in v1.

Instead, Symphony should poll GitHub for:

- new comments on Symphony-managed pull requests
- pull request state changes needed for reconciliation

Recommended starting point:

- `github.poll_interval: 30s`

Tradeoff:

- lower poll intervals reduce command latency but increase GitHub API traffic
- higher poll intervals reduce API traffic but make comment-driven workflows feel slower

## Step 4: Generate The Private Key

After the app is created:

1. Open the GitHub App settings page.
2. Select `Private keys` in the left sidebar.
3. Click `Generate a private key`.
4. GitHub downloads a `.pem` file immediately.
5. Move that file to a path outside the repository, for example `/etc/symphony/github-app.pem`.
6. Restrict file permissions so only root and the Symphony service account can read it.
7. Record the GitHub App ID if you have not already done so.

Important details:

- Treat the downloaded `.pem` file as the secret itself. Do not paste it into git-tracked files.
- Store it immediately. GitHub will let you create new private keys later, but you should not rely on the browser download being recoverable from the original download step.
- Keep the PEM file content intact, including the `BEGIN` and `END` lines.
- If the private key is lost or exposed, generate a replacement key in GitHub and update the deployed file.

Example install command on Linux:

```bash
sudo install -o root -g symphony -m 0440 \
  "$HOME/Downloads/<github-app-private-key>.pem" \
  /etc/symphony/github-app.pem
```

Adjust the group to match the Symphony service account or service group on the host.

## Step 5: Install The App On Target Repositories

Install the app on every repository Symphony should manage.

Typical install flow:

1. Open the GitHub App settings page.
2. Click `Install App`.
3. Choose the target organization or user account.
4. Select either all repositories or only the specific repositories Symphony should manage.
5. Approve the installation.

If you only want Symphony to manage a subset of repositories, install it only on that subset.

After installation, record the installation ID if your deployment uses a single installation.

Ways to find the installation ID:

- open the installation settings page in GitHub; the numeric installation ID appears in the URL
- or use GitHub CLI for any installed repository:

```bash
gh api /repos/<owner>/<repo>/installation --jq '.id'
```

The installation ID is different from the app ID. Symphony needs both if you run it against one fixed installation.

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
```

Config values should include:

- `github.poll_interval`, such as `30s`
- base branch, usually `main`
- repo routing rules
- allowed GitHub users
- allowed apply agents

Common source of confusion:

- `SYMPHONY_GITHUB_APP_ID` is the app-wide ID from the GitHub App settings page
- `SYMPHONY_GITHUB_INSTALLATION_ID` is the repository-owner installation ID from the app installation
- `SYMPHONY_GITHUB_PRIVATE_KEY_FILE` points to the downloaded `.pem` file on disk

## Step 8: Verify Polling Works

Once Symphony is running:

1. Confirm the GitHub App is installed on the target repository.
2. Create or reuse a Symphony-managed pull request.
3. Add a test comment such as `/symphony status`.
4. Wait one GitHub poll interval.
5. Confirm Symphony discovers the comment and responds through the pull request or logs.

## Minimal Per-Repo Checklist

For each managed repository, verify:

- the app is installed
- `main` is the intended base branch
- the repo accepts PRs from `symphony/*` branches
- the repo appears in `/etc/symphony/config.yaml`
- `allowed_users` contains the operators who may run commands
- `allowed_agents` contains the agent names permitted for `/opsx-apply`

## Easy-To-Miss GitHub Details

- No public Symphony URL is required in v1.
- Comment-command latency is bounded by the configured GitHub poll interval.
- Installing the app on the organization is not enough if it is not granted access to the target repositories.
- The GitHub App private key is a critical recovery asset.
- Symphony should only process mutation commands on PRs that it created.
