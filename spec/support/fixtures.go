// Package support provides test helpers and fixtures for the backlog CLI specs.
package support

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// CommentFixture represents a comment for fixture loading.
type CommentFixture struct {
	Author string `yaml:"author"`
	Date   string `yaml:"date"`
	Body   string `yaml:"body"`
}

// TaskFixture represents a task for fixture loading.
type TaskFixture struct {
	ID          string           `yaml:"id"`
	Title       string           `yaml:"title"`
	Description string           `yaml:"description,omitempty"`
	Status      string           `yaml:"status"`
	Priority    string           `yaml:"priority,omitempty"`
	Assignee    string           `yaml:"assignee,omitempty"`
	Labels      []string         `yaml:"labels,omitempty"`
	AgentID     string           `yaml:"agent_id,omitempty"`
	Comments    []CommentFixture `yaml:"comments,omitempty"`
}

// BacklogFixture represents a complete backlog directory fixture.
type BacklogFixture struct {
	Tasks  []TaskFixture     `yaml:"tasks"`
	Config map[string]any    `yaml:"config,omitempty"`
	Locks  map[string]string `yaml:"locks,omitempty"` // task ID -> agent ID
}

// FixtureLoader loads pre-built .backlog/ directories from fixtures.
type FixtureLoader struct {
	// FixturesDir is the directory containing fixture files
	FixturesDir string
}

// NewFixtureLoader creates a new fixture loader.
// If fixturesDir is empty, it defaults to "fixtures" in the spec directory.
func NewFixtureLoader(fixturesDir string) *FixtureLoader {
	if fixturesDir == "" {
		fixturesDir = "fixtures"
	}
	return &FixtureLoader{
		FixturesDir: fixturesDir,
	}
}

// LoadFixture loads a fixture by name and applies it to the test environment.
// The fixture can be either a YAML file (name.yaml) or a directory (name/).
func (l *FixtureLoader) LoadFixture(env *TestEnv, name string) error {
	// Try YAML file first
	yamlPath := filepath.Join(l.FixturesDir, name+".yaml")
	if _, err := os.Stat(yamlPath); err == nil {
		return l.loadYAMLFixture(env, yamlPath)
	}

	// Try directory
	dirPath := filepath.Join(l.FixturesDir, name)
	if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
		return l.loadDirectoryFixture(env, dirPath)
	}

	return fmt.Errorf("fixture not found: %s (tried %s.yaml and %s/)", name, name, name)
}

// LoadFromYAML loads a fixture from a YAML string.
func (l *FixtureLoader) LoadFromYAML(env *TestEnv, yamlContent string) error {
	var fixture BacklogFixture
	if err := yaml.Unmarshal([]byte(yamlContent), &fixture); err != nil {
		return fmt.Errorf("failed to parse fixture YAML: %w", err)
	}
	return l.applyFixture(env, &fixture)
}

// LoadTasks loads tasks from a slice of TaskFixture structs.
// This is useful for programmatic fixture creation in tests.
func (l *FixtureLoader) LoadTasks(env *TestEnv, tasks []TaskFixture) error {
	fixture := &BacklogFixture{Tasks: tasks}
	return l.applyFixture(env, fixture)
}

// loadYAMLFixture loads a fixture from a YAML file.
func (l *FixtureLoader) loadYAMLFixture(env *TestEnv, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read fixture file: %w", err)
	}

	var fixture BacklogFixture
	if err := yaml.Unmarshal(content, &fixture); err != nil {
		return fmt.Errorf("failed to parse fixture YAML: %w", err)
	}

	return l.applyFixture(env, &fixture)
}

// loadDirectoryFixture loads a fixture from a directory by copying it.
func (l *FixtureLoader) loadDirectoryFixture(env *TestEnv, srcDir string) error {
	return copyDir(srcDir, env.BacklogDir)
}

// applyFixture applies a parsed fixture to the test environment.
func (l *FixtureLoader) applyFixture(env *TestEnv, fixture *BacklogFixture) error {
	// Ensure .backlog directory exists
	if err := env.CreateBacklogDir(); err != nil {
		return fmt.Errorf("failed to create backlog directory: %w", err)
	}

	// Create config.yaml if provided
	if fixture.Config != nil {
		configContent, err := yaml.Marshal(fixture.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		if err := env.CreateFile(".backlog/config.yaml", string(configContent)); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
	}

	// Create task files
	for _, task := range fixture.Tasks {
		if err := l.createTaskFile(env, &task); err != nil {
			return fmt.Errorf("failed to create task %s: %w", task.ID, err)
		}
	}

	// Create lock files
	for taskID, agentID := range fixture.Locks {
		if err := l.createLockFile(env, taskID, agentID); err != nil {
			return fmt.Errorf("failed to create lock for %s: %w", taskID, err)
		}
	}

	return nil
}

// createTaskFile creates a task markdown file in the appropriate status directory.
func (l *FixtureLoader) createTaskFile(env *TestEnv, task *TaskFixture) error {
	// Default status to "backlog" if not specified
	status := task.Status
	if status == "" {
		status = "backlog"
	}

	// Validate status
	validStatuses := map[string]bool{
		"backlog":     true,
		"todo":        true,
		"in-progress": true,
		"review":      true,
		"done":        true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}

	// Build filename
	filename := fmt.Sprintf("%s-%s.md", task.ID, slugify(task.Title))

	// Build frontmatter
	var frontmatter strings.Builder
	frontmatter.WriteString("---\n")
	frontmatter.WriteString(fmt.Sprintf("id: %q\n", task.ID))
	frontmatter.WriteString(fmt.Sprintf("title: %s\n", task.Title))

	if task.Priority != "" {
		frontmatter.WriteString(fmt.Sprintf("priority: %s\n", task.Priority))
	}

	if task.Assignee != "" {
		frontmatter.WriteString(fmt.Sprintf("assignee: %s\n", task.Assignee))
	} else {
		frontmatter.WriteString("assignee: null\n")
	}

	if len(task.Labels) > 0 {
		frontmatter.WriteString("labels: [")
		for i, label := range task.Labels {
			if i > 0 {
				frontmatter.WriteString(", ")
			}
			frontmatter.WriteString(label)
		}
		frontmatter.WriteString("]\n")
	} else {
		frontmatter.WriteString("labels: []\n")
	}

	if task.AgentID != "" {
		frontmatter.WriteString(fmt.Sprintf("agent_id: %s\n", task.AgentID))
	}

	frontmatter.WriteString("---\n")

	// Build content
	var content strings.Builder
	content.WriteString(frontmatter.String())
	content.WriteString("\n")

	if task.Description != "" {
		content.WriteString("## Description\n\n")
		content.WriteString(task.Description)
		content.WriteString("\n")
	}

	// Add comments section if there are comments
	if len(task.Comments) > 0 {
		content.WriteString("\n## Comments\n")
		for _, comment := range task.Comments {
			content.WriteString(fmt.Sprintf("\n### %s @%s\n", comment.Date, comment.Author))
			content.WriteString(comment.Body)
			content.WriteString("\n")
		}
	}

	// Write file
	path := filepath.Join(".backlog", status, filename)
	return env.CreateFile(path, content.String())
}

// createLockFile creates a lock file for a task.
func (l *FixtureLoader) createLockFile(env *TestEnv, taskID, agentID string) error {
	content := fmt.Sprintf("agent: %s\nclaimed_at: 2025-01-01T00:00:00Z\nexpires_at: 2025-01-01T00:30:00Z\n", agentID)
	path := filepath.Join(".backlog", ".locks", taskID+".lock")
	return env.CreateFile(path, content)
}

// slugify converts a title to a URL-friendly slug.
func slugify(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)
	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove non-alphanumeric characters (except hyphens)
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	// Remove multiple consecutive hyphens
	slug = result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	// Trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")
	return slug
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			content, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, content, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}
