Feature: Deleting Tasks
  As a user of the backlog CLI
  I want to delete tasks
  So that I can remove completed or obsolete tasks from the backlog

  Background:
    Given a backlog with the following tasks:
      | id    | title              | status      | priority | description          |
      | task1 | Task to delete     | backlog     | medium   | This will be deleted |
      | task2 | Another task       | todo        | high     | Keep this one        |
      | task3 | In progress work   | in-progress | low      | Working on this      |

  Scenario: Delete a task
    When I run "backlog delete task1"
    Then the exit code should be 0
    And stdout should contain "task1"
    And stdout should contain "Deleted"

  Scenario: Deleted task no longer appears in list
    When I run "backlog delete task1"
    Then the exit code should be 0
    When I run "backlog list"
    Then stdout should not contain "Task to delete"
    And stdout should contain "Another task"

  Scenario: Show deleted task returns not found
    When I run "backlog delete task1"
    Then the exit code should be 0
    When I run "backlog show task1"
    Then the exit code should be 3
    And stderr should contain "not found"

  Scenario: Delete non-existent task returns exit code 3
    When I run "backlog delete nonexistent-task"
    Then the exit code should be 3
    And stderr should contain "not found"

  Scenario: Delete task in JSON format
    When I run "backlog delete task1 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task1"
    And the JSON output should have "deleted" equal to "true"

  Scenario: Delete task in plain format
    When I run "backlog delete task1 -f plain"
    Then the exit code should be 0
    And stdout should contain "task1"

  Scenario: Delete task in id-only format
    When I run "backlog delete task1 -f id-only"
    Then the exit code should be 0
    And stdout should contain "task1"

  Scenario: Delete task from todo status
    When I run "backlog delete task2"
    Then the exit code should be 0
    When I run "backlog show task2"
    Then the exit code should be 3

  Scenario: Delete task from in-progress status
    When I run "backlog delete task3"
    Then the exit code should be 0
    When I run "backlog show task3"
    Then the exit code should be 3

  Scenario: Remaining tasks are still accessible after delete
    When I run "backlog delete task1"
    Then the exit code should be 0
    When I run "backlog show task2"
    Then the exit code should be 0
    And stdout should contain "Another task"
