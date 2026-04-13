## 1. Linear API Contract

- [x] 1.1 Finalize the representative Linear GraphQL polling query against the current public schema and developer docs for issues, state, team, project, labels, pagination, and updated-time filtering.
- [x] 1.2 Update the relevant Linear setup and workflow docs to describe the static API key polling contract, configured project-name scope from `.env`, and retry or rate-limit expectations.

## 2. Linear GraphQL Polling Client

- [x] 2.1 Implement the Linear GraphQL client against `https://api.linear.app/graphql` with static API key authentication and structured handling for HTTP and GraphQL errors.
- [x] 2.2 Implement project-scoped issue polling with `first` and `after` pagination, updated-time ordering or filtering, and only the GraphQL fields needed to build normalized work items.
- [x] 2.3 Add dotenv configuration parsing and validation for the required `HEIMDALL_LINEAR_PROJECT_NAME` setting used by v1 polling.
- [x] 2.4 Read and persist the Linear provider checkpoint in SQLite, including safe overlap semantics and no checkpoint advance on failed poll cycles.

## 3. Transition Detection And App Wiring

- [x] 3.1 Map polled Linear issues into normalized work items and lifecycle buckets by using the configured active state names.
- [x] 3.2 Compare stored snapshots and emit deduplicated `entered_active_state` events only on real inactive-to-active transitions.
- [x] 3.3 Wire the Linear poll loop into the running application so successful Linear poll cycles can feed the existing activation workflow path without webhooks.

## 4. Testing

- [x] 4.1 Write Gherkin behavior tests for successful project-scoped Linear polling, multi-page Linear polling, missing project-name configuration, invalid API key or rate-limited poll failures, and active-state transition detection.
- [x] 4.2 Implement or update the step bindings, fake Linear GraphQL fixtures, and test runner integration needed to execute the Linear polling behavior tests.
- [x] 4.3 Add focused Go unit or integration tests for GraphQL response parsing, checkpoint handling, and duplicate transition suppression.

## 5. Verification

- [x] 5.1 Run the relevant automated test suite and verify the Linear polling scenarios pass.
