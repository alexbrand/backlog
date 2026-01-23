Feature: Showing Tasks
  As a user of the backlog CLI
  I want to view detailed information about a specific task
  So that I can understand what needs to be done

  Scenario: Show task displays all fields
    Given a backlog with the following tasks:
      | id    | title           | status      | priority | assignee | labels        | description                  |
      | task1 | Implement auth  | in-progress | high     | alex     | feature,auth  | OAuth2 implementation needed |
    When I run "backlog show task1"
    Then the exit code should be 0
    And stdout should contain "task1"
    And stdout should contain "Implement auth"
    And stdout should contain "in-progress"
    And stdout should contain "high"
    And stdout should contain "alex"
    And stdout should contain "feature"
    And stdout should contain "auth"
    And stdout should contain "OAuth2 implementation needed"

  Scenario: Show task in JSON format
    Given a backlog with the following tasks:
      | id    | title           | status      | priority | assignee | labels        | description                  |
      | task1 | Implement auth  | in-progress | high     | alex     | feature,auth  | OAuth2 implementation needed |
    When I run "backlog show task1 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task1"
    And the JSON output should have "title" equal to "Implement auth"
    And the JSON output should have "status" equal to "in-progress"
    And the JSON output should have "priority" equal to "high"
    And the JSON output should have "assignee" equal to "alex"
    And the JSON output should have "labels" as an array
    And the JSON output should have array "labels" containing "feature"
    And the JSON output should have array "labels" containing "auth"

  Scenario: Show task with comments
    Given a backlog with the following tasks:
      | id    | title           | status      | priority | assignee | labels        | description                  |
      | task1 | Implement auth  | in-progress | high     | alex     | feature,auth  | OAuth2 implementation needed |
    And task "task1" has the following comments:
      | author | date       | body                                |
      | alex   | 2025-01-16 | Started research on OAuth providers |
      | bot    | 2025-01-17 | Found relevant documentation        |
    When I run "backlog show task1 --comments"
    Then the exit code should be 0
    And stdout should contain "Started research on OAuth providers"
    And stdout should contain "Found relevant documentation"
    And stdout should contain "@alex"
    And stdout should contain "@bot"

  Scenario: Show non-existent task returns exit code 3
    Given a fresh backlog directory
    When I run "backlog show nonexistent-task"
    Then the exit code should be 3
    And stderr should contain "not found"
