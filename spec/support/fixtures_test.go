package support

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"Implement Auth Flow", "implement-auth-flow"},
		{"Fix bug #123", "fix-bug-123"},
		{"Add rate-limiting", "add-rate-limiting"},
		{"Test!@#$%^&*()", "test"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"--Leading Trailing--", "leading-trailing"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := slugify(tt.input)
			if result != tt.expected {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFixtureLoader_LoadFromYAML(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	loader := NewFixtureLoader("")

	yaml := `
tasks:
  - id: "001"
    title: Implement auth flow
    status: todo
    priority: high
    labels: [feature, auth]
  - id: "002"
    title: Fix login bug
    status: in-progress
    priority: urgent
    assignee: alex
`

	err = loader.LoadFromYAML(env, yaml)
	if err != nil {
		t.Fatalf("LoadFromYAML failed: %v", err)
	}

	// Check that task files were created
	if !env.FileExists(".backlog/todo/001-implement-auth-flow.md") {
		t.Error("Task 001 file not created in todo directory")
	}
	if !env.FileExists(".backlog/in-progress/002-fix-login-bug.md") {
		t.Error("Task 002 file not created in in-progress directory")
	}

	// Check task 001 content
	content, err := env.ReadFile(".backlog/todo/001-implement-auth-flow.md")
	if err != nil {
		t.Fatalf("Failed to read task 001: %v", err)
	}

	if !strings.Contains(content, `id: "001"`) {
		t.Error("Task 001 missing id field")
	}
	if !strings.Contains(content, "title: Implement auth flow") {
		t.Error("Task 001 missing title field")
	}
	if !strings.Contains(content, "priority: high") {
		t.Error("Task 001 missing priority field")
	}
	if !strings.Contains(content, "labels: [feature, auth]") {
		t.Error("Task 001 missing labels field")
	}

	// Check task 002 content
	content, err = env.ReadFile(".backlog/in-progress/002-fix-login-bug.md")
	if err != nil {
		t.Fatalf("Failed to read task 002: %v", err)
	}

	if !strings.Contains(content, "assignee: alex") {
		t.Error("Task 002 missing assignee field")
	}
	if !strings.Contains(content, "priority: urgent") {
		t.Error("Task 002 missing priority field")
	}
}

func TestFixtureLoader_LoadTasks(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	loader := NewFixtureLoader("")

	tasks := []TaskFixture{
		{
			ID:       "003",
			Title:    "Add tests",
			Status:   "backlog",
			Priority: "medium",
		},
	}

	err = loader.LoadTasks(env, tasks)
	if err != nil {
		t.Fatalf("LoadTasks failed: %v", err)
	}

	if !env.FileExists(".backlog/backlog/003-add-tests.md") {
		t.Error("Task 003 file not created")
	}
}

func TestFixtureLoader_LoadFromYAML_WithDescription(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	loader := NewFixtureLoader("")

	yaml := `
tasks:
  - id: "004"
    title: Complex task
    status: review
    description: |
      This is a multi-line description.

      It has multiple paragraphs.
`

	err = loader.LoadFromYAML(env, yaml)
	if err != nil {
		t.Fatalf("LoadFromYAML failed: %v", err)
	}

	content, err := env.ReadFile(".backlog/review/004-complex-task.md")
	if err != nil {
		t.Fatalf("Failed to read task 004: %v", err)
	}

	if !strings.Contains(content, "## Description") {
		t.Error("Task 004 missing description header")
	}
	if !strings.Contains(content, "multi-line description") {
		t.Error("Task 004 missing description content")
	}
}

func TestFixtureLoader_LoadFromYAML_WithConfig(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	loader := NewFixtureLoader("")

	yaml := `
config:
  version: 1
  defaults:
    format: table
    workspace: main
tasks:
  - id: "005"
    title: Test task
    status: todo
`

	err = loader.LoadFromYAML(env, yaml)
	if err != nil {
		t.Fatalf("LoadFromYAML failed: %v", err)
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
}

func TestFixtureLoader_LoadFromYAML_WithLocks(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	loader := NewFixtureLoader("")

	yaml := `
tasks:
  - id: "006"
    title: Locked task
    status: in-progress
    agent_id: claude-1
locks:
  "006": claude-1
`

	err = loader.LoadFromYAML(env, yaml)
	if err != nil {
		t.Fatalf("LoadFromYAML failed: %v", err)
	}

	if !env.FileExists(".backlog/.locks/006.lock") {
		t.Error("Lock file not created")
	}

	content, err := env.ReadFile(".backlog/.locks/006.lock")
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	if !strings.Contains(content, "agent: claude-1") {
		t.Error("Lock file missing agent field")
	}
}

func TestFixtureLoader_LoadFromYAML_InvalidStatus(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	loader := NewFixtureLoader("")

	yaml := `
tasks:
  - id: "007"
    title: Invalid task
    status: invalid-status
`

	err = loader.LoadFromYAML(env, yaml)
	if err == nil {
		t.Error("Expected error for invalid status, got nil")
	}
	if !strings.Contains(err.Error(), "invalid status") {
		t.Errorf("Expected 'invalid status' error, got: %v", err)
	}
}

func TestFixtureLoader_LoadFromYAML_DefaultStatus(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	loader := NewFixtureLoader("")

	yaml := `
tasks:
  - id: "008"
    title: No status task
`

	err = loader.LoadFromYAML(env, yaml)
	if err != nil {
		t.Fatalf("LoadFromYAML failed: %v", err)
	}

	// Should default to backlog status
	if !env.FileExists(".backlog/backlog/008-no-status-task.md") {
		t.Error("Task 008 file not created in backlog directory (default status)")
	}
}

func TestFixtureLoader_LoadFixture_YAMLFile(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	// Create fixtures directory and YAML file
	fixturesDir := filepath.Join(env.TempDir, "fixtures")
	if err := os.MkdirAll(fixturesDir, 0755); err != nil {
		t.Fatalf("Failed to create fixtures dir: %v", err)
	}

	yamlContent := `
tasks:
  - id: "009"
    title: Fixture task
    status: todo
`
	if err := os.WriteFile(filepath.Join(fixturesDir, "test-fixture.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create fixture file: %v", err)
	}

	loader := NewFixtureLoader(fixturesDir)

	err = loader.LoadFixture(env, "test-fixture")
	if err != nil {
		t.Fatalf("LoadFixture failed: %v", err)
	}

	if !env.FileExists(".backlog/todo/009-fixture-task.md") {
		t.Error("Task 009 file not created")
	}
}

func TestFixtureLoader_LoadFixture_Directory(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	// Create fixtures directory with a subdirectory fixture
	fixturesDir := filepath.Join(env.TempDir, "fixtures")
	fixtureDir := filepath.Join(fixturesDir, "dir-fixture")
	todoDir := filepath.Join(fixtureDir, "todo")
	if err := os.MkdirAll(todoDir, 0755); err != nil {
		t.Fatalf("Failed to create fixture dir: %v", err)
	}

	// Create a task file in the fixture directory
	taskContent := `---
id: "010"
title: Directory fixture task
---
`
	if err := os.WriteFile(filepath.Join(todoDir, "010-directory-fixture-task.md"), []byte(taskContent), 0644); err != nil {
		t.Fatalf("Failed to create task file: %v", err)
	}

	loader := NewFixtureLoader(fixturesDir)

	err = loader.LoadFixture(env, "dir-fixture")
	if err != nil {
		t.Fatalf("LoadFixture failed: %v", err)
	}

	if !env.FileExists(".backlog/todo/010-directory-fixture-task.md") {
		t.Error("Task 010 file not copied from directory fixture")
	}
}

func TestFixtureLoader_LoadFixture_NotFound(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	loader := NewFixtureLoader(filepath.Join(env.TempDir, "fixtures"))

	err = loader.LoadFixture(env, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent fixture, got nil")
	}
	if !strings.Contains(err.Error(), "fixture not found") {
		t.Errorf("Expected 'fixture not found' error, got: %v", err)
	}
}

func TestFixtureLoader_AllStatuses(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("Failed to create test env: %v", err)
	}
	defer env.Cleanup()

	loader := NewFixtureLoader("")

	statuses := []string{"backlog", "todo", "in-progress", "review", "done"}
	for i, status := range statuses {
		tasks := []TaskFixture{
			{
				ID:     fmt.Sprintf("0%d0", i+1),
				Title:  fmt.Sprintf("Task in %s", status),
				Status: status,
			},
		}
		err = loader.LoadTasks(env, tasks)
		if err != nil {
			t.Fatalf("LoadTasks failed for status %s: %v", status, err)
		}
	}

	// Verify all files were created in correct directories
	for i, status := range statuses {
		id := fmt.Sprintf("0%d0", i+1)
		path := fmt.Sprintf(".backlog/%s/%s-task-in-%s.md", status, id, status)
		if !env.FileExists(path) {
			t.Errorf("Task file not created for status %s at path %s", status, path)
		}
	}
}

