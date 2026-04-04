Feature: Pull request command handling
  As an operator
  I want to control Symphony workflows through PR comments
  So that I can refine specs and trigger implementation

  Background:
    Given a Symphony-managed pull request exists
    And the PR author is in the allowed users list

  Rule: Authorized users can trigger commands

    Scenario: Status command
      Given a Symphony-managed pull request exists
      When the user comments "/symphony status"
      Then Symphony should reply with the current proposal status

    Scenario: Refine command
      Given a Symphony-managed pull request exists
      When the user comments "/symphony refine Add error handling section"
      Then Symphony should update the proposal artifacts
      And Symphony should commit the changes
      And Symphony should push the updated branch

    Scenario: Apply command with allowed agent
      Given a Symphony-managed pull request exists
      And the repository allows agent "gpt-5.4"
      When the user comments "/opsx-apply --agent gpt-5.4"
      Then Symphony should execute the apply workflow
      And Symphony should commit implementation changes
      And Symphony should comment with the execution results

  Rule: Unauthorized commands are rejected

    Scenario: User not in allowed list
      Given a user not in the allowed users list
      When they comment "/symphony status"
      Then the command should be rejected
      And no workflow should be triggered

    Scenario: Agent not in allowed list
      Given the repository does not allow agent "unauthorized-agent"
      When the user comments "/opsx-apply --agent unauthorized-agent"
      Then the command should be rejected
      And Symphony should comment that the agent is not authorized

  Rule: Duplicate commands are safe

    Scenario: Same comment delivered twice
      Given a command has already been processed
      When the same comment is delivered again
      Then the duplicate should be detected
      And the command should not be executed again

    Scenario: Comment is edited
      Given a command comment exists
      When the comment is edited
      Then the edit should not trigger a new command execution
