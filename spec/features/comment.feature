Feature: Adding Comments
  As a user of the backlog CLI
  I want to add comments to tasks
  So that I can track progress and communicate about work items

  Background:
    Given a backlog with the following tasks:
      | id    | title               | status      | priority | assignee | labels   | description              |
      | task1 | Implement auth      | in-progress | high     | alex     | feature  | OAuth2 implementation    |
      | task2 | Fix login bug       | todo        | urgent   | bob      | bug      | Login fails on mobile    |
      | task3 | Write documentation | backlog     | low      |          | docs     | Update API documentation |

  Scenario: Add comment to task
    When I run "backlog comment task1 'Starting work on OAuth integration'"
    Then the exit code should be 0
    And stdout should contain "task1"
    And stdout should contain "Comment added"
    And the task "task1" should have comment containing "Starting work on OAuth integration"

  Scenario: Add comment with body-file
    Given a file "comment.md" with content "Detailed analysis of the bug.\n\nSteps to reproduce:\n1. Open app\n2. Try to login"
    When I run "backlog comment task2 --body-file=comment.md"
    Then the exit code should be 0
    And the task "task2" should have comment containing "Detailed analysis of the bug"
    And the task "task2" should have comment containing "Steps to reproduce"

  Scenario: Comment appears in task file
    When I run "backlog comment task3 'Initial thoughts on documentation structure'"
    Then the exit code should be 0
    And the task "task3" should have comment containing "Initial thoughts on documentation structure"

  Scenario: Comment on non-existent task returns exit code 3
    When I run "backlog comment nonexistent-task 'This should fail'"
    Then the exit code should be 3
    And stderr should contain "not found"

  Scenario: Add comment in JSON format
    When I run "backlog comment task1 'JSON comment test' -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task1"

  Scenario: Add multiple comments to same task
    When I run "backlog comment task1 'First comment'"
    And I run "backlog comment task1 'Second comment'"
    Then the exit code should be 0
    And the task "task1" should have comment containing "First comment"
    And the task "task1" should have comment containing "Second comment"

  Scenario: Comment with empty message fails
    When I run "backlog comment task1 ''"
    Then the exit code should be 1
    And stderr should contain "message"

  Scenario: Comment requires task ID argument
    When I run "backlog comment"
    Then the exit code should be 1
    And stderr should contain "requires"
