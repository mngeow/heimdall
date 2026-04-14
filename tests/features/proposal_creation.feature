Feature: OpenSpec proposal pull request creation from activated work item
  As an operator
  I want Heimdall to automatically create OpenSpec proposal pull requests when Linear issues enter active state
  So that the activation-to-PR path creates reviewable OpenSpec changes instead of temporary bootstrap files

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
      And Heimdall should create a workflow run for proposal generation

    Scenario: Duplicate activation is idempotent
      Given a Linear issue "ENG-123" is already in state "In Progress"
      And a proposal branch already exists for the issue
      When Heimdall polls Linear again
      Then Heimdall should not create a duplicate workflow run
      And Heimdall should reuse the existing proposal

  Rule: Heimdall creates deterministic proposal branches from issue title

    Scenario: Proposal branch naming uses issue title
      Given a Linear issue "ENG-123" with title "Add rate limiting"
      When the issue enters active state
      Then the proposal branch should be named "heimdall/ENG-123-add-rate-limiting"

    Scenario: Proposal branch naming cleans special characters
      Given a Linear issue "ENG-123" with title "Feature: add rate limiting, please"
      When the issue enters active state
      Then the proposal branch should be named "heimdall/ENG-123-feature-add-rate-limiting-please"

  Rule: Heimdall creates pull requests for OpenSpec proposal changes

    Scenario: Proposal PR creation
      Given a Linear issue enters active state
      When Heimdall generates the OpenSpec proposal
      Then Heimdall should push the proposal branch
      And Heimdall should create a pull request to main
      And Heimdall should include the issue description in the proposal pull request body
      And the pull request title should indicate an OpenSpec proposal

    Scenario: Proposal PR is labeled for Heimdall monitoring
      Given a Linear issue enters active state
      And the repository configures PR monitor label "heimdall-monitored"
      When Heimdall generates the OpenSpec proposal
      Then Heimdall should create or reuse repository label "heimdall-monitored"
      And Heimdall should apply the monitor label "heimdall-monitored" to the proposal pull request

    Scenario: Proposal generation fails when no file changes are produced
      Given a Linear issue enters active state
      And the proposal execution produces no file changes
      When Heimdall generates the OpenSpec proposal
      Then Heimdall should mark the workflow run as blocked
      And Heimdall should record the no-change reason

    Scenario: Activation proposal logs expose workflow progress
      Given a Linear issue enters active state
      When Heimdall generates the OpenSpec proposal
      Then Heimdall should emit activation proposal logs with workflow step names
      And Heimdall should not log installation tokens or raw proposal prompts

  Rule: Heimdall discovers OpenSpec changes from CLI output

    Scenario: Proposal generation discovers change from OpenSpec list output
      Given a Linear issue enters active state
      And the target repository worktree has no existing OpenSpec changes
      When Heimdall generates the OpenSpec proposal
      And the proposal creates a new OpenSpec change "eng-123-add-rate-limiting"
      Then Heimdall should discover the new change from the OpenSpec list output
      And Heimdall should request apply instructions for the discovered change
      And Heimdall should persist the discovered change name in the repository binding

    Scenario: Proposal generation readiness check succeeds
      Given a Linear issue enters active state
      And the target repository worktree has no existing OpenSpec changes
      When Heimdall generates the OpenSpec proposal
      And the proposal creates a new OpenSpec change "eng-123-add-rate-limiting"
      And the apply instructions for the discovered change indicate state "ready"
      Then Heimdall should commit the proposal branch
      And Heimdall should push the proposal branch
      And Heimdall should create a pull request to main
