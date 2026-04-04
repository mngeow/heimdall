Feature: Security and authorization
  As an operator
  I want Symphony to enforce security boundaries
  So that unauthorized users cannot trigger workflows

  Background:
    Given Symphony is running with security configuration

  Rule: Commands are only accepted on Symphony-managed PRs

    Scenario: Command on non-Symphony PR
      Given a pull request not created by Symphony
      When a user comments "/symphony status"
      And Symphony polls GitHub
      Then the command should be rejected
      And Symphony should record that the PR is not eligible

  Rule: Polling does not require a public webhook endpoint

    Scenario: Managed PR command is discovered without webhook delivery
      Given Symphony runs without a public GitHub webhook endpoint
      And a Symphony-managed pull request exists
      When a user comments "/symphony status"
      And Symphony polls GitHub
      Then Symphony should discover the comment during polling
      And the command-intake path should not require a public webhook endpoint

  Rule: Secrets are not exposed

    Scenario: GitHub token handling
      Given Symphony uses a GitHub App
      When installation tokens are minted
      Then tokens should not appear in logs
      And tokens should not be stored in SQLite
