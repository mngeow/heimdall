## Why

Heimdall can already detect PR slash commands such as `/heimdall refine` and start `opencode run`, but today that work is largely invisible unless an operator is watching host logs. Users do not get immediate confirmation that the command was recognized, and operators cannot follow the live run output from the Heimdall UI even though the runtime already receives a structured JSON event stream with a canonical `sessionID` in the first chunk.

This change is needed now because comment-driven refine and apply flows are becoming a primary interaction surface. Heimdall needs a clear acknowledgment path in GitHub plus a private HTMX-based UI that turns opencode's machine-readable event stream into a human-readable live run view.

## What Changes

- Add a read-only Heimdall UI view for active PR-command opencode runs so operators can see which commands are currently running and inspect live human-readable output for each run.
- Require Heimdall to capture the canonical opencode `sessionID` from the first structured JSON event, associate it with the originating command request, and keep enough ordered run-output state to drive the UI while the command is active.
- Require Heimdall to parse `opencode run --format json` events into a sanitized, human-readable output stream for the UI instead of exposing raw JSON blobs.
- Require Heimdall to react to an accepted `/heimdall ...` pull-request comment with an acknowledgment emoji as soon as the command is recognized.
- Require Heimdall to post a follow-up pull-request comment with a link to the relevant Heimdall UI page so the user can open the live run output directly from GitHub.
- Keep the new UI private, read-only, and HTMX-driven, and do not expose raw prompt bodies, secrets, or unparsed provider payloads.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `feature-pr-command-workflows`: add visible GitHub acknowledgment and UI-link feedback when Heimdall accepts a PR command, plus user-visible live-run observability expectations.
- `service-execution-runtime`: define how Heimdall parses the opencode JSON event stream, captures the first-event `sessionID`, and derives human-readable live output from structured events.
- `service-runtime-state`: define the durable and in-progress state Heimdall keeps for command-linked opencode runs, session identities, and ordered UI-ready output entries.
- `service-operator-dashboard`: extend the private HTMX dashboard with active command-run views and live run-detail output for PR-command executions.
- `service-github-scm`: require GitHub comment reactions and pull-request reply comments that link accepted commands back to the Heimdall UI.
- `service-configuration`: define the stable public Heimdall base URL used to build operator-dashboard links posted back into GitHub comments.

## Impact

- Affected code: PR comment intake and acknowledgment, GitHub reactions/comments, operator-URL configuration, opencode event parsing, runtime-state persistence for active runs and output entries, dashboard handlers/query services/templates, and behavior tests.
- Affected systems: GitHub pull-request comments, Heimdall's private operator UI, SQLite runtime state, and local `opencode run --format json` execution.
- Operator impact: the Heimdall operator URL must be stable enough to include in PR reply comments, and the linked UI remains a private read-only surface rather than a workflow-control surface.
- Safety impact: Heimdall will transform structured opencode events into sanitized display output rather than exposing raw JSON event payloads directly in the UI.
