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
Heimdall MUST treat `/heimdall refine` as an artifact-only operation that targets a single OpenSpec change, requires an explicitly selected allowed agent, and does not run implementation apply steps.

#### Scenario: User refines an open proposal with a selected agent
- **WHEN** an authorized user comments `/heimdall refine --agent gpt-5.4 -- Clarify rollback behavior and add non-goals.` on an active Heimdall pull request that resolves to one active change
- **THEN** Heimdall runs refinement for that change with agent `gpt-5.4`
- **AND** it updates the relevant OpenSpec proposal artifacts for that change
- **AND** it does not run implementation task execution as part of the refine command

### Requirement: Apply uses an allowed agent and commits results to the same branch
Heimdall MUST run `/heimdall apply` and `/opsx-apply` only with an explicitly selected agent allowed for the target repository, MAY include additional apply guidance after a `--` separator, and MUST commit the resulting task and code changes back to the same proposal branch.

#### Scenario: Authorized apply command selects an allowed agent
- **WHEN** an authorized user comments `/heimdall apply --agent gpt-5.4 -- Focus on the PR comment executor first.` on a Heimdall pull request whose repository allows `gpt-5.4`
- **THEN** Heimdall runs the apply workflow with that selected agent and additional prompt guidance
- **AND** Heimdall commits and pushes the resulting task updates and implementation changes to the same branch

#### Scenario: Compatibility alias is used for apply
- **WHEN** an authorized user comments `/opsx-apply --agent gpt-5.4` on a Heimdall pull request whose repository allows `gpt-5.4`
- **THEN** Heimdall treats the request as the same workflow as `/heimdall apply --agent gpt-5.4`
- **AND** it applies the same authorization, change-resolution, and commit behavior

## ADDED Requirements

### Requirement: Agent-driven PR commands resolve a single target change
Heimdall MUST resolve `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, and `/heimdall opencode` against exactly one OpenSpec change before execution starts. If the comment omits `change-name`, Heimdall MUST infer it only when the pull request has exactly one active OpenSpec change.

#### Scenario: Omitted change name resolves from a single active change
- **WHEN** an authorized user comments `/heimdall apply --agent gpt-5.4` on a Heimdall pull request that is bound to exactly one active OpenSpec change
- **THEN** Heimdall resolves that single active change as the command target
- **AND** it proceeds without requiring the user to restate the change name

#### Scenario: Omitted change name is ambiguous
- **WHEN** an authorized user comments `/heimdall refine --agent gpt-5.4 -- Add more rollback detail.` on a Heimdall pull request that contains more than one active OpenSpec change
- **THEN** Heimdall does not guess which change to mutate
- **AND** it records and reports that the command must be retried with an explicit change name

### Requirement: Generic opencode commands are allowlisted and policy-bounded
Heimdall MUST run `/heimdall opencode <command-alias>` only when that alias is allowlisted for the target repository and bound to a configured opencode command profile.

#### Scenario: Allowed generic opencode command runs with a selected agent
- **WHEN** an authorized user comments `/heimdall opencode explore-change --agent gpt-5.4 -- Compare two permission-handling options.` on a Heimdall pull request and `explore-change` is an allowed alias for that repository
- **THEN** Heimdall runs the configured opencode command bound to `explore-change` by using agent `gpt-5.4`
- **AND** it scopes that execution to the pull request worktree and resolved change context

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

### Requirement: Pending permission requests require an explicit Heimdall approval command
Heimdall MUST allow an authorized user to approve a pending opencode permission request only by issuing `/heimdall approve <permission-request-id>` on the same Heimdall-managed pull request where that request was reported.

#### Scenario: Authorized user approves a pending permission request
- **WHEN** Heimdall has already reported a pending permission request `perm_123` on a Heimdall-managed pull request and an authorized user comments `/heimdall approve perm_123`
- **THEN** Heimdall approves that exact pending permission request once
- **AND** it resumes the blocked command execution and reports the resumed outcome on the pull request

#### Scenario: Unknown or stale permission request approval is rejected
- **WHEN** an authorized user comments `/heimdall approve perm_123` on a Heimdall-managed pull request but `perm_123` is unknown, already resolved, expired, or belongs to a different pull request
- **THEN** Heimdall does not send any permission approval for that request ID
- **AND** it records and reports that the approval command was rejected
