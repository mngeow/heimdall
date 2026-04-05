## ADDED Requirements

### Requirement: Repository configuration can declare a GitHub PR monitor label
Symphony MUST support an optional `SYMPHONY_REPO_<ID>_PR_MONITOR_LABEL` setting for each managed repository. When present, Symphony MUST treat that value as the GitHub label name used to mark and narrow monitoring for that repository's Symphony pull requests, and it MUST reject empty or whitespace-only values.

#### Scenario: Repository config declares a PR monitor label
- **WHEN** Symphony loads a repository block that includes `SYMPHONY_REPO_PLATFORM_PR_MONITOR_LABEL=symphony-monitored`
- **THEN** it stores `symphony-monitored` as that repository's GitHub PR monitor label
- **AND** the GitHub adapter uses that label name for PR reconciliation and polling filters in that repository

#### Scenario: Repository config omits a PR monitor label
- **WHEN** Symphony loads a managed repository block that does not define `SYMPHONY_REPO_PLATFORM_PR_MONITOR_LABEL`
- **THEN** it accepts that repository configuration
- **AND** GitHub PR monitoring for that repository continues without label-based filtering

#### Scenario: Repository config declares an empty PR monitor label
- **WHEN** Symphony loads a repository block where `SYMPHONY_REPO_PLATFORM_PR_MONITOR_LABEL` is empty or whitespace only
- **THEN** it does not report ready
- **AND** it emits a validation error for that repository setting
