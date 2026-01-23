// Package steps provides step definitions for the backlog CLI Gherkin specs.
package steps

import (
	"context"
	"fmt"
	"strings"

	"github.com/cucumber/godog"

	"github.com/alexbrand/backlog/spec/support"
)

// contextKey is a type for context keys to avoid collisions.
type contextKey string

const (
	testEnvKey    contextKey = "testEnv"
	cliRunnerKey  contextKey = "cliRunner"
	lastResultKey contextKey = "lastResult"
)

// getTestEnv retrieves the TestEnv from context.
func getTestEnv(ctx context.Context) *support.TestEnv {
	if env, ok := ctx.Value(testEnvKey).(*support.TestEnv); ok {
		return env
	}
	return nil
}

// getCLIRunner retrieves the CLIRunner from context.
func getCLIRunner(ctx context.Context) *support.CLIRunner {
	if runner, ok := ctx.Value(cliRunnerKey).(*support.CLIRunner); ok {
		return runner
	}
	return nil
}

// getLastResult retrieves the last command result from context.
func getLastResult(ctx context.Context) *support.CommandResult {
	if result, ok := ctx.Value(lastResultKey).(*support.CommandResult); ok {
		return result
	}
	return nil
}

// InitializeCommonSteps registers all common step definitions.
func InitializeCommonSteps(ctx *godog.ScenarioContext) {
	// Before each scenario: set up test environment
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		env, err := support.NewTestEnv()
		if err != nil {
			return ctx, fmt.Errorf("failed to create test environment: %w", err)
		}

		// Create CLI runner pointing to the built binary
		// Assumes `go build` has been run and backlog binary is in PATH or current dir
		runner := support.NewCLIRunner("")
		runner.WorkDir = env.TempDir

		ctx = context.WithValue(ctx, testEnvKey, env)
		ctx = context.WithValue(ctx, cliRunnerKey, runner)

		return ctx, nil
	})

	// After each scenario: clean up test environment
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		env := getTestEnv(ctx)
		if env != nil {
			if cleanupErr := env.Cleanup(); cleanupErr != nil {
				// Log but don't fail on cleanup errors
				fmt.Printf("Warning: cleanup failed: %v\n", cleanupErr)
			}
		}
		return ctx, nil
	})

	// Given steps
	ctx.Step(`^a fresh backlog directory$`, aFreshBacklogDirectory)
	ctx.Step(`^a backlog with the following tasks:$`, aBacklogWithTheFollowingTasks)

	// When steps
	ctx.Step(`^I run "([^"]*)"$`, iRun)

	// Then steps
	ctx.Step(`^the exit code should be (\d+)$`, theExitCodeShouldBe)
	ctx.Step(`^stdout should contain "([^"]*)"$`, stdoutShouldContain)
	ctx.Step(`^stderr should contain "([^"]*)"$`, stderrShouldContain)
	ctx.Step(`^stdout should be empty$`, stdoutShouldBeEmpty)
	ctx.Step(`^stderr should be empty$`, stderrShouldBeEmpty)
	ctx.Step(`^the output should match:$`, theOutputShouldMatch)
	ctx.Step(`^the JSON output should have "([^"]*)" equal to "([^"]*)"$`, theJSONOutputShouldHaveEqualTo)
}

// aFreshBacklogDirectory creates a new empty .backlog directory.
func aFreshBacklogDirectory(ctx context.Context) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	if err := env.CreateBacklogDir(); err != nil {
		return ctx, fmt.Errorf("failed to create backlog directory: %w", err)
	}

	return ctx, nil
}

// aBacklogWithTheFollowingTasks creates a backlog with tasks from a data table.
// Table columns: id, title, status, priority, labels, assignee
func aBacklogWithTheFollowingTasks(ctx context.Context, table *godog.Table) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	// Create backlog directory first
	if err := env.CreateBacklogDir(); err != nil {
		return ctx, fmt.Errorf("failed to create backlog directory: %w", err)
	}

	// Parse table header to get column indices
	if len(table.Rows) < 2 {
		return ctx, fmt.Errorf("table must have at least a header row and one data row")
	}

	header := table.Rows[0]
	colIndex := make(map[string]int)
	for i, cell := range header.Cells {
		colIndex[cell.Value] = i
	}

	// Required columns
	if _, ok := colIndex["id"]; !ok {
		return ctx, fmt.Errorf("table must have 'id' column")
	}
	if _, ok := colIndex["title"]; !ok {
		return ctx, fmt.Errorf("table must have 'title' column")
	}

	// Load fixtures
	loader := support.NewFixtureLoader("")
	var tasks []support.TaskFixture

	for _, row := range table.Rows[1:] {
		task := support.TaskFixture{}

		// Get cell value by column name
		getValue := func(col string) string {
			if idx, ok := colIndex[col]; ok && idx < len(row.Cells) {
				return row.Cells[idx].Value
			}
			return ""
		}

		task.ID = getValue("id")
		task.Title = getValue("title")
		task.Status = getValue("status")
		if task.Status == "" {
			task.Status = "backlog"
		}
		task.Priority = getValue("priority")
		task.Assignee = getValue("assignee")
		task.Description = getValue("description")

		// Parse labels as comma-separated list
		labelsStr := getValue("labels")
		if labelsStr != "" {
			for _, label := range strings.Split(labelsStr, ",") {
				label = strings.TrimSpace(label)
				if label != "" {
					task.Labels = append(task.Labels, label)
				}
			}
		}

		task.AgentID = getValue("agent_id")

		tasks = append(tasks, task)
	}

	if err := loader.LoadTasks(env, tasks); err != nil {
		return ctx, fmt.Errorf("failed to load tasks: %w", err)
	}

	return ctx, nil
}

// iRun executes a CLI command.
func iRun(ctx context.Context, command string) (context.Context, error) {
	runner := getCLIRunner(ctx)
	if runner == nil {
		return ctx, fmt.Errorf("CLI runner not initialized")
	}

	result := runner.Run(command)
	ctx = context.WithValue(ctx, lastResultKey, result)

	return ctx, nil
}

// theExitCodeShouldBe verifies the exit code of the last command.
func theExitCodeShouldBe(ctx context.Context, expected int) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	if result.ExitCode != expected {
		return fmt.Errorf("expected exit code %d, got %d\nstdout: %s\nstderr: %s",
			expected, result.ExitCode, result.Stdout, result.Stderr)
	}

	return nil
}

// stdoutShouldContain verifies stdout contains a substring.
func stdoutShouldContain(ctx context.Context, expected string) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	if !strings.Contains(result.Stdout, expected) {
		return fmt.Errorf("expected stdout to contain %q, got:\n%s", expected, result.Stdout)
	}

	return nil
}

// stderrShouldContain verifies stderr contains a substring.
func stderrShouldContain(ctx context.Context, expected string) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	if !strings.Contains(result.Stderr, expected) {
		return fmt.Errorf("expected stderr to contain %q, got:\n%s", expected, result.Stderr)
	}

	return nil
}

// stdoutShouldBeEmpty verifies stdout is empty.
func stdoutShouldBeEmpty(ctx context.Context) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	if strings.TrimSpace(result.Stdout) != "" {
		return fmt.Errorf("expected stdout to be empty, got:\n%s", result.Stdout)
	}

	return nil
}

// stderrShouldBeEmpty verifies stderr is empty.
func stderrShouldBeEmpty(ctx context.Context) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	if strings.TrimSpace(result.Stderr) != "" {
		return fmt.Errorf("expected stderr to be empty, got:\n%s", result.Stderr)
	}

	return nil
}

// theOutputShouldMatch verifies stdout matches a docstring exactly (ignoring leading/trailing whitespace).
func theOutputShouldMatch(ctx context.Context, expected *godog.DocString) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	actual := strings.TrimSpace(result.Stdout)
	expectedTrimmed := strings.TrimSpace(expected.Content)

	if actual != expectedTrimmed {
		return fmt.Errorf("output did not match\nExpected:\n%s\n\nActual:\n%s", expectedTrimmed, actual)
	}

	return nil
}

// theJSONOutputShouldHaveEqualTo verifies a JSON path has the expected value.
func theJSONOutputShouldHaveEqualTo(ctx context.Context, path, expected string) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	jsonResult := support.ParseJSON(result.Stdout)
	if !jsonResult.Valid() {
		return fmt.Errorf("stdout is not valid JSON: %s\nstdout:\n%s", jsonResult.Error(), result.Stdout)
	}

	actual := jsonResult.GetString(path)
	if actual != expected {
		return fmt.Errorf("expected JSON path %q to be %q, got %q", path, expected, actual)
	}

	return nil
}
