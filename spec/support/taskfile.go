// Package support provides test helpers and fixtures for the backlog CLI specs.
package support

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// TaskFile represents a parsed task markdown file with frontmatter.
type TaskFile struct {
	// Path is the absolute path to the task file
	Path string

	// Status is derived from the directory name (backlog, todo, in-progress, review, done)
	Status string

	// Frontmatter fields
	ID       string   `yaml:"id"`
	Title    string   `yaml:"title"`
	Priority string   `yaml:"priority,omitempty"`
	Assignee string   `yaml:"assignee,omitempty"`
	Labels   []string `yaml:"labels,omitempty"`
	AgentID  string   `yaml:"agent_id,omitempty"`
	Created  string   `yaml:"created,omitempty"`
	Updated  string   `yaml:"updated,omitempty"`

	// Description is the content after the frontmatter
	Description string

	// RawFrontmatter is the raw YAML frontmatter content
	RawFrontmatter string

	// RawContent is the raw content after frontmatter
	RawContent string

	// ParseErr holds any error that occurred during parsing
	ParseErr error
}

// TaskFileReader reads and parses task files from the .backlog directory.
type TaskFileReader struct {
	// BacklogDir is the path to the .backlog directory
	BacklogDir string
}

// NewTaskFileReader creates a new task file reader.
func NewTaskFileReader(backlogDir string) *TaskFileReader {
	return &TaskFileReader{
		BacklogDir: backlogDir,
	}
}

// ReadTask reads a task file by its ID.
// It searches all status directories to find the task.
func (r *TaskFileReader) ReadTask(id string) *TaskFile {
	statuses := []string{"backlog", "todo", "in-progress", "review", "done"}

	for _, status := range statuses {
		statusDir := filepath.Join(r.BacklogDir, status)
		entries, err := os.ReadDir(statusDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			// Check if filename starts with the task ID
			name := entry.Name()
			if strings.HasPrefix(name, id+"-") || name == id+".md" {
				path := filepath.Join(statusDir, name)
				return r.ReadTaskFile(path)
			}
		}
	}

	return &TaskFile{
		ParseErr: fmt.Errorf("task not found: %s", id),
	}
}

// ReadTaskFile reads and parses a task file at the given path.
func (r *TaskFileReader) ReadTaskFile(path string) *TaskFile {
	task := &TaskFile{
		Path: path,
	}

	// Derive status from directory name
	dir := filepath.Dir(path)
	task.Status = filepath.Base(dir)

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		task.ParseErr = fmt.Errorf("failed to read file: %w", err)
		return task
	}

	// Parse frontmatter and content
	frontmatter, body, err := parseFrontmatter(string(content))
	if err != nil {
		task.ParseErr = fmt.Errorf("failed to parse frontmatter: %w", err)
		return task
	}

	task.RawFrontmatter = frontmatter
	task.RawContent = body

	// Parse YAML frontmatter
	if err := yaml.Unmarshal([]byte(frontmatter), task); err != nil {
		task.ParseErr = fmt.Errorf("failed to parse YAML: %w", err)
		return task
	}

	// Extract description (content after frontmatter, excluding "## Description" header if present)
	task.Description = extractDescription(body)

	return task
}

// ListTasks lists all tasks in the backlog directory.
func (r *TaskFileReader) ListTasks() []*TaskFile {
	return r.ListTasksByStatus("")
}

// ListTasksByStatus lists tasks filtered by status.
// If status is empty, returns tasks from all statuses.
func (r *TaskFileReader) ListTasksByStatus(status string) []*TaskFile {
	var tasks []*TaskFile
	statuses := []string{"backlog", "todo", "in-progress", "review", "done"}

	if status != "" {
		statuses = []string{status}
	}

	for _, s := range statuses {
		statusDir := filepath.Join(r.BacklogDir, s)
		entries, err := os.ReadDir(statusDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			path := filepath.Join(statusDir, entry.Name())
			task := r.ReadTaskFile(path)
			if task.ParseErr == nil {
				tasks = append(tasks, task)
			}
		}
	}

	return tasks
}

// TaskExists checks if a task with the given ID exists.
func (r *TaskFileReader) TaskExists(id string) bool {
	task := r.ReadTask(id)
	return task.ParseErr == nil
}

// Valid returns true if the task was parsed successfully.
func (t *TaskFile) Valid() bool {
	return t.ParseErr == nil
}

// Error returns the parse error message, or empty string if no error.
func (t *TaskFile) Error() string {
	if t.ParseErr == nil {
		return ""
	}
	return t.ParseErr.Error()
}

// HasLabel checks if the task has a specific label.
func (t *TaskFile) HasLabel(label string) bool {
	for _, l := range t.Labels {
		if l == label {
			return true
		}
	}
	return false
}

// HasAgentLabel checks if the task has an agent label (e.g., "agent:claude-1").
func (t *TaskFile) HasAgentLabel(prefix string) bool {
	for _, l := range t.Labels {
		if strings.HasPrefix(l, prefix+":") {
			return true
		}
	}
	return false
}

// GetAgentFromLabel extracts the agent ID from an agent label.
// Returns empty string if no agent label is found.
func (t *TaskFile) GetAgentFromLabel(prefix string) string {
	for _, l := range t.Labels {
		if strings.HasPrefix(l, prefix+":") {
			return strings.TrimPrefix(l, prefix+":")
		}
	}
	return ""
}

// IsAssigned returns true if the task has an assignee.
func (t *TaskFile) IsAssigned() bool {
	return t.Assignee != "" && t.Assignee != "null"
}

// IsClaimed returns true if the task has an agent_id set.
func (t *TaskFile) IsClaimed() bool {
	return t.AgentID != ""
}

// GetField returns the value of a field by name.
// Supported fields: id, title, status, priority, assignee, agent_id, created, updated, description
func (t *TaskFile) GetField(name string) string {
	switch strings.ToLower(name) {
	case "id":
		return t.ID
	case "title":
		return t.Title
	case "status":
		return t.Status
	case "priority":
		return t.Priority
	case "assignee":
		return t.Assignee
	case "agent_id", "agentid":
		return t.AgentID
	case "created":
		return t.Created
	case "updated":
		return t.Updated
	case "description":
		return t.Description
	default:
		return ""
	}
}

// parseFrontmatter splits content into frontmatter and body.
// Frontmatter is delimited by "---" at the start and end.
func parseFrontmatter(content string) (frontmatter, body string, err error) {
	scanner := bufio.NewScanner(strings.NewReader(content))

	// Skip any leading whitespace/newlines and find the opening ---
	foundStart := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			foundStart = true
			break
		}
		if strings.TrimSpace(line) != "" {
			return "", content, fmt.Errorf("expected frontmatter to start with ---")
		}
	}

	if !foundStart {
		return "", content, fmt.Errorf("no frontmatter found")
	}

	// Collect frontmatter until closing ---
	var fmLines []string
	foundEnd := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			foundEnd = true
			break
		}
		fmLines = append(fmLines, line)
	}

	if !foundEnd {
		return "", content, fmt.Errorf("frontmatter not closed with ---")
	}

	frontmatter = strings.Join(fmLines, "\n")

	// Collect the rest as body
	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}

	body = strings.Join(bodyLines, "\n")

	return frontmatter, body, nil
}

// extractDescription extracts the description from the body content.
// It removes the "## Description" header if present.
func extractDescription(body string) string {
	body = strings.TrimSpace(body)

	// Check if it starts with "## Description"
	lines := strings.Split(body, "\n")
	startIndex := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.EqualFold(trimmed, "## description") {
			startIndex = i + 1
			break
		}
		// If first non-empty line isn't "## Description", keep everything
		break
	}

	if startIndex > 0 && startIndex < len(lines) {
		// Skip the header and any following blank line
		remaining := lines[startIndex:]
		for len(remaining) > 0 && strings.TrimSpace(remaining[0]) == "" {
			remaining = remaining[1:]
		}
		return strings.TrimSpace(strings.Join(remaining, "\n"))
	}

	return body
}
