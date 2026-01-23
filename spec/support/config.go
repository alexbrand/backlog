// Package support provides test helpers and fixtures for the backlog CLI specs.
package support

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// WorkspaceConfig represents a workspace configuration.
type WorkspaceConfig struct {
	Backend           string            `yaml:"backend"`
	Path              string            `yaml:"path,omitempty"`               // For local backend
	Repo              string            `yaml:"repo,omitempty"`               // For GitHub backend
	Team              string            `yaml:"team,omitempty"`               // For Linear backend
	Project           int               `yaml:"project,omitempty"`            // GitHub Project number
	StatusField       string            `yaml:"status_field,omitempty"`       // Project field name for status
	AgentID           string            `yaml:"agent_id,omitempty"`           // Agent ID for this workspace
	AgentLabelPrefix  string            `yaml:"agent_label_prefix,omitempty"` // Prefix for agent labels
	Default           bool              `yaml:"default,omitempty"`            // Whether this is the default workspace
	LockMode          string            `yaml:"lock_mode,omitempty"`          // "file" or "git"
	GitSync           bool              `yaml:"git_sync,omitempty"`           // Auto-commit on changes
	APIKeyEnv         string            `yaml:"api_key_env,omitempty"`        // Env var for API key
	StatusMap         map[string]any    `yaml:"status_map,omitempty"`         // Custom status mapping
	DefaultFilters    map[string]any    `yaml:"default_filters,omitempty"`    // Default list filters
	Extra             map[string]any    `yaml:"-"`                            // Extra fields to merge
}

// DefaultsConfig represents the defaults section of config.
type DefaultsConfig struct {
	Format    string `yaml:"format,omitempty"`
	Workspace string `yaml:"workspace,omitempty"`
	AgentID   string `yaml:"agent_id,omitempty"`
}

// Config represents the full backlog configuration file.
type Config struct {
	Version    int                         `yaml:"version"`
	Defaults   *DefaultsConfig             `yaml:"defaults,omitempty"`
	Workspaces map[string]*WorkspaceConfig `yaml:"workspaces,omitempty"`
}

// ConfigGenerator creates config.yaml files for test workspaces.
type ConfigGenerator struct {
	// No fields needed for now
}

// NewConfigGenerator creates a new config generator.
func NewConfigGenerator() *ConfigGenerator {
	return &ConfigGenerator{}
}

// Generate creates a config file from a Config struct.
func (g *ConfigGenerator) Generate(env *TestEnv, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Set default version if not specified
	if cfg.Version == 0 {
		cfg.Version = 1
	}

	// Convert to map for YAML marshaling to handle extra fields
	configMap := g.configToMap(cfg)

	content, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure .backlog directory exists
	if err := env.CreateBacklogDir(); err != nil {
		return fmt.Errorf("failed to create backlog directory: %w", err)
	}

	return env.CreateFile(".backlog/config.yaml", string(content))
}

// GenerateFromYAML creates a config file from a YAML string.
func (g *ConfigGenerator) GenerateFromYAML(env *TestEnv, yamlContent string) error {
	// Validate the YAML is parseable
	var cfg map[string]any
	if err := yaml.Unmarshal([]byte(yamlContent), &cfg); err != nil {
		return fmt.Errorf("failed to parse config YAML: %w", err)
	}

	// Ensure .backlog directory exists
	if err := env.CreateBacklogDir(); err != nil {
		return fmt.Errorf("failed to create backlog directory: %w", err)
	}

	return env.CreateFile(".backlog/config.yaml", yamlContent)
}

// GenerateDefault creates a default config for local backend testing.
func (g *ConfigGenerator) GenerateDefault(env *TestEnv) error {
	cfg := &Config{
		Version: 1,
		Defaults: &DefaultsConfig{
			Format:    "table",
			Workspace: "local",
		},
		Workspaces: map[string]*WorkspaceConfig{
			"local": {
				Backend: "local",
				Path:    "./.backlog",
				Default: true,
			},
		},
	}
	return g.Generate(env, cfg)
}

// GenerateLocalBackend creates a config for local backend with common options.
func (g *ConfigGenerator) GenerateLocalBackend(env *TestEnv, opts *LocalBackendOptions) error {
	if opts == nil {
		opts = &LocalBackendOptions{}
	}

	// Set defaults
	workspaceName := opts.WorkspaceName
	if workspaceName == "" {
		workspaceName = "local"
	}

	path := opts.Path
	if path == "" {
		path = "./.backlog"
	}

	lockMode := opts.LockMode
	if lockMode == "" {
		lockMode = "file"
	}

	workspace := &WorkspaceConfig{
		Backend:          "local",
		Path:             path,
		Default:          true,
		LockMode:         lockMode,
		GitSync:          opts.GitSync,
		AgentID:          opts.AgentID,
		AgentLabelPrefix: opts.AgentLabelPrefix,
	}

	cfg := &Config{
		Version: 1,
		Defaults: &DefaultsConfig{
			Format:    "table",
			Workspace: workspaceName,
			AgentID:   opts.DefaultAgentID,
		},
		Workspaces: map[string]*WorkspaceConfig{
			workspaceName: workspace,
		},
	}

	return g.Generate(env, cfg)
}

// LocalBackendOptions contains options for generating a local backend config.
type LocalBackendOptions struct {
	WorkspaceName    string
	Path             string
	LockMode         string // "file" or "git"
	GitSync          bool
	AgentID          string
	AgentLabelPrefix string
	DefaultAgentID   string
}

// GenerateMultiWorkspace creates a config with multiple workspaces.
func (g *ConfigGenerator) GenerateMultiWorkspace(env *TestEnv, workspaces map[string]*WorkspaceConfig, defaultWorkspace string) error {
	// Validate we have at least one workspace
	if len(workspaces) == 0 {
		return fmt.Errorf("at least one workspace is required")
	}

	// Set the default workspace
	if defaultWorkspace != "" {
		if ws, ok := workspaces[defaultWorkspace]; ok {
			ws.Default = true
		}
	}

	cfg := &Config{
		Version: 1,
		Defaults: &DefaultsConfig{
			Format:    "table",
			Workspace: defaultWorkspace,
		},
		Workspaces: workspaces,
	}

	return g.Generate(env, cfg)
}

// configToMap converts a Config struct to a map for flexible YAML marshaling.
func (g *ConfigGenerator) configToMap(cfg *Config) map[string]any {
	result := make(map[string]any)
	result["version"] = cfg.Version

	if cfg.Defaults != nil {
		defaults := make(map[string]any)
		if cfg.Defaults.Format != "" {
			defaults["format"] = cfg.Defaults.Format
		}
		if cfg.Defaults.Workspace != "" {
			defaults["workspace"] = cfg.Defaults.Workspace
		}
		if cfg.Defaults.AgentID != "" {
			defaults["agent_id"] = cfg.Defaults.AgentID
		}
		if len(defaults) > 0 {
			result["defaults"] = defaults
		}
	}

	if len(cfg.Workspaces) > 0 {
		workspaces := make(map[string]any)
		for name, ws := range cfg.Workspaces {
			workspaces[name] = g.workspaceToMap(ws)
		}
		result["workspaces"] = workspaces
	}

	return result
}

// workspaceToMap converts a WorkspaceConfig to a map for flexible YAML marshaling.
func (g *ConfigGenerator) workspaceToMap(ws *WorkspaceConfig) map[string]any {
	result := make(map[string]any)

	if ws.Backend != "" {
		result["backend"] = ws.Backend
	}
	if ws.Path != "" {
		result["path"] = ws.Path
	}
	if ws.Repo != "" {
		result["repo"] = ws.Repo
	}
	if ws.Team != "" {
		result["team"] = ws.Team
	}
	if ws.Project != 0 {
		result["project"] = ws.Project
	}
	if ws.StatusField != "" {
		result["status_field"] = ws.StatusField
	}
	if ws.AgentID != "" {
		result["agent_id"] = ws.AgentID
	}
	if ws.AgentLabelPrefix != "" {
		result["agent_label_prefix"] = ws.AgentLabelPrefix
	}
	if ws.Default {
		result["default"] = ws.Default
	}
	if ws.LockMode != "" {
		result["lock_mode"] = ws.LockMode
	}
	if ws.GitSync {
		result["git_sync"] = ws.GitSync
	}
	if ws.APIKeyEnv != "" {
		result["api_key_env"] = ws.APIKeyEnv
	}
	if ws.StatusMap != nil {
		result["status_map"] = ws.StatusMap
	}
	if ws.DefaultFilters != nil {
		result["default_filters"] = ws.DefaultFilters
	}

	// Merge any extra fields
	for k, v := range ws.Extra {
		result[k] = v
	}

	return result
}
