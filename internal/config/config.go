package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	env "github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

const (
	dotenvFileName       = ".env"
	legacyConfigBaseName = "config"
)

var repoIDPattern = regexp.MustCompile(`^[A-Z0-9_]+$`)

// Config represents the Heimdall configuration.
type Config struct {
	Server  ServerConfig
	Storage StorageConfig
	Linear  LinearConfig
	GitHub  GitHubConfig
	Repos   []RepoConfig
}

// ServerConfig represents HTTP server configuration.
type ServerConfig struct {
	ListenAddress string `env:"LISTEN_ADDRESS" envDefault:":8080"`
	PublicURL     string `env:"PUBLIC_URL"`
}

// StorageConfig represents database configuration.
type StorageConfig struct {
	Driver string `env:"DRIVER" envDefault:"sqlite"`
	DSN    string `env:"DSN" envDefault:"/var/lib/heimdall/state/heimdall.db"`
}

// LinearConfig represents Linear integration configuration.
type LinearConfig struct {
	PollInterval time.Duration `env:"POLL_INTERVAL" envDefault:"30s"`
	ActiveStates []string      `env:"ACTIVE_STATES,required" envSeparator:","`
	ProjectName  string        `env:"PROJECT_NAME,required,notEmpty"`
	APIToken     string        `env:"API_TOKEN,required,notEmpty"`
}

// GitHubConfig represents GitHub integration configuration.
type GitHubConfig struct {
	BaseBranch     string        `env:"BASE_BRANCH" envDefault:"main"`
	PollInterval   time.Duration `env:"POLL_INTERVAL" envDefault:"30s"`
	LookbackWindow time.Duration `env:"LOOKBACK_WINDOW" envDefault:"2m"`
	AppID          string        `env:"APP_ID,required,notEmpty"`
	InstallationID int64         `env:"INSTALLATION_ID,required"`
	PrivateKey     string
}

// OpencodeCommandAlias maps a repository alias to an opencode command name and permission profile.
type OpencodeCommandAlias struct {
	Name              string
	Command           string
	PermissionProfile string
}

// RepoConfig represents a managed repository.
type RepoConfig struct {
	ID                      string
	Name                    string   `env:"NAME,required,notEmpty"`
	LocalMirrorPath         string   `env:"LOCAL_MIRROR_PATH,required,notEmpty"`
	DefaultBranch           string   `env:"DEFAULT_BRANCH" envDefault:"main"`
	BranchPrefix            string   `env:"BRANCH_PREFIX" envDefault:"heimdall"`
	PRMonitorLabel          string   `env:"PR_MONITOR_LABEL"`
	LinearTeamKeys          []string `env:"LINEAR_TEAM_KEYS" envSeparator:","`
	AllowedAgents           []string `env:"ALLOWED_AGENTS,required" envSeparator:","`
	AllowedUsers            []string `env:"ALLOWED_USERS,required" envSeparator:","`
	DefaultSpecWritingAgent string   `env:"DEFAULT_SPEC_WRITING_AGENT,required,notEmpty"`
	OpencodeCommands        []string `env:"OPENCODE_COMMANDS" envSeparator:","`
	OpencodeAliases         map[string]OpencodeCommandAlias
}

type rootEnvConfig struct {
	Server  ServerConfig    `envPrefix:"HEIMDALL_SERVER_"`
	Storage StorageConfig   `envPrefix:"HEIMDALL_STORAGE_"`
	Linear  LinearConfig    `envPrefix:"HEIMDALL_LINEAR_"`
	GitHub  githubEnvConfig `envPrefix:"HEIMDALL_GITHUB_"`
	RepoIDs []string        `env:"HEIMDALL_REPOS,required" envSeparator:","`
}

type githubEnvConfig struct {
	BaseBranch     string        `env:"BASE_BRANCH" envDefault:"main"`
	PollInterval   time.Duration `env:"POLL_INTERVAL" envDefault:"30s"`
	LookbackWindow time.Duration `env:"LOOKBACK_WINDOW" envDefault:"2m"`
	AppID          string        `env:"APP_ID,required,notEmpty"`
	InstallationID int64         `env:"INSTALLATION_ID,required"`
	PrivateKey     string        `env:"PRIVATE_KEY"`
	PrivateKeyFile string        `env:"PRIVATE_KEY_FILE,file"`
}

// Load loads configuration from environment variables and an optional project-root .env file.
func Load() (*Config, error) {
	projectRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project root: %w", err)
	}

	return LoadFromDir(projectRoot)
}

// LoadFromDir loads Heimdall configuration for a specific project root.
func LoadFromDir(projectRoot string) (*Config, error) {
	environment, err := loadEnvironment(projectRoot)
	if err != nil {
		return nil, err
	}

	root, err := env.ParseAsWithOptions[rootEnvConfig](env.Options{Environment: environment})
	if err != nil {
		return nil, fmt.Errorf("failed to parse environment configuration: %w", err)
	}

	config := &Config{
		Server:  root.Server,
		Storage: root.Storage,
		Linear: LinearConfig{
			PollInterval: root.Linear.PollInterval,
			ActiveStates: trimNonEmpty(root.Linear.ActiveStates),
			ProjectName:  strings.TrimSpace(root.Linear.ProjectName),
			APIToken:     strings.TrimSpace(root.Linear.APIToken),
		},
		GitHub: GitHubConfig{
			BaseBranch:     strings.TrimSpace(root.GitHub.BaseBranch),
			PollInterval:   root.GitHub.PollInterval,
			LookbackWindow: root.GitHub.LookbackWindow,
			AppID:          strings.TrimSpace(root.GitHub.AppID),
			InstallationID: root.GitHub.InstallationID,
			PrivateKey:     strings.TrimSpace(selectGitHubPrivateKey(root.GitHub)),
		},
	}

	repoIDs := trimNonEmpty(root.RepoIDs)
	for _, repoID := range repoIDs {
		repoConfig, err := loadRepoConfig(environment, repoID)
		if err != nil {
			return nil, err
		}
		config.Repos = append(config.Repos, repoConfig)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate ensures the loaded configuration is internally consistent.
func (c *Config) Validate() error {
	if len(c.Linear.ActiveStates) == 0 {
		return fmt.Errorf("HEIMDALL_LINEAR_ACTIVE_STATES must include at least one state")
	}
	if c.Linear.ProjectName == "" {
		return fmt.Errorf("HEIMDALL_LINEAR_PROJECT_NAME must not be empty")
	}
	if c.GitHub.PrivateKey == "" {
		return fmt.Errorf("either HEIMDALL_GITHUB_PRIVATE_KEY or HEIMDALL_GITHUB_PRIVATE_KEY_FILE must be set")
	}
	if c.GitHub.PollInterval <= 0 {
		return fmt.Errorf("HEIMDALL_GITHUB_POLL_INTERVAL must be greater than zero")
	}
	if c.GitHub.LookbackWindow <= 0 {
		return fmt.Errorf("HEIMDALL_GITHUB_LOOKBACK_WINDOW must be greater than zero")
	}
	if len(c.Repos) == 0 {
		return fmt.Errorf("HEIMDALL_REPOS must declare at least one repository")
	}

	repoRefs := make(map[string]string, len(c.Repos))
	routedTeams := make(map[string]string)
	for _, repo := range c.Repos {
		if repo.Name == "" {
			return fmt.Errorf("HEIMDALL_REPO_%s_NAME must not be empty", repo.ID)
		}
		if repo.LocalMirrorPath == "" {
			return fmt.Errorf("HEIMDALL_REPO_%s_LOCAL_MIRROR_PATH must not be empty", repo.ID)
		}
		if len(repo.AllowedUsers) == 0 {
			return fmt.Errorf("HEIMDALL_REPO_%s_ALLOWED_USERS must include at least one user", repo.ID)
		}
		if len(repo.AllowedAgents) == 0 {
			return fmt.Errorf("HEIMDALL_REPO_%s_ALLOWED_AGENTS must include at least one agent", repo.ID)
		}
		if repo.DefaultSpecWritingAgent == "" {
			return fmt.Errorf("HEIMDALL_REPO_%s_DEFAULT_SPEC_WRITING_AGENT must not be empty", repo.ID)
		}
		if existingRepoID, exists := repoRefs[repo.Name]; exists {
			return fmt.Errorf("repository %q is defined more than once by HEIMDALL_REPO_%s_NAME and HEIMDALL_REPO_%s_NAME", repo.Name, existingRepoID, repo.ID)
		}
		repoRefs[repo.Name] = repo.ID

		if len(c.Repos) > 1 && len(repo.LinearTeamKeys) == 0 {
			return fmt.Errorf("HEIMDALL_REPO_%s_LINEAR_TEAM_KEYS must include at least one team key when multiple repositories are configured", repo.ID)
		}

		for _, teamKey := range repo.LinearTeamKeys {
			if existingRepoID, exists := routedTeams[teamKey]; exists {
				return fmt.Errorf("team key %q is configured for both HEIMDALL_REPO_%s_LINEAR_TEAM_KEYS and HEIMDALL_REPO_%s_LINEAR_TEAM_KEYS", teamKey, existingRepoID, repo.ID)
			}
			routedTeams[teamKey] = repo.ID
		}
	}

	return nil
}

func loadEnvironment(projectRoot string) (map[string]string, error) {
	environment := currentEnvironment()

	dotenvPath := filepath.Join(projectRoot, dotenvFileName)

	dotenvExists, err := fileExists(dotenvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect %s: %w", dotenvPath, err)
	}
	legacyYAMLFiles, err := findLegacyYAMLFiles(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect project root for legacy yaml config: %w", err)
	}

	if !dotenvExists && len(legacyYAMLFiles) > 0 {
		return nil, fmt.Errorf("legacy YAML configuration files are no longer supported; use %s or environment variables", dotenvPath)
	}

	if !dotenvExists {
		return environment, nil
	}

	fileEnvironment, err := godotenv.Read(dotenvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", dotenvPath, err)
	}

	for key, value := range fileEnvironment {
		if _, exists := environment[key]; exists {
			continue
		}
		environment[key] = value
	}

	return environment, nil
}

func loadRepoConfig(environment map[string]string, repoID string) (RepoConfig, error) {
	repoID = strings.TrimSpace(repoID)
	if repoID == "" {
		return RepoConfig{}, fmt.Errorf("HEIMDALL_REPOS must not contain empty repository identifiers")
	}
	if !repoIDPattern.MatchString(repoID) {
		return RepoConfig{}, fmt.Errorf("HEIMDALL_REPOS entry %q must use only A-Z, 0-9, and _ characters", repoID)
	}

	repoConfig, err := env.ParseAsWithOptions[RepoConfig](env.Options{
		Environment: environment,
		Prefix:      repoEnvPrefix(repoID),
	})
	if err != nil {
		return RepoConfig{}, fmt.Errorf("failed to parse repository %s configuration: %w", repoID, err)
	}
	if rawMonitorLabel, ok := environment[repoEnvPrefix(repoID)+"PR_MONITOR_LABEL"]; ok && strings.TrimSpace(rawMonitorLabel) == "" {
		return RepoConfig{}, fmt.Errorf("HEIMDALL_REPO_%s_PR_MONITOR_LABEL must not be empty when set", repoID)
	}

	repoConfig.ID = repoID
	repoConfig.Name = strings.TrimSpace(repoConfig.Name)
	repoConfig.LocalMirrorPath = strings.TrimSpace(repoConfig.LocalMirrorPath)
	repoConfig.DefaultBranch = strings.TrimSpace(repoConfig.DefaultBranch)
	repoConfig.BranchPrefix = strings.TrimSpace(repoConfig.BranchPrefix)
	repoConfig.PRMonitorLabel = strings.TrimSpace(repoConfig.PRMonitorLabel)
	repoConfig.LinearTeamKeys = trimNonEmpty(repoConfig.LinearTeamKeys)
	repoConfig.AllowedAgents = trimNonEmpty(repoConfig.AllowedAgents)
	repoConfig.AllowedUsers = trimNonEmpty(repoConfig.AllowedUsers)
	repoConfig.DefaultSpecWritingAgent = strings.TrimSpace(repoConfig.DefaultSpecWritingAgent)
	repoConfig.OpencodeCommands = trimNonEmpty(repoConfig.OpencodeCommands)

	aliases, err := loadOpencodeAliases(environment, repoID, repoConfig.OpencodeCommands)
	if err != nil {
		return RepoConfig{}, err
	}
	repoConfig.OpencodeAliases = aliases

	return repoConfig, nil
}

func loadOpencodeAliases(environment map[string]string, repoID string, aliasNames []string) (map[string]OpencodeCommandAlias, error) {
	aliases := make(map[string]OpencodeCommandAlias, len(aliasNames))
	for _, name := range aliasNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		upper := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
		prefix := repoEnvPrefix(repoID) + "OPENCODE_COMMAND_" + upper + "_"
		cmd := strings.TrimSpace(environment[prefix+"COMMAND"])
		profile := strings.TrimSpace(environment[prefix+"PERMISSION_PROFILE"])
		if cmd == "" {
			return nil, fmt.Errorf("HEIMDALL_REPO_%s_OPENCODE_COMMAND_%s_COMMAND must not be empty", repoID, upper)
		}
		validProfiles := map[string]bool{"readonly": true, "openspec-write": true, "repo-write": true}
		if profile == "" || !validProfiles[profile] {
			return nil, fmt.Errorf("HEIMDALL_REPO_%s_OPENCODE_COMMAND_%s_PERMISSION_PROFILE must be one of readonly, openspec-write, repo-write", repoID, upper)
		}
		if _, exists := aliases[name]; exists {
			return nil, fmt.Errorf("duplicate opencode alias %q for repository %s", name, repoID)
		}
		aliases[name] = OpencodeCommandAlias{
			Name:              name,
			Command:           cmd,
			PermissionProfile: profile,
		}
	}
	return aliases, nil
}

func selectGitHubPrivateKey(cfg githubEnvConfig) string {
	if strings.TrimSpace(cfg.PrivateKeyFile) != "" {
		return cfg.PrivateKeyFile
	}

	return cfg.PrivateKey
}

func currentEnvironment() map[string]string {
	environment := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		environment[key] = value
	}
	return environment
}

func fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return !info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func findLegacyYAMLFiles(projectRoot string) ([]string, error) {
	entries, err := os.ReadDir(projectRoot)
	if err != nil {
		return nil, err
	}

	legacyFiles := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == legacyConfigBaseName+".yaml" || name == legacyConfigBaseName+".yml" {
			legacyFiles = append(legacyFiles, name)
		}
	}

	return legacyFiles, nil
}

func repoEnvPrefix(repoID string) string {
	return "HEIMDALL_REPO_" + repoID + "_"
}

func trimNonEmpty(values []string) []string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	return trimmed
}
