## MODIFIED Requirements

### Requirement: Agent selection is explicit and policy-controlled
The execution runtime MUST use the repository's configured default spec-writing agent only for activation-triggered proposal generation. For `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, and `/heimdall opencode`, Heimdall MUST require an explicitly selected agent that is allowed for the repository, and `/heimdall opencode` MUST also require an allowlisted command alias for that repository.

#### Scenario: Activation proposal is started
- **WHEN** an activated work item starts the proposal pull request workflow
- **THEN** Heimdall runs the local OpenCode execution by using the repository's configured default spec-writing agent
- **AND** it does not require per-run agent input for that activation path

#### Scenario: User runs refine with an allowed agent
- **WHEN** a pull request comment requests `/heimdall refine --agent claude-sonnet -- Clarify rollback behavior.` and the repository allows `claude-sonnet`
- **THEN** Heimdall runs the refinement execution by using the selected agent `claude-sonnet`
- **AND** it does not fall back to the repository default spec-writing agent for that comment-driven run

#### Scenario: User runs a PR command without an allowed agent
- **WHEN** a pull request comment requests `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, or `/heimdall opencode` with an agent that is not allowed for the repository
- **THEN** Heimdall does not start the requested execution
- **AND** it records and reports that the requested agent is not authorized for that repository

#### Scenario: User runs a generic opencode command with a disallowed alias
- **WHEN** a pull request comment requests `/heimdall opencode explore-change --agent gpt-5.4` and `explore-change` is not allowlisted for that repository
- **THEN** Heimdall does not start the generic opencode execution
- **AND** it records and reports that the requested command alias is not authorized for that repository

## ADDED Requirements

### Requirement: PR-comment opencode execution is non-interactive and approval-aware
Heimdall MUST run PR-comment-driven opencode executions without interactive stdin approval loops. If opencode requests additional user input, Heimdall MUST classify the run as blocked and require a retried command. If opencode requests a permission outside the selected execution profile, Heimdall MUST persist the pending request, expose its request ID to the pull request, and resume only after an authorized `/heimdall approve <request-id>` command replies to that exact pending request.

#### Scenario: Clarification input is requested during a PR-comment run
- **WHEN** Heimdall is executing `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, or `/heimdall opencode` and opencode asks for more user input before it can continue
- **THEN** Heimdall marks the run as blocked because it needs input
- **AND** it stops treating the process as an interactive session waiting on stdin

#### Scenario: Additional permission is requested during a PR-comment run
- **WHEN** Heimdall is executing an agent-driven PR command and opencode asks for a permission outside the selected execution profile such as broader git access than the run allows
- **THEN** Heimdall marks the run as blocked because it needs permission
- **AND** it persists the pending permission request identity and opencode resume state needed for a later approval command

#### Scenario: Authorized approval command resumes a pending permission request
- **WHEN** Heimdall receives `/heimdall approve perm_123` for a still-pending permission request `perm_123` on the same pull request from an authorized user
- **THEN** it replies once to that exact opencode permission request
- **AND** it resumes the blocked opencode execution instead of starting a brand-new run from scratch

#### Scenario: Approval command targets an unknown or resolved request
- **WHEN** Heimdall receives `/heimdall approve perm_123` but `perm_123` is unknown, already resolved, expired, or scoped to another pull request
- **THEN** it does not send a permission reply to opencode for that request
- **AND** it reports that the approval command was rejected
