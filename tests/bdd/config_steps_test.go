package bdd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cucumber/godog"
	"github.com/mngeow/heimdall/internal/config"
	"github.com/mngeow/heimdall/internal/workflow"
)

func registerConfigurationSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a project root with a valid Heimdall \.env file$`, projectRootWithValidHeimdallDotenv)
	sc.Step(`^a project root with only a legacy Heimdall YAML config$`, projectRootWithLegacyYAMLOnly)
	sc.Step(`^a project root with multi-repository Heimdall \.env configuration$`, projectRootWithMultiRepositoryHeimdallDotenv)
	sc.Step(`^a project root with an invalid Heimdall \.env file$`, projectRootWithInvalidHeimdallDotenv)
	sc.Step(`^a project root with a Heimdall \.env file missing the Linear project name$`, projectRootWithMissingLinearProjectName)
	sc.Step(`^the environment overrides "([^"]*)" with "([^"]*)"$`, environmentOverridesValue)
	sc.Step(`^Heimdall loads configuration from that project root$`, heimdallLoadsConfigurationFromProjectRoot)
	sc.Step(`^configuration loading should succeed$`, configurationLoadingShouldSucceed)
	sc.Step(`^configuration loading should fail with "([^"]*)"$`, configurationLoadingShouldFailWith)
	sc.Step(`^the loaded configuration should include repository "([^"]*)"$`, loadedConfigurationShouldIncludeRepository)
	sc.Step(`^the loaded repository "([^"]*)" should use PR monitor label "([^"]*)"$`, loadedRepositoryShouldUsePRMonitorLabel)
	sc.Step(`^repository routing for team "([^"]*)" should resolve to "([^"]*)"$`, repositoryRoutingShouldResolveTo)
	sc.Step(`^the loaded GitHub base branch should be "([^"]*)"$`, loadedGitHubBaseBranchShouldBe)
}

func snapshotHeimdallEnv() map[string]envState {
	snapshot := make(map[string]envState)
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if !found || !strings.HasPrefix(key, "HEIMDALL_") {
			continue
		}
		snapshot[key] = envState{value: value, present: true}
		_ = os.Unsetenv(key)
	}
	return snapshot
}

func restoreHeimdallEnv(snapshot map[string]envState) {
	for _, entry := range os.Environ() {
		key, _, found := strings.Cut(entry, "=")
		if !found || !strings.HasPrefix(key, "HEIMDALL_") {
			continue
		}
		_ = os.Unsetenv(key)
	}

	for key, state := range snapshot {
		if state.present {
			_ = os.Setenv(key, state.value)
		}
	}
}

func projectRootWithValidHeimdallDotenv(ctx context.Context) error {
	return writeProjectFile(getTC(ctx), ".env", validConfigurationDotenv("main"))
}

func projectRootWithLegacyYAMLOnly(ctx context.Context) error {
	return writeProjectFile(getTC(ctx), "config.yml", "server:\n  listen_address: ':8080'\n")
}

func projectRootWithMultiRepositoryHeimdallDotenv(ctx context.Context) error {
	return writeProjectFile(getTC(ctx), ".env", multiRepositoryDotenv())
}

func projectRootWithInvalidHeimdallDotenv(ctx context.Context) error {
	invalid := strings.ReplaceAll(validConfigurationDotenv("main"), "HEIMDALL_GITHUB_LOOKBACK_WINDOW=2m", "HEIMDALL_GITHUB_LOOKBACK_WINDOW=0s")
	return writeProjectFile(getTC(ctx), ".env", invalid)
}

func projectRootWithMissingLinearProjectName(ctx context.Context) error {
	invalid := strings.ReplaceAll(validConfigurationDotenv("main"), "HEIMDALL_LINEAR_PROJECT_NAME=Core Platform\n", "")
	return writeProjectFile(getTC(ctx), ".env", invalid)
}

func environmentOverridesValue(ctx context.Context, key, value string) error {
	return os.Setenv(key, value)
}

func heimdallLoadsConfigurationFromProjectRoot(ctx context.Context) error {
	tc := getTC(ctx)
	tc.config, tc.configLoadErr = config.LoadFromDir(tc.projectRoot)
	return nil
}

func configurationLoadingShouldSucceed(ctx context.Context) error {
	tc := getTC(ctx)
	if tc.configLoadErr != nil {
		return fmt.Errorf("expected configuration load to succeed, got %v", tc.configLoadErr)
	}
	if tc.config == nil {
		return fmt.Errorf("expected loaded config, got nil")
	}
	return nil
}

func configurationLoadingShouldFailWith(ctx context.Context, message string) error {
	tc := getTC(ctx)
	if tc.configLoadErr == nil {
		return fmt.Errorf("expected configuration load to fail with %q", message)
	}
	if !strings.Contains(tc.configLoadErr.Error(), message) {
		return fmt.Errorf("expected configuration load error to contain %q, got %v", message, tc.configLoadErr)
	}
	return nil
}

func loadedConfigurationShouldIncludeRepository(ctx context.Context, repoRef string) error {
	tc := getTC(ctx)
	for _, repo := range tc.config.Repos {
		if repo.Name == repoRef {
			return nil
		}
	}
	return fmt.Errorf("expected repository %q in config, got %#v", repoRef, tc.config.Repos)
}

func loadedRepositoryShouldUsePRMonitorLabel(ctx context.Context, repoRef, label string) error {
	tc := getTC(ctx)
	for _, repo := range tc.config.Repos {
		if repo.Name != repoRef {
			continue
		}
		if repo.PRMonitorLabel != label {
			return fmt.Errorf("expected repository %q to use PR monitor label %q, got %q", repoRef, label, repo.PRMonitorLabel)
		}
		return nil
	}
	return fmt.Errorf("expected repository %q in config, got %#v", repoRef, tc.config.Repos)
}

func repositoryRoutingShouldResolveTo(ctx context.Context, teamKey, repoRef string) error {
	tc := getTC(ctx)
	result := workflow.NewRouter(tc.config.Repos).Resolve(teamKey)
	if !result.Matched {
		return fmt.Errorf("expected route for team %q, got unmatched result: %s", teamKey, result.Reason)
	}
	if result.Repository.Name != repoRef {
		return fmt.Errorf("expected repo %q, got %q", repoRef, result.Repository.Name)
	}
	return nil
}

func loadedGitHubBaseBranchShouldBe(ctx context.Context, branch string) error {
	tc := getTC(ctx)
	if tc.config == nil {
		return fmt.Errorf("expected loaded config, got nil")
	}
	if tc.config.GitHub.BaseBranch != branch {
		return fmt.Errorf("expected base branch %q, got %q", branch, tc.config.GitHub.BaseBranch)
	}
	return nil
}

func writeProjectFile(tc *testContext, relativePath, contents string) error {
	if tc.projectRoot == "" {
		projectRoot, err := os.MkdirTemp("", "heimdall-config-*")
		if err != nil {
			return err
		}
		tc.projectRoot = projectRoot
	}

	path := filepath.Join(tc.projectRoot, relativePath)
	return os.WriteFile(path, []byte(contents), 0o600)
}

func validConfigurationDotenv(baseBranch string) string {
	return strings.Join([]string{
		"HEIMDALL_SERVER_LISTEN_ADDRESS=:8080",
		"HEIMDALL_SERVER_PUBLIC_URL=http://127.0.0.1:8080",
		"HEIMDALL_STORAGE_DRIVER=sqlite",
		"HEIMDALL_STORAGE_DSN=/tmp/heimdall.db",
		"HEIMDALL_LINEAR_POLL_INTERVAL=30s",
		"HEIMDALL_LINEAR_ACTIVE_STATES=In Progress",
		"HEIMDALL_LINEAR_PROJECT_NAME=Core Platform",
		"HEIMDALL_LINEAR_API_TOKEN=linear-token",
		"HEIMDALL_GITHUB_BASE_BRANCH=" + baseBranch,
		"HEIMDALL_GITHUB_POLL_INTERVAL=30s",
		"HEIMDALL_GITHUB_LOOKBACK_WINDOW=2m",
		"HEIMDALL_GITHUB_APP_ID=12345",
		"HEIMDALL_GITHUB_INSTALLATION_ID=99",
		"HEIMDALL_GITHUB_PRIVATE_KEY=test-private-key",
		"HEIMDALL_REPOS=PLATFORM",
		"HEIMDALL_REPO_PLATFORM_NAME=github.com/acme/platform",
		"HEIMDALL_REPO_PLATFORM_LOCAL_MIRROR_PATH=/var/lib/heimdall/repos/github.com/acme/platform.git",
		"HEIMDALL_REPO_PLATFORM_DEFAULT_BRANCH=main",
		"HEIMDALL_REPO_PLATFORM_BRANCH_PREFIX=heimdall",
		"HEIMDALL_REPO_PLATFORM_LINEAR_TEAM_KEYS=ENG",
		"HEIMDALL_REPO_PLATFORM_ALLOWED_AGENTS=gpt-5.4,claude-sonnet",
		"HEIMDALL_REPO_PLATFORM_ALLOWED_USERS=mngeow",
		"HEIMDALL_REPO_PLATFORM_DEFAULT_SPEC_WRITING_AGENT=gpt-5.4",
	}, "\n") + "\n"
}

func multiRepositoryDotenv() string {
	return strings.Join([]string{
		"HEIMDALL_SERVER_LISTEN_ADDRESS=:8080",
		"HEIMDALL_STORAGE_DRIVER=sqlite",
		"HEIMDALL_STORAGE_DSN=/tmp/heimdall.db",
		"HEIMDALL_LINEAR_POLL_INTERVAL=30s",
		"HEIMDALL_LINEAR_ACTIVE_STATES=In Progress",
		"HEIMDALL_LINEAR_PROJECT_NAME=Core Platform",
		"HEIMDALL_LINEAR_API_TOKEN=linear-token",
		"HEIMDALL_GITHUB_BASE_BRANCH=main",
		"HEIMDALL_GITHUB_POLL_INTERVAL=30s",
		"HEIMDALL_GITHUB_LOOKBACK_WINDOW=2m",
		"HEIMDALL_GITHUB_APP_ID=12345",
		"HEIMDALL_GITHUB_INSTALLATION_ID=99",
		"HEIMDALL_GITHUB_PRIVATE_KEY=test-private-key",
		"HEIMDALL_REPOS=PLATFORM,MOBILE",
		"HEIMDALL_REPO_PLATFORM_NAME=github.com/acme/platform",
		"HEIMDALL_REPO_PLATFORM_LOCAL_MIRROR_PATH=/var/lib/heimdall/repos/github.com/acme/platform.git",
		"HEIMDALL_REPO_PLATFORM_DEFAULT_BRANCH=main",
		"HEIMDALL_REPO_PLATFORM_BRANCH_PREFIX=heimdall",
		"HEIMDALL_REPO_PLATFORM_LINEAR_TEAM_KEYS=ENG",
		"HEIMDALL_REPO_PLATFORM_ALLOWED_AGENTS=gpt-5.4,claude-sonnet",
		"HEIMDALL_REPO_PLATFORM_ALLOWED_USERS=mngeow",
		"HEIMDALL_REPO_PLATFORM_DEFAULT_SPEC_WRITING_AGENT=gpt-5.4",
		"HEIMDALL_REPO_MOBILE_NAME=github.com/acme/mobile",
		"HEIMDALL_REPO_MOBILE_LOCAL_MIRROR_PATH=/var/lib/heimdall/repos/github.com/acme/mobile.git",
		"HEIMDALL_REPO_MOBILE_DEFAULT_BRANCH=main",
		"HEIMDALL_REPO_MOBILE_BRANCH_PREFIX=heimdall",
		"HEIMDALL_REPO_MOBILE_LINEAR_TEAM_KEYS=MOBILE",
		"HEIMDALL_REPO_MOBILE_ALLOWED_AGENTS=gpt-5.4",
		"HEIMDALL_REPO_MOBILE_ALLOWED_USERS=mngeow",
		"HEIMDALL_REPO_MOBILE_DEFAULT_SPEC_WRITING_AGENT=gpt-5.4",
	}, "\n") + "\n"
}
