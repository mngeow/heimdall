Feature: Pull request command handling
  As an operator
  I want to control Symphony workflows through PR comments
  So that I can refine specs and trigger implementation

  Background:
    Given Symphony is configured with GitHub polling
    Given a Symphony-managed pull request exists
    And the PR author is in the allowed users list

  Rule: Authorized users can trigger commands discovered during polling

    Scenario: Status command
      Given a Symphony-managed pull request exists
      When the user comments "/symphony status"
      And Symphony polls GitHub
      Then Symphony should discover the comment during polling
      And Symphony should reply with the current proposal status

    Scenario: Refine command
      Given a Symphony-managed pull request exists
      When the user comments "/symphony refine Add error handling section"
      And Symphony polls GitHub
      Then Symphony should discover the comment during polling
      And Symphony should update the proposal artifacts
      And Symphony should commit the changes
      And Symphony should push the updated branch

    Scenario: Apply command with allowed agent
      Given a Symphony-managed pull request exists
      And the repository allows agent "gpt-5.4"
      When the user comments "/opsx-apply --agent gpt-5.4"
      And Symphony polls GitHub
      Then Symphony should discover the comment during polling
      And Symphony should execute the apply workflow
      And Symphony should commit implementation changes
      And Symphony should comment with the execution results

  Rule: Unauthorized commands are rejected

    Scenario: User not in allowed list
      Given a user not in the allowed users list
      When they comment "/symphony status"
      And Symphony polls GitHub
      Then the command should be rejected
      And no workflow should be triggered

    Scenario: Agent not in allowed list
      Given the repository does not allow agent "unauthorized-agent"
      When the user comments "/opsx-apply --agent unauthorized-agent"
      And Symphony polls GitHub
      Then the command should be rejected
      And Symphony should comment that the agent is not authorized

  Rule: Duplicate commands are safe

    Scenario: Same comment observed twice across polling windows
      Given a command has already been processed
      When the same comment is observed in another GitHub poll
      Then the duplicate should be detected
      And the command should not be executed again

    Scenario: Comment is edited after initial discovery
      Given a command comment exists
      When Symphony polls an edited version of the same comment
      Then the edit should not trigger a new command execution
