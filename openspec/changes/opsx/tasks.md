## 1. Command intake feedback and link generation

- [ ] 1.1 Load and validate `HEIMDALL_SERVER_PUBLIC_URL`, then add a command-detail URL builder for GitHub-linked dashboard pages.
- [ ] 1.2 Update PR command intake to add the GitHub `eyes` reaction for every accepted non-duplicate command and to skip that accepted-command feedback for rejected or duplicate observations.
- [ ] 1.3 Post an immediate PR reply comment with the command-detail dashboard link for accepted opencode-backed commands, while keeping non-opencode commands such as `/heimdall status` free of misleading live-output links.

## 2. Live opencode timeline persistence

- [ ] 2.1 Extend SQLite runtime state and store/query APIs for command-linked opencode run records, ordered human-readable timeline entries, explicit state transitions, canonical `sessionID`, and terminal summaries.
- [ ] 2.2 Update the opencode execution adapter to parse `opencode run --format json`, capture the first structured-event `sessionID`, normalize structured events into sanitized timeline entries according to an explicit mapping, and preserve blocker or terminal events on the same command timeline.
- [ ] 2.3 Wire the PR-command worker to create and update queued, starting, running, blocked, completed, and failed command-linked run state without requiring the dashboard to tail host logs.

## 3. HTMX dashboard views for command runs

- [ ] 3.1 Add a private `/ui/command-runs` dashboard screen and supporting read queries that list currently queued, starting, running, or blocked opencode-backed commands with repository, PR, actor, status, start time, and session context.
- [ ] 3.2 Add a private `/ui/command-runs/{commandRequestID}` detail page plus HTMX fragment endpoints that show queued pre-session state, explicit state transitions, refresh the live human-readable timeline as a live tail while the command is non-terminal, and render terminal states clearly once the command finishes.
- [ ] 3.3 Link the new command-run views from the existing dashboard navigation and pull-request detail experience while keeping the UI read-only.

## 4. Behavior-test coverage

- [ ] 4.1 Write or update Gherkin behavior tests for accepted-command acknowledgment, including `eyes` reactions, live-output link comments for opencode-backed commands, no live-output link for `/heimdall status`, and no accepted-command feedback for rejected or duplicate observations.
- [ ] 4.2 Write or update Gherkin behavior tests for active command-run dashboard views, queued pre-session detail pages, first-event `sessionID` capture, and live-tail human-readable timeline rendering from opencode JSON events.
- [ ] 4.3 Implement or update the step bindings, fixtures, fake GitHub interactions, fake opencode event streams, and test runner wiring needed to execute the new command-feedback and live-dashboard scenarios.

## 5. Documentation and verification

- [ ] 5.1 Update the relevant docs in `docs/` to describe accepted-command reactions, live command-run dashboard pages, HTMX refresh behavior, the explicit opencode event normalization mapping, and the `HEIMDALL_SERVER_PUBLIC_URL` requirement for GitHub link comments.
- [ ] 5.2 Run the relevant automated tests for the changed PR-command, opencode-runtime, dashboard, and behavior-test coverage, and verify they pass before considering the change complete.

## 6. Edge case robustness

- [ ] 6.1 Add idempotency tracking for accepted-command feedback (reactions and link comments) so retries or duplicate observations do not produce duplicates.
- [ ] 6.2 Ensure partial GitHub acknowledgment failures do not prevent command execution; queue the command for the worker regardless.
- [ ] 6.3 Implement explicit run state machine transitions (queued → starting → running → terminal) in runtime state and dashboard queries.
- [ ] 6.4 Render terminal states (completed, failed, blocked) prominently on the command detail page with status indicators and summaries.
- [ ] 6.5 Bound live-tail fragment queries to a default limit and provide a way to load earlier entries.
- [ ] 6.6 Define and implement the explicit opencode event normalization mapping in the execution adapter.
