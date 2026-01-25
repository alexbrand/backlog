Feature: Next Task
  As an agent using the backlog CLI
  I want to get the next recommended task to work on
  So that I can efficiently pick up work without manually searching

  Background:
    Given a backlog with the following tasks:
      | id    | title               | status      | priority | assignee | labels        | agent_id  |
      | task1 | Urgent task         | todo        | urgent   |          | backend       |           |
      | task2 | High priority task  | todo        | high     |          | frontend      |           |
      | task3 | Medium task         | todo        | medium   |          | backend       |           |
      | task4 | Low priority task   | todo        | low      |          | frontend      |           |
      | task5 | No priority task    | todo        | none     |          | backend,api   |           |
      | task6 | Claimed task        | in-progress | urgent   | alex     | backend       | claude-1  |
      | task7 | Done task           | done        | high     |          |               |           |

  Scenario: Next returns highest priority unclaimed task
    When I run "backlog next"
    Then the exit code should be 0
    And stdout should contain "task1"
    And stdout should not contain "task6"
    And stdout should not contain "task7"

  Scenario: Next with label filter
    When I run "backlog next --label=frontend"
    Then the exit code should be 0
    And stdout should contain "task2"
    And stdout should not contain "task1"
    And stdout should not contain "task3"

  Scenario: Next with --claim atomically claims
    Given the environment variable "BACKLOG_AGENT_ID" is "test-agent"
    When I run "backlog next --claim"
    Then the exit code should be 0
    And stdout should contain "task1"
    And the task "task1" should have status "in-progress"
    And the task "task1" should have label "agent:test-agent"
    And a lock file should exist for task "task1"

  Scenario: Next when no tasks available returns empty
    Given a backlog with the following tasks:
      | id    | title           | status | priority | assignee | labels   | agent_id  |
      | taskA | Claimed task    | todo   | high     | alex     | backend  | claude-1  |
      | taskB | Done task       | done   | high     |          |          |           |
    When I run "backlog next"
    Then the exit code should be 0
    And stdout should be empty

  Scenario: Next skips tasks claimed by other agents
    When I run "backlog next"
    Then the exit code should be 0
    And stdout should contain "task1"
    And stdout should not contain "task6"

  Scenario: Next respects priority ordering
    When I run "backlog next -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task1"
    And the JSON output should have "priority" equal to "urgent"

  Scenario: Next in id-only format
    When I run "backlog next -f id-only"
    Then the exit code should be 0
    And stdout should contain "task1"
    And stdout should not contain "Urgent task"
    And stdout should not contain "urgent"

  Scenario: Next in JSON format
    When I run "backlog next -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task1"
    And the JSON output should have "title" equal to "Urgent task"
    And the JSON output should have "priority" equal to "urgent"

  Scenario: Next with multiple label filters
    When I run "backlog next --label=api"
    Then the exit code should be 0
    And stdout should contain "task5"
    And stdout should not contain "task1"
    And stdout should not contain "task2"

  Scenario: Next for non-existent label returns empty
    When I run "backlog next --label=nonexistent"
    Then the exit code should be 0
    And stdout should be empty

  Scenario: Next returns plain format by default
    When I run "backlog next"
    Then the exit code should be 0
    And stdout should contain "task1"

  Scenario: Next with claim when no tasks available
    Given a backlog with the following tasks:
      | id    | title        | status | priority | assignee | labels   | agent_id |
      | taskA | Done task    | done   | high     |          |          |          |
    When I run "backlog next --claim"
    Then the exit code should be 0
    And stdout should be empty

  Scenario: Next returns oldest task when priorities are equal
    Given a backlog with the following tasks:
      | id    | title        | status | priority | assignee | labels   | agent_id |
      | taskA | First task   | todo   | none     |          |          |          |
      | taskB | Second task  | todo   | none     |          |          |          |
      | taskC | Third task   | todo   | none     |          |          |          |
    When I run "backlog next -f json"
    Then the exit code should be 0
    And the JSON output should have "id" equal to "taskA"
    And the JSON output should have "title" equal to "First task"
