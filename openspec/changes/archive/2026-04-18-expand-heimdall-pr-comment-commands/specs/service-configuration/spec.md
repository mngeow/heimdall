## MODIFIED Requirements

### Requirement: Repository configuration declares a default spec-writing agent
Heimdall MUST require `HEIMDALL_REPO_<ID>_DEFAULT_SPEC_WRITING_AGENT` for each managed repository. Heimdall MUST use that value as the default OpenCode agent for activation-triggered proposal generation only, and Heimdall MUST reject empty or whitespace-only values.

#### Scenario: Repository config declares a default spec-writing agent
- **WHEN** Heimdall loads a repository block that includes `HEIMDALL_REPO_PLATFORM_DEFAULT_SPEC_WRITING_AGENT=gpt-5.4`
- **THEN** it stores `gpt-5.4` as that repository's default spec-writing agent
- **AND** activation proposal workflows use that agent when they invoke local OpenCode execution

#### Scenario: Repository config omits the default spec-writing agent
- **WHEN** Heimdall loads a managed repository block that does not define `HEIMDALL_REPO_PLATFORM_DEFAULT_SPEC_WRITING_AGENT`
- **THEN** it does not report ready
- **AND** it emits a validation error for the missing repository agent setting

#### Scenario: Repository config declares an empty default spec-writing agent
- **WHEN** Heimdall loads a repository block where `HEIMDALL_REPO_PLATFORM_DEFAULT_SPEC_WRITING_AGENT` is empty or whitespace only
- **THEN** it does not report ready
- **AND** it emits a validation error for that repository setting

## ADDED Requirements

### Requirement: Repository configuration declares allowed PR-comment agents
Heimdall MUST require `HEIMDALL_REPO_<ID>_ALLOWED_AGENTS` for each managed repository that accepts PR-comment mutation commands. Heimdall MUST parse that setting as a comma-separated list of agent names, and it MUST reject missing, empty, or whitespace-only values.

#### Scenario: Repository config declares allowed PR-comment agents
- **WHEN** Heimdall loads a repository block that includes `HEIMDALL_REPO_PLATFORM_ALLOWED_AGENTS=gpt-5.4,claude-sonnet`
- **THEN** it stores `gpt-5.4` and `claude-sonnet` as that repository's allowed PR-comment agents
- **AND** `/heimdall refine`, `/heimdall apply`, `/opsx-apply`, and `/heimdall opencode` may select only from that set

#### Scenario: Repository config omits allowed PR-comment agents
- **WHEN** Heimdall loads a managed repository block that does not define `HEIMDALL_REPO_PLATFORM_ALLOWED_AGENTS`
- **THEN** it does not report ready
- **AND** it emits a validation error for the missing allowed-agent setting

#### Scenario: Repository config declares an empty allowed-agent list
- **WHEN** Heimdall loads a repository block where `HEIMDALL_REPO_PLATFORM_ALLOWED_AGENTS` is empty or whitespace only
- **THEN** it does not report ready
- **AND** it emits a validation error for that repository setting

### Requirement: Repository configuration can declare generic opencode command aliases
Heimdall MUST support an optional repository-scoped alias schema for `/heimdall opencode`. When aliases are configured, each alias MUST resolve to exactly one opencode command name and one permission profile, and Heimdall MUST reject empty aliases, empty command targets, duplicate aliases, or unknown permission-profile values.

#### Scenario: Repository config declares a generic opencode command alias
- **WHEN** Heimdall loads a repository block that includes `HEIMDALL_REPO_PLATFORM_OPENCODE_COMMANDS=explore-change`, `HEIMDALL_REPO_PLATFORM_OPENCODE_COMMAND_EXPLORE_CHANGE_COMMAND=opsx-explore`, and `HEIMDALL_REPO_PLATFORM_OPENCODE_COMMAND_EXPLORE_CHANGE_PERMISSION_PROFILE=readonly`
- **THEN** it stores `explore-change` as an allowed generic opencode alias for that repository
- **AND** `/heimdall opencode explore-change --agent gpt-5.4` resolves to the configured command and permission profile

#### Scenario: Repository config omits generic opencode command aliases
- **WHEN** Heimdall loads a managed repository block that does not define `HEIMDALL_REPO_PLATFORM_OPENCODE_COMMANDS`
- **THEN** it accepts that repository configuration
- **AND** `/heimdall opencode` remains unavailable for that repository until aliases are configured

#### Scenario: Repository config declares an invalid generic opencode alias mapping
- **WHEN** Heimdall loads a repository block where an opencode alias has an empty command target or an unknown permission profile
- **THEN** it does not report ready
- **AND** it emits a validation error for that alias mapping
