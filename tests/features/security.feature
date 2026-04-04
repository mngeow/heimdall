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
      Then the command should be rejected
      And Symphony should record that the PR is not eligible

  Rule: Webhook signatures are verified

    Scenario: Valid webhook signature
      Given a GitHub webhook delivery
      When the signature is valid
      Then the webhook should be processed

    Scenario: Invalid webhook signature
      Given a GitHub webhook delivery
      When the signature is invalid
      Then the webhook should be rejected
      And a 401 response should be returned

  Rule: Secrets are not exposed

    Scenario: GitHub token handling
      Given Symphony uses a GitHub App
      When installation tokens are minted
      Then tokens should not appear in logs
      And tokens should not be stored in SQLite
