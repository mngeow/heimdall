## MODIFIED Requirements

### Requirement: Agent-driven PR commands resolve a single target change
Heimdall MUST resolve `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, and `/heimdall opencode` against exactly one OpenSpec change before execution starts. If the comment omits `change-name`, Heimdall MUST infer it only from active bindings linked to that pull request's durable binding and repository context, MUST prepare the canonical pull-request worktree before validating the resolved change, and MUST NOT execute an agent-driven command with an empty unresolved change name or a target that only matches another repository's branch.

#### Scenario: Omitted change name resolves from a single active change
- **WHEN** an authorized user comments `/heimdall apply --agent gpt-5.4` on a Heimdall pull request that is bound to exactly one active OpenSpec change
- **THEN** Heimdall resolves that single active change as the command target
- **AND** it proceeds without requiring the user to restate the change name

#### Scenario: Omitted change name is ambiguous
- **WHEN** an authorized user comments `/heimdall refine --agent gpt-5.4 -- Add more rollback detail.` on a Heimdall pull request that contains more than one active OpenSpec change
- **THEN** Heimdall does not guess which change to mutate
- **AND** it records and reports that the command must be retried with an explicit change name

#### Scenario: Omitted change name has no active target
- **WHEN** an authorized user comments `/heimdall refine --agent gpt-5.4 -- Add more rollback detail.` on a Heimdall pull request that is not bound to any active OpenSpec change
- **THEN** Heimdall does not start refine execution with an empty change name
- **AND** it records and reports that no active change could be resolved for that pull request

#### Scenario: Resolved change no longer exists in the prepared pull request worktree
- **WHEN** an agent-driven PR command resolves a change name from Heimdall runtime bindings but that change is missing from the prepared pull request worktree at execution time
- **THEN** Heimdall rejects the command as a stale target instead of sending that missing change into opencode
- **AND** it reports that the pull request binding no longer matches a real OpenSpec change in the repository

#### Scenario: Branch-name collision in another repository does not affect target resolution
- **WHEN** an agent-driven PR command executes for a Heimdall-managed pull request and another repository also has an active binding with the same branch name
- **THEN** Heimdall resolves candidate target changes only from the same pull request and repository context
- **AND** it does not treat the other repository's binding as an eligible target for that command

### Requirement: Agent-driven PR commands are not aborted by valid large opencode event lines
Heimdall MUST continue `/heimdall refine`, `/heimdall apply`, and `/opsx-apply` execution when `opencode run --format json` emits valid newline-delimited JSON events whose single lines are much larger than typical log output. Heimdall MUST classify the command from the structured event stream instead of failing the run because its local event reader hit a token-length limit.

#### Scenario: Large valid text event precedes the real outcome
- **WHEN** Heimdall executes an agent-driven PR command and opencode emits a valid large `text` event line before later permission, error, or completion events
- **THEN** Heimdall continues parsing the event stream
- **AND** it reports the later structured outcome instead of failing the command with a local token-length reader error

### Requirement: Agent-driven PR command results follow the terminal opencode session outcome
Heimdall MUST report `/heimdall refine`, `/heimdall apply`, and `/opsx-apply` outcomes from the terminal opencode session result rather than from the first intermediate generic error event. If the run truly fails without a detailed structured message, Heimdall MUST still produce a non-empty failure summary instead of a blank `refine failed:` or `apply failed:` result.

#### Scenario: Intermediate generic tool error is followed by successful refine completion
- **WHEN** Heimdall executes `/heimdall refine` and opencode emits an intermediate generic `tool_use` error event with empty output before later completion evidence shows the session succeeded
- **THEN** Heimdall reports the refine command as successful
- **AND** it does not leave the queued job or command request in a failed state just because of the earlier empty intermediate error event

#### Scenario: True generic failure still has a useful failure message
- **WHEN** Heimdall executes `/heimdall refine` or `/heimdall apply` and the opencode session truly fails without a detailed structured error message
- **THEN** Heimdall records and reports a non-empty fallback failure summary
- **AND** operators do not have to debug a blank failure message to know that the run failed
