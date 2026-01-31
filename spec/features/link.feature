Feature: Task Dependencies
  As a user of the backlog CLI
  I want to link tasks with dependency relationships
  So that I can track which tasks block other tasks

  Background:
    Given a backlog with the following tasks:
      | id    | title          | status  | priority | assignee | labels  |
      | task1 | First task     | todo    | high     |          | backend |
      | task2 | Second task    | todo    | urgent   |          | backend |
      | task3 | Third task     | todo    | medium   |          | backend |
      | task4 | Done blocker   | done    | high     |          | backend |

  Scenario: Link two tasks with blocks
    When I run "backlog link task1 --blocks task2"
    Then the exit code should be 0
    And stdout should contain "task1"
    And stdout should contain "task2"

  Scenario: Link two tasks with blocked-by
    When I run "backlog link task2 --blocked-by task1"
    Then the exit code should be 0
    And stdout should contain "task2"
    And stdout should contain "task1"

  Scenario: Show displays blocking relationships
    When I run "backlog link task1 --blocks task2"
    And I run "backlog show task1"
    Then the exit code should be 0
    And stdout should contain "Blocks:"
    And stdout should contain "task2"

  Scenario: Show displays blocked-by relationships
    When I run "backlog link task2 --blocked-by task1"
    And I run "backlog show task2"
    Then the exit code should be 0
    And stdout should contain "Blocked by:"
    And stdout should contain "task1"

  Scenario: Unlink removes dependency
    When I run "backlog link task1 --blocks task2"
    And I run "backlog unlink task1 --blocks task2"
    Then the exit code should be 0
    When I run "backlog show task1"
    Then stdout should not contain "Blocks:"

  Scenario: Next skips blocked tasks
    When I run "backlog link task2 --blocked-by task1"
    And I run "backlog next"
    Then the exit code should be 0
    And stdout should contain "task1"
    And stdout should not contain "task2"

  Scenario: Next returns blocked task when blocker is done
    When I run "backlog link task3 --blocked-by task4"
    And I run "backlog next"
    Then the exit code should be 0
    And stdout should contain "task2"

  Scenario: Add task with --blocked-by flag
    When I run "backlog add 'New task' --blocked-by task1"
    Then the exit code should be 0
    When I run "backlog show task1"
    Then stdout should contain "Blocks:"

  Scenario: Edit task with --blocks flag
    When I run "backlog edit task1 --blocks task3"
    Then the exit code should be 0
    When I run "backlog show task1"
    Then stdout should contain "Blocks:"
    And stdout should contain "task3"

  Scenario: Link requires exactly one flag
    When I run "backlog link task1"
    Then the exit code should be 1
    And stderr should contain "one of --blocks or --blocked-by is required"

  Scenario: Link with both flags fails
    When I run "backlog link task1 --blocks task2 --blocked-by task3"
    Then the exit code should be 1
    And stderr should contain "only one of --blocks or --blocked-by can be specified"

  Scenario: Link non-existent source task fails
    When I run "backlog link nonexistent --blocks task2"
    Then the exit code should be 1
    And stderr should contain "not found"

  Scenario: Link non-existent target task fails
    When I run "backlog link task1 --blocks nonexistent"
    Then the exit code should be 1
    And stderr should contain "not found"

  Scenario: Show dependencies in JSON format
    When I run "backlog link task1 --blocks task2"
    And I run "backlog show task1 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task1"

  Scenario: Link in JSON format
    When I run "backlog link task1 --blocks task2 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
