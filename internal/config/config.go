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
	Version    int                  `mapstructure:"version" json:"version"`
	Defaults   Defaults             `mapstructure:"defaults" json:"defaults"`
	Workspaces map[string]Workspace `mapstructure:"workspaces" json:"workspaces"`
}

// Defaults contains global default settings.
type Defaults struct {
	Format    string `mapstructure:"format" json:"format,omitempty"`
	Workspace string `mapstructure:"workspace" json:"workspace,omitempty"`
	AgentID   string `mapstructure:"agent_id" json:"agent_id,omitempty"`
}

// Workspace represents a configured connection to a backend.
type Workspace struct {
	Backend          string            `mapstructure:"backend" json:"backend,omitempty"`
	Repo             string            `mapstructure:"repo" json:"repo,omitempty"`
	Team             string            `mapstructure:"team" json:"team,omitempty"`
	Path             string            `mapstructure:"path" json:"path,omitempty"`
	Project          int               `mapstructure:"project" json:"project,omitempty"`
	StatusField      string            `mapstructure:"status_field" json:"status_field,omitempty"`
	AgentID          string            `mapstructure:"agent_id" json:"agent_id,omitempty"`
	AgentLabelPrefix string            `mapstructure:"agent_label_prefix" json:"agent_label_prefix,omitempty"`
	Default          bool              `mapstructure:"default" json:"default,omitempty"`
	APIKeyEnv        string            `mapstructure:"api_key_env" json:"api_key_env,omitempty"`
	LockMode         string            `mapstructure:"lock_mode" json:"lock_mode,omitempty"`
	GitSync          bool              `mapstructure:"git_sync" json:"git_sync,omitempty"`
	StatusMap        map[string]Status `mapstructure:"status_map" json:"status_map,omitempty"`
	DefaultFilters   DefaultFilters    `mapstructure:"default_filters" json:"default_filters,omitempty"`
}

// Status represents a status mapping configuration.
type Status struct {
	State  string   `mapstructure:"state" json:"state,omitempty"`
	Labels []string `mapstructure:"labels" json:"labels,omitempty"`
}

// DefaultFilters represents default filters for a workspace.
type DefaultFilters struct {
	Labels []string `mapstructure:"labels" json:"labels,omitempty"`
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
