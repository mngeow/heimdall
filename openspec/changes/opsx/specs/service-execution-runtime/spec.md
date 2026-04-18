## ADDED Requirements

### Requirement: Structured opencode events are normalized into live UI output
Heimdall MUST turn the newline-delimited JSON event stream from PR-comment opencode executions into an ordered human-readable output timeline for the originating command request. It MUST capture the canonical `sessionID` from the first structured event, associate later rendered entries with that same command-linked run, preserve those rendered entries in observed stream order so the dashboard can present them as a live tail, and MUST NOT expose raw JSON lines directly in the operator UI.

#### Scenario: Text and tool events are rendered as human-readable live output
- **WHEN** Heimdall runs `/heimdall refine --agent gpt-5.4` and the opencode JSON stream emits a first structured event with `sessionID` `ses_abc`, followed by `text` and tool-related events
- **THEN** Heimdall records `ses_abc` as the canonical session identity for that command-linked run
- **AND** it appends ordered human-readable live-output entries that summarize the text and tool activity instead of exposing raw JSON lines to the dashboard

#### Scenario: Unknown structured events fall back to readable status entries
- **WHEN** a PR-comment opencode run emits a valid structured event type that Heimdall does not yet map to a richer presentation
- **THEN** Heimdall still appends a readable timeline entry that identifies the event type and its operational meaning as best it can
- **AND** it does not surface the raw JSON blob directly in the dashboard as the only user-visible output

#### Scenario: Blocked events appear in the same live timeline
- **WHEN** a PR-comment opencode run emits a structured blocker event such as a permission or input request after live output has already started
- **THEN** Heimdall appends a human-readable blocker entry to the same command-linked timeline
- **AND** the linked command view continues to reference the same canonical session identity for that run

#### Scenario: Later output stays append-ordered for live tailing
- **WHEN** a PR-comment opencode run emits additional structured output events after earlier display entries were already recorded
- **THEN** Heimdall records the newer human-readable entries after the older ones in the same observed order
- **AND** the dashboard can render the command detail page as a live tail rather than a reordered or replaced transcript

### Requirement: Opencode event normalization uses an explicit mapping
Heimdall MUST normalize structured opencode events into human-readable display entries according to an explicit, versioned mapping. The normalization MUST cover at minimum the event types required for live-tail observability.

#### Scenario: Text event renders as assistant message
- **WHEN** the opencode JSON stream emits a `text` event containing assistant-generated content
- **THEN** Heimdall creates a display entry that presents the text content as an assistant message block

#### Scenario: Tool use event renders as tool status line
- **WHEN** the opencode JSON stream emits a `tool_use` or `tool_start` event
- **THEN** Heimdall creates a display entry that identifies the tool name and its operational intent as a concise status line

#### Scenario: Permission request event renders as blocker notice
- **WHEN** the opencode JSON stream emits a `permission_request` event
- **THEN** Heimdall creates a display entry that presents the permission request as a highlighted blocker notice with the request ID

#### Scenario: Error event renders as error summary
- **WHEN** the opencode JSON stream emits an `error` event
- **THEN** Heimdall creates a display entry that presents the error type and message as a human-readable error summary

#### Scenario: Session start event captures session identity
- **WHEN** the opencode JSON stream emits a `session_start` or equivalent first structured event carrying `sessionID`
- **THEN** Heimdall records the `sessionID` and creates a display entry indicating that the session has started

#### Scenario: Unknown event falls back to generic status
- **WHEN** the opencode JSON stream emits a structured event type not covered by the explicit mapping
- **THEN** Heimdall creates a generic display entry that identifies the event type and its operational meaning
- **AND** it does not expose the raw JSON payload directly
