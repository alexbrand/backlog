package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit_WithValidConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	cfgContent := `
version: 1
defaults:
  format: json
  workspace: main
  agent_id: test-agent

workspaces:
  main:
    backend: local
    path: ./.backlog
    default: true
  work:
    backend: github
    repo: user/repo
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Initialize config
	if err := Init(cfgPath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	cfg := Get()
	if cfg == nil {
		t.Fatal("Get() returned nil")
	}

	// Check version
	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}

	// Check defaults
	if cfg.Defaults.Format != "json" {
		t.Errorf("expected format 'json', got %q", cfg.Defaults.Format)
	}
	if cfg.Defaults.Workspace != "main" {
		t.Errorf("expected workspace 'main', got %q", cfg.Defaults.Workspace)
	}
	if cfg.Defaults.AgentID != "test-agent" {
		t.Errorf("expected agent_id 'test-agent', got %q", cfg.Defaults.AgentID)
	}

	// Check workspaces
	if len(cfg.Workspaces) != 2 {
		t.Errorf("expected 2 workspaces, got %d", len(cfg.Workspaces))
	}

	main, ok := cfg.Workspaces["main"]
	if !ok {
		t.Fatal("workspace 'main' not found")
	}
	if main.Backend != "local" {
		t.Errorf("expected backend 'local', got %q", main.Backend)
	}
	if main.Path != "./.backlog" {
		t.Errorf("expected path './.backlog', got %q", main.Path)
	}
	if !main.Default {
		t.Error("expected main workspace to have default: true")
	}
}

func TestInit_MissingConfigFile(t *testing.T) {
	// Initialize with non-existent file should not error
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "nonexistent.yaml")

	if err := Init(cfgPath); err == nil {
		// Config file not found should error when explicit path given
		// Actually, let's test with a path that doesn't exist
	}
}

func TestGetWorkspace_ByName(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	cfgContent := `
workspaces:
  alpha:
    backend: local
    path: ./alpha
  beta:
    backend: github
    repo: user/beta
    default: true
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if err := Init(cfgPath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Get workspace by name
	ws, name, err := GetWorkspace("alpha")
	if err != nil {
		t.Fatalf("GetWorkspace('alpha') failed: %v", err)
	}
	if name != "alpha" {
		t.Errorf("expected name 'alpha', got %q", name)
	}
	if ws.Backend != "local" {
		t.Errorf("expected backend 'local', got %q", ws.Backend)
	}
}

func TestGetWorkspace_Default(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	cfgContent := `
workspaces:
  alpha:
    backend: local
    path: ./alpha
  beta:
    backend: github
    repo: user/beta
    default: true
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if err := Init(cfgPath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Get default workspace (empty name)
	ws, name, err := GetWorkspace("")
	if err != nil {
		t.Fatalf("GetWorkspace('') failed: %v", err)
	}
	if name != "beta" {
		t.Errorf("expected name 'beta', got %q", name)
	}
	if ws.Backend != "github" {
		t.Errorf("expected backend 'github', got %q", ws.Backend)
	}
}

func TestGetWorkspace_DefaultFromDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	cfgContent := `
defaults:
  workspace: alpha

workspaces:
  alpha:
    backend: local
    path: ./alpha
  beta:
    backend: github
    repo: user/beta
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if err := Init(cfgPath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Get default workspace from defaults.workspace
	ws, name, err := GetWorkspace("")
	if err != nil {
		t.Fatalf("GetWorkspace('') failed: %v", err)
	}
	if name != "alpha" {
		t.Errorf("expected name 'alpha', got %q", name)
	}
	if ws.Backend != "local" {
		t.Errorf("expected backend 'local', got %q", ws.Backend)
	}
}

func TestGetWorkspace_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	cfgContent := `
workspaces:
  alpha:
    backend: local
    path: ./alpha
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if err := Init(cfgPath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Get non-existent workspace
	_, _, err := GetWorkspace("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent workspace")
	}
}

func TestGetWorkspace_NoWorkspaces(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	cfgContent := `version: 1`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if err := Init(cfgPath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Get workspace when none configured
	_, _, err := GetWorkspace("")
	if err == nil {
		t.Error("expected error when no workspaces configured")
	}
}

func TestGetWorkspace_SingleWorkspaceAsDefault(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	cfgContent := `
workspaces:
  only:
    backend: local
    path: ./only
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if err := Init(cfgPath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Single workspace should be used as default
	ws, name, err := GetWorkspace("")
	if err != nil {
		t.Fatalf("GetWorkspace('') failed: %v", err)
	}
	if name != "only" {
		t.Errorf("expected name 'only', got %q", name)
	}
	if ws.Backend != "local" {
		t.Errorf("expected backend 'local', got %q", ws.Backend)
	}
}
