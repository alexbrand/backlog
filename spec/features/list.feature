Feature: Listing Tasks
  As a user of the backlog CLI
  I want to list tasks from my backlog
  So that I can see what work needs to be done

  Scenario: List all tasks (excludes done by default)
    Given a backlog with the following tasks:
      | id    | title           | status      | priority |
      | task1 | First task      | todo        | high     |
      | task2 | Second task     | in-progress | medium   |
      | task3 | Completed task  | done        | low      |
      | task4 | Backlog task    | backlog     | none     |
    When I run "backlog list"
    Then the exit code should be 0
    And stdout should contain "First task"
    And stdout should contain "Second task"
    And stdout should contain "Backlog task"
    And stdout should not contain "Completed task"
