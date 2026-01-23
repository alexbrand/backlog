Feature: GitHub Status Mapping
  As a user of the backlog CLI
  I want GitHub issue labels to be mapped to canonical statuses
  So that I can work with consistent status values across backends

  # Note: These scenarios test the GitHub backend's status mapping functionality.
  # All scenarios require a mock GitHub API server for testing without real credentials.
  #
  # Default status mapping (labels):
  #   backlog     -> open issue with no status label
  #   todo        -> open issue with "ready" label
  #   in-progress -> open issue with "in-progress" label
  #   review      -> open issue with "needs-review" label
  #   done        -> closed issue (regardless of labels)

  @github
  Scenario: Default status mapping - ready label maps to todo
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
    And the mock GitHub API has the following issues:
      | number | title            | state | labels |
      | 1      | Ready for work   | open  | ready  |
    When I run "backlog show GH-1 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "todo"

  @github
  Scenario: Default status mapping - in-progress label maps to in-progress
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
    And the mock GitHub API has the following issues:
      | number | title           | state | labels      |
      | 2      | Work in flight  | open  | in-progress |
    When I run "backlog show GH-2 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "in-progress"

  @github
  Scenario: Default status mapping - needs-review label maps to review
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
    And the mock GitHub API has the following issues:
      | number | title            | state | labels       |
      | 3      | Needs review     | open  | needs-review |
    When I run "backlog show GH-3 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "review"

  @github
  Scenario: Default status mapping - no status label maps to backlog
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
    And the mock GitHub API has the following issues:
      | number | title          | state | labels     |
      | 4      | Unsorted issue | open  | bug,urgent |
    When I run "backlog show GH-4 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "backlog"

  @github
  Scenario: Default status mapping - closed issue maps to done regardless of labels
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
    And the mock GitHub API has the following issues:
      | number | title           | state  | labels      |
      | 5      | Completed task  | closed | in-progress |
    When I run "backlog list --status=all -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks[0].status" equal to "done"

  @github
  Scenario: Custom status_map in config - custom label for todo
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
          status_map:
            backlog:
              state: open
              labels: []
            todo:
              state: open
              labels:
                - prioritized
            in-progress:
              state: open
              labels:
                - wip
            review:
              state: open
              labels:
                - review-needed
            done:
              state: closed
      """
    And the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running
    And the mock GitHub API has the following issues:
      | number | title              | state | labels      |
      | 10     | Custom todo        | open  | prioritized |
      | 11     | Custom in progress | open  | wip         |
      | 12     | Custom review      | open  | review-needed |
    When I run "backlog list --status=all -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks[0].status" equal to "todo"
    And the JSON output should have "tasks[1].status" equal to "in-progress"
    And the JSON output should have "tasks[2].status" equal to "review"

  @github
  Scenario: Custom status_map - move command applies custom labels
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
          status_map:
            backlog:
              state: open
              labels: []
            todo:
              state: open
              labels:
                - prioritized
            in-progress:
              state: open
              labels:
                - wip
            review:
              state: open
              labels:
                - review-needed
            done:
              state: closed
      """
    And the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running
    And the mock GitHub API has the following issues:
      | number | title          | state | labels |
      | 20     | Task to move   | open  |        |
    When I run "backlog move GH-20 todo -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "todo"
    And the JSON output should have array "labels" containing "prioritized"

  @github
  Scenario: Unknown status maps to backlog with warning
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
    And the mock GitHub API has the following issues:
      | number | title                | state | labels         |
      | 30     | Issue with odd label | open  | unknown-status |
    When I run "backlog show GH-30 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "backlog"

  @github
  Scenario: Move to unmapped status fails
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
    And the mock GitHub API has the following issues:
      | number | title            | state | labels |
      | 40     | Task to move     | open  | ready  |
    When I run "backlog move GH-40 invalid-status"
    Then the exit code should be 1
    And stderr should contain "invalid status"
