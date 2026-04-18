Feature: Pull request command handling
  As an operator
  I want to control Heimdall workflows through PR comments
  So that I can refine specs, trigger implementation, run generic opencode commands, and approve blocked permission requests

  Background:
    Given Heimdall is configured with GitHub polling
    Given a Heimdall-managed pull request exists
    And the PR author is in the allowed users list

  Rule: Authorized users can trigger commands discovered during polling

    Scenario: Status command
      Given a Heimdall-managed pull request exists
      When the user comments "/heimdall status"
      And Heimdall polls GitHub
      Then Heimdall should discover the comment during polling
      And Heimdall should reply with the current proposal status

    Scenario: Missing persisted PR-command worker state is terminal
      Given a Heimdall-managed pull request exists
      And the pull request record was removed from the database
      When the user comments "/heimdall status"
      And Heimdall polls GitHub
      And the PR-command worker processes the queued job
      Then Heimdall should discover the comment during polling
      And Heimdall should report that the command failed and not leave it silently queued

    Scenario: Multiline refine prompt is preserved and executed
      Given a Heimdall-managed pull request exists
      And the repository allows agent "gpt-5.4"
      When the user comments a multiline refine command with trailing separator
      And Heimdall polls GitHub
      And the PR-command worker processes the queued job
      Then Heimdall should discover the comment during polling
      And Heimdall should update the proposal artifacts using the full prompt body
      And Heimdall should commit the changes
      And Heimdall should push the updated branch

    Scenario: Refine with omitted change name and no active target is rejected
      Given a Heimdall-managed pull request exists with no active changes
      And the repository allows agent "gpt-5.4"
      When the user comments "/heimdall refine --agent gpt-5.4 -- Add detail"
      And Heimdall polls GitHub
      And the PR-command worker processes the queued job
      Then Heimdall should discover the comment during polling
      And Heimdall should reject the command because no active change could be resolved

    Scenario: Refine command with explicit agent
      Given a Heimdall-managed pull request exists
      And the repository allows agent "gpt-5.4"
      When the user comments "/heimdall refine --agent gpt-5.4 -- Add error handling section"
      And Heimdall polls GitHub
      Then Heimdall should discover the comment during polling
      And Heimdall should update the proposal artifacts
      And Heimdall should commit the changes
      And Heimdall should push the updated branch

    Scenario: Apply command with allowed agent
      Given a Heimdall-managed pull request exists
      And the repository allows agent "gpt-5.4"
      When the user comments "/heimdall apply --agent gpt-5.4"
      And Heimdall polls GitHub
      Then Heimdall should discover the comment during polling
      And Heimdall should execute the apply workflow
      And Heimdall should commit implementation changes
      And Heimdall should comment with the execution results

    Scenario: Compatibility alias for apply
      Given a Heimdall-managed pull request exists
      And the repository allows agent "gpt-5.4"
      When the user comments "/opsx-apply --agent gpt-5.4"
      And Heimdall polls GitHub
      Then Heimdall should discover the comment during polling
      And Heimdall should execute the apply workflow

    Scenario: Generic opencode command with allowed alias
      Given a Heimdall-managed pull request exists
      And the repository allows agent "gpt-5.4"
      And the repository configures opencode alias "explore-change"
      When the user comments "/heimdall opencode explore-change --agent gpt-5.4 -- Compare options"
      And Heimdall polls GitHub
      Then Heimdall should discover the comment during polling
      And Heimdall should run the configured opencode command
      And Heimdall should comment with the execution results

  Rule: Agent-driven commands resolve a single target change

    Scenario: Omitted change name resolves from a single active change
      Given a Heimdall-managed pull request exists with exactly one active change
      When the user comments "/heimdall apply --agent gpt-5.4"
      And Heimdall polls GitHub
      Then Heimdall should resolve that single active change as the target

    Scenario: Omitted change name is ambiguous
      Given a Heimdall-managed pull request exists with more than one active change
      When the user comments "/heimdall refine --agent gpt-5.4 -- Add detail"
      And Heimdall polls GitHub
      Then Heimdall should reject the command as ambiguous
      And Heimdall should comment that the change name must be specified

  Rule: PR commands use a canonical prepared worktree

    Scenario: Refine validates after preparing the PR worktree
      Given a Heimdall-managed pull request exists with exactly one active change
      And the repository allows agent "gpt-5.4"
      When the user comments "/heimdall refine --agent gpt-5.4 -- Add detail"
      And Heimdall polls GitHub
      And the PR-command worker processes the queued job
      Then Heimdall should derive the worktree path from the repository mirror and PR head branch
      And Heimdall should prepare that worktree before validating the resolved change
      And Heimdall should run opencode in the same prepared worktree

    Scenario: Cross-repository branch name collision is ignored
      Given a Heimdall-managed pull request exists with exactly one active change
      And another repository has an active binding with the same branch name
      When the user comments "/heimdall apply --agent gpt-5.4"
      And Heimdall polls GitHub
      Then Heimdall should resolve the change only from the pull request's own repository context
      And Heimdall should not include the other repository's binding as a candidate target

  Rule: Large valid opencode events do not abort PR commands

    Scenario: Refine succeeds when opencode emits a large text event
      Given a Heimdall-managed pull request exists with exactly one active change
      And the repository allows agent "gpt-5.4"
      And opencode emits a large text event before the final outcome
      When the user comments "/heimdall refine --agent gpt-5.4 -- Add detail"
      And Heimdall polls GitHub
      And the PR-command worker processes the queued job
      Then Heimdall should discover the comment during polling
      And Heimdall should update the proposal artifacts
      And Heimdall should commit the changes
      And Heimdall should push the updated branch

  Rule: Unauthorized commands are rejected

    Scenario: User not in allowed list
      Given a user not in the allowed users list
      When they comment "/heimdall status"
      And Heimdall polls GitHub
      Then the command should be rejected
      And no workflow should be triggered

    Scenario: Agent not in allowed list
      Given the repository does not allow agent "unauthorized-agent"
      When the user comments "/heimdall apply --agent unauthorized-agent"
      And Heimdall polls GitHub
      Then the command should be rejected
      And Heimdall should comment that the agent is not authorized

    Scenario: Unknown generic opencode alias
      Given the repository does not configure opencode alias "unknown-alias"
      When the user comments "/heimdall opencode unknown-alias --agent gpt-5.4"
      And Heimdall polls GitHub
      Then the command should be rejected
      And Heimdall should comment that the alias is not authorized

  Rule: Duplicate commands are safe

    Scenario: Same comment observed twice across polling windows
      Given a command has already been processed
      When the same comment is observed in another GitHub poll
      Then the duplicate should be detected
      And the command should not be executed again

    Scenario: Comment is edited after initial discovery
      Given a command comment exists
      When Heimdall polls an edited version of the same comment
      Then the edit should not trigger a new command execution

    Scenario: Queued status command is executed by the worker
      Given a Heimdall-managed pull request exists
      When the user comments "/heimdall status"
      And Heimdall polls GitHub
      And the PR-command worker processes the queued job
      Then Heimdall should discover the comment during polling
      And Heimdall should reply with the current proposal status

  Rule: Blocked opencode requests surface actionable PR feedback

    Scenario: Opencode asks for clarification input
      Given a Heimdall-managed pull request exists
      When the user comments "/heimdall refine --agent gpt-5.4 -- Incomplete prompt"
      And Heimdall polls GitHub
      And the opencode run blocks on clarification input
      Then Heimdall should post a comment that the command is blocked on missing input
      And Heimdall should suggest how to retry the command

    Scenario: Opencode asks for additional permission
      Given a Heimdall-managed pull request exists
      When the user comments "/heimdall apply --agent gpt-5.4"
      And Heimdall polls GitHub
      And the opencode run blocks on a permission request
      Then Heimdall should post a comment with the permission request ID
      And Heimdall should include the exact approval command to run next

    Scenario: Approve a pending permission request
      Given a Heimdall-managed pull request exists
      And a pending permission request "perm_123" was reported on that pull request
      When the user comments "/heimdall approve perm_123"
      And Heimdall polls GitHub
      Then Heimdall should approve that exact pending permission request once
      And Heimdall should resume the blocked command execution
      And Heimdall should comment with the resumed outcome

    Scenario: Unknown or stale permission request approval
      Given a Heimdall-managed pull request exists
      When the user comments "/heimdall approve perm_999"
      And Heimdall polls GitHub
      Then Heimdall should reject the approval command
      And Heimdall should comment that the request ID is unknown or already resolved

  Rule: Label-scoped polling ignores unlabeled pull requests

    Scenario: Unlabeled Heimdall pull request is ignored when monitor label is configured
      Given a Heimdall-managed pull request exists
      And the repository configures PR monitor label "heimdall-monitored"
      When the user comments "/heimdall status"
      And Heimdall polls GitHub
      Then Heimdall should ignore the pull request because it is missing monitor label
