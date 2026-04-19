## Context

Heimdall already polls GitHub PR comments, persists command requests, and executes opencode-backed commands such as `/heimdall refine` through a background worker. The execution adapter already depends on `opencode run --format json` for machine-readable outcome classification, and the first structured event in that stream carries the canonical `sessionID` for the run.

What is missing is an operator-facing live view of that execution. Today, an accepted PR command can look silent until a terminal PR comment appears, and the private dashboard only shows tracked PR history rather than live opencode output. The requested behavior also needs GitHub-side acknowledgment as soon as Heimdall accepts the command, but the link in that acknowledgment cannot depend on `sessionID` because the session identity does not exist until after the worker starts consuming opencode output.

A critical bug was discovered during implementation: the timeline entries are accumulated in memory inside the opencode adapter and only persisted to SQLite **after** the process exits. This means the HTMX live-tail polls the database during execution and finds no rows, so the UI appears frozen or empty until the run completes. The design must be corrected to stream entries into the store as they are parsed.

```mermaid
flowchart LR
    Comment["GitHub PR comment"] --> Poller["GitHub poll + auth + dedupe"]
    Poller --> Command["command_request + job"]
    Command --> Feedback["GitHub eyes reaction + UI link reply"]
    Command --> Worker["PR-command worker"]
    Worker --> OpenCode["opencode run --format json"]
    OpenCode --> Parser["structured event parser"]
    Parser -->|stream entries| Store["SQLite command timeline"]
    Store --> UI["/ui/command-runs + detail page"]
    UI --> HTMX["HTMX polling fragments"]
```

## Goals / Non-Goals

**Goals:**
- Add immediate visible GitHub acknowledgment when Heimdall accepts a supported PR command.
- Post a stable Heimdall UI link for accepted opencode-backed commands as soon as intake succeeds.
- Show currently active opencode-backed PR commands in the private dashboard.
- Render live opencode output in a human-readable form by parsing the structured JSON event stream.
- **Stream timeline entries to the database in real time as opencode emits them**, so the HTMX live tail has actual data to display during execution.
- Capture the canonical `sessionID` from the first structured event and bind it to the same command-linked view.
- Render tool events with rich detail: tool name, status, input (command, pattern, file path, etc.), output preview, and execution metadata.
- Render step events with token breakdowns (total, input, output, reasoning, cache) and cost.
- Keep the implementation inside the existing single Go binary, SQLite store, and HTMX-based operator UI.

**Non-Goals:**
- Introducing a SPA, WebSocket stack, or separate frontend deployment artifact.
- Exposing workflow-control buttons such as rerun, cancel, approve, or comment-posting from the UI.
- Rendering raw opencode JSON blobs, raw prompt bodies, or secrets directly in the dashboard.
- Mirroring every GitHub comment body or every provider payload into a new generic event archive.
- Changing the supported PR command surface or authorization model beyond acknowledgment and observability.

## Decisions

### Decision: Anchor GitHub live-output links on command request IDs, not session IDs
The GitHub reply comment should link to a stable command detail page such as `/ui/command-runs/{commandRequestID}`. That route exists as soon as Heimdall accepts and persists the command request, so the URL can be posted immediately during intake.

The detail page will resolve from existing `command_requests` and job state before opencode starts, and it will show queued or starting state until the worker creates and updates the live run timeline. Once the first structured event arrives, the same page will display the canonical `sessionID` and live output for the current run.

Rationale:
- `sessionID` is unavailable when Heimdall first acknowledges the command.
- retries and resumed runs are easier to reason about from the stable command-request identity.
- the same GitHub link remains valid before, during, and after execution.

Alternatives considered:
- Link by `sessionID`. Rejected because the ID arrives too late and cannot support immediate feedback.
- Link only to the PR detail page. Rejected because operators need a directly addressable run view from the GitHub reply comment.

### Decision: Stream timeline entries via a TimelineWriter callback instead of batching at the end
The opencode adapter will accept an optional `TimelineWriter` callback interface. As `parseOpencodeEvents` reads each NDJSON line from the process stdout pipe, it will immediately call the writer for that event before continuing to the next line. The executor will provide a writer implementation that calls `store.AppendTimelineEntry`, so entries appear in SQLite in real time.

Rationale:
- fixes the live-tail bug: HTMX polls during execution will see newly inserted rows.
- avoids holding the entire timeline in memory for long-running commands.
- keeps the adapter testable: unit tests can pass a no-op or recording writer.

Alternatives considered:
- Batching at the end (current broken implementation). Rejected because it produces an empty database during execution.
- Writing raw stdout to a file and tailing it from the UI. Rejected because it complicates the storage model and still requires parsing.
- Using a goroutine with a channel. Rejected because a simple callback interface is easier to test and reason about.

### Decision: Persist a normalized command timeline in SQLite rather than exposing raw event streams
Heimdall will extend runtime state with additive command-run records and ordered timeline entries for opencode-backed PR commands. The model should remain command-centric and include at least the command-request linkage, attempt or continuity metadata, current state, canonical `sessionID` when known, timestamps, terminal summary, and ordered display entries.

The stored timeline entries should be UI-ready records with rich metadata captured from the JSON event rather than raw NDJSON lines. Each entry carries an `entry_type` (e.g., `tool_use`, `step_start`, `step_finish`, `text`, `error`) and a JSON `metadata` column that preserves tool inputs, outputs, token counts, and execution metadata for the template to render. The dashboard will read this normalized state and render tool-specific cards, step summaries, and text blocks.

Rationale:
- preserves live-output state across worker boundaries and page refreshes.
- keeps the dashboard independent of provider-specific raw JSON shapes.
- avoids making operators reconstruct runs from journald output.
- rich metadata in SQLite allows the template to show tool commands, file paths, output previews, and token counts without re-parsing raw events.

Alternatives considered:
- Tail journald for live output. Rejected because logs are not a stable per-command UI contract.
- Store only raw JSON and transform it in the UI. Rejected because it leaks provider-specific payloads into the view layer and risks exposing sensitive content verbatim.

### Decision: Normalize opencode events into rich structured display entries at ingestion time
The opencode adapter should continue treating the NDJSON stream as the source of truth for session state, but it should now emit rich structured display entries while it classifies the run.

For `tool_use` events, the adapter will extract:
- `tool`: the tool name (bash, glob, read, skill, todowrite, apply_patch, etc.)
- `status`: completed, pending, or error
- `input`: the full input object (command, pattern, filePath, patchText, etc.)
- `output`: the tool output string
- `metadata`: execution metadata (exit code, execution time, OpenAI item IDs, etc.)

For `step_finish` events, the adapter will extract:
- `reason`: e.g., "tool-calls"
- `tokens`: total, input, output, reasoning, cache read/write
- `cost`: the computed cost

For `step_start` and `text` events, the adapter will capture the relevant display text and minimal metadata.

Unknown but valid event types should fall back to a compact generic status entry instead of surfacing raw JSON.

The first structured event remains the canonical source of `sessionID`. Once captured, that same session identity is attached to the command-linked run state and shown in logs and the dashboard header.

Rationale:
- one parser pass drives both machine classification and human observability.
- UI rendering stays stable even if low-level event payloads are noisy or large.
- secrecy rules are easier to enforce before data reaches templates.
- rich metadata enables per-tool-type rendering (bash commands, file paths, output previews, token counts) in the dashboard.

Alternatives considered:
- Parse raw JSON separately for outcome classification and UI rendering. Rejected because duplicate parsers increase drift risk.
- Expose only coarse status without timeline entries. Rejected because the user explicitly asked for live output visibility.
- Store only a simple display string. Rejected because it loses tool inputs, outputs, and token counts that the user wants visible.

### Decision: Use HTMX polling fragments for live-tail list and detail views
The operator UI will remain server-rendered HTML. A new `/ui/command-runs` screen will show active queued, starting, running, or blocked opencode-backed commands, and a `/ui/command-runs/{commandRequestID}` detail page will show the selected command timeline.

HTMX will drive incremental refresh. The active-run list can refresh as a normal fragment on a short interval. The detail page should poll lightweight timeline fragments while the command is non-terminal. Because entries are streamed to SQLite in real time, each poll will return newly available rows. The visible output should behave like a live tail: previously rendered lines stay in place while newly available entries are appended in stream order. Polling can stop or slow down after a terminal state is reached.

Rationale:
- matches the existing dashboard direction and the user's explicit HTMX request.
- keeps deployment simple and avoids a new push protocol.
- preserves a read-only HTML-first operator surface.

Alternatives considered:
- WebSockets or SSE. Rejected because they add more runtime complexity than v1 needs.
- Full-page refresh only. Rejected because live output would feel clumsy and delay observation.

### Decision: Render the timeline top-to-bottom in chronological order
The dashboard should display timeline entries oldest-first at the top and newest-last at the bottom, like terminal scrollback. The query returns entries in descending sequence order (newest first) for efficient pagination; the handler reverses the slice before rendering so the visual order is chronological. New entries emitted during a live run appear at the bottom, which is the natural direction for reading ongoing output.

Rationale:
- matches how developers read terminal output and log streams
- avoids the jarring experience of new entries pushing old content downward
- HTMX outerHTML swap works unchanged because the full list is re-rendered in the correct order each poll

Alternatives considered:
- Newest-first ordering. Rejected because it feels unnatural for a live tail; users expect the latest output at the bottom.
- Client-side JavaScript reversal. Rejected because it adds unnecessary complexity; server-side slice reversal is trivial.

### Decision: Render long text and tool output inside a styled markdown/code box
Large blocks of text (skill output, file contents, bash stdout, patch text) should not flow as plain text inside the card. Instead, they render inside a dark-background bordered container with padding, rounded corners, and monospace font — a "markdown box" within the card. This keeps the card structure clean and prevents long output from visually dominating the timeline.

Rationale:
- visually contains large output blocks so they do not overwhelm the timeline
- distinguishes raw tool output from card chrome (headers, metadata, badges)
- consistent with how code blocks and terminal output are displayed in documentation UIs

Alternatives considered:
- Plain `<pre>` without a container. Rejected because it blends into the card background and lacks visual boundaries.
- Collapsible sections. Rejected as over-engineering for v1; can be added later if output length becomes a real problem.

### Decision: Separate intake acknowledgment from worker outcome reporting
Heimdall should publish acceptance feedback as part of command intake after the comment has been authorized, parsed, deduplicated as new, and persisted. All accepted supported PR commands get the GitHub `eyes` reaction. Accepted commands that start or resume opencode execution also get a PR reply comment with the stable command detail URL.

The worker continues to own terminal success, failure, or blocked result comments. Acknowledgment failures should be recorded and retried if needed, but they must not cause duplicate command execution or invalidate the accepted command request.

Rationale:
- users learn immediately that Heimdall saw the command.
- the PR feedback path becomes clearer without waiting for the worker to finish.
- keeps execution and GitHub feedback responsibilities explicit.

Alternatives considered:
- Wait until the worker starts before acknowledging. Rejected because queued commands would still appear silent.
- Replace terminal result comments with the live-output link comment. Rejected because the PR still needs a visible final outcome.

### Decision: Require a validated public operator URL for GitHub links
Heimdall will treat `HEIMDALL_SERVER_PUBLIC_URL` as the required absolute base URL for any dashboard link posted into GitHub comments. The service should validate that the value is present and absolute at startup so link comments are never built from relative paths or host-local guesses.

Rationale:
- GitHub comments need absolute URLs.
- startup validation is better than posting broken links at runtime.
- the repository already documents this setting in operations guidance.

Alternatives considered:
- Build links from the incoming request host header. Rejected because PR comments are posted outside any browser request context.
- Post relative paths. Rejected because GitHub comments cannot resolve them against the Heimdall server.

### Decision: Deduplicate accepted-command feedback at the command-request level
Heimdall will record whether the `eyes` reaction and the link comment have already been posted for a given command request. Retry or duplicate poll logic will check these flags before calling the GitHub API again.

Rationale:
- prevents duplicate reactions and link comments on retries or overlapping polls
- keeps the GitHub comment thread clean

Alternatives considered:
- Deduplicate by querying GitHub API before posting. Rejected because it adds API overhead and race conditions.

### Decision: Continue command execution even if acknowledgment fails
If the reaction or link comment step fails due to a transient GitHub API error, Heimdall will still enqueue the command for worker execution. A background reconciliation job can retry the missing acknowledgment later.

Rationale:
- avoids blocking real work because of a secondary feedback failure
- matches the existing reconciliation pattern for retrying failed comment status posts

Alternatives considered:
- Roll back the command acceptance if feedback fails. Rejected because it would create a poor user experience where valid commands are silently ignored.

### Decision: Define a bounded six-state command run model
The command-linked opencode run will have explicit states: `queued`, `starting`, `running`, `blocked`, `completed`, `failed`. The dashboard will render the current state label prominently, and the live tail will become active once the state reaches `running`.

Rationale:
- gives operators clear visibility into where the run is in its lifecycle
- distinguishes "accepted but not yet started" from "actively producing output"

Alternatives considered:
- Use only two states (active/terminal). Rejected because it hides useful pre-session and blocked states.

### Decision: Cap live-tail fragment size with a default limit
To prevent unbounded HTMX response growth, the dashboard will query the most recent N display entries (e.g., 200) for live-tail refreshes. A separate action will allow loading earlier history if needed.

Rationale:
- protects UI performance for very long runs
- keeps the default live-tail experience fast

Alternatives considered:
- Return all entries always. Rejected because very long runs could produce multi-megabyte HTML fragments.

### Decision: Maintain an explicit normalization mapping for opencode events
The opencode adapter will use a versioned mapping from known event types to rich structured display entries. New event types will fall back to a generic template rather than raw JSON. The mapping is explicit about which JSON fields are extracted for each event type so the template can render tool-specific cards, step summaries, and text blocks.

Rationale:
- ensures consistent UI output across different commands and runs
- makes it easy to add richer rendering for new event types later
- preserves execution metadata (token counts, exit codes, timing) in the database for later analysis

Alternatives considered:
- Ad-hoc normalization per event. Rejected because it leads to inconsistent UI output and harder testing.
- Store only a simple string per event. Rejected because it cannot support the requested rich rendering.

## Risks / Trade-offs

- Private operator URLs become visible in PR comments to the PR's readers → Mitigation: keep the dashboard read-only, rely on the private/internal deployment boundary already assumed by Heimdall, and document the trust implication.
- Timeline storage can grow if long-running commands emit many display entries → Mitigation: store compact normalized entries with rich metadata, scope the feature to opencode-backed PR commands, and revisit retention separately if needed.
- HTMX polling is near-real-time rather than true push streaming → Mitigation: use short active polling intervals and stream entries to the database as they arrive so each poll returns fresh rows.
- Opencode may add new event types over time → Mitigation: keep adapter-level fallback rendering for unknown valid events so observability degrades gracefully instead of breaking.
- Streaming writes to SQLite during execution may add latency → Mitigation: use simple INSERTs with pre-computed sequence numbers; SQLite handles this well for a single-writer workload.

## Migration Plan

1. Validate and plumb `HEIMDALL_SERVER_PUBLIC_URL` for dashboard link generation.
2. Add GitHub accepted-command acknowledgment flow: `eyes` reaction for accepted commands and reply-comment links for opencode-backed commands.
3. **Fix the live-tail bug by introducing a `TimelineWriter` callback interface and streaming entries to SQLite as they are parsed.**
4. **Update the opencode adapter to extract rich structured metadata from each JSON event type (tool inputs/outputs, token counts, execution metadata).**
5. **Update the dashboard templates to render rich HTML cards per event type instead of simple text lines.**
6. Extend runtime state and the opencode adapter to persist command-linked live timeline entries, current run status, and canonical `sessionID` from the first structured event.
7. Add dashboard routes, query services, templates, and HTMX fragments for active command runs and command detail pages.
8. Update behavior tests, step bindings, and operator docs for the new acknowledgment and live-output surfaces.

Rollback is low risk because the schema changes are additive. If the feature must be disabled, Heimdall can stop writing the new timeline records and stop serving the new dashboard routes while leaving additive tables unused until a later cleanup migration.

## Open Questions

None.
