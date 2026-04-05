Feature: Linear polling
  As an operator
  I want Symphony to poll Linear through the GraphQL API
  So that project-scoped issue activation can trigger proposal workflows without webhooks

  Background:
    Given Symphony is configured with a Linear project and GitHub repository

  Rule: Project-scoped polling succeeds when the API responds normally

    Scenario: Successful project-scoped poll
      Given a Linear GraphQL server with a valid project-scoped response
      When Symphony polls Linear through the board provider
      Then the Linear poll should succeed
      And the board provider should scope requests to the configured project name
      And the board provider should load the project-scoped issue "ENG-123"
      And the Linear poll checkpoint should be persisted

    Scenario: Multi-page project-scoped poll
      Given a Linear GraphQL server with multiple pages of project issues
      When Symphony polls Linear through the board provider
      Then the Linear poll should succeed
      And the board provider should load 2 project-scoped issues

  Rule: Poll failures do not advance checkpoints

    Scenario: Invalid API key fails safely
      Given a Linear GraphQL server that rejects the API key
      And an existing Linear poll checkpoint
      When Symphony polls Linear through the board provider
      Then the Linear poll should fail with "authentication"
      And the Linear poll checkpoint should remain unchanged

    Scenario: Rate-limited polling fails safely
      Given a Linear GraphQL server with a rate-limited response
      And an existing Linear poll checkpoint
      When Symphony polls Linear through the board provider
      Then the Linear poll should fail with "rate limited"
      And the Linear poll checkpoint should remain unchanged

  Rule: Active-state transitions are emitted once

    Scenario: Active-state transition is detected without duplication
      Given a Linear GraphQL server with a valid project-scoped response
      And an existing inactive snapshot for issue "ENG-123"
      When Symphony polls Linear through the board provider
      Then the board provider should emit an entered_active_state event
      When Symphony processes the same Linear poll result again
      Then the board provider should not emit a duplicate entered_active_state event
