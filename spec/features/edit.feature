Feature: Editing Tasks
  As a user of the backlog CLI
  I want to edit existing tasks
  So that I can update task details as requirements change

  Background:
    Given a backlog with the following tasks:
      | id    | title              | status      | priority | assignee | labels       | description              |
      | task1 | Original title     | backlog     | medium   |          | feature      | Original description     |
      | task2 | Bug to fix         | todo        | high     | alex     | bug,critical | Fix the login bug        |
      | task3 | In progress work   | in-progress | low      |          | backend      | Working on API           |

  Scenario: Edit task title
    When I run "backlog edit task1 --title='Updated title'"
    Then the exit code should be 0
    And stdout should contain "task1"
    And the task "task1" should have title "Updated title"

  Scenario: Edit task priority
    When I run "backlog edit task1 --priority=urgent"
    Then the exit code should be 0
    And the task "task1" should have priority "urgent"

  Scenario: Edit task description
    When I run "backlog edit task1 --description='New description here'"
    Then the exit code should be 0
    And the task "task1" should have description containing "New description here"

  Scenario: Add label to task
    When I run "backlog edit task1 --add-label=backend"
    Then the exit code should be 0
    And the task "task1" should have label "feature"
    And the task "task1" should have label "backend"

  Scenario: Remove label from task
    When I run "backlog edit task2 --remove-label=critical"
    Then the exit code should be 0
    And the task "task2" should have label "bug"
    And the task "task2" should not have label "critical"

  Scenario: Edit multiple fields at once
    When I run "backlog edit task1 --title='Multi-edit task' --priority=high --add-label=urgent"
    Then the exit code should be 0
    And the task "task1" should have title "Multi-edit task"
    And the task "task1" should have priority "high"
    And the task "task1" should have label "urgent"

  Scenario: Edit non-existent task returns exit code 3
    When I run "backlog edit nonexistent-task --title='New title'"
    Then the exit code should be 3
    And stderr should contain "not found"

  Scenario: Edit preserves unmodified fields
    When I run "backlog edit task2 --title='New bug title'"
    Then the exit code should be 0
    And the task "task2" should have title "New bug title"
    And the task "task2" should have priority "high"
    And the task "task2" should have assignee "alex"
    And the task "task2" should have label "bug"
    And the task "task2" should have label "critical"
    And the task "task2" should have status "todo"

  Scenario: Edit task in JSON format
    When I run "backlog edit task1 --title='JSON edit' -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task1"
    And the JSON output should have "title" equal to "JSON edit"

  Scenario: Edit task with invalid priority fails
    When I run "backlog edit task1 --priority=invalid"
    Then the exit code should be 1
    And stderr should contain "invalid priority"
