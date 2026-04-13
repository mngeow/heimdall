Feature: Security and authorization
  As an operator
  I want Heimdall to enforce security boundaries
  So that unauthorized users cannot trigger workflows

  Background:
    Given Heimdall is running with security configuration

  Rule: Commands are only accepted on Heimdall-managed PRs

    Scenario: Command on non-Heimdall PR
      Given a pull request not created by Heimdall
      When a user comments "/heimdall status"
      And Heimdall polls GitHub
      Then the command should be rejected
      And Heimdall should record that the PR is not eligible

  Rule: Polling does not require a public webhook endpoint

    Scenario: Managed PR command is discovered without webhook delivery
      Given Heimdall runs without a public GitHub webhook endpoint
      And a Heimdall-managed pull request exists
      When a user comments "/heimdall status"
      And Heimdall polls GitHub
      Then Heimdall should discover the comment during polling
      And the command-intake path should not require a public webhook endpoint

  Rule: Secrets are not exposed

    Scenario: GitHub token handling
      Given Heimdall uses a GitHub App
      When installation tokens are minted
      Then tokens should not appear in logs
      And tokens should not be stored in SQLite
