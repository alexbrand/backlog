package local

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/alexbrand/backlog/internal/backend"
	"gopkg.in/yaml.v3"
)

// taskFrontmatter represents the YAML frontmatter of a task file.
type taskFrontmatter struct {
	ID        string           `yaml:"id"`
	Title     string           `yaml:"title"`
	Priority  backend.Priority `yaml:"priority,omitempty"`
	Assignee  string           `yaml:"assignee,omitempty"`
	Labels    []string         `yaml:"labels,omitempty"`
	Blocks    []string         `yaml:"blocks,omitempty"`
	BlockedBy []string         `yaml:"blocked_by,omitempty"`
	SortOrder float64          `yaml:"sort_order,omitempty"`
	Created   time.Time        `yaml:"created"`
	Updated   time.Time        `yaml:"updated"`
}

// readTaskFile reads a task from a markdown file with YAML frontmatter.
func (l *Local) readTaskFile(filePath string, status backend.Status) (*backend.Task, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	frontmatter, body, err := parseFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	var fm taskFrontmatter
	if err := yaml.Unmarshal(frontmatter, &fm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal frontmatter: %w", err)
	}

	// Extract description from body (everything before ## Comments section)
	description, comments := parseBody(body)

	task := &backend.Task{
		ID:          fm.ID,
		Title:       fm.Title,
		Description: description,
		Status:      status,
		Priority:    fm.Priority,
		Assignee:    fm.Assignee,
		Labels:      fm.Labels,
		SortOrder:   fm.SortOrder,
		Created:     fm.Created,
		Updated:     fm.Updated,
	}

	// Set default priority if empty
	if task.Priority == "" {
		task.Priority = backend.PriorityNone
	}

	// Initialize meta for comments and relations
	if len(comments) > 0 || len(fm.Blocks) > 0 || len(fm.BlockedBy) > 0 {
		if task.Meta == nil {
			task.Meta = make(map[string]any)
		}
		if len(comments) > 0 {
			task.Meta["comments"] = comments
		}
		if len(fm.Blocks) > 0 {
			task.Meta["blocks"] = fm.Blocks
		}
		if len(fm.BlockedBy) > 0 {
			task.Meta["blocked_by"] = fm.BlockedBy
		}
	}

	return task, nil
}

// writeTask writes a task to a markdown file with YAML frontmatter.
func (l *Local) writeTask(task *backend.Task) error {
	// Ensure the status directory exists
	statusDir := filepath.Join(l.path, string(task.Status))
	if err := os.MkdirAll(statusDir, 0755); err != nil {
		return fmt.Errorf("failed to create status directory: %w", err)
	}

	// Generate filename
	filename := generateFilename(task.ID, task.Title)
	filePath := filepath.Join(statusDir, filename)

	// Extract blocks/blocked_by from meta
	var blocks, blockedBy []string
	if task.Meta != nil {
		if b, ok := task.Meta["blocks"].([]string); ok {
			blocks = b
		}
		if b, ok := task.Meta["blocked_by"].([]string); ok {
			blockedBy = b
		}
	}

	// Build frontmatter
	fm := taskFrontmatter{
		ID:        task.ID,
		Title:     task.Title,
		Priority:  task.Priority,
		Assignee:  task.Assignee,
		Labels:    task.Labels,
		Blocks:    blocks,
		BlockedBy: blockedBy,
		SortOrder: task.SortOrder,
		Created:   task.Created,
		Updated:   task.Updated,
	}

	frontmatterBytes, err := yaml.Marshal(&fm)
	if err != nil {
		return fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	// Build file content
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(frontmatterBytes)
	buf.WriteString("---\n\n")

	// Add description if present
	if task.Description != "" {
		buf.WriteString("## Description\n\n")
		buf.WriteString(task.Description)
		buf.WriteString("\n")
	}

	// Add comments if present
	if task.Meta != nil {
		if comments, ok := task.Meta["comments"].([]backend.Comment); ok && len(comments) > 0 {
			buf.WriteString("\n## Comments\n")
			for _, comment := range comments {
				buf.WriteString(fmt.Sprintf("\n### %s @%s\n\n",
					comment.Created.Format("2006-01-02"),
					comment.Author))
				buf.WriteString(comment.Body)
				buf.WriteString("\n")
			}
		}
	}

	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// parseFrontmatter parses YAML frontmatter from markdown content.
// Returns the frontmatter bytes and the remaining body.
func parseFrontmatter(content []byte) ([]byte, []byte, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))

	// Check for opening delimiter
	if !scanner.Scan() {
		return nil, nil, fmt.Errorf("empty file")
	}
	if strings.TrimSpace(scanner.Text()) != "---" {
		return nil, nil, fmt.Errorf("file does not start with frontmatter delimiter")
	}

	// Read frontmatter until closing delimiter
	var frontmatter bytes.Buffer
	foundClose := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			foundClose = true
			break
		}
		frontmatter.WriteString(line)
		frontmatter.WriteString("\n")
	}

	if !foundClose {
		return nil, nil, fmt.Errorf("frontmatter not closed")
	}

	// Read the rest as body
	var body bytes.Buffer
	for scanner.Scan() {
		body.WriteString(scanner.Text())
		body.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading file: %w", err)
	}

	return frontmatter.Bytes(), body.Bytes(), nil
}

// parseBody parses the markdown body to extract description and comments.
func parseBody(body []byte) (string, []backend.Comment) {
	content := string(body)

	// Find the ## Comments section
	commentsIdx := strings.Index(content, "\n## Comments\n")
	if commentsIdx == -1 {
		// No comments section, entire body is description
		return extractDescription(content), nil
	}

	// Split into description and comments
	descPart := content[:commentsIdx]
	commentsPart := content[commentsIdx+len("\n## Comments\n"):]

	description := extractDescription(descPart)
	comments := parseComments(commentsPart)

	return description, comments
}

// extractDescription extracts the description from the body part.
func extractDescription(content string) string {
	content = strings.TrimSpace(content)

	// Remove ## Description header if present
	if strings.HasPrefix(content, "## Description\n") {
		content = strings.TrimPrefix(content, "## Description\n")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "## Description\r\n") {
		content = strings.TrimPrefix(content, "## Description\r\n")
		content = strings.TrimSpace(content)
	}

	return content
}

// parseComments parses the comments section of a task file.
func parseComments(content string) []backend.Comment {
	var comments []backend.Comment

	// Match comment headers: ### 2025-01-16 @alex
	commentHeaderRe := regexp.MustCompile(`###\s+(\d{4}-\d{2}-\d{2})\s+@(\S+)`)

	// Split by comment headers
	parts := commentHeaderRe.Split(content, -1)
	matches := commentHeaderRe.FindAllStringSubmatch(content, -1)

	for i, match := range matches {
		if i+1 >= len(parts) {
			break
		}

		dateStr := match[1]
		author := match[2]
		body := strings.TrimSpace(parts[i+1])

		created, _ := time.Parse("2006-01-02", dateStr)

		comments = append(comments, backend.Comment{
			ID:      fmt.Sprintf("c%d", i+1),
			Author:  author,
			Body:    body,
			Created: created,
		})
	}

	return comments
}

// generateFilename generates a filename from task ID and title.
func generateFilename(id, title string) string {
	// Sanitize title for filename
	slug := slugify(title)
	if slug == "" {
		return id + ".md"
	}

	// Limit slug length
	if len(slug) > 50 {
		slug = slug[:50]
	}

	return fmt.Sprintf("%s-%s.md", id, slug)
}

// slugify converts a string to a URL-safe slug.
func slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces and underscores with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove any character that isn't alphanumeric or hyphen
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	// Replace multiple hyphens with single hyphen
	s = result.String()
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	// Trim hyphens from ends
	s = strings.Trim(s, "-")

	return s
}
