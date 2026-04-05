# Symphony

A Linux-hosted Go service that turns kanban movement into OpenSpec-driven engineering work by converting board activity into branches, specs, PRs, and agent-driven implementation flows.

## Overview

Symphony polls Linear for issues entering an active state, then:
1. Creates a git branch for the target repository
2. Generates an OpenSpec change from the issue title and description
3. Commits and pushes the generated spec artifacts
4. Opens a GitHub pull request to `main`
5. Polls GitHub for PR comments on Symphony-managed pull requests so it can refine specs or run `/opsx-apply` with an allowed agent
6. Commits any resulting changes back to the same branch

## Quick Start

### Prerequisites

- Go 1.21+
- `git`
- `openspec` CLI
- `opencode` CLI
- SQLite

### Building

```bash
# Build the binary
go build -o symphony ./cmd/symphony

# Run tests
go test ./...

# Create a local runtime config
cp dist.env .env

# Run with the project-root .env file
./symphony
```

### Configuration

Copy `dist.env` to `.env` in the project root and edit the values for your environment:

```bash
cp dist.env .env
```

Example project-root `.env`:

```dotenv
SYMPHONY_SERVER_LISTEN_ADDRESS=:8080
SYMPHONY_SERVER_PUBLIC_URL=http://127.0.0.1:8080
SYMPHONY_STORAGE_DRIVER=sqlite
SYMPHONY_STORAGE_DSN=/var/lib/symphony/state/symphony.db
SYMPHONY_LINEAR_POLL_INTERVAL=30s
SYMPHONY_LINEAR_ACTIVE_STATES=In Progress
SYMPHONY_LINEAR_PROJECT_NAME=Core Platform
SYMPHONY_LINEAR_API_TOKEN=your-linear-token
SYMPHONY_GITHUB_BASE_BRANCH=main
SYMPHONY_GITHUB_POLL_INTERVAL=30s
SYMPHONY_GITHUB_LOOKBACK_WINDOW=2m
SYMPHONY_GITHUB_APP_ID=your-app-id
SYMPHONY_GITHUB_INSTALLATION_ID=your-installation-id
SYMPHONY_GITHUB_PRIVATE_KEY_FILE=/etc/symphony/github-app.pem
SYMPHONY_REPOS=PLATFORM
SYMPHONY_REPO_PLATFORM_NAME=github.com/acme/platform
SYMPHONY_REPO_PLATFORM_LOCAL_MIRROR_PATH=/var/lib/symphony/repos/github.com/acme/platform.git
SYMPHONY_REPO_PLATFORM_DEFAULT_BRANCH=main
SYMPHONY_REPO_PLATFORM_BRANCH_PREFIX=symphony
SYMPHONY_REPO_PLATFORM_LINEAR_TEAM_KEYS=ENG
SYMPHONY_REPO_PLATFORM_ALLOWED_AGENTS=gpt-5.4,claude-sonnet
SYMPHONY_REPO_PLATFORM_ALLOWED_USERS=your-github-username
```

The local `.env` file is ignored by git. `dist.env` stays committed as the supported settings template.
For v1, Linear polling is scoped to the configured project name in `SYMPHONY_LINEAR_PROJECT_NAME`.

### Running

```bash
# Start the service
./symphony

# Check health
curl http://localhost:8080/healthz

# Check readiness (validates dependencies)
curl http://localhost:8080/readyz
```

## Project Structure

```
.
├── cmd/symphony/          # Main application entrypoint
├── internal/
│   ├── app/               # Application lifecycle
│   ├── board/linear/      # Linear integration
│   ├── config/            # Configuration loading
│   ├── exec/              # OpenSpec/OpenCode wrappers
│   ├── repo/              # Git repository management
│   ├── scm/github/        # GitHub App auth and polling
│   ├── slashcmd/          # PR command parsing and intake
│   ├── store/             # SQLite persistence
│   ├── validation/        # Dependency validation
│   └── workflow/          # Workflow orchestration
├── tests/
│   ├── bdd/               # Godog step definitions
│   └── features/          # Gherkin behavior tests
└── docs/                  # Documentation
```

## Documentation

- [Product Design](docs/product.md)
- [Architecture](docs/architecture.md)
- [Workflows](docs/workflows.md)
- [Authentication](docs/authentication.md)
- [Operations](docs/operations.md)
- [Setup Guide](docs/setup/)

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run BDD tests
go test ./tests/bdd/... -v

# Run specific package tests
go test ./internal/store/...
go test ./internal/workflow/...

# Run with coverage
go test -cover ./...
```

### Adding Features

1. Create or update OpenSpec specs in `openspec/specs/`
2. Implement the feature
3. Add tests
4. Update documentation

## License

MIT
