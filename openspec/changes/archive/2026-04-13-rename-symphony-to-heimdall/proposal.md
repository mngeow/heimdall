## Why

The current `Symphony` name is embedded across the product, repository, runtime conventions, and operator documentation even though the service's role is closer to a vigilant bridge and workflow gatekeeper than a music metaphor. Renaming the system to `Heimdall` now keeps the brand aligned with the product's purpose before more repositories, specs, commands, and deployment defaults harden around the old identity.

## What Changes

- Rename the canonical product, repository, module, binary, and operator-facing identity from `Symphony` to `Heimdall` across the maintained codebase, documentation, OpenSpec artifacts, and tests.
- **BREAKING** Replace `Symphony`-branded automation namespaces with `Heimdall`-branded ones, including slash commands, branch prefixes, bootstrap paths, GitHub label examples, environment-variable prefixes, default filesystem paths, and default runtime database naming.
- Update managed pull request, bootstrap, and observability requirements so operator-visible workflow surfaces consistently use the new `Heimdall` identity.
- Rebrand existing in-repo OpenSpec references, including archived change artifacts, so the repository no longer carries mixed product naming.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `feature-kanban-activation`: rename the activation workflow's operator-facing ownership language from Symphony to Heimdall.
- `feature-openspec-proposal-pr`: rename deterministic bootstrap branch and repository scaffolding conventions to use the Heimdall namespace.
- `feature-pr-command-workflows`: rename the slash-command namespace from `/symphony` to `/heimdall` while preserving the existing narrow command surface.
- `service-behavior-testing`: rename the canonical behavior-suite identity and examples to Heimdall.
- `service-board-provider`: rename board-provider requirements and operator-facing references from Symphony to Heimdall.
- `service-configuration`: rename the canonical dotenv schema, configuration examples, and operator-facing naming defaults from `SYMPHONY_*` conventions to `HEIMDALL_*` conventions.
- `service-execution-runtime`: rename runtime-facing workflow ownership language and execution examples to Heimdall while keeping the existing execution model.
- `service-github-scm`: rename the managed pull request and monitor-label identity from Symphony-branded conventions to Heimdall-branded conventions.
- `service-observability`: rename operator-visible service and log references so deployment and debugging guidance consistently refer to Heimdall.
- `service-runtime-state`: rename runtime-state ownership language and default storage naming to Heimdall.

## Impact

- Go module path, import paths, binary/entrypoint naming, repository slug, and other source-level `symphony` identifiers
- Branch names, bootstrap file paths, PR metadata, and command examples used by workflow automation
- Dotenv keys, default filesystem paths, and runtime database naming used by operators and deployment docs
- OpenSpec specs, active change artifacts, archived change artifacts, documentation, and BDD fixtures that currently reference `Symphony`
