// Package steps provides step definitions for the backlog CLI Gherkin specs.
package steps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	ctx.Step(`^a file "([^"]*)" with content "([^"]*)"$`, aFileWithContent)
	ctx.Step(`^a task "([^"]*)" exists with status "([^"]*)"$`, aTaskExistsWithStatus)
	ctx.Step(`^a task "([^"]*)" exists with priority "([^"]*)"$`, aTaskExistsWithPriority)
	ctx.Step(`^a task "([^"]*)" exists with labels "([^"]*)"$`, aTaskExistsWithLabels)

	// When steps
	ctx.Step(`^I run "([^"]*)"$`, iRun)

	// Then steps
	ctx.Step(`^the exit code should be (\d+)$`, theExitCodeShouldBe)
	ctx.Step(`^stdout should contain "([^"]*)"$`, stdoutShouldContain)
	ctx.Step(`^stdout should not contain "([^"]*)"$`, stdoutShouldNotContain)
	ctx.Step(`^stderr should contain "([^"]*)"$`, stderrShouldContain)
	ctx.Step(`^stdout should be empty$`, stdoutShouldBeEmpty)
	ctx.Step(`^stderr should be empty$`, stderrShouldBeEmpty)
	ctx.Step(`^the output should match:$`, theOutputShouldMatch)
	ctx.Step(`^the JSON output should have "([^"]*)" equal to "([^"]*)"$`, theJSONOutputShouldHaveEqualTo)
	ctx.Step(`^the directory "([^"]*)" should exist$`, theDirectoryShouldExist)
	ctx.Step(`^the file "([^"]*)" should exist$`, theFileShouldExist)
	ctx.Step(`^a task file should exist in "([^"]*)" directory$`, aTaskFileShouldExistInDirectory)
	ctx.Step(`^the created task should have priority "([^"]*)"$`, theCreatedTaskShouldHavePriority)
	ctx.Step(`^the created task should have label "([^"]*)"$`, theCreatedTaskShouldHaveLabel)
	ctx.Step(`^the created task should have description containing "([^"]*)"$`, theCreatedTaskShouldHaveDescriptionContaining)
	ctx.Step(`^the task count should be (\d+)$`, theTaskCountShouldBe)
	ctx.Step(`^stdout should match pattern "([^"]*)"$`, stdoutShouldMatchPattern)
	ctx.Step(`^the JSON output should be valid$`, theJSONOutputShouldBeValid)
	ctx.Step(`^the JSON output should have "([^"]*)" as an array$`, theJSONOutputShouldHaveAsAnArray)
	ctx.Step(`^the JSON output should have array "([^"]*)" containing "([^"]*)"$`, theJSONOutputShouldHaveArrayContaining)
	ctx.Step(`^the JSON output should have array length "([^"]*)" equal to (\d+)$`, theJSONOutputShouldHaveArrayLengthEqualTo)
	ctx.Step(`^the JSON output should have "([^"]*)" as an object$`, theJSONOutputShouldHaveAsAnObject)
	ctx.Step(`^the JSON output should have "([^"]*)" containing "([^"]*)"$`, theJSONOutputShouldHaveContaining)
	ctx.Step(`^task "([^"]*)" has the following comments:$`, taskHasTheFollowingComments)

	// Config steps
	ctx.Step(`^a config file with the following content:$`, aConfigFileWithTheFollowingContent)
	ctx.Step(`^the config file is removed$`, theConfigFileIsRemoved)

	// Task state verification steps
	ctx.Step(`^the task "([^"]*)" should have status "([^"]*)"$`, theTaskShouldHaveStatus)
	ctx.Step(`^the task "([^"]*)" should be in directory "([^"]*)"$`, theTaskShouldBeInDirectory)
	ctx.Step(`^the task "([^"]*)" should have title "([^"]*)"$`, theTaskShouldHaveTitle)
	ctx.Step(`^the task "([^"]*)" should have priority "([^"]*)"$`, theTaskShouldHavePriority)
	ctx.Step(`^the task "([^"]*)" should have assignee "([^"]*)"$`, theTaskShouldHaveAssignee)
	ctx.Step(`^the task "([^"]*)" should have label "([^"]*)"$`, theTaskShouldHaveLabel)
	ctx.Step(`^the task "([^"]*)" should have comment containing "([^"]*)"$`, theTaskShouldHaveCommentContaining)
	ctx.Step(`^the task "([^"]*)" should not have label "([^"]*)"$`, theTaskShouldNotHaveLabel)
	ctx.Step(`^the task "([^"]*)" should have description containing "([^"]*)"$`, theTaskShouldHaveDescriptionContaining)
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

// stdoutShouldNotContain verifies stdout does not contain a substring.
func stdoutShouldNotContain(ctx context.Context, unexpected string) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	if strings.Contains(result.Stdout, unexpected) {
		return fmt.Errorf("expected stdout to not contain %q, but it does:\n%s", unexpected, result.Stdout)
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

// aFileWithContent creates a file with the specified content.
func aFileWithContent(ctx context.Context, path, content string) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	if err := env.CreateFile(path, content); err != nil {
		return ctx, fmt.Errorf("failed to create file %q: %w", path, err)
	}

	return ctx, nil
}

// theDirectoryShouldExist verifies that a directory exists in the test environment.
func theDirectoryShouldExist(ctx context.Context, path string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	fullPath := env.Path(path)
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory %q does not exist", path)
		}
		return fmt.Errorf("error checking directory %q: %w", path, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path %q exists but is not a directory", path)
	}

	return nil
}

// theFileShouldExist verifies that a file exists in the test environment.
func theFileShouldExist(ctx context.Context, path string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	fullPath := env.Path(path)
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file %q does not exist", path)
		}
		return fmt.Errorf("error checking file %q: %w", path, err)
	}

	if info.IsDir() {
		return fmt.Errorf("path %q exists but is a directory, not a file", path)
	}

	return nil
}

// aTaskFileShouldExistInDirectory verifies that at least one task file exists in the specified status directory.
func aTaskFileShouldExistInDirectory(ctx context.Context, status string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	tasks := reader.ListTasksByStatus(status)

	if len(tasks) == 0 {
		return fmt.Errorf("no task files found in %q directory", status)
	}

	return nil
}

// getCreatedTask returns the most recently created task (assumes only one or uses the last in list).
func getCreatedTask(ctx context.Context) (*support.TaskFile, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return nil, fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	tasks := reader.ListTasks()

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks found")
	}

	// Return the last task (most recently created in simple cases)
	return tasks[len(tasks)-1], nil
}

// theCreatedTaskShouldHavePriority verifies the created task has the expected priority.
func theCreatedTaskShouldHavePriority(ctx context.Context, expected string) error {
	task, err := getCreatedTask(ctx)
	if err != nil {
		return err
	}

	if task.Priority != expected {
		return fmt.Errorf("expected task priority to be %q, got %q", expected, task.Priority)
	}

	return nil
}

// theCreatedTaskShouldHaveLabel verifies the created task has a specific label.
func theCreatedTaskShouldHaveLabel(ctx context.Context, label string) error {
	task, err := getCreatedTask(ctx)
	if err != nil {
		return err
	}

	if !task.HasLabel(label) {
		return fmt.Errorf("expected task to have label %q, but it has labels: %v", label, task.Labels)
	}

	return nil
}

// theCreatedTaskShouldHaveDescriptionContaining verifies the task description contains expected text.
func theCreatedTaskShouldHaveDescriptionContaining(ctx context.Context, expected string) error {
	task, err := getCreatedTask(ctx)
	if err != nil {
		return err
	}

	if !strings.Contains(task.Description, expected) {
		return fmt.Errorf("expected task description to contain %q, got:\n%s", expected, task.Description)
	}

	return nil
}

// theTaskCountShouldBe verifies the total number of tasks.
func theTaskCountShouldBe(ctx context.Context, expected int) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	tasks := reader.ListTasks()

	if len(tasks) != expected {
		return fmt.Errorf("expected %d tasks, got %d", expected, len(tasks))
	}

	return nil
}

// stdoutShouldMatchPattern verifies stdout matches a regular expression pattern.
func stdoutShouldMatchPattern(ctx context.Context, pattern string) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern %q: %w", pattern, err)
	}

	if !re.MatchString(result.Stdout) {
		return fmt.Errorf("expected stdout to match pattern %q, got:\n%s", pattern, result.Stdout)
	}

	return nil
}

// theJSONOutputShouldBeValid verifies that stdout is valid JSON.
func theJSONOutputShouldBeValid(ctx context.Context) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	jsonResult := support.ParseJSON(result.Stdout)
	if !jsonResult.Valid() {
		return fmt.Errorf("stdout is not valid JSON: %s\nstdout:\n%s", jsonResult.Error(), result.Stdout)
	}

	return nil
}

// theJSONOutputShouldHaveAsAnArray verifies a JSON path contains an array.
func theJSONOutputShouldHaveAsAnArray(ctx context.Context, path string) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	jsonResult := support.ParseJSON(result.Stdout)
	if !jsonResult.Valid() {
		return fmt.Errorf("stdout is not valid JSON: %s\nstdout:\n%s", jsonResult.Error(), result.Stdout)
	}

	if !jsonResult.IsArray(path) {
		return fmt.Errorf("expected JSON path %q to be an array, but it is not\nstdout:\n%s", path, result.Stdout)
	}

	return nil
}

// theJSONOutputShouldHaveArrayContaining verifies an array at a JSON path contains a value.
func theJSONOutputShouldHaveArrayContaining(ctx context.Context, path, expected string) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	jsonResult := support.ParseJSON(result.Stdout)
	if !jsonResult.Valid() {
		return fmt.Errorf("stdout is not valid JSON: %s\nstdout:\n%s", jsonResult.Error(), result.Stdout)
	}

	if !jsonResult.ContainsString(path, expected) {
		arr := jsonResult.GetArray(path)
		return fmt.Errorf("expected JSON array at %q to contain %q, but it contains: %v", path, expected, arr)
	}

	return nil
}

// theJSONOutputShouldHaveArrayLengthEqualTo verifies an array at a JSON path has the expected length.
func theJSONOutputShouldHaveArrayLengthEqualTo(ctx context.Context, path string, expected int) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	jsonResult := support.ParseJSON(result.Stdout)
	if !jsonResult.Valid() {
		return fmt.Errorf("stdout is not valid JSON: %s\nstdout:\n%s", jsonResult.Error(), result.Stdout)
	}

	arr := jsonResult.GetArray(path)
	if len(arr) != expected {
		return fmt.Errorf("expected JSON array at %q to have length %d, got %d", path, expected, len(arr))
	}

	return nil
}

// theJSONOutputShouldHaveAsAnObject verifies a JSON path contains an object.
func theJSONOutputShouldHaveAsAnObject(ctx context.Context, path string) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	jsonResult := support.ParseJSON(result.Stdout)
	if !jsonResult.Valid() {
		return fmt.Errorf("stdout is not valid JSON: %s\nstdout:\n%s", jsonResult.Error(), result.Stdout)
	}

	if !jsonResult.IsObject(path) {
		return fmt.Errorf("expected JSON path %q to be an object, but it is not\nstdout:\n%s", path, result.Stdout)
	}

	return nil
}

// theJSONOutputShouldHaveContaining verifies a string value at a JSON path contains a substring.
func theJSONOutputShouldHaveContaining(ctx context.Context, path, expected string) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	jsonResult := support.ParseJSON(result.Stdout)
	if !jsonResult.Valid() {
		return fmt.Errorf("stdout is not valid JSON: %s\nstdout:\n%s", jsonResult.Error(), result.Stdout)
	}

	actual := jsonResult.GetString(path)
	if !strings.Contains(actual, expected) {
		return fmt.Errorf("expected JSON path %q to contain %q, got %q", path, expected, actual)
	}

	return nil
}

// taskHasTheFollowingComments adds comments to an existing task.
func taskHasTheFollowingComments(ctx context.Context, taskID string, table *godog.Table) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
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

	// Parse comments from table
	var comments []support.CommentFixture
	for _, row := range table.Rows[1:] {
		getValue := func(col string) string {
			if idx, ok := colIndex[col]; ok && idx < len(row.Cells) {
				return row.Cells[idx].Value
			}
			return ""
		}

		comment := support.CommentFixture{
			Author: getValue("author"),
			Date:   getValue("date"),
			Body:   getValue("body"),
		}
		comments = append(comments, comment)
	}

	// Read the existing task file
	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr != nil {
		return ctx, fmt.Errorf("failed to read task %s: %w", taskID, task.ParseErr)
	}

	// Build comments section
	var commentsSection strings.Builder
	commentsSection.WriteString("\n## Comments\n")
	for _, comment := range comments {
		commentsSection.WriteString(fmt.Sprintf("\n### %s @%s\n", comment.Date, comment.Author))
		commentsSection.WriteString(comment.Body)
		commentsSection.WriteString("\n")
	}

	// Read the original file content and append comments
	content, err := os.ReadFile(task.Path)
	if err != nil {
		return ctx, fmt.Errorf("failed to read task file: %w", err)
	}

	newContent := string(content) + commentsSection.String()
	if err := os.WriteFile(task.Path, []byte(newContent), 0644); err != nil {
		return ctx, fmt.Errorf("failed to write task file: %w", err)
	}

	return ctx, nil
}

// theTaskShouldHaveStatus verifies a task has the expected status.
func theTaskShouldHaveStatus(ctx context.Context, taskID, expectedStatus string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr != nil {
		return fmt.Errorf("failed to read task %s: %w", taskID, task.ParseErr)
	}

	if task.Status != expectedStatus {
		return fmt.Errorf("expected task %s to have status %q, got %q", taskID, expectedStatus, task.Status)
	}

	return nil
}

// theTaskShouldBeInDirectory verifies a task file exists in the expected status directory.
func theTaskShouldBeInDirectory(ctx context.Context, taskID, expectedDir string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr != nil {
		return fmt.Errorf("failed to read task %s: %w", taskID, task.ParseErr)
	}

	// Check the directory from the path
	dir := filepath.Dir(task.Path)
	actualDir := filepath.Base(dir)
	if actualDir != expectedDir {
		return fmt.Errorf("expected task %s to be in directory %q, got %q", taskID, expectedDir, actualDir)
	}

	return nil
}

// theTaskShouldHaveTitle verifies a task has the expected title.
func theTaskShouldHaveTitle(ctx context.Context, taskID, expectedTitle string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr != nil {
		return fmt.Errorf("failed to read task %s: %w", taskID, task.ParseErr)
	}

	if task.Title != expectedTitle {
		return fmt.Errorf("expected task %s to have title %q, got %q", taskID, expectedTitle, task.Title)
	}

	return nil
}

// theTaskShouldHavePriority verifies a task has the expected priority.
func theTaskShouldHavePriority(ctx context.Context, taskID, expectedPriority string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr != nil {
		return fmt.Errorf("failed to read task %s: %w", taskID, task.ParseErr)
	}

	if task.Priority != expectedPriority {
		return fmt.Errorf("expected task %s to have priority %q, got %q", taskID, expectedPriority, task.Priority)
	}

	return nil
}

// theTaskShouldHaveAssignee verifies a task has the expected assignee.
func theTaskShouldHaveAssignee(ctx context.Context, taskID, expectedAssignee string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr != nil {
		return fmt.Errorf("failed to read task %s: %w", taskID, task.ParseErr)
	}

	if task.Assignee != expectedAssignee {
		return fmt.Errorf("expected task %s to have assignee %q, got %q", taskID, expectedAssignee, task.Assignee)
	}

	return nil
}

// theTaskShouldHaveLabel verifies a task has a specific label.
func theTaskShouldHaveLabel(ctx context.Context, taskID, expectedLabel string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr != nil {
		return fmt.Errorf("failed to read task %s: %w", taskID, task.ParseErr)
	}

	if !task.HasLabel(expectedLabel) {
		return fmt.Errorf("expected task %s to have label %q, but it has labels: %v", taskID, expectedLabel, task.Labels)
	}

	return nil
}

// theTaskShouldHaveCommentContaining verifies a task has a comment containing specific text.
func theTaskShouldHaveCommentContaining(ctx context.Context, taskID, expectedText string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr != nil {
		return fmt.Errorf("failed to read task %s: %w", taskID, task.ParseErr)
	}

	// Read the full file content to check for comments
	content, err := os.ReadFile(task.Path)
	if err != nil {
		return fmt.Errorf("failed to read task file: %w", err)
	}

	if !strings.Contains(string(content), expectedText) {
		return fmt.Errorf("expected task %s to have comment containing %q, but it doesn't.\nFile content:\n%s", taskID, expectedText, string(content))
	}

	return nil
}

// theTaskShouldNotHaveLabel verifies a task does not have a specific label.
func theTaskShouldNotHaveLabel(ctx context.Context, taskID, unexpectedLabel string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr != nil {
		return fmt.Errorf("failed to read task %s: %w", taskID, task.ParseErr)
	}

	if task.HasLabel(unexpectedLabel) {
		return fmt.Errorf("expected task %s to not have label %q, but it has labels: %v", taskID, unexpectedLabel, task.Labels)
	}

	return nil
}

// theTaskShouldHaveDescriptionContaining verifies a task description contains expected text.
func theTaskShouldHaveDescriptionContaining(ctx context.Context, taskID, expectedText string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr != nil {
		return fmt.Errorf("failed to read task %s: %w", taskID, task.ParseErr)
	}

	if !strings.Contains(task.Description, expectedText) {
		return fmt.Errorf("expected task %s description to contain %q, got:\n%s", taskID, expectedText, task.Description)
	}

	return nil
}

// aTaskExistsWithStatus creates a task with the given title and status.
func aTaskExistsWithStatus(ctx context.Context, title, status string) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	// Ensure backlog directory exists
	if !env.FileExists(".backlog") {
		if err := env.CreateBacklogDir(); err != nil {
			return ctx, fmt.Errorf("failed to create backlog directory: %w", err)
		}
	}

	// Generate a simple ID from the title
	id := generateTaskID(title)

	loader := support.NewFixtureLoader("")
	task := support.TaskFixture{
		ID:     id,
		Title:  title,
		Status: status,
	}

	if err := loader.LoadTasks(env, []support.TaskFixture{task}); err != nil {
		return ctx, fmt.Errorf("failed to create task: %w", err)
	}

	return ctx, nil
}

// aTaskExistsWithPriority creates a task with the given title and priority.
func aTaskExistsWithPriority(ctx context.Context, title, priority string) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	// Ensure backlog directory exists
	if !env.FileExists(".backlog") {
		if err := env.CreateBacklogDir(); err != nil {
			return ctx, fmt.Errorf("failed to create backlog directory: %w", err)
		}
	}

	// Generate a simple ID from the title
	id := generateTaskID(title)

	loader := support.NewFixtureLoader("")
	task := support.TaskFixture{
		ID:       id,
		Title:    title,
		Status:   "backlog",
		Priority: priority,
	}

	if err := loader.LoadTasks(env, []support.TaskFixture{task}); err != nil {
		return ctx, fmt.Errorf("failed to create task: %w", err)
	}

	return ctx, nil
}

// aTaskExistsWithLabels creates a task with the given title and labels.
func aTaskExistsWithLabels(ctx context.Context, title, labelsStr string) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	// Ensure backlog directory exists
	if !env.FileExists(".backlog") {
		if err := env.CreateBacklogDir(); err != nil {
			return ctx, fmt.Errorf("failed to create backlog directory: %w", err)
		}
	}

	// Generate a simple ID from the title
	id := generateTaskID(title)

	// Parse labels as comma-separated list
	var labels []string
	for _, label := range strings.Split(labelsStr, ",") {
		label = strings.TrimSpace(label)
		if label != "" {
			labels = append(labels, label)
		}
	}

	loader := support.NewFixtureLoader("")
	task := support.TaskFixture{
		ID:     id,
		Title:  title,
		Status: "backlog",
		Labels: labels,
	}

	if err := loader.LoadTasks(env, []support.TaskFixture{task}); err != nil {
		return ctx, fmt.Errorf("failed to create task: %w", err)
	}

	return ctx, nil
}

// generateTaskID generates a simple task ID from the title.
func generateTaskID(title string) string {
	// Create a slug from the title - take first few words, lowercase, replace spaces with dashes
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove non-alphanumeric characters except dashes
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()
	// Truncate if too long
	if len(slug) > 20 {
		slug = slug[:20]
	}
	// Remove trailing dashes
	slug = strings.TrimRight(slug, "-")
	return slug
}

// aConfigFileWithTheFollowingContent creates a config file with the specified YAML content.
func aConfigFileWithTheFollowingContent(ctx context.Context, content *godog.DocString) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	// Ensure .backlog directory exists
	if !env.FileExists(".backlog") {
		if err := env.CreateBacklogDir(); err != nil {
			return ctx, fmt.Errorf("failed to create backlog directory: %w", err)
		}
	}

	if err := env.CreateFile(".backlog/config.yaml", content.Content); err != nil {
		return ctx, fmt.Errorf("failed to create config file: %w", err)
	}

	return ctx, nil
}

// theConfigFileIsRemoved removes the config file from the backlog directory.
func theConfigFileIsRemoved(ctx context.Context) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	configPath := env.Path(".backlog/config.yaml")
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return ctx, fmt.Errorf("failed to remove config file: %w", err)
	}

	return ctx, nil
}
