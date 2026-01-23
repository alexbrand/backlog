package support

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple arguments",
			input:    "list --status=todo",
			expected: []string{"list", "--status=todo"},
		},
		{
			name:     "multiple flags",
			input:    "list --status=todo -f json",
			expected: []string{"list", "--status=todo", "-f", "json"},
		},
		{
			name:     "double quoted string",
			input:    `add "My new task"`,
			expected: []string{"add", "My new task"},
		},
		{
			name:     "single quoted string",
			input:    `add 'My new task'`,
			expected: []string{"add", "My new task"},
		},
		{
			name:     "mixed quotes",
			input:    `add "Task with 'inner' quotes"`,
			expected: []string{"add", "Task with 'inner' quotes"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "multiple spaces",
			input:    "list    --status=todo",
			expected: []string{"list", "--status=todo"},
		},
		{
			name:     "complex command",
			input:    `edit GH-123 --title="New title" --priority=high`,
			expected: []string{"edit", "GH-123", "--title=New title", "--priority=high"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseArgs(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseArgs(%q) = %v (len %d), want %v (len %d)",
					tt.input, result, len(result), tt.expected, len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("parseArgs(%q)[%d] = %q, want %q",
						tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestNewCLIRunner(t *testing.T) {
	t.Run("with empty path uses default", func(t *testing.T) {
		runner := NewCLIRunner("")
		if runner.BinaryPath != "backlog" {
			t.Errorf("BinaryPath = %q, want %q", runner.BinaryPath, "backlog")
		}
	})

	t.Run("with custom path", func(t *testing.T) {
		runner := NewCLIRunner("/usr/local/bin/backlog")
		if runner.BinaryPath != "/usr/local/bin/backlog" {
			t.Errorf("BinaryPath = %q, want %q", runner.BinaryPath, "/usr/local/bin/backlog")
		}
	})
}

func TestCLIRunnerSetEnv(t *testing.T) {
	runner := NewCLIRunner("")
	runner.SetEnv("BACKLOG_AGENT_ID", "test-agent")
	runner.SetEnv("GITHUB_TOKEN", "fake-token")

	if len(runner.Env) != 2 {
		t.Errorf("Env length = %d, want 2", len(runner.Env))
	}
	if runner.Env[0] != "BACKLOG_AGENT_ID=test-agent" {
		t.Errorf("Env[0] = %q, want %q", runner.Env[0], "BACKLOG_AGENT_ID=test-agent")
	}
	if runner.Env[1] != "GITHUB_TOKEN=fake-token" {
		t.Errorf("Env[1] = %q, want %q", runner.Env[1], "GITHUB_TOKEN=fake-token")
	}

	runner.ClearEnv()
	if len(runner.Env) != 0 {
		t.Errorf("Env length after clear = %d, want 0", len(runner.Env))
	}
}

func TestCLIRunnerRun(t *testing.T) {
	// Use 'echo' as a simple test command
	echoPath := "echo"
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows - echo behavior differs")
	}

	runner := NewCLIRunner(echoPath)

	t.Run("captures stdout", func(t *testing.T) {
		result := runner.RunArgs("hello", "world")
		if result.ExitCode != 0 {
			t.Errorf("ExitCode = %d, want 0", result.ExitCode)
		}
		if result.StdoutTrimmed() != "hello world" {
			t.Errorf("Stdout = %q, want %q", result.StdoutTrimmed(), "hello world")
		}
		if !result.Success() {
			t.Error("Success() = false, want true")
		}
	})

	t.Run("Run parses command string", func(t *testing.T) {
		result := runner.Run("hello world")
		if result.StdoutTrimmed() != "hello world" {
			t.Errorf("Stdout = %q, want %q", result.StdoutTrimmed(), "hello world")
		}
	})

	t.Run("stores last result", func(t *testing.T) {
		runner.Run("test output")
		if runner.LastResult == nil {
			t.Error("LastResult is nil")
		}
		if !runner.LastResult.StdoutContains("test output") {
			t.Error("LastResult.Stdout should contain 'test output'")
		}
	})
}

func TestCLIRunnerExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows - shell behavior differs")
	}

	runner := NewCLIRunner("sh")

	t.Run("captures non-zero exit code", func(t *testing.T) {
		result := runner.RunArgs("-c", "exit 42")
		if result.ExitCode != 42 {
			t.Errorf("ExitCode = %d, want 42", result.ExitCode)
		}
		if result.Success() {
			t.Error("Success() = true, want false")
		}
	})

	t.Run("captures stderr", func(t *testing.T) {
		result := runner.RunArgs("-c", "echo error >&2; exit 1")
		if !result.StderrContains("error") {
			t.Errorf("Stderr = %q, should contain 'error'", result.Stderr)
		}
		if result.ExitCode != 1 {
			t.Errorf("ExitCode = %d, want 1", result.ExitCode)
		}
	})
}

func TestCLIRunnerWorkDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows - pwd behavior differs")
	}

	tmpDir, err := os.MkdirTemp("", "cli-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	runner := NewCLIRunner("pwd")
	runner.WorkDir = tmpDir

	result := runner.RunArgs()
	// Resolve symlinks for comparison (macOS /tmp is a symlink)
	expectedDir, _ := filepath.EvalSymlinks(tmpDir)
	gotDir, _ := filepath.EvalSymlinks(result.StdoutTrimmed())

	if gotDir != expectedDir {
		t.Errorf("Working directory = %q, want %q", result.StdoutTrimmed(), tmpDir)
	}
}

func TestCLIRunnerBinaryNotFound(t *testing.T) {
	runner := NewCLIRunner("/nonexistent/binary/path")
	result := runner.RunArgs("test")

	if result.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1 for missing binary", result.ExitCode)
	}
	if result.Err == nil {
		t.Error("Err should not be nil for missing binary")
	}
}

func TestCommandResultHelpers(t *testing.T) {
	result := &CommandResult{
		Stdout:   "line1\nline2\nline3\n",
		Stderr:   "error1\nerror2\n",
		ExitCode: 0,
	}

	t.Run("StdoutLines", func(t *testing.T) {
		lines := result.StdoutLines()
		if len(lines) != 3 {
			t.Errorf("StdoutLines() len = %d, want 3", len(lines))
		}
		if lines[0] != "line1" {
			t.Errorf("StdoutLines()[0] = %q, want %q", lines[0], "line1")
		}
	})

	t.Run("StderrLines", func(t *testing.T) {
		lines := result.StderrLines()
		if len(lines) != 2 {
			t.Errorf("StderrLines() len = %d, want 2", len(lines))
		}
	})

	t.Run("StdoutContains", func(t *testing.T) {
		if !result.StdoutContains("line2") {
			t.Error("StdoutContains('line2') = false, want true")
		}
		if result.StdoutContains("notfound") {
			t.Error("StdoutContains('notfound') = true, want false")
		}
	})

	t.Run("empty stdout", func(t *testing.T) {
		emptyResult := &CommandResult{Stdout: ""}
		lines := emptyResult.StdoutLines()
		if len(lines) != 0 {
			t.Errorf("StdoutLines() for empty = %v, want empty slice", lines)
		}
	})
}
