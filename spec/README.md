# Executable Specification

This directory contains the Gherkin-based executable specification for the `backlog` CLI. The specs are written using [Cucumber](https://cucumber.io/) syntax and executed with [godog](https://github.com/cucumber/godog).

## Directory Structure

```
spec/
├── features/           # Gherkin .feature files
│   ├── init.feature
│   ├── add.feature
│   ├── list.feature
│   └── ...
├── steps/              # Step definitions (Go)
│   └── common_steps.go
├── support/            # Test helpers and utilities
│   ├── testenv.go      # Test environment setup/teardown
│   ├── cli.go          # CLI runner helper
│   ├── fixtures.go     # Fixture loader
│   ├── json.go         # JSON output parser
│   ├── taskfile.go     # Task file reader
│   ├── config.go       # Config file generator
│   ├── mockgithub.go   # Mock GitHub API server
│   └── mocklinear.go   # Mock Linear API server
├── cmd/
│   └── genreport/      # HTML report generator
├── main_test.go        # Godog test runner entry point
└── README.md           # This file
```

## Running Specs

### Prerequisites

1. Go 1.22 or later
2. The `backlog` binary must be built and available in your PATH or current directory

Build the CLI first:

```bash
go build -o backlog ./cmd/backlog
```

### Make Targets

The project provides several Make targets for running specs:

```bash
# Run all specs (excludes @remote tests by default)
make spec

# Run only local backend specs
make spec-local

# Run GitHub backend specs (uses mock server)
make spec-github

# Run Linear backend specs (uses mock server)
make spec-linear

# Run all specs including remote backend tests
make spec-all

# Run specs with coverage reporting
make spec-coverage

# Generate HTML coverage report
make spec-coverage-html

# Generate Cucumber JSON report
make spec-report

# Generate HTML spec report for documentation
make spec-report-html
```

### Running Directly with Go

You can also run specs directly using `go test`:

```bash
# Run all specs (excludes @remote by default)
cd spec && go test -run TestFeatures -v .

# Run with specific tags
cd spec && GODOG_TAGS="@github" go test -run TestFeatures -v .

# Run all specs including remote backends
cd spec && GODOG_TAGS="" go test -run TestFeatures -v .
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GODOG_TAGS` | Filter scenarios by tag (e.g., `@github`, `~@remote`) | `~@remote` |
| `GODOG_FORMAT` | Output format (`pretty`, `progress`, `cucumber`) | `pretty` |
| `GODOG_JSON_OUTPUT` | Path to write Cucumber JSON report | (none) |

## Tags

Scenarios are tagged to allow selective execution:

| Tag | Description |
|-----|-------------|
| `@local` | Local backend specs |
| `@github` | GitHub backend specs |
| `@linear` | Linear backend specs |
| `@remote` | All remote backend specs (excluded by default) |
| `@wip` | Work in progress (not yet implemented) |

Examples:

```bash
# Run only GitHub specs
GODOG_TAGS="@github" make spec

# Run everything except remote backends
GODOG_TAGS="~@remote" make spec

# Run only local specs
GODOG_TAGS="@local" make spec
```

## Feature Files

Feature files are written in Gherkin syntax. Each file describes a feature of the CLI with scenarios that test specific behaviors.

Example (`features/init.feature`):

```gherkin
Feature: Initialization
  As a user of the backlog CLI
  I want to initialize a backlog in my project directory
  So that I can start tracking tasks

  Scenario: Initialize backlog in empty directory
    When I run "backlog init"
    Then the exit code should be 0
    And stdout should contain "Initialized"
    And the directory ".backlog" should exist
```

## Step Definitions

Step definitions map Gherkin steps to Go code. They are located in `steps/common_steps.go`.

### Available Steps

#### Given Steps

| Step | Description |
|------|-------------|
| `Given a fresh backlog directory` | Creates an initialized `.backlog/` directory |
| `Given a backlog with the following tasks:` | Sets up tasks from a table |
| `Given a file "path" with content "content"` | Creates a file with content |
| `Given a task "title" exists with status "status"` | Creates a task with specific status |
| `Given a task "title" exists with priority "priority"` | Creates a task with specific priority |
| `Given a task "title" exists with labels "labels"` | Creates a task with labels |
| `Given the environment variable "name" is "value"` | Sets an environment variable |
| `Given task "title" is claimed by agent "agent"` | Creates a claimed task |
| `Given the agent ID is "id"` | Sets the agent ID for claims |

#### When Steps

| Step | Description |
|------|-------------|
| `When I run "command"` | Executes a CLI command |

#### Then Steps

| Step | Description |
|------|-------------|
| `Then the exit code should be N` | Verifies exit code |
| `Then stdout should contain "text"` | Checks stdout contains text |
| `Then stdout should not contain "text"` | Checks stdout does not contain text |
| `Then stderr should contain "text"` | Checks stderr contains text |
| `Then stdout should be empty` | Verifies stdout is empty |
| `Then stderr should be empty` | Verifies stderr is empty |
| `Then the output should match:` | Compares output to docstring |
| `Then the JSON output should have "path" equal to "value"` | Checks JSON field value |
| `Then the JSON output should be valid` | Verifies output is valid JSON |
| `Then the directory "path" should exist` | Checks directory exists |
| `Then the file "path" should exist` | Checks file exists |
| `Then a task file should exist in "status" directory` | Checks task file location |
| `Then the task "title" should have status "status"` | Verifies task status |
| `Then the task "title" should have priority "priority"` | Verifies task priority |
| `Then the task "title" should have label "label"` | Verifies task has label |
| `Then the task "title" should not have label "label"` | Verifies task lacks label |
| `Then task "title" should be claimed by "agent"` | Verifies task claim |
| `Then task "title" should not be claimed` | Verifies task is unclaimed |

## Test Infrastructure

### TestEnv

The `TestEnv` struct (`support/testenv.go`) manages test isolation:

- Creates a temporary directory for each scenario
- Cleans up after each scenario
- Provides helpers for file operations

### CLIRunner

The `CLIRunner` struct (`support/cli.go`) executes CLI commands:

- Captures stdout, stderr, and exit code
- Supports environment variable injection
- Works from the test environment's temp directory

### Mock Servers

For testing remote backends, mock servers simulate API responses:

- `mockgithub.go` - Mock GitHub REST and GraphQL APIs
- `mocklinear.go` - Mock Linear GraphQL API

## Reports

### Coverage Report

Generate a coverage report to see which code is exercised by the specs:

```bash
make spec-coverage-html
# Open spec/coverage.html in a browser
```

### HTML Spec Report

Generate a human-readable HTML report of all scenarios:

```bash
make spec-report-html
# Open spec/report.html in a browser
```

The HTML report serves as living documentation, showing all features and their scenarios with pass/fail status.

## Adding New Scenarios

This section explains how to add new test scenarios to the executable specification.

### 1. Choose or Create a Feature File

Feature files live in `spec/features/` and are organized by functionality:

- **CRUD operations**: `init.feature`, `add.feature`, `list.feature`, `show.feature`, `move.feature`, `edit.feature`
- **Agent coordination**: `claim.feature`, `release.feature`, `next.feature`, `comment.feature`, `locking.feature`
- **Output formats**: `output_table.feature`, `output_json.feature`, `output_plain.feature`
- **Configuration**: `config.feature`, `global_flags.feature`
- **Error handling**: `errors.feature`
- **Backend-specific**: `github_*.feature`, `linear_*.feature`, `git_*.feature`
- **Integration**: `multi_backend.feature`, `agent_workflow.feature`

If your scenario fits an existing feature, add it there. Otherwise, create a new `.feature` file.

### 2. Write the Feature Description

Every feature file should start with a description explaining the feature's purpose:

```gherkin
Feature: Feature Name
  As a [type of user]
  I want to [do something]
  So that [benefit/value]
```

### 3. Write the Scenario

Scenarios follow the Given-When-Then structure:

```gherkin
Scenario: Descriptive name of what is being tested
  Given [precondition - set up the test state]
  And [additional precondition if needed]
  When [action - the command being tested]
  Then [expected outcome]
  And [additional expectations]
```

**Example:**

```gherkin
Scenario: Add task with custom priority
  Given a fresh backlog directory
  When I run "backlog add 'Important task' --priority=high"
  Then the exit code should be 0
  And a task file should exist in "backlog" directory
  And the created task should have priority "high"
```

### 4. Use Existing Step Definitions

Before creating new step definitions, check if an existing step can be reused. Common steps are documented in the "Available Steps" section above.

**Tips:**

- Use `Given a fresh backlog directory` to start with a clean state
- Use `Given a backlog with the following tasks:` for scenarios that need pre-existing data
- Use `When I run "command"` for all CLI invocations
- Use appropriate `Then` steps for verification

### 5. Use Data Tables for Multiple Items

When setting up multiple tasks or verifying multiple values, use Gherkin data tables:

```gherkin
Scenario: List multiple tasks
  Given a backlog with the following tasks:
    | id  | title           | status      | priority |
    | 001 | First task      | todo        | high     |
    | 002 | Second task     | in-progress | medium   |
    | 003 | Third task      | backlog     | low      |
  When I run "backlog list -f json"
  Then the exit code should be 0
  And the JSON output should have array length "tasks" equal to 3
```

### 6. Use Scenario Outlines for Repeated Tests

When testing the same behavior with different inputs, use Scenario Outlines:

```gherkin
Scenario Outline: Move task to each valid status
  Given a fresh backlog directory
  And a task "Test task" exists with status "backlog"
  When I run "backlog move 001 <status>"
  Then the exit code should be 0
  And the task "Test task" should have status "<status>"

  Examples:
    | status      |
    | todo        |
    | in-progress |
    | review      |
    | done        |
```

### 7. Use Tags for Categorization

Apply appropriate tags to scenarios for selective execution:

```gherkin
@local
Feature: Local Backend Tasks
  ...

@github @remote
Scenario: GitHub-specific behavior
  ...

@wip
Scenario: Work in progress (not yet implemented)
  ...
```

### 8. Create New Step Definitions (If Needed)

If no existing step matches your needs, add a new step definition in `spec/steps/common_steps.go`:

1. **Define the step function:**

```go
// myNewStep implements a custom verification.
func myNewStep(ctx context.Context, param string) error {
    env := getTestEnv(ctx)
    if env == nil {
        return fmt.Errorf("test environment not initialized")
    }

    // Your verification logic here

    return nil
}
```

2. **Register the step in `InitializeCommonSteps`:**

```go
func InitializeCommonSteps(ctx *godog.ScenarioContext) {
    // ... existing steps ...

    ctx.Step(`^my new step with "([^"]*)"$`, myNewStep)
}
```

3. **Follow the naming conventions** (see "Step Definition Conventions" below).

### 9. Run Your Scenario

Test your new scenario:

```bash
# Run all specs to see if it passes
make spec

# Run with verbose output
cd spec && go test -run TestFeatures -v .

# Run specific tag
GODOG_TAGS="@local" make spec
```

### 10. Update spec-tasks.md

After adding scenarios, update the `spec-tasks.md` file to track what was added:

1. Mark any related tasks as complete
2. Update the Progress Summary table with new scenario counts
