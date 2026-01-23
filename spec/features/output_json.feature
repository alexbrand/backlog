Feature: JSON Output Format
  As a user of the backlog CLI
  I want JSON output to be well-structured and complete
  So that I can parse it programmatically for automation

  Background:
    Given a backlog with the following tasks:
      | id    | title           | status      | priority | assignee | labels        | description           |
      | task1 | First task      | todo        | high     | alice    | feature,api   | First task details    |
      | task2 | Second task     | in-progress | medium   |          | bug           | Second task details   |
      | task3 | Third task      | backlog     | low      | bob      |               | Third task details    |

  Scenario: JSON output is valid JSON
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    When I run "backlog show task1 -f json"
    Then the exit code should be 0
    And the JSON output should be valid

  Scenario: JSON output includes all task fields
    When I run "backlog show task1 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task1"
    And the JSON output should have "title" equal to "First task"
    And the JSON output should have "status" equal to "todo"
    And the JSON output should have "priority" equal to "high"
    And the JSON output should have "assignee" equal to "alice"
    And the JSON output should have "labels" as an array
    And the JSON output should have array "labels" containing "feature"
    And the JSON output should have array "labels" containing "api"
    When I run "backlog show task2 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task2"
    And the JSON output should have "title" equal to "Second task"
    And the JSON output should have "status" equal to "in-progress"
    And the JSON output should have "priority" equal to "medium"
    # Empty assignee should still be present
    And the JSON output should have "assignee" equal to ""

  Scenario: JSON output includes count and hasMore
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks" as an array
    And the JSON output should have "count" equal to "3"
    And the JSON output should have "hasMore" equal to "false"
    # Test hasMore when limited
    When I run "backlog list --limit=2 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "count" equal to "2"
    And the JSON output should have "hasMore" equal to "true"

  Scenario: JSON error output format
    When I run "backlog show nonexistent-task -f json"
    Then the exit code should be 3
    And the JSON output should be valid
    And the JSON output should have "error" as an object
    And the JSON output should have "error.code" equal to "NOT_FOUND"
    And the JSON output should have "error.message" containing "not found"
