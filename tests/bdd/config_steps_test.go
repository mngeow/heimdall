package bdd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cucumber/godog"
	"github.com/mngeow/symphony/internal/config"
	"github.com/mngeow/symphony/internal/workflow"
)

func registerConfigurationSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a project root with a valid Symphony \.env file$`, projectRootWithValidSymphonyDotenv)
	sc.Step(`^a project root with only a legacy Symphony YAML config$`, projectRootWithLegacyYAMLOnly)
	sc.Step(`^a project root with multi-repository Symphony \.env configuration$`, projectRootWithMultiRepositorySymphonyDotenv)
	sc.Step(`^a project root with an invalid Symphony \.env file$`, projectRootWithInvalidSymphonyDotenv)
	sc.Step(`^the environment overrides "([^"]*)" with "([^"]*)"$`, environmentOverridesValue)
	sc.Step(`^Symphony loads configuration from that project root$`, symphonyLoadsConfigurationFromProjectRoot)
	sc.Step(`^configuration loading should succeed$`, configurationLoadingShouldSucceed)
	sc.Step(`^configuration loading should fail with "([^"]*)"$`, configurationLoadingShouldFailWith)
	sc.Step(`^the loaded configuration should include repository "([^"]*)"$`, loadedConfigurationShouldIncludeRepository)
	sc.Step(`^repository routing for team "([^"]*)" should resolve to "([^"]*)"$`, repositoryRoutingShouldResolveTo)
	sc.Step(`^the loaded GitHub base branch should be "([^"]*)"$`, loadedGitHubBaseBranchShouldBe)
}

func snapshotSymphonyEnv() map[string]envState {
	snapshot := make(map[string]envState)
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if !found || !strings.HasPrefix(key, "SYMPHONY_") {
			continue
		}
		snapshot[key] = envState{value: value, present: true}
		_ = os.Unsetenv(key)
	}
	return snapshot
}

func restoreSymphonyEnv(snapshot map[string]envState) {
	for _, entry := range os.Environ() {
		key, _, found := strings.Cut(entry, "=")
		if !found || !strings.HasPrefix(key, "SYMPHONY_") {
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

func projectRootWithValidSymphonyDotenv(ctx context.Context) error {
	return writeProjectFile(getTC(ctx), ".env", validConfigurationDotenv("main"))
}

func projectRootWithLegacyYAMLOnly(ctx context.Context) error {
	return writeProjectFile(getTC(ctx), "config.yml", "server:\n  listen_address: ':8080'\n")
}

func projectRootWithMultiRepositorySymphonyDotenv(ctx context.Context) error {
	return writeProjectFile(getTC(ctx), ".env", multiRepositoryDotenv())
}

func projectRootWithInvalidSymphonyDotenv(ctx context.Context) error {
	invalid := strings.ReplaceAll(validConfigurationDotenv("main"), "SYMPHONY_GITHUB_LOOKBACK_WINDOW=2m", "SYMPHONY_GITHUB_LOOKBACK_WINDOW=0s")
	return writeProjectFile(getTC(ctx), ".env", invalid)
}

func environmentOverridesValue(ctx context.Context, key, value string) error {
	return os.Setenv(key, value)
}

func symphonyLoadsConfigurationFromProjectRoot(ctx context.Context) error {
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
		projectRoot, err := os.MkdirTemp("", "symphony-config-*")
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
		"SYMPHONY_SERVER_LISTEN_ADDRESS=:8080",
		"SYMPHONY_SERVER_PUBLIC_URL=http://127.0.0.1:8080",
		"SYMPHONY_STORAGE_DRIVER=sqlite",
		"SYMPHONY_STORAGE_DSN=/tmp/symphony.db",
		"SYMPHONY_LINEAR_POLL_INTERVAL=30s",
		"SYMPHONY_LINEAR_ACTIVE_STATES=In Progress",
		"SYMPHONY_LINEAR_TEAM_KEYS=ENG",
		"SYMPHONY_LINEAR_API_TOKEN=linear-token",
		"SYMPHONY_GITHUB_BASE_BRANCH=" + baseBranch,
		"SYMPHONY_GITHUB_POLL_INTERVAL=30s",
		"SYMPHONY_GITHUB_LOOKBACK_WINDOW=2m",
		"SYMPHONY_GITHUB_APP_ID=12345",
		"SYMPHONY_GITHUB_INSTALLATION_ID=99",
		"SYMPHONY_GITHUB_PRIVATE_KEY=test-private-key",
		"SYMPHONY_REPOS=PLATFORM",
		"SYMPHONY_REPO_PLATFORM_NAME=github.com/acme/platform",
		"SYMPHONY_REPO_PLATFORM_LOCAL_MIRROR_PATH=/var/lib/symphony/repos/github.com/acme/platform.git",
		"SYMPHONY_REPO_PLATFORM_DEFAULT_BRANCH=main",
		"SYMPHONY_REPO_PLATFORM_BRANCH_PREFIX=symphony",
		"SYMPHONY_REPO_PLATFORM_LINEAR_TEAM_KEYS=ENG",
		"SYMPHONY_REPO_PLATFORM_ALLOWED_AGENTS=gpt-5.4,claude-sonnet",
		"SYMPHONY_REPO_PLATFORM_ALLOWED_USERS=mngeow",
	}, "\n") + "\n"
}

func multiRepositoryDotenv() string {
	return strings.Join([]string{
		"SYMPHONY_SERVER_LISTEN_ADDRESS=:8080",
		"SYMPHONY_STORAGE_DRIVER=sqlite",
		"SYMPHONY_STORAGE_DSN=/tmp/symphony.db",
		"SYMPHONY_LINEAR_POLL_INTERVAL=30s",
		"SYMPHONY_LINEAR_ACTIVE_STATES=In Progress",
		"SYMPHONY_LINEAR_TEAM_KEYS=ENG,MOBILE",
		"SYMPHONY_LINEAR_API_TOKEN=linear-token",
		"SYMPHONY_GITHUB_BASE_BRANCH=main",
		"SYMPHONY_GITHUB_POLL_INTERVAL=30s",
		"SYMPHONY_GITHUB_LOOKBACK_WINDOW=2m",
		"SYMPHONY_GITHUB_APP_ID=12345",
		"SYMPHONY_GITHUB_INSTALLATION_ID=99",
		"SYMPHONY_GITHUB_PRIVATE_KEY=test-private-key",
		"SYMPHONY_REPOS=PLATFORM,MOBILE",
		"SYMPHONY_REPO_PLATFORM_NAME=github.com/acme/platform",
		"SYMPHONY_REPO_PLATFORM_LOCAL_MIRROR_PATH=/var/lib/symphony/repos/github.com/acme/platform.git",
		"SYMPHONY_REPO_PLATFORM_DEFAULT_BRANCH=main",
		"SYMPHONY_REPO_PLATFORM_BRANCH_PREFIX=symphony",
		"SYMPHONY_REPO_PLATFORM_LINEAR_TEAM_KEYS=ENG",
		"SYMPHONY_REPO_PLATFORM_ALLOWED_AGENTS=gpt-5.4,claude-sonnet",
		"SYMPHONY_REPO_PLATFORM_ALLOWED_USERS=mngeow",
		"SYMPHONY_REPO_MOBILE_NAME=github.com/acme/mobile",
		"SYMPHONY_REPO_MOBILE_LOCAL_MIRROR_PATH=/var/lib/symphony/repos/github.com/acme/mobile.git",
		"SYMPHONY_REPO_MOBILE_DEFAULT_BRANCH=main",
		"SYMPHONY_REPO_MOBILE_BRANCH_PREFIX=symphony",
		"SYMPHONY_REPO_MOBILE_LINEAR_TEAM_KEYS=MOBILE",
		"SYMPHONY_REPO_MOBILE_ALLOWED_AGENTS=gpt-5.4",
		"SYMPHONY_REPO_MOBILE_ALLOWED_USERS=mngeow",
	}, "\n") + "\n"
}
