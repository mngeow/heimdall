# Service: Behavior Testing

## ADDED Requirements

### Requirement: Critical Heimdall workflows have executable behavior coverage
Heimdall MUST maintain executable behavior tests for the critical operator workflows: proposal creation from an activated work item, pull request refinement, agent-driven apply, unauthorized command rejection, and duplicate event safety.

#### Scenario: The behavior suite is run for a release candidate
- **WHEN** the Heimdall behavior test suite is executed
- **THEN** it includes coverage for activated-work-item proposal creation, pull request refine, pull request apply, authorization denial, and duplicate event handling
- **AND** failures in those scenarios block a release candidate from being considered behaviorally verified

### Requirement: Behavior tests are written in Gherkin feature files
Heimdall behavior tests MUST be written in Gherkin `.feature` files, with exactly one `Feature` block per file and scenario names written in domain language.

#### Scenario: A new behavior area is added to the test suite
- **WHEN** a new end-to-end Heimdall behavior area is introduced
- **THEN** it is described in its own `.feature` file with a single `Feature` block
- **AND** related business rules may be grouped with `Rule` blocks instead of mixing unrelated behaviors into one feature file

### Requirement: Gherkin scenarios use Given, When, and Then correctly
Heimdall behavior tests MUST use `Given` for initial context, `When` for actions or events, and `Then` for observable outcomes, with `And` and `But` used only as readable continuations.

#### Scenario: A workflow scenario is authored in Gherkin
- **WHEN** an author writes a Gherkin scenario for a Heimdall workflow
- **THEN** the preconditions are expressed as `Given` steps
- **AND** the triggering action or event is expressed as a `When` step
- **AND** the expected user-visible or externally observable outcome is expressed as one or more `Then` steps

### Requirement: Gherkin scenarios focus on observable behavior
Heimdall behavior tests MUST assert observable outcomes such as branches, pull requests, comments, statuses, commits, or audit-visible results rather than depending solely on hidden internal state.

#### Scenario: A scenario verifies a rejected pull request command
- **WHEN** a behavior test checks how Heimdall handles an unauthorized apply request
- **THEN** the scenario asserts an observable rejection outcome such as no workflow mutation, an audit-visible denial, or a user-visible status response
- **AND** it does not rely only on direct database inspection as the proof of correct behavior

### Requirement: Scenario structure uses Gherkin data features appropriately
Heimdall behavior tests MUST use `Background` only for shared context, MUST use `Scenario Outline` with `Examples` for data permutations, and MUST keep scenarios concise enough to remain readable as executable specifications.

#### Scenario: The same command behavior must be tested across multiple agent values
- **WHEN** a behavior author needs to cover the same workflow against several allowed and disallowed agent selections
- **THEN** the behavior is expressed as a `Scenario Outline` with `Examples`
- **AND** the test suite does not duplicate nearly identical scenarios solely to vary table-driven inputs

### Requirement: Gherkin behavior tests remain executable from the Go codebase
Heimdall MUST bind its Gherkin feature files to a Go-compatible test runner so behavior scenarios execute as part of the Go-based test suite or equivalent CI workflow.

#### Scenario: A new feature file is added for pull request refine behavior
- **WHEN** a new Gherkin feature file is introduced for Heimdall behavior coverage
- **THEN** the Go test suite includes the step bindings required to execute that feature file
- **AND** the behavior can run in automated verification without a separate non-Go test harness becoming the primary source of truth
