Feature: Runtime configuration
  As an operator
  I want Symphony to load configuration from environment variables and a project-root .env file
  So that deployment stays simple without a YAML runtime config

  Rule: Symphony reads dotenv-based runtime configuration

    Scenario: Service starts with a valid project-root .env file
      Given a project root with a valid Symphony .env file
      When Symphony loads configuration from that project root
      Then configuration loading should succeed
      And the loaded configuration should include repository "github.com/acme/platform"

    Scenario: Legacy YAML configuration is rejected
      Given a project root with only a legacy Symphony YAML config
      When Symphony loads configuration from that project root
      Then configuration loading should fail with "legacy"

    Scenario: Multiple repositories route explicitly from dotenv configuration
      Given a project root with multi-repository Symphony .env configuration
      When Symphony loads configuration from that project root
      Then configuration loading should succeed
      And repository routing for team "ENG" should resolve to "github.com/acme/platform"

    Scenario: Environment variables override .env values
      Given a project root with a valid Symphony .env file
      And the environment overrides "SYMPHONY_GITHUB_BASE_BRANCH" with "release"
      When Symphony loads configuration from that project root
      Then configuration loading should succeed
      And the loaded GitHub base branch should be "release"

    Scenario: Invalid dotenv values fail validation before readiness
      Given a project root with an invalid Symphony .env file
      When Symphony loads configuration from that project root
      Then configuration loading should fail with "SYMPHONY_GITHUB_LOOKBACK_WINDOW"

    Scenario: Linear project name is required for polling
      Given a project root with a Symphony .env file missing the Linear project name
      When Symphony loads configuration from that project root
      Then configuration loading should fail with "SYMPHONY_LINEAR_PROJECT_NAME"
