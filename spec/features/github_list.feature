Feature: GitHub List
  As a user of the backlog CLI
  I want to list tasks from GitHub Issues
  So that I can see my GitHub issues in a unified format

  # Note: These scenarios test the GitHub backend's list functionality.
  # All scenarios require a mock GitHub API server for testing without real credentials.

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
          api_key_env: GITHUB_TOKEN
          default: true
      """
    And the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running

  @github
  Scenario: List fetches issues from repository
    Given the mock GitHub API has the following issues:
      | number | title              | state | labels         |
      | 1      | First issue        | open  | ready          |
      | 2      | Second issue       | open  | in-progress    |
      | 3      | Third issue        | open  |                |
      | 4      | Closed issue       | closed|                |
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks" as an array
    And the JSON output should have "tasks[0].title" equal to "First issue"
    And the JSON output should have "tasks[1].title" equal to "Second issue"
    And the JSON output should have "tasks[2].title" equal to "Third issue"

  @github
  Scenario: List maps issue labels to status
    Given the mock GitHub API has the following issues:
      | number | title              | state  | labels         |
      | 1      | Ready task         | open   | ready          |
      | 2      | In progress task   | open   | in-progress    |
      | 3      | Review task        | open   | needs-review   |
      | 4      | Backlog task       | open   |                |
      | 5      | Done task          | closed |                |
    When I run "backlog list --status=all -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks[0].status" equal to "todo"
    And the JSON output should have "tasks[1].status" equal to "in-progress"
    And the JSON output should have "tasks[2].status" equal to "review"
    And the JSON output should have "tasks[3].status" equal to "backlog"
    And the JSON output should have "tasks[4].status" equal to "done"

  @github
  Scenario: List filters by status via labels
    Given the mock GitHub API has the following issues:
      | number | title              | state | labels         |
      | 1      | Ready task one     | open  | ready          |
      | 2      | In progress task   | open  | in-progress    |
      | 3      | Ready task two     | open  | ready          |
      | 4      | Backlog task       | open  |                |
    When I run "backlog list --status=todo -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "count" equal to "2"
    And the JSON output should have "tasks[0].title" equal to "Ready task one"
    And the JSON output should have "tasks[1].title" equal to "Ready task two"

  @github
  Scenario: List filters by assignee
    Given the mock GitHub API has the following issues:
      | number | title              | state | labels | assignee |
      | 1      | Alice's task       | open  | ready  | alice    |
      | 2      | Bob's task         | open  | ready  | bob      |
      | 3      | Alice's other task | open  | ready  | alice    |
      | 4      | Unassigned task    | open  | ready  |          |
    When I run "backlog list --assignee=alice -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "count" equal to "2"
    And the JSON output should have "tasks[0].title" equal to "Alice's task"
    And the JSON output should have "tasks[0].assignee" equal to "alice"
    And the JSON output should have "tasks[1].title" equal to "Alice's other task"
    And the JSON output should have "tasks[1].assignee" equal to "alice"

  @github
  Scenario: List respects limit
    Given the mock GitHub API has the following issues:
      | number | title              | state | labels |
      | 1      | First issue        | open  | ready  |
      | 2      | Second issue       | open  | ready  |
      | 3      | Third issue        | open  | ready  |
      | 4      | Fourth issue       | open  | ready  |
      | 5      | Fifth issue        | open  | ready  |
    When I run "backlog list --limit=3 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "count" equal to "3"
    And the JSON output should have "hasMore" equal to "true"
    And the JSON output should have "tasks[0].title" equal to "First issue"
    And the JSON output should have "tasks[1].title" equal to "Second issue"
    And the JSON output should have "tasks[2].title" equal to "Third issue"
