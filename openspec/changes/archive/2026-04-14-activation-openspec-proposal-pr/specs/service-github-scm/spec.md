## MODIFIED Requirements

### Requirement: GitHub integration uses a GitHub App
Heimdall MUST use a GitHub App for GitHub API and git push operations and MUST rely on short-lived installation tokens for repository mutations.

#### Scenario: Heimdall needs to push an activation proposal branch
- **WHEN** Heimdall needs to push an activation-triggered OpenSpec proposal branch to GitHub
- **THEN** it mints a short-lived installation token from its GitHub App configuration
- **AND** it uses that token for the branch push instead of a long-lived personal access token

### Requirement: GitHub repository operations support the automation lifecycle
The GitHub SCM service MUST create or reuse branches, open or reuse pull requests, apply the configured PR monitor label when present, and publish pull request comments for Heimdall workflows.

#### Scenario: Heimdall publishes an activation proposal pull request
- **WHEN** an activation-triggered OpenSpec proposal workflow succeeds for a mapped repository
- **THEN** the GitHub SCM service ensures the proposal branch exists in the repository
- **AND** it opens or reuses a pull request against the configured base branch with the proposal title and description derived from the source issue and generated change
- **AND** it applies the configured PR monitor label when the repository declares one
