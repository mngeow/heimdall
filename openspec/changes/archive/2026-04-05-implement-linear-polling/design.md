## Context

Heimdall's product flow starts when a Linear issue enters an active state, but the current provider implementation is still a stub and the durable specs do not yet pin the runtime contract to Linear's actual GraphQL API behavior. Linear's developer documentation describes a single GraphQL endpoint at `https://api.linear.app/graphql`, support for personal or static API keys via the `Authorization: <API_KEY>` header, Relay-style cursor pagination with `first` and `after`, filtering on paginated queries, and API request or complexity rate limits exposed through response headers.

The implementation needs to stay inside Heimdall's current v1 constraints: one Linux-hosted binary, SQLite-backed checkpoints and snapshots, explicit provider adapters, and no reliance on inbound Linear webhooks. It also needs to be careful not to turn a single polling loop into an API-heavy crawler, because Linear explicitly recommends filtering and ordering by recently updated data when polling is unavoidable. For v1, the polling scope is intentionally narrow: Heimdall will only poll issues in one configured Linear project, and that project name must come from the `HEIMDALL_LINEAR_PROJECT_NAME` setting in the application's `.env` configuration.

## Goals / Non-Goals

**Goals:**
- Implement Linear polling against the official GraphQL API by using a secret-backed static API key.
- Poll only the explicitly configured Linear project scope for v1, using a project name supplied in `.env`.
- Page through recently updated issues safely, normalize the required issue fields, and compare the result with SQLite-backed snapshots before emitting activation events.
- Persist a durable poll checkpoint and avoid advancing it when a cycle fails because of auth errors, GraphQL errors, or rate limiting.
- Keep Linear-specific query, pagination, and state semantics inside the Linear adapter.

**Non-Goals:**
- Introducing OAuth for Linear in v1.
- Using Linear webhooks for the standard activation path.
- Polling every issue independently or mirroring the entire Linear workspace into SQLite.
- Expanding v1 scope beyond a single configured Linear project.
- Implementing unrelated proposal, refine, or apply workflow logic outside what is needed to feed the existing activation path.

## Decisions

### Decision: Use Linear's static API key model for v1 poll authentication
Heimdall will authenticate Linear poll requests with a secret-backed static API key using Linear's documented `Authorization: <API_KEY>` header form.

Rationale:
- matches the user's requested operating model
- aligns with Linear's documented simplest auth path for personal or service-style scripts
- avoids OAuth complexity for Heimdall's single-operator v1 deployment

Alternatives considered:
- OAuth 2.0. Rejected because it adds token lifecycle and app registration complexity without solving a current v1 problem.
- Webhooks plus token auth. Rejected because the project explicitly prefers polling for the Linear activation path in v1.

### Decision: Poll the `issues` GraphQL connection with project-name filtering, updated-time ordering, and Relay pagination
The Linear adapter will poll the `issues` connection, constrain it to the configured Linear project name from `.env`, and page through results with `first` and `after` until `pageInfo.hasNextPage` is false.

Representative query shape:

```graphql
query PollIssues($first: Int!, $after: String, $updatedSince: DateTime, $projectName: String!) {
  issues(
    first: $first
    after: $after
    orderBy: updatedAt
    filter: {
      updatedAt: { gte: $updatedSince }
      project: { name: { eq: $projectName } }
    }
  ) {
    nodes {
      id
      identifier
      title
      description
      updatedAt
      state {
        id
        name
        type
      }
      team {
        id
        key
        name
      }
      project {
        id
        name
      }
      labels {
        nodes {
          id
          name
        }
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
```

Rationale:
- matches Linear's documented GraphQL pagination model
- keeps the query minimal and targeted to the fields Heimdall actually normalizes
- supports safe workspace polling without per-issue reads
- matches the requested v1 operating model of project-level polling only

Alternatives considered:
- Poll individual issues separately. Rejected because it is inefficient and directly conflicts with Linear's guidance against excessive polling.
- Fetch all issues and filter locally. Rejected because Linear explicitly recommends server-side filtering to reduce API load.
- Filter by team key in v1. Rejected because the requested v1 scope is project-level polling only.

### Decision: Persist a successful poll timestamp checkpoint, and use page cursors only within a single poll cycle
Heimdall will store the last successful Linear poll timestamp as its durable provider cursor. Within a single poll cycle, it will use the GraphQL page cursor only to drain additional pages. The next cycle will query by updated-time filter from a safe overlapping checkpoint window rather than trying to resume a stale GraphQL page cursor across restarts.

Rationale:
- fits the existing SQLite provider-cursor model
- keeps restart behavior simple and resilient
- works with overlap plus snapshot comparison to avoid missed or duplicated activation events

Alternatives considered:
- Persist the GraphQL `endCursor` across poll cycles. Rejected because page cursors are tied to one query sequence and are more brittle across restarts or filter changes.
- Persist only the last seen issue identifier. Rejected because update-time filtering is a better match for recently changed issue polling.

### Decision: Require `HEIMDALL_LINEAR_PROJECT_NAME` in dotenv configuration
Heimdall will require `HEIMDALL_LINEAR_PROJECT_NAME` in dotenv configuration, and the provider will refuse to start polling when that value is missing.

Rationale:
- keeps the project-scoping contract explicit and operator-visible
- makes the v1 polling scope narrow and predictable
- aligns configuration with the requested operating model instead of inferring scope indirectly

Alternatives considered:
- Resolve the project only from team keys. Rejected because v1 scope is explicitly project-level.
- Poll all projects visible to the API key. Rejected because it would broaden scope and increase accidental activations.

### Decision: Keep active-state mapping inside the adapter, driven by configured state names
The Linear adapter will continue mapping provider-specific state names into Heimdall's normalized `active` lifecycle bucket by using configured active state names, while still reading GraphQL `state.type` and `state.name` as part of the normalized payload.

Rationale:
- preserves the existing explicit operator configuration model
- keeps provider-specific state semantics out of the workflow engine
- leaves room for future provider expansion without forcing the core onto Linear's workflow model

Alternatives considered:
- Treat Linear `state.type` as the only activation source. Rejected because operator-configured active-state names are already part of the product and setup model.

### Decision: Treat GraphQL errors, auth failures, and rate limits as failed poll cycles that do not advance checkpoints
If a Linear poll cycle receives an auth failure, an HTTP failure, a GraphQL response containing an `errors` array, or a rate-limit response, Heimdall will treat the cycle as unsuccessful and leave the durable checkpoint unchanged.

Rationale:
- avoids data loss from advancing the checkpoint after a partial or failed read
- matches the repo's preference for deterministic reconcile-before-create behavior
- gives operators a clear, retryable failure mode

Alternatives considered:
- Advance the checkpoint on partial success. Rejected because it risks silently skipping later issues in the failed window.
- Ignore rate-limit headers. Rejected because Linear documents explicit request and complexity headers that are useful for observability and backoff decisions.

## Risks / Trade-offs

- Linear discourages polling when webhooks are available -> Mitigation: keep the query narrow, use server-side project filters, order by recent updates, and retain operator-configurable poll intervals.
- Overlap windows can cause the same issue to be re-read across cycles -> Mitigation: rely on stored snapshots and idempotency keys so repeated reads do not create duplicate activation events.
- Static API keys are tied to a Linear user context -> Mitigation: document the use of a dedicated service account and keep the key in a secret-backed `.env` value.
- Project-name scoping depends on a human-readable identifier that may change in Linear -> Mitigation: validate that the configured project name resolves during polling and document that renaming the Linear project requires updating `.env`.
- Schema changes could break a hand-written GraphQL query -> Mitigation: keep the selected field set minimal and isolate query parsing inside the Linear adapter.

## Migration Plan

1. Update the Linear adapter to call the official GraphQL endpoint with the configured static API key.
2. Implement paginated issue polling with updated-time filtering for the configured project-name scope.
3. Reuse SQLite provider cursors and work-item snapshots to detect real inactive-to-active transitions.
4. Wire the Linear poll loop into application startup.
5. Validate with behavior tests and a real-workspace smoke test.

Rollback is straightforward because this change adds the real polling path in place of the current stubbed implementation. If needed, revert the release or disable the Linear poll loop while keeping the existing configuration contract intact.

## Open Questions

None.
