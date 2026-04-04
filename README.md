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

# Run with config
SYMPHONY_CONFIG_PATH=/etc/symphony/config.yaml ./symphony
```

### Configuration

Create `/etc/symphony/config.yaml`:

```yaml
server:
  listen_address: ":8080"
  public_url: "http://127.0.0.1:8080"

storage:
  driver: sqlite
  dsn: "/var/lib/symphony/state/symphony.db"

linear:
  poll_interval: 30s
  active_states:
    - "In Progress"
  team_keys:
    - "ENG"

github:
  base_branch: "main"
  poll_interval: 30s
  lookback_window: 2m

repos:
  - name: "github.com/acme/platform"
    local_mirror_path: "/var/lib/symphony/repos/github.com/acme/platform.git"
    default_branch: "main"
    branch_prefix: "symphony"
    linear_team_keys:
      - "ENG"
    allowed_agents:
      - "gpt-5.4"
      - "claude-sonnet"
    allowed_users:
      - "your-github-username"
```

Set environment variables for secrets:

```bash
export SYMPHONY_LINEAR_API_TOKEN="your-linear-token"
export SYMPHONY_GITHUB_APP_ID="your-app-id"
export SYMPHONY_GITHUB_INSTALLATION_ID="your-installation-id"
export SYMPHONY_GITHUB_PRIVATE_KEY_FILE="/etc/symphony/github-app.pem"
```

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
