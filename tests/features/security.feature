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

  Rule: GitHub polling works without public ingress

    Scenario: New PR comments are discovered by polling
      Given a Symphony-managed pull request exists
      When Symphony polls GitHub for new pull request comments
      Then new command comments should be discovered for processing
      And Symphony should not require a public HTTPS endpoint

    Scenario: Pull request state changes are discovered by polling
      Given a Symphony-managed pull request exists
      When Symphony polls GitHub for pull request state changes
      Then state changes should be available for reconciliation

  Rule: Secrets are not exposed

    Scenario: GitHub token handling
      Given Symphony uses a GitHub App
      When installation tokens are minted
      Then tokens should not appear in logs
      And tokens should not be stored in SQLite
