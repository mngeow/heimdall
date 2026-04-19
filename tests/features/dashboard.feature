Feature: Private operator dashboard
  As an operator
  I want a read-only dashboard inside the Heimdall service
  So that I can inspect queued work items, active pull requests, and tracked activity without querying SQLite directly

  Background:
    Given Heimdall is configured with a Linear team and GitHub repository
    And the dashboard is available on the operator HTTP surface

  Rule: Dashboard overview shows summary counts

    Scenario: Operator opens the overview
      Given a Linear issue "ENG-123" with title "Add rate limiting" exists
      And the issue is in state "In Progress"
      And an active Heimdall-managed pull request exists for the issue
      When the operator requests the dashboard overview
      Then the response should be a server-rendered HTML page
      And the overview should show at least one tracked work item
      And the overview should show at least one active pull request

  Rule: Dashboard work-item queue lists all tracked statuses

    Scenario: Operator views the work-item queue
      Given a Linear issue "ENG-123" with title "Add rate limiting" exists
      And the issue is in state "In Progress"
      When the operator requests the work-item queue
      Then the response should be a server-rendered HTML page
      And the queue should list the work item with key "ENG-123"
      And the queue should show the work item status "In Progress"
      And the queue should show the work item lifecycle bucket

    Scenario: Operator filters the work-item queue
      Given a Linear issue "ENG-123" with title "Add rate limiting" exists
      And the issue is in state "In Progress"
      When the operator requests the work-item queue filtered by status "In Progress"
      Then the queue should list the work item with key "ENG-123"
      And the response should not trigger any repository mutation

  Rule: Dashboard shows active pull requests and PR detail

    Scenario: Operator views active pull requests
      Given a Linear issue "ENG-123" with title "Add rate limiting" exists
      And an active Heimdall-managed pull request exists for the issue
      When the operator requests the active pull-request list
      Then the response should be a server-rendered HTML page
      And the list should include the pull request for "ENG-123"
      And the list should link to the pull-request detail page

    Scenario: Operator opens a pull-request detail view
      Given a Linear issue "ENG-123" with title "Add rate limiting" exists
      And an active Heimdall-managed pull request exists for the issue
      And a command request was recorded for the pull request
      When the operator requests the pull-request detail view
      Then the response should be a server-rendered HTML page
      And the detail view should show the linked work item
      And the detail view should label the timeline as Heimdall-tracked command/activity history
      And the detail view should include a link to the canonical GitHub pull request
      And the rendered page should not contain secrets or raw sensitive payloads

  Rule: Dashboard interactions remain read-only

    Scenario: Dashboard filter refresh does not mutate state
      Given a Linear issue "ENG-123" with title "Add rate limiting" exists
      When the operator requests a filtered queue refresh via HTMX
      Then the response should contain an HTML fragment
      And no new workflow run should be created
      And no repository mutation should occur

  Rule: Dashboard shows active command runs and live output

    Scenario: Operator views active command runs
      Given a Heimdall-managed pull request exists
      And the repository allows agent "gpt-5.4"
      And a command run exists for an accepted opencode-backed command
      When the operator requests the active command-run list
      Then the response should be a server-rendered HTML page
      And the list should include the command run with status "queued"
      And the list should link to the command run detail page

    Scenario: Operator opens a command run detail view
      Given a Heimdall-managed pull request exists
      And the repository allows agent "gpt-5.4"
      And a command run exists for an accepted opencode-backed command
      And the command run has timeline entries
      When the operator requests the command run detail view
      Then the response should be a server-rendered HTML page
      And the detail view should show the command run status
      And the detail view should show the live output timeline
      And the rendered page should not contain secrets or raw sensitive payloads
