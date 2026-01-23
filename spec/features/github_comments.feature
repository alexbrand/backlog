Feature: GitHub Comments
  As a user of the backlog CLI with GitHub backend
  I want to add and view comments on GitHub Issues
  So that I can track progress and communicate about work items

  # Note: These scenarios test the GitHub backend's comment operations.
  # All scenarios require a mock GitHub API server for testing without real credentials.

  Background:
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          api_key_env: GITHUB_TOKEN
          default: true
      """
    And the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running

  @github
  Scenario: Comment adds issue comment via API
    Given the mock GitHub API has the following issues:
      | number | title           | state | labels | assignee | body               |
      | 100    | Task to comment | open  | ready  | alice    | A task for testing |
    When I run "backlog comment GH-100 'Starting work on this feature'"
    Then the exit code should be 0
    And stdout should contain "Comment added"
    And stdout should contain "GH-100"

  @github
  Scenario: Comment adds issue comment with JSON output
    Given the mock GitHub API has the following issues:
      | number | title           | state | labels | assignee | body               |
      | 101    | Task to comment | open  | ready  | alice    | A task for testing |
    When I run "backlog comment GH-101 'Progress update: 50% complete' -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "GH-101"

  @github
  Scenario: Comment on non-existent issue returns exit code 3
    When I run "backlog comment GH-9999 'This should fail'"
    Then the exit code should be 3
    And stderr should contain "not found"

  @github
  Scenario: Comment on non-existent issue returns JSON error
    When I run "backlog comment GH-9999 'This should fail' -f json"
    Then the exit code should be 3
    And the JSON output should be valid
    And the JSON output should have "error" as an object
    And the JSON output should have "error.code" equal to "NOT_FOUND"

  @github
  Scenario: Comment with empty message fails
    Given the mock GitHub API has the following issues:
      | number | title           | state | labels | assignee | body               |
      | 102    | Task to comment | open  | ready  |          | A task for testing |
    When I run "backlog comment GH-102 ''"
    Then the exit code should be 1
    And stderr should contain "message"

  @github
  Scenario: Comment requires task ID argument
    When I run "backlog comment"
    Then the exit code should be 1
    And stderr should contain "requires"

  @github
  Scenario: Show with --comments fetches comment thread
    Given the mock GitHub API has the following issues:
      | number | title           | state | labels | assignee | body               |
      | 110    | Task with notes | open  | ready  | alice    | A task with history |
    And the mock GitHub issue "110" has the following comments:
      | author | body                                |
      | alice  | Started research on this feature    |
      | bob    | Found some relevant documentation   |
      | alice  | Updated the implementation approach |
    When I run "backlog show GH-110 --comments"
    Then the exit code should be 0
    And stdout should contain "Started research on this feature"
    And stdout should contain "Found some relevant documentation"
    And stdout should contain "Updated the implementation approach"
    And stdout should contain "alice"
    And stdout should contain "bob"

  @github
  Scenario: Show with --comments in JSON format includes comments array
    Given the mock GitHub API has the following issues:
      | number | title           | state | labels | assignee | body               |
      | 111    | Task with notes | open  | ready  | alice    | A task with history |
    And the mock GitHub issue "111" has the following comments:
      | author | body                          |
      | alice  | First comment on the issue    |
      | bob    | Second comment with feedback  |
    When I run "backlog show GH-111 --comments -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "GH-111"
    And the JSON output should have "comments" as an array
    And the JSON output array "comments" should have length 2

  @github
  Scenario: Show without --comments does not include comment thread
    Given the mock GitHub API has the following issues:
      | number | title           | state | labels | assignee | body               |
      | 112    | Task with notes | open  | ready  | alice    | A task with history |
    And the mock GitHub issue "112" has the following comments:
      | author | body                    |
      | alice  | This is a test comment  |
    When I run "backlog show GH-112"
    Then the exit code should be 0
    And stdout should not contain "This is a test comment"

  @github
  Scenario: Show with --comments on issue with no comments shows empty
    Given the mock GitHub API has the following issues:
      | number | title             | state | labels | assignee | body                   |
      | 113    | Task without notes| open  | ready  |          | A task with no comments |
    When I run "backlog show GH-113 --comments -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "comments" as an array
    And the JSON output array "comments" should have length 0

  @github
  Scenario: Comment with body-file flag
    Given the mock GitHub API has the following issues:
      | number | title           | state | labels | assignee | body               |
      | 114    | Task to comment | open  | ready  |          | A task for testing |
    And a file "analysis.md" with content "## Detailed Analysis\n\nThis is a multi-line comment with:\n- Bullet points\n- Technical details"
    When I run "backlog comment GH-114 --body-file=analysis.md"
    Then the exit code should be 0
    And stdout should contain "Comment added"
    And stdout should contain "GH-114"
