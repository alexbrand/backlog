package support

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewTestEnv(t *testing.T) {
	originalDir, _ := os.Getwd()

	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("NewTestEnv() error = %v", err)
	}
	defer env.Cleanup()

	// Check temp directory was created
	if env.TempDir == "" {
		t.Error("TempDir should not be empty")
	}

	if !strings.Contains(env.TempDir, "backlog-test-") {
		t.Errorf("TempDir should contain 'backlog-test-', got %s", env.TempDir)
	}

	// Check we changed to temp directory
	currentDir, _ := os.Getwd()
	if currentDir != env.TempDir {
		t.Errorf("Should be in temp directory, got %s, want %s", currentDir, env.TempDir)
	}

	// Check original directory was saved
	if env.OriginalDir != originalDir {
		t.Errorf("OriginalDir = %s, want %s", env.OriginalDir, originalDir)
	}

	// Check BacklogDir is set correctly
	expectedBacklogDir := filepath.Join(env.TempDir, ".backlog")
	if env.BacklogDir != expectedBacklogDir {
		t.Errorf("BacklogDir = %s, want %s", env.BacklogDir, expectedBacklogDir)
	}
}

func TestTestEnv_Cleanup(t *testing.T) {
	originalDir, _ := os.Getwd()

	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("NewTestEnv() error = %v", err)
	}

	tempDir := env.TempDir

	// Cleanup should restore directory and remove temp dir
	err = env.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	// Check we're back in original directory
	currentDir, _ := os.Getwd()
	if currentDir != originalDir {
		t.Errorf("After cleanup, should be in %s, got %s", originalDir, currentDir)
	}

	// Check temp directory was removed
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("Temp directory should be removed after cleanup")
	}
}

func TestTestEnv_SetEnv(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("NewTestEnv() error = %v", err)
	}
	defer env.Cleanup()

	// Set a new env var
	env.SetEnv("TEST_BACKLOG_VAR", "test-value")

	if got := os.Getenv("TEST_BACKLOG_VAR"); got != "test-value" {
		t.Errorf("SetEnv did not set variable, got %s, want test-value", got)
	}

	// Cleanup should restore (unset) the variable
	env.Cleanup()

	if got := os.Getenv("TEST_BACKLOG_VAR"); got != "" {
		t.Errorf("After cleanup, env var should be unset, got %s", got)
	}
}

func TestTestEnv_CreateBacklogDir(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("NewTestEnv() error = %v", err)
	}
	defer env.Cleanup()

	err = env.CreateBacklogDir()
	if err != nil {
		t.Fatalf("CreateBacklogDir() error = %v", err)
	}

	// Check all directories exist
	expectedDirs := []string{
		".backlog",
		".backlog/backlog",
		".backlog/todo",
		".backlog/in-progress",
		".backlog/review",
		".backlog/done",
		".backlog/.locks",
	}

	for _, dir := range expectedDirs {
		fullPath := filepath.Join(env.TempDir, dir)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Errorf("Directory %s should exist: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s should be a directory", dir)
		}
	}
}

func TestTestEnv_CreateFile(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("NewTestEnv() error = %v", err)
	}
	defer env.Cleanup()

	content := "test content"
	err = env.CreateFile("test/nested/file.txt", content)
	if err != nil {
		t.Fatalf("CreateFile() error = %v", err)
	}

	// Read it back
	got, err := env.ReadFile("test/nested/file.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if got != content {
		t.Errorf("ReadFile() = %s, want %s", got, content)
	}
}

func TestTestEnv_FileExists(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("NewTestEnv() error = %v", err)
	}
	defer env.Cleanup()

	// File shouldn't exist yet
	if env.FileExists("nonexistent.txt") {
		t.Error("FileExists() should return false for nonexistent file")
	}

	// Create a file
	env.CreateFile("exists.txt", "content")

	// Now it should exist
	if !env.FileExists("exists.txt") {
		t.Error("FileExists() should return true for existing file")
	}
}

func TestTestEnv_Path(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("NewTestEnv() error = %v", err)
	}
	defer env.Cleanup()

	got := env.Path("some/path")
	want := filepath.Join(env.TempDir, "some/path")

	if got != want {
		t.Errorf("Path() = %s, want %s", got, want)
	}
}
