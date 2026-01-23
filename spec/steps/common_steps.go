// Package steps provides step definitions for the backlog CLI Gherkin specs.
package steps

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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

	// After each scenario: clean up test environment and mock server
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		// Clean up mock GitHub server if running
		if server := getMockGitHubServer(ctx); server != nil {
			server.Close()
		}

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
	ctx.Step(`^the environment variable "([^"]*)" is "([^"]*)"$`, theEnvironmentVariableIs)
	ctx.Step(`^task "([^"]*)" is claimed by agent "([^"]*)"$`, taskIsClaimedByAgent)
	ctx.Step(`^the agent ID is "([^"]*)"$`, theAgentIDIs)

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
	ctx.Step(`^the JSON output should have "([^"]*)" matching pattern "([^"]*)"$`, theJSONOutputShouldHaveMatchingPattern)
	ctx.Step(`^the JSON output should not have array "([^"]*)" containing "([^"]*)"$`, theJSONOutputShouldNotHaveArrayContaining)
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

	// Claim-specific verification steps
	ctx.Step(`^the task "([^"]*)" should be assigned$`, theTaskShouldBeAssigned)
	ctx.Step(`^the task "([^"]*)" should have agent label$`, theTaskShouldHaveAgentLabel)
	ctx.Step(`^a lock file should exist for task "([^"]*)"$`, aLockFileShouldExistForTask)
	ctx.Step(`^no lock file should exist for task "([^"]*)"$`, noLockFileShouldExistForTask)
	ctx.Step(`^task "([^"]*)" should be claimed by "([^"]*)"$`, taskShouldBeClaimedBy)
	ctx.Step(`^task "([^"]*)" should not be claimed$`, taskShouldNotBeClaimed)

	// Lock file content verification steps
	ctx.Step(`^the lock file for task "([^"]*)" should contain agent "([^"]*)"$`, theLockFileForTaskShouldContainAgent)
	ctx.Step(`^the lock file for task "([^"]*)" should have a valid claimed_at timestamp$`, theLockFileForTaskShouldHaveValidClaimedAt)
	ctx.Step(`^the lock file for task "([^"]*)" should have a valid expires_at timestamp$`, theLockFileForTaskShouldHaveValidExpiresAt)
	ctx.Step(`^the lock file for task "([^"]*)" should have expires_at after claimed_at$`, theLockFileForTaskShouldHaveExpiresAtAfterClaimedAt)

	// Stale/expired lock setup steps
	ctx.Step(`^task "([^"]*)" has a stale lock from agent "([^"]*)" that expired (\d+) hour(?:s)? ago$`, taskHasStaleLockHoursAgo)
	ctx.Step(`^task "([^"]*)" has a lock from agent "([^"]*)" that expired (\d+) minute(?:s)? ago$`, taskHasExpiredLockMinutesAgo)
	ctx.Step(`^task "([^"]*)" has an active lock from agent "([^"]*)"$`, taskHasActiveLock)

	// Git sync setup steps
	ctx.Step(`^a git repository is initialized$`, aGitRepositoryIsInitialized)
	ctx.Step(`^git_sync is enabled in the config$`, gitSyncIsEnabledInTheConfig)
	ctx.Step(`^git_sync is disabled in the config$`, gitSyncIsDisabledInTheConfig)
	ctx.Step(`^lock_mode is "([^"]*)" in the config$`, lockModeIsInTheConfig)
	ctx.Step(`^a remote git repository$`, aRemoteGitRepository)
	ctx.Step(`^the remote has different content than local$`, theRemoteHasDifferentContentThanLocal)
	ctx.Step(`^the remote has been updated by another agent$`, theRemoteHasBeenUpdatedByAnotherAgent)
	ctx.Step(`^there are uncommitted changes in the repository$`, thereAreUncommittedChangesInTheRepository)
	ctx.Step(`^the remote has a new commit$`, theRemoteHasANewCommit)
	ctx.Step(`^another agent has claimed task "([^"]*)" and pushed while we were working$`, anotherAgentHasClaimedTaskAndPushed)
	ctx.Step(`^task "([^"]*)" has a stale lock file$`, taskHasStaleLockFile)
	ctx.Step(`^the remote repository is unreachable$`, theRemoteRepositoryIsUnreachable)

	// Git sync verification steps
	ctx.Step(`^a git commit should exist with message containing "([^"]*)"$`, aGitCommitShouldExistWithMessageContaining)
	ctx.Step(`^the last git commit message should match pattern "([^"]*)"$`, theLastGitCommitMessageShouldMatchPattern)
	ctx.Step(`^the local repository should be in sync with remote$`, theLocalRepositoryShouldBeInSyncWithRemote)
	ctx.Step(`^the local repository should match the remote$`, theLocalRepositoryShouldMatchTheRemote)
	ctx.Step(`^no new git commits should exist$`, noNewGitCommitsShouldExist)
	ctx.Step(`^the remote should have the latest commit$`, theRemoteShouldHaveTheLatestCommit)
	ctx.Step(`^the local repository should include the remote commit$`, theLocalRepositoryShouldIncludeTheRemoteCommit)

	// Mock GitHub API steps
	ctx.Step(`^a mock GitHub API server is running$`, aMockGitHubAPIServerIsRunning)
	ctx.Step(`^the mock GitHub API returns auth error for invalid tokens$`, theMockGitHubAPIReturnsAuthErrorForInvalidTokens)
	ctx.Step(`^the mock GitHub API expects token "([^"]*)"$`, theMockGitHubAPIExpectsToken)
	ctx.Step(`^the mock GitHub API has the following issues:$`, theMockGitHubAPIHasTheFollowingIssues)
	ctx.Step(`^a credentials file with the following content:$`, aCredentialsFileWithTheFollowingContent)
	ctx.Step(`^the environment variable "([^"]*)" is not set$`, theEnvironmentVariableIsNotSet)
	ctx.Step(`^the environment variable "([^"]*)" is set to a valid token$`, theEnvironmentVariableIsSetToAValidToken)
	ctx.Step(`^the mock GitHub API authenticated user is "([^"]*)"$`, theMockGitHubAPIAuthenticatedUserIs)
	ctx.Step(`^the mock GitHub issue "([^"]*)" has the following comments:$`, theMockGitHubIssueHasTheFollowingComments)
	ctx.Step(`^the JSON output array "([^"]*)" should have length (\d+)$`, theJSONOutputArrayShouldHaveLength)

	// GitHub assertion steps
	ctx.Step(`^a GitHub repository "([^"]*)" with issues:$`, aGitHubRepositoryWithIssues)
	ctx.Step(`^the GitHub token is "([^"]*)"$`, theGitHubTokenIs)
	ctx.Step(`^the GitHub issue "([^"]*)" should have label "([^"]*)"$`, theGitHubIssueShouldHaveLabel)
	ctx.Step(`^the GitHub issue "([^"]*)" should be assigned to "([^"]*)"$`, theGitHubIssueShouldBeAssignedTo)
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

// theJSONOutputShouldHaveMatchingPattern verifies a value at a JSON path matches a regex pattern.
func theJSONOutputShouldHaveMatchingPattern(ctx context.Context, path, pattern string) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	jsonResult := support.ParseJSON(result.Stdout)
	if !jsonResult.Valid() {
		return fmt.Errorf("stdout is not valid JSON: %s\nstdout:\n%s", jsonResult.Error(), result.Stdout)
	}

	actual := jsonResult.GetString(path)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern %q: %v", pattern, err)
	}

	if !re.MatchString(actual) {
		return fmt.Errorf("expected JSON path %q to match pattern %q, got %q", path, pattern, actual)
	}

	return nil
}

// theJSONOutputShouldNotHaveArrayContaining verifies an array at a JSON path does NOT contain a value.
func theJSONOutputShouldNotHaveArrayContaining(ctx context.Context, path, unexpected string) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	jsonResult := support.ParseJSON(result.Stdout)
	if !jsonResult.Valid() {
		return fmt.Errorf("stdout is not valid JSON: %s\nstdout:\n%s", jsonResult.Error(), result.Stdout)
	}

	if jsonResult.ContainsString(path, unexpected) {
		arr := jsonResult.GetArray(path)
		return fmt.Errorf("expected JSON array at %q to NOT contain %q, but it does: %v", path, unexpected, arr)
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

// theEnvironmentVariableIs sets an environment variable for the test.
func theEnvironmentVariableIs(ctx context.Context, key, value string) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	env.SetEnv(key, value)
	return ctx, nil
}

// taskIsClaimedByAgent sets up a task as claimed by a specific agent.
func taskIsClaimedByAgent(ctx context.Context, taskID, agentID string) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	// Read the task file
	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr != nil {
		return ctx, fmt.Errorf("failed to read task %s: %w", taskID, task.ParseErr)
	}

	// Read the file content
	content, err := os.ReadFile(task.Path)
	if err != nil {
		return ctx, fmt.Errorf("failed to read task file: %w", err)
	}

	// Add agent_id to frontmatter if not present
	contentStr := string(content)
	if !strings.Contains(contentStr, "agent_id:") {
		// Insert agent_id before the closing ---
		contentStr = strings.Replace(contentStr, "\n---\n", fmt.Sprintf("\nagent_id: %s\n---\n", agentID), 1)
	}

	// Add agent label if not present
	agentLabel := fmt.Sprintf("agent:%s", agentID)
	if !strings.Contains(contentStr, agentLabel) {
		// Update the labels line
		if strings.Contains(contentStr, "labels: []") {
			contentStr = strings.Replace(contentStr, "labels: []", fmt.Sprintf("labels: [%s]", agentLabel), 1)
		} else if strings.Contains(contentStr, "labels: [") {
			// Add to existing labels
			contentStr = strings.Replace(contentStr, "labels: [", fmt.Sprintf("labels: [%s, ", agentLabel), 1)
		}
	}

	if err := os.WriteFile(task.Path, []byte(contentStr), 0644); err != nil {
		return ctx, fmt.Errorf("failed to write task file: %w", err)
	}

	// Create lock file
	lockContent := fmt.Sprintf("agent: %s\nclaimed_at: 2025-01-01T00:00:00Z\nexpires_at: 2025-01-01T00:30:00Z\n", agentID)
	lockPath := filepath.Join(".backlog", ".locks", taskID+".lock")
	if err := env.CreateFile(lockPath, lockContent); err != nil {
		return ctx, fmt.Errorf("failed to create lock file: %w", err)
	}

	return ctx, nil
}

// theAgentIDIs is an alias for setting BACKLOG_AGENT_ID environment variable.
func theAgentIDIs(ctx context.Context, agentID string) (context.Context, error) {
	return theEnvironmentVariableIs(ctx, "BACKLOG_AGENT_ID", agentID)
}

// theTaskShouldBeAssigned verifies that a task has an assignee set.
func theTaskShouldBeAssigned(ctx context.Context, taskID string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr != nil {
		return fmt.Errorf("failed to read task %s: %w", taskID, task.ParseErr)
	}

	if !task.IsAssigned() {
		return fmt.Errorf("expected task %s to be assigned, but assignee is %q", taskID, task.Assignee)
	}

	return nil
}

// theTaskShouldHaveAgentLabel verifies that a task has an agent label (agent:*).
func theTaskShouldHaveAgentLabel(ctx context.Context, taskID string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr != nil {
		return fmt.Errorf("failed to read task %s: %w", taskID, task.ParseErr)
	}

	if !task.HasAgentLabel("agent") {
		return fmt.Errorf("expected task %s to have an agent label, but it has labels: %v", taskID, task.Labels)
	}

	return nil
}

// aLockFileShouldExistForTask verifies that a lock file exists for the task.
func aLockFileShouldExistForTask(ctx context.Context, taskID string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	lockPath := filepath.Join(".backlog", ".locks", taskID+".lock")
	if !env.FileExists(lockPath) {
		return fmt.Errorf("expected lock file %s to exist, but it doesn't", lockPath)
	}

	return nil
}

// noLockFileShouldExistForTask verifies that no lock file exists for the task.
func noLockFileShouldExistForTask(ctx context.Context, taskID string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	lockPath := filepath.Join(".backlog", ".locks", taskID+".lock")
	if env.FileExists(lockPath) {
		return fmt.Errorf("expected no lock file at %s, but it exists", lockPath)
	}

	return nil
}

// taskShouldBeClaimedBy verifies that a task is claimed by a specific agent.
func taskShouldBeClaimedBy(ctx context.Context, taskID, agentID string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr != nil {
		return fmt.Errorf("failed to read task %s: %w", taskID, task.ParseErr)
	}

	expectedLabel := fmt.Sprintf("agent:%s", agentID)
	if !task.HasLabel(expectedLabel) {
		return fmt.Errorf("expected task %s to be claimed by %s (have label %s), but it has labels: %v",
			taskID, agentID, expectedLabel, task.Labels)
	}

	return nil
}

// taskShouldNotBeClaimed verifies that a task is not claimed by any agent.
func taskShouldNotBeClaimed(ctx context.Context, taskID string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr != nil {
		return fmt.Errorf("failed to read task %s: %w", taskID, task.ParseErr)
	}

	if task.HasAgentLabel("agent") {
		agentID := task.GetAgentFromLabel("agent")
		return fmt.Errorf("expected task %s to not be claimed, but it is claimed by %s", taskID, agentID)
	}

	return nil
}

// LockFile represents the parsed contents of a lock file.
type LockFile struct {
	Agent     string
	ClaimedAt time.Time
	ExpiresAt time.Time
}

// readLockFile reads and parses a lock file for a task.
func readLockFile(env *support.TestEnv, taskID string) (*LockFile, error) {
	lockPath := filepath.Join(".backlog", ".locks", taskID+".lock")
	content, err := env.ReadFile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	lock := &LockFile{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "agent:") {
			lock.Agent = strings.TrimSpace(strings.TrimPrefix(line, "agent:"))
		} else if strings.HasPrefix(line, "claimed_at:") {
			ts := strings.TrimSpace(strings.TrimPrefix(line, "claimed_at:"))
			t, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				return nil, fmt.Errorf("invalid claimed_at timestamp: %w", err)
			}
			lock.ClaimedAt = t
		} else if strings.HasPrefix(line, "expires_at:") {
			ts := strings.TrimSpace(strings.TrimPrefix(line, "expires_at:"))
			t, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				return nil, fmt.Errorf("invalid expires_at timestamp: %w", err)
			}
			lock.ExpiresAt = t
		}
	}

	return lock, nil
}

// theLockFileForTaskShouldContainAgent verifies the lock file contains the expected agent.
func theLockFileForTaskShouldContainAgent(ctx context.Context, taskID, expectedAgent string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	lock, err := readLockFile(env, taskID)
	if err != nil {
		return err
	}

	if lock.Agent != expectedAgent {
		return fmt.Errorf("expected lock file for task %s to contain agent %q, got %q", taskID, expectedAgent, lock.Agent)
	}

	return nil
}

// theLockFileForTaskShouldHaveValidClaimedAt verifies the lock file has a valid claimed_at timestamp.
func theLockFileForTaskShouldHaveValidClaimedAt(ctx context.Context, taskID string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	lock, err := readLockFile(env, taskID)
	if err != nil {
		return err
	}

	if lock.ClaimedAt.IsZero() {
		return fmt.Errorf("expected lock file for task %s to have a valid claimed_at timestamp, but it was zero or missing", taskID)
	}

	return nil
}

// theLockFileForTaskShouldHaveValidExpiresAt verifies the lock file has a valid expires_at timestamp.
func theLockFileForTaskShouldHaveValidExpiresAt(ctx context.Context, taskID string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	lock, err := readLockFile(env, taskID)
	if err != nil {
		return err
	}

	if lock.ExpiresAt.IsZero() {
		return fmt.Errorf("expected lock file for task %s to have a valid expires_at timestamp, but it was zero or missing", taskID)
	}

	return nil
}

// theLockFileForTaskShouldHaveExpiresAtAfterClaimedAt verifies expires_at is after claimed_at.
func theLockFileForTaskShouldHaveExpiresAtAfterClaimedAt(ctx context.Context, taskID string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	lock, err := readLockFile(env, taskID)
	if err != nil {
		return err
	}

	if !lock.ExpiresAt.After(lock.ClaimedAt) {
		return fmt.Errorf("expected expires_at (%v) to be after claimed_at (%v) for task %s",
			lock.ExpiresAt, lock.ClaimedAt, taskID)
	}

	return nil
}

// taskHasStaleLockHoursAgo creates a stale lock file that expired N hours ago.
func taskHasStaleLockHoursAgo(ctx context.Context, taskID, agentID string, hoursAgo int) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	now := time.Now().UTC()
	claimedAt := now.Add(-time.Duration(hoursAgo+1) * time.Hour)
	expiresAt := now.Add(-time.Duration(hoursAgo) * time.Hour)

	content := fmt.Sprintf("agent: %s\nclaimed_at: %s\nexpires_at: %s\n",
		agentID,
		claimedAt.Format(time.RFC3339),
		expiresAt.Format(time.RFC3339))

	lockPath := filepath.Join(".backlog", ".locks", taskID+".lock")
	if err := env.CreateFile(lockPath, content); err != nil {
		return ctx, fmt.Errorf("failed to create stale lock file: %w", err)
	}

	return ctx, nil
}

// taskHasExpiredLockMinutesAgo creates a lock file that expired N minutes ago.
func taskHasExpiredLockMinutesAgo(ctx context.Context, taskID, agentID string, minutesAgo int) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	now := time.Now().UTC()
	claimedAt := now.Add(-time.Duration(minutesAgo+30) * time.Minute)
	expiresAt := now.Add(-time.Duration(minutesAgo) * time.Minute)

	content := fmt.Sprintf("agent: %s\nclaimed_at: %s\nexpires_at: %s\n",
		agentID,
		claimedAt.Format(time.RFC3339),
		expiresAt.Format(time.RFC3339))

	lockPath := filepath.Join(".backlog", ".locks", taskID+".lock")
	if err := env.CreateFile(lockPath, content); err != nil {
		return ctx, fmt.Errorf("failed to create expired lock file: %w", err)
	}

	// Also add the agent label to the task to simulate a previous claim
	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr == nil {
		// Read the file content
		taskContent, err := os.ReadFile(task.Path)
		if err == nil {
			contentStr := string(taskContent)
			agentLabel := fmt.Sprintf("agent:%s", agentID)
			if !strings.Contains(contentStr, agentLabel) {
				if strings.Contains(contentStr, "labels: []") {
					contentStr = strings.Replace(contentStr, "labels: []", fmt.Sprintf("labels: [%s]", agentLabel), 1)
				} else if strings.Contains(contentStr, "labels: [") {
					contentStr = strings.Replace(contentStr, "labels: [", fmt.Sprintf("labels: [%s, ", agentLabel), 1)
				}
				os.WriteFile(task.Path, []byte(contentStr), 0644)
			}
		}
	}

	return ctx, nil
}

// taskHasActiveLock creates an active (non-expired) lock file.
func taskHasActiveLock(ctx context.Context, taskID, agentID string) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	now := time.Now().UTC()
	claimedAt := now.Add(-5 * time.Minute)
	expiresAt := now.Add(25 * time.Minute) // 30 min TTL, 5 min elapsed

	content := fmt.Sprintf("agent: %s\nclaimed_at: %s\nexpires_at: %s\n",
		agentID,
		claimedAt.Format(time.RFC3339),
		expiresAt.Format(time.RFC3339))

	lockPath := filepath.Join(".backlog", ".locks", taskID+".lock")
	if err := env.CreateFile(lockPath, content); err != nil {
		return ctx, fmt.Errorf("failed to create active lock file: %w", err)
	}

	// Also add the agent label to the task
	reader := support.NewTaskFileReader(env.Path(".backlog"))
	task := reader.ReadTask(taskID)
	if task.ParseErr == nil {
		taskContent, err := os.ReadFile(task.Path)
		if err == nil {
			contentStr := string(taskContent)
			agentLabel := fmt.Sprintf("agent:%s", agentID)
			if !strings.Contains(contentStr, agentLabel) {
				if strings.Contains(contentStr, "labels: []") {
					contentStr = strings.Replace(contentStr, "labels: []", fmt.Sprintf("labels: [%s]", agentLabel), 1)
				} else if strings.Contains(contentStr, "labels: [") {
					contentStr = strings.Replace(contentStr, "labels: [", fmt.Sprintf("labels: [%s, ", agentLabel), 1)
				}
				os.WriteFile(task.Path, []byte(contentStr), 0644)
			}
		}
	}

	return ctx, nil
}

// ============================================================================
// Git Sync Step Definitions
// ============================================================================

const (
	initialCommitCountKey contextKey = "initialCommitCount"
	remoteRepoPathKey     contextKey = "remoteRepoPath"
)

// aGitRepositoryIsInitialized initializes a git repository in the test environment.
func aGitRepositoryIsInitialized(ctx context.Context) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = env.TempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to initialize git repository: %w\nOutput: %s", err, output)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = env.TempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to configure git email: %w\nOutput: %s", err, output)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = env.TempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to configure git name: %w\nOutput: %s", err, output)
	}

	return ctx, nil
}

// gitSyncIsEnabledInTheConfig creates a config with git_sync enabled.
func gitSyncIsEnabledInTheConfig(ctx context.Context) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	configContent := `version: 1
defaults:
  format: table
  workspace: local
workspaces:
  local:
    backend: local
    path: ./.backlog
    default: true
    lock_mode: git
    git_sync: true
`
	if err := env.CreateFile(".backlog/config.yaml", configContent); err != nil {
		return ctx, fmt.Errorf("failed to create config file: %w", err)
	}

	// Add and commit the initial state
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = env.TempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to git add: %w\nOutput: %s", err, output)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = env.TempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to git commit: %w\nOutput: %s", err, output)
	}

	// Store the initial commit count
	count, err := getGitCommitCount(env.TempDir)
	if err != nil {
		return ctx, fmt.Errorf("failed to get commit count: %w", err)
	}
	ctx = context.WithValue(ctx, initialCommitCountKey, count)

	return ctx, nil
}

// gitSyncIsDisabledInTheConfig creates a config with git_sync disabled.
func gitSyncIsDisabledInTheConfig(ctx context.Context) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	configContent := `version: 1
defaults:
  format: table
  workspace: local
workspaces:
  local:
    backend: local
    path: ./.backlog
    default: true
    lock_mode: file
    git_sync: false
`
	if err := env.CreateFile(".backlog/config.yaml", configContent); err != nil {
		return ctx, fmt.Errorf("failed to create config file: %w", err)
	}

	// Add and commit the initial state
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = env.TempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to git add: %w\nOutput: %s", err, output)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = env.TempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to git commit: %w\nOutput: %s", err, output)
	}

	// Store the initial commit count
	count, err := getGitCommitCount(env.TempDir)
	if err != nil {
		return ctx, fmt.Errorf("failed to get commit count: %w", err)
	}
	ctx = context.WithValue(ctx, initialCommitCountKey, count)

	return ctx, nil
}

// aGitCommitShouldExistWithMessageContaining verifies a git commit exists with a message containing the substring.
func aGitCommitShouldExistWithMessageContaining(ctx context.Context, expected string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	// Get the last few commit messages
	cmd := exec.Command("git", "log", "--oneline", "-10")
	cmd.Dir = env.TempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get git log: %w\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), expected) {
		return fmt.Errorf("expected git log to contain commit message with %q, got:\n%s", expected, output)
	}

	return nil
}

// theLastGitCommitMessageShouldMatchPattern verifies the last commit message matches a pattern.
func theLastGitCommitMessageShouldMatchPattern(ctx context.Context, pattern string) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	// Get the last commit message
	cmd := exec.Command("git", "log", "-1", "--format=%s")
	cmd.Dir = env.TempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get last git commit message: %w\nOutput: %s", err, output)
	}

	message := strings.TrimSpace(string(output))
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern %q: %w", pattern, err)
	}

	if !re.MatchString(message) {
		return fmt.Errorf("expected last commit message to match pattern %q, got: %q", pattern, message)
	}

	return nil
}

// aRemoteGitRepository sets up a bare git repository as a remote.
func aRemoteGitRepository(ctx context.Context) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	// Create a bare repository in a temporary location
	remoteDir, err := os.MkdirTemp("", "backlog-remote-*")
	if err != nil {
		return ctx, fmt.Errorf("failed to create remote directory: %w", err)
	}

	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to initialize bare repository: %w\nOutput: %s", err, output)
	}

	// Add the remote to the local repository
	cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
	cmd.Dir = env.TempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to add remote: %w\nOutput: %s", err, output)
	}

	// Push the current state to the remote
	cmd = exec.Command("git", "push", "-u", "origin", "master")
	cmd.Dir = env.TempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try main branch instead of master
		cmd = exec.Command("git", "push", "-u", "origin", "main")
		cmd.Dir = env.TempDir
		if output2, err2 := cmd.CombinedOutput(); err2 != nil {
			return ctx, fmt.Errorf("failed to push to remote: %w\nOutput: %s\nAlternate: %s", err, output, output2)
		}
	}

	ctx = context.WithValue(ctx, remoteRepoPathKey, remoteDir)

	return ctx, nil
}

// theLocalRepositoryShouldBeInSyncWithRemote verifies local and remote are in sync.
func theLocalRepositoryShouldBeInSyncWithRemote(ctx context.Context) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	// Fetch from remote
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = env.TempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fetch from remote: %w\nOutput: %s", err, output)
	}

	// Check if local and remote are at the same commit
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = env.TempDir
	localHead, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get local HEAD: %w", err)
	}

	// Try main or master
	cmd = exec.Command("git", "rev-parse", "origin/main")
	cmd.Dir = env.TempDir
	remoteHead, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("git", "rev-parse", "origin/master")
		cmd.Dir = env.TempDir
		remoteHead, err = cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get remote HEAD: %w", err)
		}
	}

	if strings.TrimSpace(string(localHead)) != strings.TrimSpace(string(remoteHead)) {
		return fmt.Errorf("local HEAD (%s) does not match remote HEAD (%s)",
			strings.TrimSpace(string(localHead)), strings.TrimSpace(string(remoteHead)))
	}

	return nil
}

// theRemoteHasDifferentContentThanLocal modifies the remote to have different content.
func theRemoteHasDifferentContentThanLocal(ctx context.Context) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	remotePath, ok := ctx.Value(remoteRepoPathKey).(string)
	if !ok || remotePath == "" {
		return ctx, fmt.Errorf("remote repository path not found in context")
	}

	// Clone the remote to a temp directory, make changes, and push
	cloneDir, err := os.MkdirTemp("", "backlog-clone-*")
	if err != nil {
		return ctx, fmt.Errorf("failed to create clone directory: %w", err)
	}
	defer os.RemoveAll(cloneDir)

	cmd := exec.Command("git", "clone", remotePath, cloneDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to clone remote: %w\nOutput: %s", err, output)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "remote@example.com")
	cmd.Dir = cloneDir
	cmd.CombinedOutput()
	cmd = exec.Command("git", "config", "user.name", "Remote User")
	cmd.Dir = cloneDir
	cmd.CombinedOutput()

	// Create a new file
	if err := os.WriteFile(filepath.Join(cloneDir, "remote-change.txt"), []byte("Remote change"), 0644); err != nil {
		return ctx, fmt.Errorf("failed to create remote change file: %w", err)
	}

	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = cloneDir
	cmd.CombinedOutput()

	cmd = exec.Command("git", "commit", "-m", "Remote change")
	cmd.Dir = cloneDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to commit remote change: %w\nOutput: %s", err, output)
	}

	cmd = exec.Command("git", "push")
	cmd.Dir = cloneDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to push remote change: %w\nOutput: %s", err, output)
	}

	return ctx, nil
}

// theLocalRepositoryShouldMatchTheRemote verifies local matches remote after sync.
func theLocalRepositoryShouldMatchTheRemote(ctx context.Context) error {
	return theLocalRepositoryShouldBeInSyncWithRemote(ctx)
}

// theRemoteHasBeenUpdatedByAnotherAgent simulates another agent pushing to the remote.
func theRemoteHasBeenUpdatedByAnotherAgent(ctx context.Context) (context.Context, error) {
	return theRemoteHasDifferentContentThanLocal(ctx)
}

// noNewGitCommitsShouldExist verifies no new commits have been made.
func noNewGitCommitsShouldExist(ctx context.Context) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	initialCount, ok := ctx.Value(initialCommitCountKey).(int)
	if !ok {
		return fmt.Errorf("initial commit count not found in context")
	}

	currentCount, err := getGitCommitCount(env.TempDir)
	if err != nil {
		return fmt.Errorf("failed to get current commit count: %w", err)
	}

	if currentCount != initialCount {
		return fmt.Errorf("expected no new commits (initial: %d), but found %d commits", initialCount, currentCount)
	}

	return nil
}

// thereAreUncommittedChangesInTheRepository creates uncommitted changes.
func thereAreUncommittedChangesInTheRepository(ctx context.Context) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	// Create an uncommitted file
	if err := env.CreateFile("uncommitted.txt", "Uncommitted changes"); err != nil {
		return ctx, fmt.Errorf("failed to create uncommitted file: %w", err)
	}

	// Stage but don't commit
	cmd := exec.Command("git", "add", "uncommitted.txt")
	cmd.Dir = env.TempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to stage file: %w\nOutput: %s", err, output)
	}

	return ctx, nil
}

// getGitCommitCount returns the number of commits in the repository.
func getGitCommitCount(dir string) (int, error) {
	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to count commits: %w", err)
	}

	count := 0
	_, err = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &count)
	if err != nil {
		return 0, fmt.Errorf("failed to parse commit count: %w", err)
	}

	return count, nil
}

// ============================================================================
// Git Claim Step Definitions
// ============================================================================

// lockModeIsInTheConfig creates a config with the specified lock_mode.
func lockModeIsInTheConfig(ctx context.Context, lockMode string) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	gitSync := "false"
	if lockMode == "git" {
		gitSync = "true"
	}

	configContent := fmt.Sprintf(`version: 1
defaults:
  format: table
  workspace: local
workspaces:
  local:
    backend: local
    path: ./.backlog
    default: true
    lock_mode: %s
    git_sync: %s
`, lockMode, gitSync)

	if err := env.CreateFile(".backlog/config.yaml", configContent); err != nil {
		return ctx, fmt.Errorf("failed to create config file: %w", err)
	}

	return ctx, nil
}

// theRemoteShouldHaveTheLatestCommit verifies the remote has the latest local commit.
func theRemoteShouldHaveTheLatestCommit(ctx context.Context) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	// First, push any pending changes
	cmd := exec.Command("git", "push")
	cmd.Dir = env.TempDir
	// Ignore push errors - we just want to verify state

	// Fetch from remote
	cmd = exec.Command("git", "fetch", "origin")
	cmd.Dir = env.TempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fetch from remote: %w\nOutput: %s", err, output)
	}

	// Get local HEAD
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = env.TempDir
	localHead, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get local HEAD: %w", err)
	}

	// Get remote HEAD (try main, then master)
	cmd = exec.Command("git", "rev-parse", "origin/main")
	cmd.Dir = env.TempDir
	remoteHead, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("git", "rev-parse", "origin/master")
		cmd.Dir = env.TempDir
		remoteHead, err = cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get remote HEAD: %w", err)
		}
	}

	localStr := strings.TrimSpace(string(localHead))
	remoteStr := strings.TrimSpace(string(remoteHead))

	if localStr != remoteStr {
		return fmt.Errorf("remote HEAD (%s) does not match local HEAD (%s)", remoteStr, localStr)
	}

	return nil
}

// theLocalRepositoryShouldIncludeTheRemoteCommit verifies local repo includes remote commits.
func theLocalRepositoryShouldIncludeTheRemoteCommit(ctx context.Context) error {
	env := getTestEnv(ctx)
	if env == nil {
		return fmt.Errorf("test environment not initialized")
	}

	// Fetch from remote
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = env.TempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fetch from remote: %w\nOutput: %s", err, output)
	}

	// Check if remote HEAD is an ancestor of local HEAD
	cmd = exec.Command("git", "rev-parse", "origin/main")
	cmd.Dir = env.TempDir
	remoteHead, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("git", "rev-parse", "origin/master")
		cmd.Dir = env.TempDir
		remoteHead, err = cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get remote HEAD: %w", err)
		}
	}

	remoteStr := strings.TrimSpace(string(remoteHead))
	cmd = exec.Command("git", "merge-base", "--is-ancestor", remoteStr, "HEAD")
	cmd.Dir = env.TempDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("local repository does not include remote commit %s", remoteStr)
	}

	return nil
}

// theRemoteHasANewCommit creates a new commit on the remote.
func theRemoteHasANewCommit(ctx context.Context) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	remotePath, ok := ctx.Value(remoteRepoPathKey).(string)
	if !ok || remotePath == "" {
		return ctx, fmt.Errorf("remote repository path not found in context")
	}

	// Clone the remote to a temp directory, make changes, and push
	cloneDir, err := os.MkdirTemp("", "backlog-clone-*")
	if err != nil {
		return ctx, fmt.Errorf("failed to create clone directory: %w", err)
	}
	defer os.RemoveAll(cloneDir)

	cmd := exec.Command("git", "clone", remotePath, cloneDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to clone remote: %w\nOutput: %s", err, output)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "remote@example.com")
	cmd.Dir = cloneDir
	cmd.CombinedOutput()
	cmd = exec.Command("git", "config", "user.name", "Remote User")
	cmd.Dir = cloneDir
	cmd.CombinedOutput()

	// Create a new file
	if err := os.WriteFile(filepath.Join(cloneDir, "new-remote-file.txt"), []byte("New remote commit"), 0644); err != nil {
		return ctx, fmt.Errorf("failed to create remote file: %w", err)
	}

	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = cloneDir
	cmd.CombinedOutput()

	cmd = exec.Command("git", "commit", "-m", "New remote commit")
	cmd.Dir = cloneDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to commit on remote: %w\nOutput: %s", err, output)
	}

	cmd = exec.Command("git", "push")
	cmd.Dir = cloneDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to push to remote: %w\nOutput: %s", err, output)
	}

	return ctx, nil
}

// anotherAgentHasClaimedTaskAndPushed simulates another agent claiming a task and pushing.
func anotherAgentHasClaimedTaskAndPushed(ctx context.Context, taskID string) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	remotePath, ok := ctx.Value(remoteRepoPathKey).(string)
	if !ok || remotePath == "" {
		return ctx, fmt.Errorf("remote repository path not found in context")
	}

	// Clone the remote to a temp directory
	cloneDir, err := os.MkdirTemp("", "backlog-other-agent-*")
	if err != nil {
		return ctx, fmt.Errorf("failed to create clone directory: %w", err)
	}
	defer os.RemoveAll(cloneDir)

	cmd := exec.Command("git", "clone", remotePath, cloneDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to clone remote: %w\nOutput: %s", err, output)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "other-agent@example.com")
	cmd.Dir = cloneDir
	cmd.CombinedOutput()
	cmd = exec.Command("git", "config", "user.name", "Other Agent")
	cmd.Dir = cloneDir
	cmd.CombinedOutput()

	// Find and modify the task file to add agent claim
	taskPath := filepath.Join(cloneDir, ".backlog", "todo", taskID+"-*.md")
	matches, _ := filepath.Glob(taskPath)
	if len(matches) == 0 {
		// Try other directories
		for _, status := range []string{"backlog", "in-progress", "review", "done"} {
			taskPath = filepath.Join(cloneDir, ".backlog", status, taskID+"-*.md")
			matches, _ = filepath.Glob(taskPath)
			if len(matches) > 0 {
				break
			}
		}
	}

	if len(matches) == 0 {
		// Create a simple change instead
		changeFile := filepath.Join(cloneDir, ".backlog", "claimed-by-other.txt")
		if err := os.WriteFile(changeFile, []byte("Claimed by other-agent"), 0644); err != nil {
			return ctx, fmt.Errorf("failed to create claim marker: %w", err)
		}
	} else {
		// Modify the task file to add agent label
		taskFile := matches[0]
		content, err := os.ReadFile(taskFile)
		if err != nil {
			return ctx, fmt.Errorf("failed to read task file: %w", err)
		}
		contentStr := string(content)
		if strings.Contains(contentStr, "labels: []") {
			contentStr = strings.Replace(contentStr, "labels: []", "labels: [agent:other-agent]", 1)
		} else if strings.Contains(contentStr, "labels: [") {
			contentStr = strings.Replace(contentStr, "labels: [", "labels: [agent:other-agent, ", 1)
		}
		if err := os.WriteFile(taskFile, []byte(contentStr), 0644); err != nil {
			return ctx, fmt.Errorf("failed to write task file: %w", err)
		}

		// Move file to in-progress if it's not already there
		taskDir := filepath.Dir(taskFile)
		if filepath.Base(taskDir) != "in-progress" {
			newPath := filepath.Join(cloneDir, ".backlog", "in-progress", filepath.Base(taskFile))
			os.MkdirAll(filepath.Dir(newPath), 0755)
			os.Rename(taskFile, newPath)
		}
	}

	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = cloneDir
	cmd.CombinedOutput()

	cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("claim: %s [agent:other-agent]", taskID))
	cmd.Dir = cloneDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to commit claim: %w\nOutput: %s", err, output)
	}

	cmd = exec.Command("git", "push")
	cmd.Dir = cloneDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to push claim: %w\nOutput: %s", err, output)
	}

	return ctx, nil
}

// taskHasStaleLockFile creates a stale (expired) lock file for a task.
func taskHasStaleLockFile(ctx context.Context, taskID string) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	now := time.Now().UTC()
	claimedAt := now.Add(-2 * time.Hour)
	expiresAt := now.Add(-1 * time.Hour)

	content := fmt.Sprintf("agent: stale-agent\nclaimed_at: %s\nexpires_at: %s\n",
		claimedAt.Format(time.RFC3339),
		expiresAt.Format(time.RFC3339))

	lockPath := filepath.Join(".backlog", ".locks", taskID+".lock")
	if err := env.CreateFile(lockPath, content); err != nil {
		return ctx, fmt.Errorf("failed to create stale lock file: %w", err)
	}

	return ctx, nil
}

// theRemoteRepositoryIsUnreachable makes the remote repository unreachable.
func theRemoteRepositoryIsUnreachable(ctx context.Context) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	// Remove the remote directory to make it unreachable
	remotePath, ok := ctx.Value(remoteRepoPathKey).(string)
	if ok && remotePath != "" {
		os.RemoveAll(remotePath)
	}

	// Alternatively, set remote to an invalid URL
	cmd := exec.Command("git", "remote", "set-url", "origin", "file:///nonexistent/path")
	cmd.Dir = env.TempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return ctx, fmt.Errorf("failed to set invalid remote URL: %w\nOutput: %s", err, output)
	}

	return ctx, nil
}

// ============================================================================
// Mock GitHub API Step Definitions
// ============================================================================

const (
	mockGitHubServerKey contextKey = "mockGitHubServer"
)

// getMockGitHubServer retrieves the MockGitHubServer from context.
func getMockGitHubServer(ctx context.Context) *support.MockGitHubServer {
	if server, ok := ctx.Value(mockGitHubServerKey).(*support.MockGitHubServer); ok {
		return server
	}
	return nil
}

// aMockGitHubAPIServerIsRunning starts a mock GitHub API server.
func aMockGitHubAPIServerIsRunning(ctx context.Context) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	// Create and start the mock server
	server := support.NewMockGitHubServer()

	// Set the GITHUB_API_URL environment variable to point to the mock server
	env.SetEnv("GITHUB_API_URL", server.URL)

	// Store the server in context for cleanup and configuration
	ctx = context.WithValue(ctx, mockGitHubServerKey, server)

	return ctx, nil
}

// theMockGitHubAPIReturnsAuthErrorForInvalidTokens configures the mock to return auth errors.
func theMockGitHubAPIReturnsAuthErrorForInvalidTokens(ctx context.Context) (context.Context, error) {
	server := getMockGitHubServer(ctx)
	if server == nil {
		return ctx, fmt.Errorf("mock GitHub API server not running - call 'a mock GitHub API server is running' first")
	}

	server.AuthErrorEnabled = true
	return ctx, nil
}

// theMockGitHubAPIExpectsToken configures the mock to expect a specific token.
func theMockGitHubAPIExpectsToken(ctx context.Context, token string) (context.Context, error) {
	server := getMockGitHubServer(ctx)
	if server == nil {
		return ctx, fmt.Errorf("mock GitHub API server not running - call 'a mock GitHub API server is running' first")
	}

	server.ExpectedToken = token
	return ctx, nil
}

// theMockGitHubAPIHasTheFollowingIssues sets up mock issues for the GitHub API.
func theMockGitHubAPIHasTheFollowingIssues(ctx context.Context, table *godog.Table) (context.Context, error) {
	server := getMockGitHubServer(ctx)
	if server == nil {
		return ctx, fmt.Errorf("mock GitHub API server not running - call 'a mock GitHub API server is running' first")
	}

	if len(table.Rows) < 2 {
		return ctx, fmt.Errorf("table must have at least a header row and one data row")
	}

	header := table.Rows[0]
	colIndex := make(map[string]int)
	for i, cell := range header.Cells {
		colIndex[cell.Value] = i
	}

	var issues []support.MockGitHubIssue
	for _, row := range table.Rows[1:] {
		getValue := func(col string) string {
			if idx, ok := colIndex[col]; ok && idx < len(row.Cells) {
				return row.Cells[idx].Value
			}
			return ""
		}

		issue := support.MockGitHubIssue{
			Title:    getValue("title"),
			State:    getValue("state"),
			Assignee: getValue("assignee"),
			Body:     getValue("body"),
		}

		// Parse number
		if numStr := getValue("number"); numStr != "" {
			fmt.Sscanf(numStr, "%d", &issue.Number)
		}

		// Parse labels as comma-separated list
		if labelsStr := getValue("labels"); labelsStr != "" {
			for _, label := range strings.Split(labelsStr, ",") {
				label = strings.TrimSpace(label)
				if label != "" {
					issue.Labels = append(issue.Labels, label)
				}
			}
		}

		issues = append(issues, issue)
	}

	// Set the issues on the mock server
	server.SetIssues(issues)
	return ctx, nil
}

// aCredentialsFileWithTheFollowingContent creates a credentials file.
func aCredentialsFileWithTheFollowingContent(ctx context.Context, content *godog.DocString) (context.Context, error) {
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

	if err := env.CreateFile(".backlog/credentials.yaml", content.Content); err != nil {
		return ctx, fmt.Errorf("failed to create credentials file: %w", err)
	}

	return ctx, nil
}

// theEnvironmentVariableIsNotSet unsets an environment variable.
func theEnvironmentVariableIsNotSet(ctx context.Context, key string) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	env.UnsetEnv(key)
	return ctx, nil
}

// theEnvironmentVariableIsSetToAValidToken sets a placeholder valid token (for remote tests).
func theEnvironmentVariableIsSetToAValidToken(ctx context.Context, key string) (context.Context, error) {
	// This step is for documentation purposes in remote integration tests
	// The actual token should be provided via environment variable before running tests
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	// Check if the token is already set in the real environment
	if os.Getenv(key) == "" {
		return ctx, fmt.Errorf("environment variable %s must be set for remote tests", key)
	}

	return ctx, nil
}

// theMockGitHubAPIAuthenticatedUserIs sets the authenticated user for the mock GitHub API.
func theMockGitHubAPIAuthenticatedUserIs(ctx context.Context, username string) (context.Context, error) {
	server := getMockGitHubServer(ctx)
	if server == nil {
		return ctx, fmt.Errorf("mock GitHub API server not running - call 'a mock GitHub API server is running' first")
	}

	server.AuthenticatedUser = username
	return ctx, nil
}

// theMockGitHubIssueHasTheFollowingComments sets up mock comments for a specific GitHub issue.
func theMockGitHubIssueHasTheFollowingComments(ctx context.Context, issueNumber string, table *godog.Table) (context.Context, error) {
	server := getMockGitHubServer(ctx)
	if server == nil {
		return ctx, fmt.Errorf("mock GitHub API server not running - call 'a mock GitHub API server is running' first")
	}

	if len(table.Rows) < 2 {
		return ctx, fmt.Errorf("table must have at least a header row and one data row")
	}

	header := table.Rows[0]
	colIndex := make(map[string]int)
	for i, cell := range header.Cells {
		colIndex[cell.Value] = i
	}

	var comments []support.MockGitHubComment
	for _, row := range table.Rows[1:] {
		getValue := func(col string) string {
			if idx, ok := colIndex[col]; ok && idx < len(row.Cells) {
				return row.Cells[idx].Value
			}
			return ""
		}

		comment := support.MockGitHubComment{
			Author: getValue("author"),
			Body:   getValue("body"),
		}

		comments = append(comments, comment)
	}

	// Parse issue number
	var issueNum int
	fmt.Sscanf(issueNumber, "%d", &issueNum)

	// Set the comments on the mock server
	server.SetComments(issueNum, comments)
	return ctx, nil
}

// theJSONOutputArrayShouldHaveLength verifies that a JSON array has the expected length.
func theJSONOutputArrayShouldHaveLength(ctx context.Context, arrayPath string, expectedLength int) error {
	result := getLastResult(ctx)
	if result == nil {
		return fmt.Errorf("no command has been run")
	}

	jsonResult := support.ParseJSON(result.Stdout)
	if !jsonResult.Valid() {
		return fmt.Errorf("stdout is not valid JSON: %s\nstdout:\n%s", jsonResult.Error(), result.Stdout)
	}

	length := jsonResult.ArrayLen(arrayPath)
	if length != expectedLength {
		return fmt.Errorf("expected JSON array %q to have length %d, got %d", arrayPath, expectedLength, length)
	}

	return nil
}

// aGitHubRepositoryWithIssues sets up a mock GitHub repository with the specified issues.
// This is a convenience step that combines starting the mock server and setting up issues.
// The repository string is parsed but primarily used for documentation; the mock server
// handles all requests regardless of the repo path.
func aGitHubRepositoryWithIssues(ctx context.Context, repo string, table *godog.Table) (context.Context, error) {
	// First ensure the mock server is running
	server := getMockGitHubServer(ctx)
	if server == nil {
		// Start the mock server
		var err error
		ctx, err = aMockGitHubAPIServerIsRunning(ctx)
		if err != nil {
			return ctx, fmt.Errorf("failed to start mock GitHub API server: %w", err)
		}
		server = getMockGitHubServer(ctx)
	}

	// Now add the issues using the existing step
	return theMockGitHubAPIHasTheFollowingIssues(ctx, table)
}

// theGitHubTokenIs sets the GitHub token for authentication.
// This sets the GITHUB_TOKEN environment variable.
func theGitHubTokenIs(ctx context.Context, token string) (context.Context, error) {
	env := getTestEnv(ctx)
	if env == nil {
		return ctx, fmt.Errorf("test environment not initialized")
	}

	env.SetEnv("GITHUB_TOKEN", token)
	return ctx, nil
}

// theGitHubIssueShouldHaveLabel verifies that a GitHub issue has the specified label.
// The issue ID should be in the format "GH-{number}" or just the number.
func theGitHubIssueShouldHaveLabel(ctx context.Context, issueID, label string) error {
	server := getMockGitHubServer(ctx)
	if server == nil {
		return fmt.Errorf("mock GitHub API server not running")
	}

	// Parse issue number from ID (handle both "GH-42" and "42" formats)
	issueNumber := parseGitHubIssueNumber(issueID)
	if issueNumber <= 0 {
		return fmt.Errorf("invalid issue ID format: %s (expected 'GH-{number}' or '{number}')", issueID)
	}

	issue := server.GetIssue(issueNumber)
	if issue == nil {
		return fmt.Errorf("GitHub issue %s not found in mock server", issueID)
	}

	for _, l := range issue.Labels {
		if l == label {
			return nil
		}
	}

	return fmt.Errorf("GitHub issue %s does not have label %q (has labels: %v)", issueID, label, issue.Labels)
}

// theGitHubIssueShouldBeAssignedTo verifies that a GitHub issue is assigned to the specified user.
// The issue ID should be in the format "GH-{number}" or just the number.
func theGitHubIssueShouldBeAssignedTo(ctx context.Context, issueID, assignee string) error {
	server := getMockGitHubServer(ctx)
	if server == nil {
		return fmt.Errorf("mock GitHub API server not running")
	}

	// Parse issue number from ID (handle both "GH-42" and "42" formats)
	issueNumber := parseGitHubIssueNumber(issueID)
	if issueNumber <= 0 {
		return fmt.Errorf("invalid issue ID format: %s (expected 'GH-{number}' or '{number}')", issueID)
	}

	issue := server.GetIssue(issueNumber)
	if issue == nil {
		return fmt.Errorf("GitHub issue %s not found in mock server", issueID)
	}

	if issue.Assignee != assignee {
		return fmt.Errorf("GitHub issue %s is assigned to %q, expected %q", issueID, issue.Assignee, assignee)
	}

	return nil
}

// parseGitHubIssueNumber extracts the issue number from an ID string.
// Handles both "GH-42" and "42" formats.
func parseGitHubIssueNumber(issueID string) int {
	// Try "GH-{number}" format first
	var num int
	if _, err := fmt.Sscanf(issueID, "GH-%d", &num); err == nil {
		return num
	}
	// Try plain number format
	if _, err := fmt.Sscanf(issueID, "%d", &num); err == nil {
		return num
	}
	return 0
}
