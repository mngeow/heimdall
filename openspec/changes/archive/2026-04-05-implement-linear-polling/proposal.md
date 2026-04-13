## Why

Heimdall's product and durable specs already depend on Linear polling as the trigger for proposal workflows, but the implementation contract is still underspecified. This change turns that expectation into an implementation-ready Linear GraphQL polling design that uses a static API key, project-level scope from `.env`, durable checkpoints, and safe retry behavior.

## What Changes

- Define the concrete Linear GraphQL polling contract against `https://api.linear.app/graphql` for Heimdall's v1 board-provider adapter.
- Require static API key authentication for poll requests by using Linear's documented API-key header model instead of OAuth or webhooks.
- Specify how Heimdall pages through the `issues` connection, filters to the configured Linear project name from `.env`, orders by update time, and maps Linear issue fields into normalized work items.
- Require durable checkpoint handling, state snapshot comparison, and safe retry behavior when Linear returns GraphQL errors, auth failures, or rate limits.
- Require the v1 Linear polling configuration to include the project name in Heimdall's dotenv schema.
- Add implementation tasks for docs, behavior tests, step bindings, and a real-workspace smoke test.

## Capabilities

### New Capabilities

### Modified Capabilities
- `service-configuration`: add the required dotenv configuration for project-scoped Linear polling.
- `service-board-provider`: refine the Linear polling requirements to specify GraphQL endpoint usage, static API key authentication, issue query shape, pagination, checkpointing, and poll failure handling.

## Impact

- Affects `internal/board/linear` and the application poll-loop wiring.
- Affects SQLite provider cursor usage and work-item snapshot handling.
- Affects operator setup and verification docs for Linear API-key-based polling and project-name configuration.
- Affects behavior tests and fixtures for the board-provider path.
