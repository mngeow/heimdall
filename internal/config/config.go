package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the Symphony configuration
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Storage StorageConfig `yaml:"storage"`
	Linear  LinearConfig  `yaml:"linear"`
	GitHub  GitHubConfig  `yaml:"github"`
	Repos   []RepoConfig  `yaml:"repos"`
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	ListenAddress string `yaml:"listen_address"`
	PublicURL     string `yaml:"public_url"`
}

// StorageConfig represents database configuration
type StorageConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

// LinearConfig represents Linear integration configuration
type LinearConfig struct {
	PollInterval time.Duration `yaml:"poll_interval"`
	ActiveStates []string      `yaml:"active_states"`
	TeamKeys     []string      `yaml:"team_keys"`
	APIToken     string        // loaded from env
}

// GitHubConfig represents GitHub integration configuration
type GitHubConfig struct {
	BaseBranch     string        `yaml:"base_branch"`
	PollInterval   time.Duration `yaml:"poll_interval"`
	LookbackWindow time.Duration `yaml:"lookback_window"`
	AppID          string        // loaded from env
	InstallationID int64         // loaded from env
	PrivateKey     string        // loaded from env or file
}

// RepoConfig represents a managed repository
type RepoConfig struct {
	Name            string   `yaml:"name"`
	LocalMirrorPath string   `yaml:"local_mirror_path"`
	DefaultBranch   string   `yaml:"default_branch"`
	BranchPrefix    string   `yaml:"branch_prefix"`
	LinearTeamKeys  []string `yaml:"linear_team_keys"`
	AllowedAgents   []string `yaml:"allowed_agents"`
	AllowedUsers    []string `yaml:"allowed_users"`
}

// Load loads configuration from file and environment
func Load() (*Config, error) {
	configPath := os.Getenv("SYMPHONY_CONFIG_PATH")
	if configPath == "" {
		configPath = "/etc/symphony/config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults
	if cfg.Server.ListenAddress == "" {
		cfg.Server.ListenAddress = ":8080"
	}
	if cfg.Storage.Driver == "" {
		cfg.Storage.Driver = "sqlite"
	}
	if cfg.Storage.DSN == "" {
		cfg.Storage.DSN = "/var/lib/symphony/state/symphony.db"
	}
	if cfg.GitHub.BaseBranch == "" {
		cfg.GitHub.BaseBranch = "main"
	}
	if cfg.GitHub.PollInterval == 0 {
		cfg.GitHub.PollInterval = 30 * time.Second
	}
	if cfg.GitHub.LookbackWindow == 0 {
		cfg.GitHub.LookbackWindow = 2 * time.Minute
	}

	// Load secrets from environment
	cfg.Linear.APIToken = os.Getenv("SYMPHONY_LINEAR_API_TOKEN")
	cfg.GitHub.AppID = os.Getenv("SYMPHONY_GITHUB_APP_ID")
	installationID := os.Getenv("SYMPHONY_GITHUB_INSTALLATION_ID")
	if installationID != "" {
		parsedInstallationID, err := strconv.ParseInt(installationID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse SYMPHONY_GITHUB_INSTALLATION_ID: %w", err)
		}
		cfg.GitHub.InstallationID = parsedInstallationID
	}

	privateKeyFile := os.Getenv("SYMPHONY_GITHUB_PRIVATE_KEY_FILE")
	if privateKeyFile != "" {
		privateKey, err := os.ReadFile(privateKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read SYMPHONY_GITHUB_PRIVATE_KEY_FILE: %w", err)
		}
		cfg.GitHub.PrivateKey = string(privateKey)
	} else {
		cfg.GitHub.PrivateKey = os.Getenv("SYMPHONY_GITHUB_PRIVATE_KEY")
	}

	return &cfg, nil
}
