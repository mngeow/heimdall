package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromDirReadsProjectRootDotenv(t *testing.T) {
	clearHeimdallEnv(t)
	projectRoot := t.TempDir()

	writeTestFile(t, filepath.Join(projectRoot, ".env"), validDotenv("main"))

	cfg, err := LoadFromDir(projectRoot)
	if err != nil {
		t.Fatalf("LoadFromDir() error = %v", err)
	}

	if cfg.GitHub.BaseBranch != "main" {
		t.Fatalf("expected base branch main, got %q", cfg.GitHub.BaseBranch)
	}
	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Name != "github.com/acme/platform" {
		t.Fatalf("expected platform repo, got %q", cfg.Repos[0].Name)
	}
}

func TestLoadFromDirEnvironmentOverridesDotenv(t *testing.T) {
	clearHeimdallEnv(t)
	projectRoot := t.TempDir()

	writeTestFile(t, filepath.Join(projectRoot, ".env"), validDotenv("main"))
	t.Setenv("HEIMDALL_GITHUB_BASE_BRANCH", "release")

	cfg, err := LoadFromDir(projectRoot)
	if err != nil {
		t.Fatalf("LoadFromDir() error = %v", err)
	}

	if cfg.GitHub.BaseBranch != "release" {
		t.Fatalf("expected base branch release, got %q", cfg.GitHub.BaseBranch)
	}
}

func TestLoadFromDirSupportsEnvironmentOnly(t *testing.T) {
	clearHeimdallEnv(t)
	projectRoot := t.TempDir()

	for key, value := range validEnvMap("main") {
		t.Setenv(key, value)
	}

	cfg, err := LoadFromDir(projectRoot)
	if err != nil {
		t.Fatalf("LoadFromDir() error = %v", err)
	}

	if cfg.Linear.APIToken != "linear-token" {
		t.Fatalf("expected linear token from environment, got %q", cfg.Linear.APIToken)
	}
}

func TestLoadFromDirRejectsLegacyYAMLOnly(t *testing.T) {
	clearHeimdallEnv(t)
	projectRoot := t.TempDir()

	writeTestFile(t, filepath.Join(projectRoot, "config.yml"), "server:\n  listen_address: ':8080'\n")

	_, err := LoadFromDir(projectRoot)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "legacy") {
		t.Fatalf("expected legacy yaml error, got %v", err)
	}
}

func TestLoadFromDirReadsPrivateKeyFromFile(t *testing.T) {
	clearHeimdallEnv(t)
	projectRoot := t.TempDir()
	privateKeyPath := filepath.Join(projectRoot, "github-app.pem")
	writeTestFile(t, privateKeyPath, "test-private-key")
	writeTestFile(t, filepath.Join(projectRoot, ".env"), strings.ReplaceAll(validDotenv("main"), "HEIMDALL_GITHUB_PRIVATE_KEY=test-private-key", "HEIMDALL_GITHUB_PRIVATE_KEY_FILE="+privateKeyPath))

	cfg, err := LoadFromDir(projectRoot)
	if err != nil {
		t.Fatalf("LoadFromDir() error = %v", err)
	}
	if cfg.GitHub.PrivateKey != "test-private-key" {
		t.Fatalf("expected private key file contents, got %q", cfg.GitHub.PrivateKey)
	}
}

func TestLoadFromDirValidatesLookbackWindow(t *testing.T) {
	clearHeimdallEnv(t)
	projectRoot := t.TempDir()
	writeTestFile(t, filepath.Join(projectRoot, ".env"), strings.ReplaceAll(validDotenv("main"), "HEIMDALL_GITHUB_LOOKBACK_WINDOW=2m", "HEIMDALL_GITHUB_LOOKBACK_WINDOW=0s"))

	_, err := LoadFromDir(projectRoot)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "HEIMDALL_GITHUB_LOOKBACK_WINDOW") {
		t.Fatalf("expected lookback validation error, got %v", err)
	}
}

func TestLoadFromDirRequiresLinearProjectName(t *testing.T) {
	clearHeimdallEnv(t)
	projectRoot := t.TempDir()
	writeTestFile(t, filepath.Join(projectRoot, ".env"), strings.ReplaceAll(validDotenv("main"), "HEIMDALL_LINEAR_PROJECT_NAME=Core Platform\n", ""))

	_, err := LoadFromDir(projectRoot)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "HEIMDALL_LINEAR_PROJECT_NAME") {
		t.Fatalf("expected linear project name validation error, got %v", err)
	}
}

func TestLoadFromDirReadsRepoPRMonitorLabel(t *testing.T) {
	clearHeimdallEnv(t)
	projectRoot := t.TempDir()
	writeTestFile(t, filepath.Join(projectRoot, ".env"), validDotenvWithExtra("main", "HEIMDALL_REPO_PLATFORM_PR_MONITOR_LABEL=heimdall-monitored"))

	cfg, err := LoadFromDir(projectRoot)
	if err != nil {
		t.Fatalf("LoadFromDir() error = %v", err)
	}
	if got := cfg.Repos[0].PRMonitorLabel; got != "heimdall-monitored" {
		t.Fatalf("expected PR monitor label heimdall-monitored, got %q", got)
	}
}

func TestLoadFromDirRejectsEmptyRepoPRMonitorLabel(t *testing.T) {
	clearHeimdallEnv(t)
	projectRoot := t.TempDir()
	writeTestFile(t, filepath.Join(projectRoot, ".env"), validDotenvWithExtra("main", "HEIMDALL_REPO_PLATFORM_PR_MONITOR_LABEL=   "))

	_, err := LoadFromDir(projectRoot)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "HEIMDALL_REPO_PLATFORM_PR_MONITOR_LABEL") {
		t.Fatalf("expected PR monitor label validation error, got %v", err)
	}
}

func TestLoadFromDirRequiresDefaultSpecWritingAgent(t *testing.T) {
	clearHeimdallEnv(t)
	projectRoot := t.TempDir()
	writeTestFile(t, filepath.Join(projectRoot, ".env"), strings.ReplaceAll(validDotenv("main"), "HEIMDALL_REPO_PLATFORM_DEFAULT_SPEC_WRITING_AGENT=gpt-5.4\n", ""))

	_, err := LoadFromDir(projectRoot)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "HEIMDALL_REPO_PLATFORM_DEFAULT_SPEC_WRITING_AGENT") {
		t.Fatalf("expected default spec-writing agent validation error, got %v", err)
	}
}

func TestLoadFromDirRejectsEmptyDefaultSpecWritingAgent(t *testing.T) {
	clearHeimdallEnv(t)
	projectRoot := t.TempDir()
	writeTestFile(t, filepath.Join(projectRoot, ".env"), validDotenvWithExtra("main", "HEIMDALL_REPO_PLATFORM_DEFAULT_SPEC_WRITING_AGENT=   "))

	_, err := LoadFromDir(projectRoot)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "HEIMDALL_REPO_PLATFORM_DEFAULT_SPEC_WRITING_AGENT") {
		t.Fatalf("expected empty default spec-writing agent validation error, got %v", err)
	}
}

func clearHeimdallEnv(t *testing.T) {
	t.Helper()

	type envValue struct {
		value   string
		present bool
	}

	snapshot := make(map[string]envValue)
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if !found || !strings.HasPrefix(key, "HEIMDALL_") {
			continue
		}
		snapshot[key] = envValue{value: value, present: true}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("os.Unsetenv(%q) error = %v", key, err)
		}
	}

	t.Cleanup(func() {
		for _, entry := range os.Environ() {
			key, _, found := strings.Cut(entry, "=")
			if !found || !strings.HasPrefix(key, "HEIMDALL_") {
				continue
			}
			os.Unsetenv(key)
		}
		for key, value := range snapshot {
			if value.present {
				os.Setenv(key, value.value)
			}
		}
	})
}

func validEnvMap(baseBranch string) map[string]string {
	return map[string]string{
		"HEIMDALL_SERVER_LISTEN_ADDRESS":                    ":8080",
		"HEIMDALL_SERVER_PUBLIC_URL":                        "http://127.0.0.1:8080",
		"HEIMDALL_STORAGE_DRIVER":                           "sqlite",
		"HEIMDALL_STORAGE_DSN":                              "/tmp/heimdall.db",
		"HEIMDALL_LINEAR_POLL_INTERVAL":                     "30s",
		"HEIMDALL_LINEAR_ACTIVE_STATES":                     "In Progress",
		"HEIMDALL_LINEAR_PROJECT_NAME":                      "Core Platform",
		"HEIMDALL_LINEAR_API_TOKEN":                         "linear-token",
		"HEIMDALL_GITHUB_BASE_BRANCH":                       baseBranch,
		"HEIMDALL_GITHUB_POLL_INTERVAL":                     "30s",
		"HEIMDALL_GITHUB_LOOKBACK_WINDOW":                   "2m",
		"HEIMDALL_GITHUB_APP_ID":                            "12345",
		"HEIMDALL_GITHUB_INSTALLATION_ID":                   "99",
		"HEIMDALL_GITHUB_PRIVATE_KEY":                       "test-private-key",
		"HEIMDALL_REPOS":                                    "PLATFORM",
		"HEIMDALL_REPO_PLATFORM_NAME":                       "github.com/acme/platform",
		"HEIMDALL_REPO_PLATFORM_LOCAL_MIRROR_PATH":          "/var/lib/heimdall/repos/github.com/acme/platform.git",
		"HEIMDALL_REPO_PLATFORM_DEFAULT_BRANCH":             "main",
		"HEIMDALL_REPO_PLATFORM_BRANCH_PREFIX":              "heimdall",
		"HEIMDALL_REPO_PLATFORM_LINEAR_TEAM_KEYS":           "ENG",
		"HEIMDALL_REPO_PLATFORM_ALLOWED_AGENTS":             "gpt-5.4,claude-sonnet",
		"HEIMDALL_REPO_PLATFORM_ALLOWED_USERS":              "mngeow",
		"HEIMDALL_REPO_PLATFORM_DEFAULT_SPEC_WRITING_AGENT": "gpt-5.4",
	}
}

func validDotenv(baseBranch string) string {
	lines := make([]string, 0, len(validEnvMap(baseBranch)))
	for key, value := range validEnvMap(baseBranch) {
		lines = append(lines, key+"="+value)
	}
	return strings.Join(lines, "\n") + "\n"
}

func validDotenvWithExtra(baseBranch string, extraLines ...string) string {
	lines := []string{strings.TrimSuffix(validDotenv(baseBranch), "\n")}
	lines = append(lines, extraLines...)
	return strings.Join(lines, "\n") + "\n"
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}
