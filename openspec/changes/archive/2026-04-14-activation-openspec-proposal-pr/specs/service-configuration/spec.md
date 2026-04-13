## ADDED Requirements

### Requirement: Repository configuration declares a default spec-writing agent
Heimdall MUST require `HEIMDALL_REPO_<ID>_DEFAULT_SPEC_WRITING_AGENT` for each managed repository. Heimdall MUST use that value as the default OpenCode agent for activation-triggered proposal generation and `/heimdall refine`, and Heimdall MUST reject empty or whitespace-only values.

#### Scenario: Repository config declares a default spec-writing agent
- **WHEN** Heimdall loads a repository block that includes `HEIMDALL_REPO_PLATFORM_DEFAULT_SPEC_WRITING_AGENT=gpt-5.4`
- **THEN** it stores `gpt-5.4` as that repository's default spec-writing agent
- **AND** activation proposal and refine workflows use that agent when they invoke local OpenCode execution

#### Scenario: Repository config omits the default spec-writing agent
- **WHEN** Heimdall loads a managed repository block that does not define `HEIMDALL_REPO_PLATFORM_DEFAULT_SPEC_WRITING_AGENT`
- **THEN** it does not report ready
- **AND** it emits a validation error for the missing repository agent setting

#### Scenario: Repository config declares an empty default spec-writing agent
- **WHEN** Heimdall loads a repository block where `HEIMDALL_REPO_PLATFORM_DEFAULT_SPEC_WRITING_AGENT` is empty or whitespace only
- **THEN** it does not report ready
- **AND** it emits a validation error for that repository setting
