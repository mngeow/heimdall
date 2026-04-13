Feature: Bootstrap pull request creation from activated work item
  As an operator
  I want Heimdall to automatically create bootstrap pull requests when Linear issues enter active state
  So that the activation-to-PR path is proven before full OpenSpec proposal generation

  Background:
    Given Heimdall is configured with a Linear project and GitHub repository
    And the required local executables are available

  Rule: Heimdall detects Linear issues entering active state

    Scenario: Linear issue moves to In Progress
      Given a Linear issue "ENG-123" with title "Add rate limiting" and description "Add rate limiting for API requests" exists
      And the issue is in state "Todo"
      When the issue is moved to state "In Progress"
      And Heimdall polls Linear
      Then Heimdall should detect the state transition
      And Heimdall should create a workflow run for bootstrap pull request creation

    Scenario: Duplicate activation is idempotent
      Given a Linear issue "ENG-123" is already in state "In Progress"
      And a bootstrap branch already exists for the issue
      When Heimdall polls Linear again
      Then Heimdall should not create a duplicate workflow run
      And Heimdall should reuse the existing bootstrap pull request binding

  Rule: Heimdall creates deterministic bootstrap branches

    Scenario: Bootstrap branch naming uses issue description first
      Given a Linear issue "ENG-123" with title "Add rate limiting" and description "Add rate limiting for API requests"
      When the issue enters active state
      Then the bootstrap branch should be named "heimdall/ENG-123-add-rate-limiting-for-api-requests"

  Rule: Heimdall creates pull requests for bootstrap changes

    Scenario: Bootstrap PR creation
      Given a Linear issue enters active state
      When Heimdall generates the activation bootstrap pull request
      Then Heimdall should push the bootstrap branch
      And Heimdall should create or reuse a bootstrap pull request to main
      And Heimdall should include the issue description in the bootstrap pull request body

    Scenario: Bootstrap PR is labeled for Heimdall monitoring
      Given a Linear issue enters active state
      And the repository configures PR monitor label "heimdall-monitored"
      When Heimdall generates the activation bootstrap pull request
      Then Heimdall should create or reuse repository label "heimdall-monitored"
      And Heimdall should apply the monitor label "heimdall-monitored" to the bootstrap pull request

    Scenario: Bootstrap execution fails when no file changes are produced
      Given a Linear issue enters active state
      And the bootstrap execution produces no file changes
      When Heimdall generates the activation bootstrap pull request
      Then Heimdall should mark the workflow run as blocked
      And Heimdall should record the no-change reason

    Scenario: Activation bootstrap logs expose workflow progress
      Given a Linear issue enters active state
      When Heimdall generates the activation bootstrap pull request
      Then Heimdall should emit activation bootstrap logs with workflow step names
      And Heimdall should not log installation tokens or raw bootstrap prompts
