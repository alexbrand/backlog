// Package support provides test helpers and fixtures for the backlog CLI specs.
package support

import (
	"os"
	"path/filepath"
)

// TestEnv holds the test environment state for a scenario.
type TestEnv struct {
	// TempDir is the temporary directory for this test run
	TempDir string
	// BacklogDir is the .backlog directory within TempDir
	BacklogDir string
	// OriginalDir is the directory we were in before the test
	OriginalDir string
	// OriginalEnv stores original environment variables to restore
	OriginalEnv map[string]string
}

// NewTestEnv creates a new isolated test environment.
// It creates a temporary directory and changes into it.
func NewTestEnv() (*TestEnv, error) {
	// Get current directory to restore later
	originalDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "backlog-test-*")
	if err != nil {
		return nil, err
	}

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		os.RemoveAll(tempDir)
		return nil, err
	}

	return &TestEnv{
		TempDir:     tempDir,
		BacklogDir:  filepath.Join(tempDir, ".backlog"),
		OriginalDir: originalDir,
		OriginalEnv: make(map[string]string),
	}, nil
}

// Cleanup removes the temporary directory and restores the original state.
func (e *TestEnv) Cleanup() error {
	// Restore original directory
	if err := os.Chdir(e.OriginalDir); err != nil {
		return err
	}

	// Restore original environment variables
	for key, value := range e.OriginalEnv {
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}

	// Remove temp directory
	return os.RemoveAll(e.TempDir)
}

// SetEnv sets an environment variable and stores the original value for restoration.
func (e *TestEnv) SetEnv(key, value string) {
	if _, exists := e.OriginalEnv[key]; !exists {
		e.OriginalEnv[key] = os.Getenv(key)
	}
	os.Setenv(key, value)
}

// UnsetEnv unsets an environment variable and stores the original value for restoration.
func (e *TestEnv) UnsetEnv(key string) {
	if _, exists := e.OriginalEnv[key]; !exists {
		e.OriginalEnv[key] = os.Getenv(key)
	}
	os.Unsetenv(key)
}

// CreateBacklogDir creates the .backlog directory structure.
// This simulates what `backlog init` would do.
func (e *TestEnv) CreateBacklogDir() error {
	// Create main .backlog directory
	if err := os.MkdirAll(e.BacklogDir, 0755); err != nil {
		return err
	}

	// Create status directories
	statusDirs := []string{"backlog", "todo", "in-progress", "review", "done"}
	for _, dir := range statusDirs {
		if err := os.MkdirAll(filepath.Join(e.BacklogDir, dir), 0755); err != nil {
			return err
		}
	}

	// Create .locks directory
	if err := os.MkdirAll(filepath.Join(e.BacklogDir, ".locks"), 0755); err != nil {
		return err
	}

	return nil
}

// CreateFile creates a file with the given content within the temp directory.
func (e *TestEnv) CreateFile(relativePath, content string) error {
	fullPath := filepath.Join(e.TempDir, relativePath)

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}

// ReadFile reads a file from the temp directory.
func (e *TestEnv) ReadFile(relativePath string) (string, error) {
	fullPath := filepath.Join(e.TempDir, relativePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// FileExists checks if a file exists within the temp directory.
func (e *TestEnv) FileExists(relativePath string) bool {
	fullPath := filepath.Join(e.TempDir, relativePath)
	_, err := os.Stat(fullPath)
	return err == nil
}

// Path returns the full path for a relative path within the temp directory.
func (e *TestEnv) Path(relativePath string) string {
	return filepath.Join(e.TempDir, relativePath)
}
