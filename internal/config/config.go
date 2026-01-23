// Package config provides configuration loading and management using Viper.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the top-level configuration structure.
type Config struct {
	Version    int                  `mapstructure:"version"`
	Defaults   Defaults             `mapstructure:"defaults"`
	Workspaces map[string]Workspace `mapstructure:"workspaces"`
}

// Defaults contains global default settings.
type Defaults struct {
	Format           string `mapstructure:"format"`
	Workspace        string `mapstructure:"workspace"`
	AgentID          string `mapstructure:"agent_id"`
}

// Workspace represents a configured connection to a backend.
type Workspace struct {
	Backend          string            `mapstructure:"backend"`
	Repo             string            `mapstructure:"repo"`
	Team             string            `mapstructure:"team"`
	Path             string            `mapstructure:"path"`
	Project          int               `mapstructure:"project"`
	StatusField      string            `mapstructure:"status_field"`
	AgentID          string            `mapstructure:"agent_id"`
	AgentLabelPrefix string            `mapstructure:"agent_label_prefix"`
	Default          bool              `mapstructure:"default"`
	APIKeyEnv        string            `mapstructure:"api_key_env"`
	LockMode         string            `mapstructure:"lock_mode"`
	GitSync          bool              `mapstructure:"git_sync"`
	StatusMap        map[string]Status `mapstructure:"status_map"`
	DefaultFilters   DefaultFilters    `mapstructure:"default_filters"`
}

// Status represents a status mapping configuration.
type Status struct {
	State  string   `mapstructure:"state"`
	Labels []string `mapstructure:"labels"`
}

// DefaultFilters represents default filters for a workspace.
type DefaultFilters struct {
	Labels []string `mapstructure:"labels"`
}

var (
	cfg     *Config
	cfgFile string
)

// configDir returns the configuration directory path.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "backlog"), nil
}

// Init initializes the configuration system.
// Config files are searched in the following order:
// 1. Explicit path via cfgPath parameter (--config flag)
// 2. Project-local: .backlog/config.yaml (current directory)
// 3. User global: ~/.config/backlog/config.yaml
func Init(cfgPath string) error {
	cfgFile = cfgPath

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Check for project-local config first
		viper.AddConfigPath(".backlog")
		// Then check user global config
		configPath, err := configDir()
		if err != nil {
			return err
		}
		viper.AddConfigPath(configPath)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// Set defaults
	viper.SetDefault("version", 1)
	viper.SetDefault("defaults.format", "table")

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK - we'll use defaults
	}

	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// Get returns the current configuration.
// Returns nil if Init has not been called.
func Get() *Config {
	return cfg
}

// GetWorkspace returns the workspace configuration for the given name.
// If name is empty, returns the default workspace.
func GetWorkspace(name string) (*Workspace, string, error) {
	if cfg == nil {
		return nil, "", fmt.Errorf("configuration not initialized")
	}

	if len(cfg.Workspaces) == 0 {
		return nil, "", fmt.Errorf("no workspaces configured")
	}

	// If name provided, look it up directly
	if name != "" {
		ws, ok := cfg.Workspaces[name]
		if !ok {
			return nil, "", fmt.Errorf("workspace %q not found", name)
		}
		return &ws, name, nil
	}

	// Check defaults.workspace
	if cfg.Defaults.Workspace != "" {
		ws, ok := cfg.Workspaces[cfg.Defaults.Workspace]
		if ok {
			return &ws, cfg.Defaults.Workspace, nil
		}
	}

	// Find workspace with default: true
	for wsName, ws := range cfg.Workspaces {
		if ws.Default {
			return &ws, wsName, nil
		}
	}

	// If only one workspace, use it
	if len(cfg.Workspaces) == 1 {
		for wsName, ws := range cfg.Workspaces {
			wsCopy := ws
			return &wsCopy, wsName, nil
		}
	}

	return nil, "", fmt.Errorf("no default workspace configured")
}

// ConfigFilePath returns the path to the config file being used.
func ConfigFilePath() string {
	return viper.ConfigFileUsed()
}

// DefaultConfigPath returns the default config file path.
func DefaultConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}
