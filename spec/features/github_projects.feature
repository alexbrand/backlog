Feature: GitHub Projects
  As a user of the backlog CLI
  I want to manage tasks stored in GitHub Projects
  So that I can use project boards for kanban-style workflow management

  # Note: These scenarios test GitHub Projects v2 integration.
  # GitHub Projects v2 uses GraphQL API for project-specific operations.
  # Issues are still managed via REST API, but project board columns are managed via GraphQL.

  Background:
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          project: 1
          status_field: Status
          api_key_env: GITHUB_TOKEN
          default: true
      """
    And the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running

  @github @projects
  Scenario: Connect to repository with project
    Given a GitHub project 1 with columns:
      | name        | id   |
      | Backlog     | COL1 |
      | Todo        | COL2 |
      | In Progress | COL3 |
      | Done        | COL4 |
    When I run "backlog config health"
    Then the exit code should be 0
    And stdout should contain "github"
    And stdout should contain "healthy"
    And stdout should contain "project"

  @github @projects
  Scenario: Connect to repository with project shows project info
    Given a GitHub project 1 with columns:
      | name        | id   |
      | Backlog     | COL1 |
      | Todo        | COL2 |
      | In Progress | COL3 |
      | Done        | COL4 |
    When I run "backlog config show -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "workspaces.github.project" equal to "1"

  @github @projects
  Scenario: Connect to repository with invalid project returns error
    Given the mock GitHub API has no project with ID 999
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          project: 999
          api_key_env: GITHUB_TOKEN
          default: true
      """
    When I run "backlog config health"
    Then the exit code should be 1
    And stderr should contain "project"

  @github @projects
  Scenario: Health check without project configured still works
    # When no project is configured, the backend should work with issues only
    Given a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          api_key_env: GITHUB_TOKEN
          default: true
      """
    When I run "backlog config health"
    Then the exit code should be 0
    And stdout should contain "github"
    And stdout should contain "healthy"

  @github @projects
  Scenario: List shows tasks from project board
    Given a GitHub project 1 with columns:
      | name        | id   |
      | Backlog     | COL1 |
      | Todo        | COL2 |
      | In Progress | COL3 |
      | Done        | COL4 |
    And the mock GitHub API has the following issues:
      | number | title              | state | labels |
      | 1      | First task         | open  |        |
      | 2      | Second task        | open  |        |
      | 3      | Third task         | open  |        |
      | 4      | Completed task     | closed|        |
    And the issue "1" is in project 1 column "Backlog"
    And the issue "2" is in project 1 column "Todo"
    And the issue "3" is in project 1 column "In Progress"
    And the issue "4" is in project 1 column "Done"
    When I run "backlog list --status=all -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks" as an array
    # When using project, status should be read from project column, not labels
    And the JSON output should have "tasks[0].status" equal to "backlog"
    And the JSON output should have "tasks[1].status" equal to "todo"
    And the JSON output should have "tasks[2].status" equal to "in-progress"
    And the JSON output should have "tasks[3].status" equal to "done"

  @github @projects
  Scenario: Move changes project column
    Given a GitHub project 1 with columns:
      | name        | id   |
      | Backlog     | COL1 |
      | Todo        | COL2 |
      | In Progress | COL3 |
      | Done        | COL4 |
    And the mock GitHub API has the following issues:
      | number | title              | state | labels |
      | 1      | Task to move       | open  |        |
    And the issue "1" is in project 1 column "Backlog"
    When I run "backlog move GH-1 in-progress"
    Then the exit code should be 0
    And the project item for issue "GH-1" should be in column "In Progress"

  @github @projects
  Scenario: Status read from project field
    Given a GitHub project 1 with columns:
      | name        | id   |
      | Backlog     | COL1 |
      | Todo        | COL2 |
      | In Progress | COL3 |
      | Review      | COL4 |
      | Done        | COL5 |
    And the mock GitHub API has the following issues:
      | number | title              | state | labels      |
      | 1      | Task with label    | open  | in-progress |
    And the issue "1" is in project 1 column "Review"
    When I run "backlog show GH-1 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    # When project is configured, status should come from project column, not labels
    And the JSON output should have "status" equal to "review"
