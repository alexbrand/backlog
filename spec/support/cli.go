// Package support provides test helpers and fixtures for the backlog CLI specs.
package support

import (
	"bytes"
	"os/exec"
	"strings"
)

// CommandResult holds the result of executing a CLI command.
type CommandResult struct {
	// Stdout is the captured standard output
	Stdout string
	// Stderr is the captured standard error
	Stderr string
	// ExitCode is the exit code of the command
	ExitCode int
	// Command is the full command that was executed
	Command string
	// Err is the underlying error if any (not including non-zero exit codes)
	Err error
}

// CLIRunner executes CLI commands and captures their output.
type CLIRunner struct {
	// BinaryPath is the path to the backlog binary
	BinaryPath string
	// WorkDir is the working directory for command execution
	WorkDir string
	// Env is additional environment variables to set
	Env []string
	// LastResult stores the result of the last command execution
	LastResult *CommandResult
}

// NewCLIRunner creates a new CLI runner with the given binary path.
// If binaryPath is empty, it defaults to "backlog" (assumes it's in PATH).
func NewCLIRunner(binaryPath string) *CLIRunner {
	if binaryPath == "" {
		binaryPath = "backlog"
	}
	return &CLIRunner{
		BinaryPath: binaryPath,
		Env:        []string{},
	}
}

// Run executes a command string and captures the result.
// The command string is parsed as shell-like arguments.
// If the first argument is "backlog", it is stripped since the binary name is already "backlog".
// Example: Run("backlog list --status=todo -f json") or Run("list --status=todo -f json")
func (r *CLIRunner) Run(commandStr string) *CommandResult {
	args := parseArgs(commandStr)
	// Strip "backlog" prefix if present since the binary is already named "backlog"
	if len(args) > 0 && args[0] == "backlog" {
		args = args[1:]
	}
	return r.RunArgs(args...)
}

// RunArgs executes a command with explicit arguments and captures the result.
// Example: RunArgs("list", "--status=todo", "-f", "json")
func (r *CLIRunner) RunArgs(args ...string) *CommandResult {
	cmd := exec.Command(r.BinaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if r.WorkDir != "" {
		cmd.Dir = r.WorkDir
	}

	// Always set cmd.Env to include parent environment plus any runner-specific env
	// This ensures env vars set via os.Setenv are passed to the child process
	cmd.Env = append(cmd.Environ(), r.Env...)

	err := cmd.Run()

	result := &CommandResult{
		Stdout:  stdout.String(),
		Stderr:  stderr.String(),
		Command: r.BinaryPath + " " + strings.Join(args, " "),
	}

	// Get exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			// Other error (e.g., binary not found)
			result.ExitCode = -1
			result.Err = err
		}
	} else {
		result.ExitCode = 0
	}

	r.LastResult = result
	return result
}

// SetEnv adds an environment variable for subsequent command executions.
func (r *CLIRunner) SetEnv(key, value string) {
	r.Env = append(r.Env, key+"="+value)
}

// ClearEnv clears all custom environment variables.
func (r *CLIRunner) ClearEnv() {
	r.Env = []string{}
}

// parseArgs parses a command string into arguments.
// Handles quoted strings and basic shell-like parsing.
func parseArgs(commandStr string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, char := range commandStr {
		switch {
		case char == '"' || char == '\'':
			if inQuote {
				if char == quoteChar {
					inQuote = false
					quoteChar = 0
				} else {
					current.WriteRune(char)
				}
			} else {
				inQuote = true
				quoteChar = char
			}
		case char == ' ' && !inQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// Success returns true if the command exited with code 0.
func (r *CommandResult) Success() bool {
	return r.ExitCode == 0
}

// StdoutContains returns true if stdout contains the given substring.
func (r *CommandResult) StdoutContains(substr string) bool {
	return strings.Contains(r.Stdout, substr)
}

// StderrContains returns true if stderr contains the given substring.
func (r *CommandResult) StderrContains(substr string) bool {
	return strings.Contains(r.Stderr, substr)
}

// StdoutLines returns stdout split into lines (trimming empty trailing lines).
func (r *CommandResult) StdoutLines() []string {
	return splitLines(r.Stdout)
}

// StderrLines returns stderr split into lines (trimming empty trailing lines).
func (r *CommandResult) StderrLines() []string {
	return splitLines(r.Stderr)
}

// splitLines splits a string into lines, removing trailing empty lines.
func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	lines := strings.Split(s, "\n")
	// Remove trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// StdoutTrimmed returns stdout with leading and trailing whitespace removed.
func (r *CommandResult) StdoutTrimmed() string {
	return strings.TrimSpace(r.Stdout)
}

// StderrTrimmed returns stderr with leading and trailing whitespace removed.
func (r *CommandResult) StderrTrimmed() string {
	return strings.TrimSpace(r.Stderr)
}
