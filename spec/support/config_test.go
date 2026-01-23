package support

import (
	"strings"
	"testing"
)

func TestConfigGenerator_Generate(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	generator := NewConfigGenerator()

	cfg := &Config{
		Version: 1,
		Defaults: &DefaultsConfig{
			Format:    "json",
			Workspace: "main",
		},
		Workspaces: map[string]*WorkspaceConfig{
			"main": {
				Backend: "local",
				Path:    "./.backlog",
				Default: true,
			},
		},
	}

	err = generator.Generate(env, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !env.FileExists(".backlog/config.yaml") {
		t.Error("Config file not created")
	}

	content, err := env.ReadFile(".backlog/config.yaml")
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if !strings.Contains(content, "version: 1") {
		t.Error("Config missing version field")
	}
	if !strings.Contains(content, "format: json") {
		t.Error("Config missing format in defaults")
	}
	if !strings.Contains(content, "backend: local") {
		t.Error("Config missing backend field")
	}
}

func TestConfigGenerator_Generate_NilConfig(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	generator := NewConfigGenerator()

	err = generator.Generate(env, nil)
	if err == nil {
		t.Error("Expected error for nil config, got nil")
	}
	if !strings.Contains(err.Error(), "config cannot be nil") {
		t.Errorf("Expected 'config cannot be nil' error, got: %v", err)
	}
}

func TestConfigGenerator_Generate_DefaultVersion(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	generator := NewConfigGenerator()

	// Config with Version 0 (unset) should default to 1
	cfg := &Config{
		Workspaces: map[string]*WorkspaceConfig{
			"test": {
				Backend: "local",
			},
		},
	}

	err = generator.Generate(env, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := env.ReadFile(".backlog/config.yaml")
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if !strings.Contains(content, "version: 1") {
		t.Error("Config should default to version 1")
	}
}

func TestConfigGenerator_GenerateFromYAML(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	generator := NewConfigGenerator()

	yamlContent := `version: 1
defaults:
  format: table
  workspace: work
workspaces:
  work:
    backend: linear
    team: ENG
`

	err = generator.GenerateFromYAML(env, yamlContent)
	if err != nil {
		t.Fatalf("GenerateFromYAML failed: %v", err)
	}

	if !env.FileExists(".backlog/config.yaml") {
		t.Error("Config file not created")
	}

	content, err := env.ReadFile(".backlog/config.yaml")
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if content != yamlContent {
		t.Error("Config content should match original YAML")
	}
}

func TestConfigGenerator_GenerateFromYAML_InvalidYAML(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	generator := NewConfigGenerator()

	invalidYAML := `version: 1
  bad indentation:
    - not valid`

	err = generator.GenerateFromYAML(env, invalidYAML)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse config YAML") {
		t.Errorf("Expected YAML parse error, got: %v", err)
	}
}

func TestConfigGenerator_GenerateDefault(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	generator := NewConfigGenerator()

	err = generator.GenerateDefault(env)
	if err != nil {
		t.Fatalf("GenerateDefault failed: %v", err)
	}

	if !env.FileExists(".backlog/config.yaml") {
		t.Error("Config file not created")
	}

	content, err := env.ReadFile(".backlog/config.yaml")
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	// Check default config has expected values
	if !strings.Contains(content, "version: 1") {
		t.Error("Default config missing version")
	}
	if !strings.Contains(content, "format: table") {
		t.Error("Default config missing format")
	}
	if !strings.Contains(content, "workspace: local") {
		t.Error("Default config missing workspace")
	}
	if !strings.Contains(content, "backend: local") {
		t.Error("Default config missing backend")
	}
}

func TestConfigGenerator_GenerateLocalBackend(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	generator := NewConfigGenerator()

	opts := &LocalBackendOptions{
		WorkspaceName:    "mylocal",
		Path:             "./tasks",
		LockMode:         "git",
		GitSync:          true,
		AgentID:          "claude-1",
		AgentLabelPrefix: "agent",
		DefaultAgentID:   "default-agent",
	}

	err = generator.GenerateLocalBackend(env, opts)
	if err != nil {
		t.Fatalf("GenerateLocalBackend failed: %v", err)
	}

	content, err := env.ReadFile(".backlog/config.yaml")
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if !strings.Contains(content, "workspace: mylocal") {
		t.Error("Config should use custom workspace name")
	}
	if !strings.Contains(content, "path: ./tasks") {
		t.Error("Config should use custom path")
	}
	if !strings.Contains(content, "lock_mode: git") {
		t.Error("Config should use git lock mode")
	}
	if !strings.Contains(content, "git_sync: true") {
		t.Error("Config should have git_sync enabled")
	}
	if !strings.Contains(content, "agent_id: claude-1") {
		t.Error("Config should have agent_id")
	}
	if !strings.Contains(content, "agent_label_prefix: agent") {
		t.Error("Config should have agent_label_prefix")
	}
}

func TestConfigGenerator_GenerateLocalBackend_NilOptions(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	generator := NewConfigGenerator()

	err = generator.GenerateLocalBackend(env, nil)
	if err != nil {
		t.Fatalf("GenerateLocalBackend with nil options failed: %v", err)
	}

	content, err := env.ReadFile(".backlog/config.yaml")
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	// Should have defaults
	if !strings.Contains(content, "workspace: local") {
		t.Error("Config should default to 'local' workspace name")
	}
	if !strings.Contains(content, "path: ./.backlog") {
		t.Error("Config should default to './.backlog' path")
	}
	if !strings.Contains(content, "lock_mode: file") {
		t.Error("Config should default to 'file' lock mode")
	}
}

func TestConfigGenerator_GenerateMultiWorkspace(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	generator := NewConfigGenerator()

	workspaces := map[string]*WorkspaceConfig{
		"local": {
			Backend: "local",
			Path:    "./.backlog",
		},
		"github": {
			Backend: "github",
			Repo:    "owner/repo",
		},
		"linear": {
			Backend: "linear",
			Team:    "ENG",
		},
	}

	err = generator.GenerateMultiWorkspace(env, workspaces, "local")
	if err != nil {
		t.Fatalf("GenerateMultiWorkspace failed: %v", err)
	}

	content, err := env.ReadFile(".backlog/config.yaml")
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	// Check all backends are present
	if !strings.Contains(content, "backend: local") {
		t.Error("Config missing local backend")
	}
	if !strings.Contains(content, "backend: github") {
		t.Error("Config missing github backend")
	}
	if !strings.Contains(content, "backend: linear") {
		t.Error("Config missing linear backend")
	}
	if !strings.Contains(content, "repo: owner/repo") {
		t.Error("Config missing github repo")
	}
	if !strings.Contains(content, "team: ENG") {
		t.Error("Config missing linear team")
	}
}

func TestConfigGenerator_GenerateMultiWorkspace_EmptyWorkspaces(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	generator := NewConfigGenerator()

	err = generator.GenerateMultiWorkspace(env, map[string]*WorkspaceConfig{}, "main")
	if err == nil {
		t.Error("Expected error for empty workspaces, got nil")
	}
	if !strings.Contains(err.Error(), "at least one workspace is required") {
		t.Errorf("Expected 'at least one workspace' error, got: %v", err)
	}
}

func TestConfigGenerator_Generate_WithGitHubOptions(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	generator := NewConfigGenerator()

	cfg := &Config{
		Version: 1,
		Workspaces: map[string]*WorkspaceConfig{
			"main": {
				Backend:          "github",
				Repo:             "owner/repo",
				Project:          1,
				StatusField:      "Status",
				AgentID:          "claude-main",
				AgentLabelPrefix: "agent",
				Default:          true,
			},
		},
	}

	err = generator.Generate(env, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := env.ReadFile(".backlog/config.yaml")
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if !strings.Contains(content, "repo: owner/repo") {
		t.Error("Config missing repo field")
	}
	if !strings.Contains(content, "project: 1") {
		t.Error("Config missing project field")
	}
	if !strings.Contains(content, "status_field: Status") {
		t.Error("Config missing status_field")
	}
}

func TestConfigGenerator_Generate_WithStatusMap(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	generator := NewConfigGenerator()

	cfg := &Config{
		Version: 1,
		Workspaces: map[string]*WorkspaceConfig{
			"main": {
				Backend: "github",
				Repo:    "owner/repo",
				StatusMap: map[string]any{
					"backlog": map[string]any{
						"state":  "open",
						"labels": []string{},
					},
					"todo": map[string]any{
						"state":  "open",
						"labels": []string{"ready"},
					},
				},
			},
		},
	}

	err = generator.Generate(env, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := env.ReadFile(".backlog/config.yaml")
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if !strings.Contains(content, "status_map:") {
		t.Error("Config missing status_map")
	}
}

func TestConfigGenerator_Generate_WithDefaultFilters(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	generator := NewConfigGenerator()

	cfg := &Config{
		Version: 1,
		Workspaces: map[string]*WorkspaceConfig{
			"backend-agent": {
				Backend: "github",
				Repo:    "myorg/myproject",
				DefaultFilters: map[string]any{
					"labels": []string{"backend", "api"},
				},
			},
		},
	}

	err = generator.Generate(env, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := env.ReadFile(".backlog/config.yaml")
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if !strings.Contains(content, "default_filters:") {
		t.Error("Config missing default_filters")
	}
}

func TestConfigGenerator_Generate_WithExtra(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	generator := NewConfigGenerator()

	cfg := &Config{
		Version: 1,
		Workspaces: map[string]*WorkspaceConfig{
			"main": {
				Backend: "local",
				Extra: map[string]any{
					"custom_field": "custom_value",
					"nested": map[string]any{
						"foo": "bar",
					},
				},
			},
		},
	}

	err = generator.Generate(env, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := env.ReadFile(".backlog/config.yaml")
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if !strings.Contains(content, "custom_field: custom_value") {
		t.Error("Config missing custom_field from Extra")
	}
}

func TestNewConfigGenerator(t *testing.T) {
	generator := NewConfigGenerator()
	if generator == nil {
		t.Error("NewConfigGenerator returned nil")
	}
}
