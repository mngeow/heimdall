Feature: Runtime configuration
  As an operator
  I want Heimdall to load configuration from environment variables and a project-root .env file
  So that deployment stays simple without a YAML runtime config

  Rule: Heimdall reads dotenv-based runtime configuration

    Scenario: Service starts with a valid project-root .env file
      Given a project root with a valid Heimdall .env file
      When Heimdall loads configuration from that project root
      Then configuration loading should succeed
      And the loaded configuration should include repository "github.com/acme/platform"

    Scenario: Legacy YAML configuration is rejected
      Given a project root with only a legacy Heimdall YAML config
      When Heimdall loads configuration from that project root
      Then configuration loading should fail with "legacy"

    Scenario: Multiple repositories route explicitly from dotenv configuration
      Given a project root with multi-repository Heimdall .env configuration
      When Heimdall loads configuration from that project root
      Then configuration loading should succeed
      And repository routing for team "ENG" should resolve to "github.com/acme/platform"

    Scenario: Environment variables override .env values
      Given a project root with a valid Heimdall .env file
      And the environment overrides "HEIMDALL_GITHUB_BASE_BRANCH" with "release"
      When Heimdall loads configuration from that project root
      Then configuration loading should succeed
      And the loaded GitHub base branch should be "release"

    Scenario: Invalid dotenv values fail validation before readiness
      Given a project root with an invalid Heimdall .env file
      When Heimdall loads configuration from that project root
      Then configuration loading should fail with "HEIMDALL_GITHUB_LOOKBACK_WINDOW"

    Scenario: Linear project name is required for polling
      Given a project root with a Heimdall .env file missing the Linear project name
      When Heimdall loads configuration from that project root
      Then configuration loading should fail with "HEIMDALL_LINEAR_PROJECT_NAME"

    Scenario: Repository config can declare a PR monitor label
      Given a project root with a valid Heimdall .env file
      And the environment overrides "HEIMDALL_REPO_PLATFORM_PR_MONITOR_LABEL" with "heimdall-monitored"
      When Heimdall loads configuration from that project root
      Then configuration loading should succeed
      And the loaded repository "github.com/acme/platform" should use PR monitor label "heimdall-monitored"

    Scenario: Empty PR monitor label is rejected when set
      Given a project root with a valid Heimdall .env file
      And the environment overrides "HEIMDALL_REPO_PLATFORM_PR_MONITOR_LABEL" with "   "
      When Heimdall loads configuration from that project root
      Then configuration loading should fail with "HEIMDALL_REPO_PLATFORM_PR_MONITOR_LABEL"
