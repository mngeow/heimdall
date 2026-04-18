## MODIFIED Requirements

### Requirement: Agent selection is explicit and policy-controlled
The execution runtime MUST use the repository's configured default spec-writing agent only for activation-triggered proposal generation. For `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, and `/heimdall opencode`, Heimdall MUST require an explicitly selected agent that is allowed for the repository, MUST preserve the raw prompt tail after the first standalone `--`, MUST resolve exactly one target change before execution starts, MUST verify that the resolved change exists in the active worktree before invoking opencode, and `/heimdall opencode` MUST also require an allowlisted command alias for that repository.

#### Scenario: Activation proposal is started
- **WHEN** an activated work item starts the proposal pull request workflow
- **THEN** Heimdall runs the local OpenCode execution by using the repository's configured default spec-writing agent
- **AND** it does not require per-run agent input for that activation path

#### Scenario: User runs refine with an allowed agent
- **WHEN** a pull request comment requests `/heimdall refine --agent claude-sonnet -- Clarify rollback behavior.` and the repository allows `claude-sonnet`
- **THEN** Heimdall runs the refinement execution by using the selected agent `claude-sonnet`
- **AND** it does not fall back to the repository default spec-writing agent for that comment-driven run

#### Scenario: User runs refine with a multiline prompt body
- **WHEN** a pull request comment requests `/heimdall refine --agent claude-sonnet --` on one line and additional prompt text continues on later lines
- **THEN** Heimdall preserves that later text as the prompt tail for the same refine run
- **AND** it passes the preserved prompt body into the refinement execution instead of truncating it at the first newline

#### Scenario: User runs a PR command without an allowed agent
- **WHEN** a pull request comment requests `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, or `/heimdall opencode` with an agent that is not allowed for the repository
- **THEN** Heimdall does not start the requested execution
- **AND** it records and reports that the requested agent is not authorized for that repository

#### Scenario: User runs a PR command without a resolved target change
- **WHEN** a pull request comment requests `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, or `/heimdall opencode` without an explicit change name and Heimdall cannot resolve exactly one active change for that pull request
- **THEN** Heimdall does not start the requested execution with an empty change name
- **AND** it records and reports that the command target must be specified or cannot be resolved

#### Scenario: User runs a PR command against a stale bound change
- **WHEN** a pull request comment requests `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, or `/heimdall opencode`, Heimdall resolves exactly one change name from runtime state, and that change is missing from the worktree it is about to use
- **THEN** Heimdall does not start the requested execution against that missing change
- **AND** it records and reports that the resolved target is stale or no longer exists in the repository worktree

#### Scenario: User runs a generic opencode command with a disallowed alias
- **WHEN** a pull request comment requests `/heimdall opencode explore-change --agent gpt-5.4` and `explore-change` is not allowlisted for that repository
- **THEN** Heimdall does not start the generic opencode execution
- **AND** it records and reports that the requested command alias is not authorized for that repository

## ADDED Requirements

### Requirement: PR-comment opencode runs use supported invocation and machine-readable events
Heimdall MUST invoke PR-comment-driven refine and apply runs by using supported opencode invocation forms, including positional messages when the CLI path is used, and MUST request machine-readable JSON events for non-interactive CLI execution. Heimdall MUST classify blocked-permission, blocked-input, resumed, and terminal error outcomes from structured event data rather than keyword matching generic stdout, stderr, or CLI help text.

#### Scenario: Refine or apply uses a supported run message and JSON event stream
- **WHEN** Heimdall starts a non-interactive `/heimdall refine`, `/heimdall apply`, or `/opsx-apply` run through the opencode CLI
- **THEN** it sends the change-specific instruction as a supported positional run message
- **AND** it requests machine-readable JSON events for outcome classification instead of depending on formatted help or log output

#### Scenario: CLI help output is treated as an execution error
- **WHEN** the opencode process returns usage text or other CLI help output because the invocation is invalid
- **THEN** Heimdall classifies that result as a normal execution failure
- **AND** it does not convert that help text into a blocked permission or blocked input state

#### Scenario: Permission request IDs come only from explicit permission events
- **WHEN** Heimdall classifies a PR-comment run as blocked on permission
- **THEN** it has observed a real machine-readable permission event such as `permission.asked`
- **AND** it extracts the exact permission request ID and session ID from that event before persisting or commenting on the blocker

### Requirement: PR-command executor outcomes reflect real execution
Heimdall MUST treat the executor entry points behind queued PR commands as real execution boundaries. For `/heimdall status`, `/heimdall refine`, `/heimdall apply`, `/heimdall opencode`, and `/heimdall approve`, Heimdall MUST derive PR feedback and persisted outcome state from the actual command work that ran, not from placeholder completion comments or state-only acknowledgments.

#### Scenario: Refine success requires a real refine run
- **WHEN** Heimdall reports a successful `/heimdall refine` outcome
- **THEN** it has already executed the real refine path for the resolved change and selected agent
- **AND** that success did not come from a placeholder completion branch that skipped refinement work

#### Scenario: Apply success requires a real apply run
- **WHEN** Heimdall reports a successful `/heimdall apply` or `/opsx-apply` outcome
- **THEN** it has already executed the real apply path for the resolved change and selected agent
- **AND** that success did not come from a placeholder completion branch that skipped apply work

#### Scenario: Generic opencode success requires a real alias run
- **WHEN** Heimdall reports a successful `/heimdall opencode <alias>` outcome
- **THEN** it has already executed the configured alias through the generic opencode path for the resolved change and selected agent
- **AND** that success did not come from a placeholder completion branch that skipped alias execution

#### Scenario: Approval success requires a real reply and resume action
- **WHEN** Heimdall reports a successful `/heimdall approve <request-id>` outcome
- **THEN** it has already sent the one-time permission reply for that pending request through the supported opencode permission API and resumed or continued the blocked execution path on the same session
- **AND** that success did not come from only updating stored request state without performing the reply/resume action

### Requirement: PR-comment opencode execution is non-interactive and approval-aware
Heimdall MUST run PR-comment-driven opencode executions without interactive stdin approval loops. If opencode requests additional user input, Heimdall MUST classify the run as blocked and require a retried command. If opencode requests a permission outside the selected execution profile, Heimdall MUST recognize that state from a real machine-readable permission event, persist the exact pending request and session identity, expose that exact request ID to the pull request, and resume only after an authorized `/heimdall approve <request-id>` command replies to that exact pending request.

#### Scenario: Clarification input is requested during a PR-comment run
- **WHEN** Heimdall is executing `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, or `/heimdall opencode` and opencode asks for more user input before it can continue
- **THEN** Heimdall marks the run as blocked because it needs input
- **AND** it stops treating the process as an interactive session waiting on stdin

#### Scenario: Additional permission is requested during a PR-comment run
- **WHEN** Heimdall is executing an agent-driven PR command and opencode asks for a permission outside the selected execution profile such as broader git access than the run allows
- **THEN** Heimdall marks the run as blocked because it needs permission
- **AND** it persists the pending permission request identity and opencode resume state needed for a later approval command

#### Scenario: Generic execution error does not create a fake pending permission request
- **WHEN** Heimdall is executing an agent-driven PR command and the adapter encounters a generic execution error without a real permission event carrying request and session identifiers
- **THEN** Heimdall reports that result as a failed execution
- **AND** it does not persist or comment on a pending permission request for that error

#### Scenario: Authorized approval command resumes a pending permission request
- **WHEN** Heimdall receives `/heimdall approve perm_123` for a still-pending permission request `perm_123` on the same pull request from an authorized user
- **THEN** it replies once to that exact opencode permission request through the supported permission-reply API
- **AND** it observes the resumed execution on the same persisted session instead of starting a brand-new run from scratch

#### Scenario: Approval command targets an unknown or resolved request
- **WHEN** Heimdall receives `/heimdall approve perm_123` but `perm_123` is unknown, already resolved, expired, or scoped to another pull request
- **THEN** it does not send a permission reply to opencode for that request
- **AND** it reports that the approval command was rejected
