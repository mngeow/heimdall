Feature: Proposal creation from activated work item
  As an operator
  I want Symphony to automatically create OpenSpec proposals when Linear issues enter active state
  So that engineering work can be prepared without manual setup

  Background:
    Given Symphony is configured with a Linear team and GitHub repository
    And the required local executables are available

  Rule: Symphony detects Linear issues entering active state

    Scenario: Linear issue moves to In Progress
      Given a Linear issue "ENG-123" with title "Add rate limiting" exists
      And the issue is in state "Todo"
      When the issue is moved to state "In Progress"
      And Symphony polls Linear
      Then Symphony should detect the state transition
      And Symphony should create a workflow run for proposal generation

    Scenario: Duplicate activation is idempotent
      Given a Linear issue "ENG-123" is already in state "In Progress"
      And a proposal branch already exists for the issue
      When Symphony polls Linear again
      Then Symphony should not create a duplicate workflow run
      And Symphony should reuse the existing proposal

  Rule: Symphony creates deterministic branches and changes

    Scenario: Proposal branch naming
      Given a Linear issue "ENG-123" with title "Add rate limiting"
      When the issue enters active state
      Then the proposal branch should be named "symphony/ENG-123-add-rate-limiting"
      And the OpenSpec change should be named "ENG-123-add-rate-limiting"

  Rule: Symphony creates pull requests for proposals

    Scenario: Proposal PR creation
      Given a Linear issue enters active state
      When Symphony generates the OpenSpec proposal
      Then Symphony should push the proposal branch
      And Symphony should create a pull request to main
      And Symphony should comment with the change name and available commands
