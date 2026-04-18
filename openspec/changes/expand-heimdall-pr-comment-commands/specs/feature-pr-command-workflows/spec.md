## MODIFIED Requirements

### Requirement: Heimdall accepts a narrow set of slash commands on automation pull requests
Heimdall MUST detect only the documented slash command surface by polling GitHub comments on pull requests that it created and MUST ignore unsupported mutation commands.

#### Scenario: Supported command is discovered during polling on a Heimdall pull request
- **WHEN** an authorized user posts `/heimdall status`, `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, `/heimdall opencode`, `/heimdall approve`, or `/opsx-archive` on a Heimdall-created pull request and a GitHub poll cycle observes that new comment
- **THEN** Heimdall parses the command and enqueues the matching workflow action
- **AND** Heimdall links the command request to the target pull request and repository binding

#### Scenario: Unsupported mutation command is discovered during polling
- **WHEN** a GitHub poll cycle observes a pull request comment that contains an unsupported Heimdall mutation command
- **THEN** Heimdall does not run repository mutation logic for that comment
- **AND** Heimdall records that the command was ignored or rejected

### Requirement: Refinement updates OpenSpec artifacts without applying implementation tasks
Heimdall MUST treat `/heimdall refine` as an artifact-only operation that targets a single OpenSpec change, requires an explicitly selected allowed agent, preserves the raw prompt tail after the first standalone `--`, and does not run implementation apply steps.

#### Scenario: User refines an open proposal with a selected agent
- **WHEN** an authorized user comments `/heimdall refine --agent gpt-5.4 -- Clarify rollback behavior and add non-goals.` on an active Heimdall pull request that resolves to one active change
- **THEN** Heimdall runs refinement for that change with agent `gpt-5.4`
- **AND** it updates the relevant OpenSpec proposal artifacts for that change
- **AND** it does not run implementation task execution as part of the refine command

#### Scenario: Multiline refine prompt is preserved after a trailing separator
- **WHEN** an authorized user comments `/heimdall refine --agent gpt-5.4 --` on one line of a Heimdall pull request comment and the following lines contain additional prompt text such as numbered implementation asks
- **THEN** Heimdall preserves those later lines as the raw prompt tail for the same refine command
- **AND** it passes that prompt tail into refinement execution instead of silently discarding the multiline body

#### Scenario: Refine reports success only after real execution succeeds
- **WHEN** an authorized user comments `/heimdall refine --agent gpt-5.4 -- Clarify rollback behavior.` on a Heimdall pull request that resolves to one active change
- **THEN** Heimdall runs the real refine execution for that change before posting a success comment
- **AND** it does not publish a successful refine outcome when no refine execution was attempted

### Requirement: Apply uses an allowed agent and commits results to the same branch
Heimdall MUST run `/heimdall apply` and `/opsx-apply` only with an explicitly selected agent allowed for the target repository, MAY include additional apply guidance after a `--` separator, and MUST commit the resulting task and code changes back to the same proposal branch.

#### Scenario: Authorized apply command selects an allowed agent
- **WHEN** an authorized user comments `/heimdall apply --agent gpt-5.4 -- Focus on the PR comment executor first.` on a Heimdall pull request whose repository allows `gpt-5.4`
- **THEN** Heimdall runs the apply workflow with that selected agent and additional prompt guidance
- **AND** Heimdall commits and pushes the resulting task updates and implementation changes to the same branch

#### Scenario: Apply reports success only after real execution succeeds
- **WHEN** an authorized user comments `/heimdall apply --agent gpt-5.4` on a Heimdall pull request whose repository allows `gpt-5.4`
- **THEN** Heimdall runs the real apply execution for the resolved target change before posting a success comment
- **AND** it does not publish a completed apply outcome when no apply execution was attempted

#### Scenario: Compatibility alias is used for apply
- **WHEN** an authorized user comments `/opsx-apply --agent gpt-5.4` on a Heimdall pull request whose repository allows `gpt-5.4`
- **THEN** Heimdall treats the request as the same workflow as `/heimdall apply --agent gpt-5.4`
- **AND** it applies the same authorization, change-resolution, and commit behavior

## ADDED Requirements

### Requirement: Queued PR commands are executed by a started background worker
Heimdall MUST start a PR-command worker as part of normal service runtime and MUST use that worker to execute every supported queued pull-request command instead of leaving accepted commands indefinitely queued.

#### Scenario: Status command produces a visible reply after intake
- **WHEN** an authorized user posts `/heimdall status` on a Heimdall-managed pull request and a GitHub poll cycle accepts that comment as a new command request
- **THEN** Heimdall starts or already has a running PR-command worker that dequeues the queued status job
- **AND** Heimdall posts one pull-request comment describing the current status outcome for that request instead of leaving it silently queued

#### Scenario: Status command reflects actual pull-request state
- **WHEN** the PR-command worker executes `/heimdall status` for a Heimdall-managed pull request
- **THEN** Heimdall loads the actual pull-request-bound change state for that pull request
- **AND** it posts a status comment derived from that real state rather than a placeholder success response

#### Scenario: Worker dispatch covers the supported queued command surface
- **WHEN** the PR-command worker dequeues a queued job created from `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, `/heimdall opencode`, or `/heimdall approve`
- **THEN** Heimdall dispatches that job to the matching command-execution path for the supported command kind
- **AND** it does not reject the job as an unknown worker command type

#### Scenario: Successful queued command does not remain running
- **WHEN** the PR-command worker finishes a queued command successfully for a Heimdall-managed pull request
- **THEN** Heimdall marks both the command request and the queued job as completed
- **AND** a later queued command for that same pull request is not blocked by a stale `running` job that already produced a terminal outcome

### Requirement: Queued PR command execution uses durable runtime identifiers
Heimdall MUST resolve queued PR-command execution from the durable command-request, pull-request, and repository records stored in runtime state, rather than reconstructing those lookups from unrelated derived values.

#### Scenario: Worker loads the queued command request from persisted job state
- **WHEN** a PR-command job is dequeued for execution
- **THEN** Heimdall loads the originating command request by the durable identifier persisted on that job
- **AND** it uses the linked pull-request and repository records from runtime state to continue execution

#### Scenario: Missing persisted state becomes a terminal visible failure
- **WHEN** the PR-command worker dequeues a job whose persisted command request, pull request, or repository record can no longer be loaded
- **THEN** Heimdall marks that queued command as failed or otherwise terminally unresolved in runtime state
- **AND** it does not leave the command appearing healthy while only later duplicate poll observations continue to appear in logs

### Requirement: Agent-driven PR commands resolve a single target change
Heimdall MUST resolve `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, and `/heimdall opencode` against exactly one OpenSpec change before execution starts. If the comment omits `change-name`, Heimdall MUST infer it only when the pull request has exactly one active OpenSpec change, and it MUST NOT execute an agent-driven command with an empty unresolved change name.

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

#### Scenario: Resolved change no longer exists in the worktree
- **WHEN** an agent-driven PR command resolves a change name from Heimdall runtime bindings but that change is missing from the pull request worktree at execution time
- **THEN** Heimdall rejects the command as a stale target instead of sending that missing change into opencode
- **AND** it reports that the pull request binding no longer matches a real OpenSpec change in the repository

### Requirement: Generic opencode commands are allowlisted and policy-bounded
Heimdall MUST run `/heimdall opencode <command-alias>` only when that alias is allowlisted for the target repository and bound to a configured opencode command profile.

#### Scenario: Allowed generic opencode command runs with a selected agent
- **WHEN** an authorized user comments `/heimdall opencode explore-change --agent gpt-5.4 -- Compare two permission-handling options.` on a Heimdall pull request and `explore-change` is an allowed alias for that repository
- **THEN** Heimdall runs the configured opencode command bound to `explore-change` by using agent `gpt-5.4`
- **AND** it scopes that execution to the pull request worktree and resolved change context

#### Scenario: Generic opencode success requires a real alias execution
- **WHEN** an authorized user comments `/heimdall opencode explore-change --agent gpt-5.4` on a Heimdall pull request and `explore-change` is an allowed alias for that repository
- **THEN** Heimdall runs the actual configured alias execution before posting a success comment
- **AND** it does not report the command as completed if the alias execution was never attempted

#### Scenario: Unknown generic opencode alias is rejected
- **WHEN** an authorized user comments `/heimdall opencode explore-change --agent gpt-5.4` on a Heimdall pull request and `explore-change` is not configured as an allowed alias for that repository
- **THEN** Heimdall does not start opencode execution for that comment
- **AND** it records and reports that the alias is unsupported for that repository

### Requirement: Interactive opencode requests surface actionable PR feedback
Heimdall MUST surface clarification requests and permission requests from agent-driven PR commands as actionable blocked command results instead of waiting for an interactive terminal response.

#### Scenario: Opencode asks for clarification input
- **WHEN** a `/heimdall refine`, `/heimdall apply`, or `/heimdall opencode` run reaches a point where opencode requests additional user input before it can continue
- **THEN** Heimdall stops waiting for interactive progress and marks the command as blocked
- **AND** it posts a pull request comment summarizing the missing input and how to retry the command

#### Scenario: Opencode asks for additional tool permission
- **WHEN** an agent-driven PR command reaches a point where opencode asks to approve a permission outside the selected command profile such as broader git access
- **THEN** Heimdall does not auto-approve that request
- **AND** it posts a pull request comment that the command is blocked on permission policy and includes the permission request ID to approve next

#### Scenario: Permission feedback uses the exact runtime request ID
- **WHEN** opencode emits a real permission request event with request ID `perm_123` during an agent-driven PR command
- **THEN** Heimdall persists and comments exactly `perm_123` as the pending approval target
- **AND** it does not publish `/heimdall approve` guidance with an empty, inferred, or synthetic permission request ID

#### Scenario: CLI usage output is not treated as a permission request
- **WHEN** an agent-driven PR command fails because the opencode invocation is invalid and opencode returns command usage or help output
- **THEN** Heimdall reports that result as a failed execution
- **AND** it does not create a pending permission request or publish an approval command for that failure

### Requirement: Pending permission requests require an explicit Heimdall approval command
Heimdall MUST allow an authorized user to approve a pending opencode permission request only by issuing `/heimdall approve <permission-request-id>` on the same Heimdall-managed pull request where that request was reported, and MUST treat approval as a real permission-reply-and-resume action rather than a state-only acknowledgment.

#### Scenario: Authorized user approves a pending permission request
- **WHEN** Heimdall has already reported a pending permission request `perm_123` on a Heimdall-managed pull request and an authorized user comments `/heimdall approve perm_123`
- **THEN** Heimdall approves that exact pending permission request once
- **AND** it resumes the blocked command execution and reports the resumed outcome on the pull request

#### Scenario: Approval is not complete until reply and resume occur
- **WHEN** Heimdall receives `/heimdall approve perm_123` for a still-pending blocked run
- **THEN** Heimdall sends the actual one-time permission reply for `perm_123`
- **AND** it does not report approval success solely because the persisted pending-request row was updated

#### Scenario: Unknown or stale permission request approval is rejected
- **WHEN** an authorized user comments `/heimdall approve perm_123` on a Heimdall-managed pull request but `perm_123` is unknown, already resolved, expired, or belongs to a different pull request
- **THEN** Heimdall does not send any permission approval for that request ID
- **AND** it records and reports that the approval command was rejected
