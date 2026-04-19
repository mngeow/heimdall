## 1. Command intake feedback and link generation

- [x] 1.1 Load and validate `HEIMDALL_SERVER_PUBLIC_URL`, then add a command-detail URL builder for GitHub-linked dashboard pages.
- [x] 1.2 Update PR command intake to add the GitHub `eyes` reaction for every accepted non-duplicate command and to skip that accepted-command feedback for rejected or duplicate observations.
- [x] 1.3 Post an immediate PR reply comment with the command-detail dashboard link for accepted opencode-backed commands, while keeping non-opencode commands such as `/heimdall status` free of misleading live-output links.

## 2. Live opencode timeline persistence

- [x] 2.1 Extend SQLite runtime state and store/query APIs for command-linked opencode run records, ordered human-readable timeline entries, explicit state transitions, canonical `sessionID`, and terminal summaries.
- [x] 2.2 **Fix the live-tail bug: introduce a `TimelineWriter` callback interface in `internal/exec` so entries are streamed to SQLite as they are parsed, instead of batched in memory and written only after the process exits.**
- [x] 2.3 **Update `parseOpencodeEvents` to call the `TimelineWriter` for each parsed event before continuing to the next line.**
- [x] 2.4 **Update `runWithJSONEvents`, `RunRefine`, `RunApply`, and `RunGeneric` to accept an optional `TimelineWriter` parameter and pass it through to the parser.**
- [x] 2.5 **Update the `prCommandExecClient` interface and `PRCommandExecutor` to construct a store-backed `TimelineWriter` that calls `store.AppendTimelineEntry` immediately for each event.**
- [x] 2.6 **Update `normalizeOpencodeEvent` to extract rich structured metadata from every JSON event type:**
  - `tool_use`: tool name, callID, status, input object (command, pattern, filePath, patchText, etc.), output string, execution metadata (exit code, time, OpenAI item IDs)
  - `step_start`: snapshot ID, message ID
  - `step_finish`: reason, token breakdown (total, input, output, reasoning, cache read/write), cost
  - `text`: content
  - `error`: error output
- [x] 2.7 **Update fake exec clients and unit tests to pass `nil` or a recording `TimelineWriter` where needed.**

## 3. HTMX dashboard views for command runs

- [x] 3.1 Add a private `/ui/command-runs` dashboard screen and supporting read queries that list currently queued, starting, running, or blocked opencode-backed commands with repository, PR, actor, status, start time, and session context.
- [x] 3.2 Add a private `/ui/command-runs/{commandRequestID}` detail page plus HTMX fragment endpoints that show queued pre-session state, explicit state transitions, refresh the live human-readable timeline as a live tail while the command is non-terminal, and render terminal states clearly once the command finishes.
- [x] 3.3 Link the new command-run views from the existing dashboard navigation and pull-request detail experience while keeping the UI read-only.
- [x] 3.4 **Replace the simple text-list timeline rendering with rich HTML cards per event type:**
  - `tool_use`: tool header (name + status badge), collapsible input section (command, pattern, file path, skill name, patch summary), output preview in `<pre>`, execution metadata footer (exit code, time)
  - `step_start`: step-started indicator
  - `step_finish`: step-completed indicator with token breakdown and cost
  - `text`: preformatted text block
  - `error`: red-highlighted error block
- [x] 3.5 **Add CSS for rich timeline rendering: tool cards, input/output sections, token summaries, status badges, error highlighting.**
- [x] 3.6 **Render the timeline top-to-bottom (chronological order): oldest events at the top, newest events appended at the bottom.** Reverse the query result in the handler before passing to the template so the natural scroll direction matches terminal scrollback.
- [x] 3.7 **Render long text and tool output inside a styled markdown/code box within the card component**, not as plain flowing text. Use a bordered, dark-background `<pre>` or `<code>` container with padding and rounded corners so large blocks of text (skill output, file contents, patch text) are visually contained.

## 4. Behavior-test coverage

- [x] 4.1 Write or update Gherkin behavior tests for accepted-command acknowledgment, including `eyes` reactions, live-output link comments for opencode-backed commands, no live-output link for `/heimdall status`, and no accepted-command feedback for rejected or duplicate observations.
- [x] 4.2 Write or update Gherkin behavior tests for active command-run dashboard views, queued pre-session detail pages, first-event `sessionID` capture, and live-tail human-readable timeline rendering from opencode JSON events.
- [x] 4.3 Implement or update the step bindings, fixtures, fake GitHub interactions, fake opencode event streams, and test runner wiring needed to execute the new command-feedback and live-dashboard scenarios.
- [x] 4.4 **Update BDD timeline assertions to check for rich content (tool names, commands, token counts) instead of generic placeholder strings.**
- [x] 4.5 **Add unit tests for the `TimelineWriter` streaming behavior in the exec adapter.**

## 5. Documentation and verification

- [x] 5.1 Update the relevant docs in `docs/` to describe accepted-command reactions, live command-run dashboard pages, HTMX refresh behavior, the explicit opencode event normalization mapping, and the `HEIMDALL_SERVER_PUBLIC_URL` requirement for GitHub link comments.
- [x] 5.2 **Update docs to describe the streaming timeline architecture, the `TimelineWriter` callback pattern, and the rich event rendering in the dashboard.**
- [x] 5.3 Run the relevant automated tests for the changed PR-command, opencode-runtime, dashboard, and behavior-test coverage, and verify they pass before considering the change complete.

## 6. Edge case robustness

- [x] 6.1 Add idempotency tracking for accepted-command feedback (reactions and link comments) so retries or duplicate observations do not produce duplicates.
- [x] 6.2 Ensure partial GitHub acknowledgment failures do not prevent command execution; queue the command for the worker regardless.
- [x] 6.3 Implement explicit run state machine transitions (queued → starting → running → terminal) in runtime state and dashboard queries.
- [x] 6.4 Render terminal states (completed, failed, blocked) prominently on the command detail page with status indicators and summaries.
- [x] 6.5 Bound live-tail fragment queries to a default limit and provide a way to load earlier entries.
- [x] 6.6 Define and implement the explicit opencode event normalization mapping in the execution adapter.
- [x] 6.7 **Handle the case where `TimelineWriter.WriteEntry` returns an error: log the error but continue parsing so the run outcome is still classified.**
- [x] 6.8 **Ensure the `TimelineWriter` callback does not block the parser for long: keep store writes simple and fast.**
