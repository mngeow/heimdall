Feature: Pull request command handling
  As an operator
  I want to control Heimdall workflows through PR comments
  So that I can refine specs and trigger implementation

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

    Scenario: Refine command
      Given a Heimdall-managed pull request exists
      When the user comments "/heimdall refine Add error handling section"
      And Heimdall polls GitHub
      Then Heimdall should discover the comment during polling
      And Heimdall should update the proposal artifacts
      And Heimdall should commit the changes
      And Heimdall should push the updated branch

    Scenario: Apply command with allowed agent
      Given a Heimdall-managed pull request exists
      And the repository allows agent "gpt-5.4"
      When the user comments "/opsx-apply --agent gpt-5.4"
      And Heimdall polls GitHub
      Then Heimdall should discover the comment during polling
      And Heimdall should execute the apply workflow
      And Heimdall should commit implementation changes
      And Heimdall should comment with the execution results

  Rule: Unauthorized commands are rejected

    Scenario: User not in allowed list
      Given a user not in the allowed users list
      When they comment "/heimdall status"
      And Heimdall polls GitHub
      Then the command should be rejected
      And no workflow should be triggered

    Scenario: Agent not in allowed list
      Given the repository does not allow agent "unauthorized-agent"
      When the user comments "/opsx-apply --agent unauthorized-agent"
      And Heimdall polls GitHub
      Then the command should be rejected
      And Heimdall should comment that the agent is not authorized

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

  Rule: Label-scoped polling ignores unlabeled pull requests

    Scenario: Unlabeled Heimdall pull request is ignored when monitor label is configured
      Given a Heimdall-managed pull request exists
      And the repository configures PR monitor label "heimdall-monitored"
      When the user comments "/heimdall status"
      And Heimdall polls GitHub
      Then Heimdall should ignore the pull request because it is missing monitor label
