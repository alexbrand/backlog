Feature: Linear CRUD Operations
  As a user of the backlog CLI
  I want to create, read, update, and move tasks via Linear Issues
  So that I can manage my Linear issues with a unified interface

  # Note: These scenarios test the Linear backend's CRUD operations.
  # All scenarios require a mock Linear API server for testing without real credentials.

  Background:
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: linear
      workspaces:
        linear:
          backend: linear
          team: ENG
          api_key_env: LINEAR_API_KEY
          default: true
      """
    And the environment variable "LINEAR_API_KEY" is "lin_api_valid_test_key"
    And a mock Linear API server is running

  @linear
  Scenario: List fetches issues from team
    Given the mock Linear API has the following issues:
      | identifier | title              | state       | priority | assignee | team |
      | ENG-1      | Implement feature  | Todo        | high     | alice    | ENG  |
      | ENG-2      | Fix critical bug   | In Progress | urgent   | bob      | ENG  |
      | ENG-3      | Update docs        | Backlog     | low      |          | ENG  |
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks" as an array

  @linear
  Scenario: List fetches issues with correct status mapping
    Given the mock Linear API has the following issues:
      | identifier | title              | state       | priority | assignee | team |
      | ENG-10     | Backlog task       | Backlog     | medium   |          | ENG  |
      | ENG-11     | Todo task          | Todo        | medium   |          | ENG  |
      | ENG-12     | In progress task   | In Progress | medium   |          | ENG  |
      | ENG-13     | Review task        | In Review   | medium   |          | ENG  |
      | ENG-14     | Done task          | Done        | medium   |          | ENG  |
    When I run "backlog list --status=todo -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks" as an array

  @linear
  Scenario: List filters by priority
    Given the mock Linear API has the following issues:
      | identifier | title              | state | priority | assignee | team |
      | ENG-20     | Urgent task        | Todo  | urgent   |          | ENG  |
      | ENG-21     | High priority task | Todo  | high     |          | ENG  |
      | ENG-22     | Low priority task  | Todo  | low      |          | ENG  |
    When I run "backlog list --priority=urgent -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks" as an array

  @linear
  Scenario: Add creates Linear issue
    When I run "backlog add 'New feature request' -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "title" equal to "New feature request"
    And the JSON output should have "id" matching pattern "ENG-[0-9]+"
    And the JSON output should have "url" containing "linear.app"

  @linear
  Scenario: Add creates Linear issue with priority
    When I run "backlog add 'Urgent bug fix' --priority=urgent -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "title" equal to "Urgent bug fix"
    And the JSON output should have "priority" equal to "urgent"

  @linear
  Scenario: Add creates Linear issue with description
    When I run "backlog add 'Feature with details' --description='This is a detailed description of the feature.' -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "title" equal to "Feature with details"

  @linear
  Scenario: Show fetches issue details
    Given the mock Linear API has the following issues:
      | identifier | title              | state | priority | assignee | description                   | team |
      | ENG-42     | Implement feature  | Todo  | high     | alice    | Detailed feature description  | ENG  |
    When I run "backlog show ENG-42 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "ENG-42"
    And the JSON output should have "title" equal to "Implement feature"
    And the JSON output should have "status" equal to "todo"
    And the JSON output should have "assignee" equal to "alice"

  @linear
  Scenario: Show fetches issue details in table format
    Given the mock Linear API has the following issues:
      | identifier | title       | state | priority | assignee | description              | team |
      | ENG-99     | Big feature | Todo  | high     |          | This is a big feature... | ENG  |
    When I run "backlog show ENG-99"
    Then the exit code should be 0
    And stdout should contain "ENG-99"
    And stdout should contain "Big feature"

  @linear
  Scenario: Show non-existent issue returns exit code 3
    When I run "backlog show ENG-9999"
    Then the exit code should be 3
    And stderr should contain "not found"

  @linear
  Scenario: Show with JSON error format for non-existent issue
    When I run "backlog show ENG-9999 -f json"
    Then the exit code should be 3
    And the JSON output should be valid
    And the JSON output should have "error" as an object
    And the JSON output should have "error.code" equal to "NOT_FOUND"

  @linear
  Scenario: Edit updates issue title
    Given the mock Linear API has the following issues:
      | identifier | title          | state | priority | assignee | description     | team |
      | ENG-10     | Original title | Todo  | medium   | bob      | Task to update  | ENG  |
    When I run "backlog edit ENG-10 --title='Updated title' -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "ENG-10"
    And the JSON output should have "title" equal to "Updated title"

  @linear
  Scenario: Edit updates issue priority
    Given the mock Linear API has the following issues:
      | identifier | title        | state | priority | assignee | description | team |
      | ENG-11     | Low priority | Todo  | low      |          | Some task   | ENG  |
    When I run "backlog edit ENG-11 --priority=urgent -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "priority" equal to "urgent"

  @linear
  Scenario: Edit non-existent issue returns exit code 3
    When I run "backlog edit ENG-9999 --title='New title'"
    Then the exit code should be 3
    And stderr should contain "not found"

  @linear
  Scenario: Move changes issue state to in-progress
    Given the mock Linear API has the following issues:
      | identifier | title            | state | priority | assignee | description   | team |
      | ENG-20     | Task to progress | Todo  | medium   |          | Move this one | ENG  |
    When I run "backlog move ENG-20 in-progress -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "ENG-20"
    And the JSON output should have "status" equal to "in-progress"

  @linear
  Scenario: Move changes issue state to review
    Given the mock Linear API has the following issues:
      | identifier | title           | state       | priority | assignee | description | team |
      | ENG-21     | Task to review  | In Progress | medium   |          | Review me   | ENG  |
    When I run "backlog move ENG-21 review -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "review"

  @linear
  Scenario: Move changes issue state to done
    Given the mock Linear API has the following issues:
      | identifier | title            | state | priority | assignee | description | team |
      | ENG-22     | Task to complete | Todo  | medium   |          | Complete me | ENG  |
    When I run "backlog move ENG-22 done -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "done"

  @linear
  Scenario: Move to invalid status fails
    Given the mock Linear API has the following issues:
      | identifier | title      | state | priority | assignee | description | team |
      | ENG-24     | Valid task | Todo  | medium   |          | Move test   | ENG  |
    When I run "backlog move ENG-24 invalid-status"
    Then the exit code should be 1
    And stderr should contain "invalid status"

  @linear
  Scenario: Move non-existent issue returns exit code 3
    When I run "backlog move ENG-9999 todo"
    Then the exit code should be 3
    And stderr should contain "not found"
