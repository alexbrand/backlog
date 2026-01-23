Feature: GitHub CRUD Operations
  As a user of the backlog CLI
  I want to create, read, update, and move tasks via GitHub Issues
  So that I can manage my GitHub issues with a unified interface

  # Note: These scenarios test the GitHub backend's CRUD operations.
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
  Scenario: Add creates GitHub issue
    When I run "backlog add 'New feature request' -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "title" equal to "New feature request"
    And the JSON output should have "id" matching pattern "GH-[0-9]+"
    And the JSON output should have "url" containing "github.com"

  @github
  Scenario: Add creates GitHub issue with priority and labels
    When I run "backlog add 'Urgent bug fix' --priority=urgent --label=bug --label=critical -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "title" equal to "Urgent bug fix"
    And the JSON output should have "labels" as an array
    And the JSON output should have array "labels" containing "bug"
    And the JSON output should have array "labels" containing "critical"

  @github
  Scenario: Add creates GitHub issue with description
    When I run "backlog add 'Feature with details' --description='This is a detailed description of the feature.' -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "title" equal to "Feature with details"

  @github
  Scenario: Add creates GitHub issue with status sets label
    When I run "backlog add 'Ready for work' --status=todo -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "todo"
    And the JSON output should have array "labels" containing "ready"

  @github
  Scenario: Show fetches issue details
    Given the mock GitHub API has the following issues:
      | number | title              | state | labels         | assignee | body                        |
      | 42     | Implement feature  | open  | ready,feature  | alice    | Detailed feature description |
    When I run "backlog show GH-42 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "GH-42"
    And the JSON output should have "title" equal to "Implement feature"
    And the JSON output should have "status" equal to "todo"
    And the JSON output should have "assignee" equal to "alice"
    And the JSON output should have "labels" as an array
    And the JSON output should have array "labels" containing "feature"

  @github
  Scenario: Show fetches issue details with full description
    Given the mock GitHub API has the following issues:
      | number | title       | state | labels | assignee | body                                    |
      | 99     | Big feature | open  | ready  |          | # Overview\n\nThis is a big feature... |
    When I run "backlog show GH-99"
    Then the exit code should be 0
    And stdout should contain "GH-99"
    And stdout should contain "Big feature"
    And stdout should contain "Overview"

  @github
  Scenario: Show non-existent issue returns exit code 3
    When I run "backlog show GH-9999"
    Then the exit code should be 3
    And stderr should contain "not found"

  @github
  Scenario: Show with JSON error format for non-existent issue
    When I run "backlog show GH-9999 -f json"
    Then the exit code should be 3
    And the JSON output should be valid
    And the JSON output should have "error" as an object
    And the JSON output should have "error.code" equal to "NOT_FOUND"

  @github
  Scenario: Edit updates issue title
    Given the mock GitHub API has the following issues:
      | number | title          | state | labels | assignee | body            |
      | 10     | Original title | open  | ready  | bob      | Task to update  |
    When I run "backlog edit GH-10 --title='Updated title' -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "GH-10"
    And the JSON output should have "title" equal to "Updated title"

  @github
  Scenario: Edit updates issue priority
    Given the mock GitHub API has the following issues:
      | number | title        | state | labels | assignee | body       |
      | 11     | Low priority | open  | ready  |          | Some task  |
    When I run "backlog edit GH-11 --priority=urgent -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "priority" equal to "urgent"

  @github
  Scenario: Edit adds label to issue
    Given the mock GitHub API has the following issues:
      | number | title      | state | labels | assignee | body      |
      | 12     | Label test | open  | ready  |          | Add label |
    When I run "backlog edit GH-12 --add-label=backend -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have array "labels" containing "ready"
    And the JSON output should have array "labels" containing "backend"

  @github
  Scenario: Edit removes label from issue
    Given the mock GitHub API has the following issues:
      | number | title        | state | labels           | assignee | body         |
      | 13     | Remove label | open  | ready,deprecated |          | Remove label |
    When I run "backlog edit GH-13 --remove-label=deprecated -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have array "labels" containing "ready"
    And the JSON output should not have array "labels" containing "deprecated"

  @github
  Scenario: Edit non-existent issue returns exit code 3
    When I run "backlog edit GH-9999 --title='New title'"
    Then the exit code should be 3
    And stderr should contain "not found"

  @github
  Scenario: Move updates status labels
    Given the mock GitHub API has the following issues:
      | number | title            | state | labels | assignee | body          |
      | 20     | Task to progress | open  | ready  |          | Move this one |
    When I run "backlog move GH-20 in-progress -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "GH-20"
    And the JSON output should have "status" equal to "in-progress"
    And the JSON output should have array "labels" containing "in-progress"
    And the JSON output should not have array "labels" containing "ready"

  @github
  Scenario: Move to review status
    Given the mock GitHub API has the following issues:
      | number | title         | state | labels      | assignee | body       |
      | 21     | Task to review| open  | in-progress |          | Review me  |
    When I run "backlog move GH-21 review -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "review"
    And the JSON output should have array "labels" containing "needs-review"
    And the JSON output should not have array "labels" containing "in-progress"

  @github
  Scenario: Move to done closes issue
    Given the mock GitHub API has the following issues:
      | number | title           | state | labels | assignee | body        |
      | 22     | Task to complete| open  | ready  |          | Complete me |
    When I run "backlog move GH-22 done -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "done"

  @github
  Scenario: Move from done reopens issue
    Given the mock GitHub API has the following issues:
      | number | title        | state  | labels | assignee | body       |
      | 23     | Closed task  | closed |        |          | Reopen me  |
    When I run "backlog move GH-23 todo -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "todo"
    And the JSON output should have array "labels" containing "ready"

  @github
  Scenario: Move to invalid status fails
    Given the mock GitHub API has the following issues:
      | number | title      | state | labels | assignee | body      |
      | 24     | Valid task | open  | ready  |          | Move test |
    When I run "backlog move GH-24 invalid-status"
    Then the exit code should be 1
    And stderr should contain "invalid status"

  @github
  Scenario: Move non-existent issue returns exit code 3
    When I run "backlog move GH-9999 todo"
    Then the exit code should be 3
    And stderr should contain "not found"

  @github
  Scenario: Move with comment adds issue comment
    Given the mock GitHub API has the following issues:
      | number | title            | state | labels | assignee | body          |
      | 25     | Task with update | open  | ready  |          | Update status |
    When I run "backlog move GH-25 in-progress --comment='Starting work on this'"
    Then the exit code should be 0
    And stdout should contain "GH-25"
