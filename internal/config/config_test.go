package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromDirReadsProjectRootDotenv(t *testing.T) {
	clearSymphonyEnv(t)
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
	clearSymphonyEnv(t)
	projectRoot := t.TempDir()

	writeTestFile(t, filepath.Join(projectRoot, ".env"), validDotenv("main"))
	t.Setenv("SYMPHONY_GITHUB_BASE_BRANCH", "release")

	cfg, err := LoadFromDir(projectRoot)
	if err != nil {
		t.Fatalf("LoadFromDir() error = %v", err)
	}

	if cfg.GitHub.BaseBranch != "release" {
		t.Fatalf("expected base branch release, got %q", cfg.GitHub.BaseBranch)
	}
}

func TestLoadFromDirSupportsEnvironmentOnly(t *testing.T) {
	clearSymphonyEnv(t)
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
	clearSymphonyEnv(t)
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
	clearSymphonyEnv(t)
	projectRoot := t.TempDir()
	privateKeyPath := filepath.Join(projectRoot, "github-app.pem")
	writeTestFile(t, privateKeyPath, "test-private-key")
	writeTestFile(t, filepath.Join(projectRoot, ".env"), strings.ReplaceAll(validDotenv("main"), "SYMPHONY_GITHUB_PRIVATE_KEY=test-private-key", "SYMPHONY_GITHUB_PRIVATE_KEY_FILE="+privateKeyPath))

	cfg, err := LoadFromDir(projectRoot)
	if err != nil {
		t.Fatalf("LoadFromDir() error = %v", err)
	}
	if cfg.GitHub.PrivateKey != "test-private-key" {
		t.Fatalf("expected private key file contents, got %q", cfg.GitHub.PrivateKey)
	}
}

func TestLoadFromDirValidatesLookbackWindow(t *testing.T) {
	clearSymphonyEnv(t)
	projectRoot := t.TempDir()
	writeTestFile(t, filepath.Join(projectRoot, ".env"), strings.ReplaceAll(validDotenv("main"), "SYMPHONY_GITHUB_LOOKBACK_WINDOW=2m", "SYMPHONY_GITHUB_LOOKBACK_WINDOW=0s"))

	_, err := LoadFromDir(projectRoot)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "SYMPHONY_GITHUB_LOOKBACK_WINDOW") {
		t.Fatalf("expected lookback validation error, got %v", err)
	}
}

func clearSymphonyEnv(t *testing.T) {
	t.Helper()

	type envValue struct {
		value   string
		present bool
	}

	snapshot := make(map[string]envValue)
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if !found || !strings.HasPrefix(key, "SYMPHONY_") {
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
			if !found || !strings.HasPrefix(key, "SYMPHONY_") {
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
		"SYMPHONY_SERVER_LISTEN_ADDRESS":           ":8080",
		"SYMPHONY_SERVER_PUBLIC_URL":               "http://127.0.0.1:8080",
		"SYMPHONY_STORAGE_DRIVER":                  "sqlite",
		"SYMPHONY_STORAGE_DSN":                     "/tmp/symphony.db",
		"SYMPHONY_LINEAR_POLL_INTERVAL":            "30s",
		"SYMPHONY_LINEAR_ACTIVE_STATES":            "In Progress",
		"SYMPHONY_LINEAR_TEAM_KEYS":                "ENG",
		"SYMPHONY_LINEAR_API_TOKEN":                "linear-token",
		"SYMPHONY_GITHUB_BASE_BRANCH":              baseBranch,
		"SYMPHONY_GITHUB_POLL_INTERVAL":            "30s",
		"SYMPHONY_GITHUB_LOOKBACK_WINDOW":          "2m",
		"SYMPHONY_GITHUB_APP_ID":                   "12345",
		"SYMPHONY_GITHUB_INSTALLATION_ID":          "99",
		"SYMPHONY_GITHUB_PRIVATE_KEY":              "test-private-key",
		"SYMPHONY_REPOS":                           "PLATFORM",
		"SYMPHONY_REPO_PLATFORM_NAME":              "github.com/acme/platform",
		"SYMPHONY_REPO_PLATFORM_LOCAL_MIRROR_PATH": "/var/lib/symphony/repos/github.com/acme/platform.git",
		"SYMPHONY_REPO_PLATFORM_DEFAULT_BRANCH":    "main",
		"SYMPHONY_REPO_PLATFORM_BRANCH_PREFIX":     "symphony",
		"SYMPHONY_REPO_PLATFORM_LINEAR_TEAM_KEYS":  "ENG",
		"SYMPHONY_REPO_PLATFORM_ALLOWED_AGENTS":    "gpt-5.4,claude-sonnet",
		"SYMPHONY_REPO_PLATFORM_ALLOWED_USERS":     "mngeow",
	}
}

func validDotenv(baseBranch string) string {
	lines := make([]string, 0, len(validEnvMap(baseBranch)))
	for key, value := range validEnvMap(baseBranch) {
		lines = append(lines, key+"="+value)
	}
	return strings.Join(lines, "\n") + "\n"
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}
