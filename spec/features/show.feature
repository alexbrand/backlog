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
