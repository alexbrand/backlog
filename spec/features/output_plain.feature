Feature: Plain Output Format
  As a user of the backlog CLI
  I want plain text output without table formatting
  So that I can easily pipe output to other commands and scripts

  Background:
    Given a backlog with the following tasks:
      | id    | title           | status      | priority | assignee | labels        | description           |
      | task1 | First task      | todo        | high     | alice    | feature,api   | First task details    |
      | task2 | Second task     | in-progress | medium   |          | bug           | Second task details   |
      | task3 | Third task      | backlog     | low      | bob      |               | Third task details    |

  Scenario: Plain output shows one task per line
    When I run "backlog list -f plain"
    Then the exit code should be 0
    # Plain format should show task ID and title on each line
    And stdout should contain "task1"
    And stdout should contain "First task"
    And stdout should contain "task2"
    And stdout should contain "Second task"
    And stdout should contain "task3"
    And stdout should contain "Third task"
    # Should not have table headers
    And stdout should not contain "ID"
    And stdout should not contain "STATUS"
    And stdout should not contain "PRIORITY"
    And stdout should not contain "TITLE"
    And stdout should not contain "ASSIGNEE"

  Scenario: Plain output format for show command
    When I run "backlog show task1 -f plain"
    Then the exit code should be 0
    # Should show task details in a readable but non-table format
    And stdout should contain "task1"
    And stdout should contain "First task"
    And stdout should contain "todo"
    And stdout should contain "high"
    And stdout should contain "alice"
    And stdout should contain "feature"
    And stdout should contain "api"
    And stdout should contain "First task details"
    # Should not have table-style headers in all caps
    And stdout should not contain "ID"
    And stdout should not contain "STATUS"
    And stdout should not contain "PRIORITY"
    And stdout should not contain "TITLE"
    And stdout should not contain "ASSIGNEE"
