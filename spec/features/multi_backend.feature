Feature: Multi-Backend Support
  As a user of the backlog CLI
  I want to switch between different backend workspaces
  So that I can manage tasks across multiple issue trackers with consistent commands

  # Note: These scenarios verify that the CLI provides a unified interface
  # across different backends (local, GitHub, Linear).

  @multi-backend
  Scenario: Switch between workspaces with different backends
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: local
      workspaces:
        local:
          backend: local
          path: ./.backlog
          default: true
        github:
          backend: github
          repo: test-owner/test-repo
          api_key_env: GITHUB_TOKEN
        linear:
          backend: linear
          team: ENG
          api_key_env: LINEAR_API_KEY
      """
    And a backlog with the following tasks:
      | id    | title           | status | priority |
      | task1 | Local task      | todo   | high     |
    And the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running
    And the mock GitHub API has the following issues:
      | number | title              | state | labels |
      | 1      | GitHub task        | open  | ready  |
    And the environment variable "LINEAR_API_KEY" is "lin_api_valid_test_key"
    And a mock Linear API server is running
    And the mock Linear API has the following issues:
      | identifier | title              | state | priority | assignee | team |
      | ENG-1      | Linear task        | Todo  | high     |          | ENG  |
    # Test local workspace (default)
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks[0].title" equal to "Local task"
    # Test GitHub workspace
    When I run "backlog list -w github -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks[0].title" equal to "GitHub task"
    # Test Linear workspace
    When I run "backlog list -w linear -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks" as an array

  @multi-backend
  Scenario: Same command syntax works across backends
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: local
      workspaces:
        local:
          backend: local
          path: ./.backlog
          default: true
        github:
          backend: github
          repo: test-owner/test-repo
          api_key_env: GITHUB_TOKEN
      """
    And a backlog with the following tasks:
      | id    | title           | status | priority |
      | task1 | Local task      | todo   | high     |
    And the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running
    And the mock GitHub API has the following issues:
      | number | title              | state | labels |
      | 1      | GitHub task        | open  | ready  |
    # Verify 'show' works with same syntax for local
    When I run "backlog show task1 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task1"
    And the JSON output should have "title" equal to "Local task"
    # Verify 'show' works with same syntax for GitHub
    When I run "backlog show GH-1 -w github -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "title" equal to "GitHub task"

  @multi-backend
  Scenario: Output format consistent across backends
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: local
      workspaces:
        local:
          backend: local
          path: ./.backlog
          default: true
        github:
          backend: github
          repo: test-owner/test-repo
          api_key_env: GITHUB_TOKEN
      """
    And a backlog with the following tasks:
      | id    | title           | status   | priority |
      | task1 | Local task      | todo     | high     |
    And the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running
    And the mock GitHub API has the following issues:
      | number | title              | state | labels |
      | 1      | GitHub task        | open  | ready  |
    # Verify JSON output structure is consistent for local
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks" as an array
    And the JSON output should have "count" equal to "1"
    # Verify JSON output structure is consistent for GitHub
    When I run "backlog list -w github -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks" as an array
    And the JSON output should have "count" equal to "1"
