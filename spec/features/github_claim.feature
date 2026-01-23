Feature: GitHub Claims
  As an agent using the backlog CLI with GitHub backend
  I want to claim and release tasks stored as GitHub Issues
  So that multiple agents don't work on the same issue simultaneously

  # Note: These scenarios test the GitHub backend's claim/release operations.
  # All scenarios require a mock GitHub API server for testing without real credentials.

  Background:
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
        agent_id: test-agent
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          api_key_env: GITHUB_TOKEN
          agent_label_prefix: agent
          default: true
      """
    And the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running

  @github
  Scenario: Claim adds agent label to issue
    Given the mock GitHub API has the following issues:
      | number | title          | state | labels | assignee | body              |
      | 50     | Unclaimed task | open  | ready  |          | Task to be claimed |
    And the environment variable "BACKLOG_AGENT_ID" is "claude-1"
    When I run "backlog claim GH-50 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "GH-50"
    And the JSON output should have array "labels" containing "agent:claude-1"

  @github
  Scenario: Claim assigns issue to authenticated user
    Given the mock GitHub API has the following issues:
      | number | title          | state | labels | assignee | body              |
      | 51     | Unclaimed task | open  | ready  |          | Task to be claimed |
    And the mock GitHub API authenticated user is "api-user"
    When I run "backlog claim GH-51 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "assignee" equal to "api-user"

  @github
  Scenario: Claim moves issue to in-progress status
    Given the mock GitHub API has the following issues:
      | number | title          | state | labels | assignee | body              |
      | 52     | Unclaimed task | open  | ready  |          | Task to be claimed |
    When I run "backlog claim GH-52 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "in-progress"
    And the JSON output should have array "labels" containing "in-progress"
    And the JSON output should not have array "labels" containing "ready"

  @github
  Scenario: Claim with explicit agent-id flag
    Given the mock GitHub API has the following issues:
      | number | title          | state | labels | assignee | body              |
      | 53     | Unclaimed task | open  | ready  |          | Task to be claimed |
    When I run "backlog claim GH-53 --agent-id=custom-agent -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have array "labels" containing "agent:custom-agent"

  @github
  Scenario: Claim already-claimed issue by same agent is no-op
    Given the mock GitHub API has the following issues:
      | number | title         | state | labels                    | assignee | body           |
      | 54     | Already mine  | open  | in-progress,agent:my-agent | api-user | Already claimed |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    When I run "backlog claim GH-54 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "GH-54"
    And the JSON output should have "status" equal to "in-progress"

  @github
  Scenario: Claim issue already claimed by different agent returns exit code 2
    Given the mock GitHub API has the following issues:
      | number | title            | state | labels                       | assignee   | body              |
      | 55     | Claimed by other | open  | in-progress,agent:other-agent | other-user | Already claimed   |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    When I run "backlog claim GH-55"
    Then the exit code should be 2
    And stderr should contain "already claimed"

  @github
  Scenario: Claim issue already claimed by different agent returns JSON error
    Given the mock GitHub API has the following issues:
      | number | title            | state | labels                       | assignee   | body              |
      | 56     | Claimed by other | open  | in-progress,agent:other-agent | other-user | Already claimed   |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    When I run "backlog claim GH-56 -f json"
    Then the exit code should be 2
    And the JSON output should be valid
    And the JSON output should have "error" as an object
    And the JSON output should have "error.code" equal to "CONFLICT"
    And the JSON output should have "error.message" containing "already claimed"

  @github
  Scenario: Claim non-existent issue returns exit code 3
    When I run "backlog claim GH-9999"
    Then the exit code should be 3
    And stderr should contain "not found"

  @github
  Scenario: Claim non-existent issue returns JSON error
    When I run "backlog claim GH-9999 -f json"
    Then the exit code should be 3
    And the JSON output should be valid
    And the JSON output should have "error" as an object
    And the JSON output should have "error.code" equal to "NOT_FOUND"

  @github
  Scenario: Release removes agent label and unassigns
    Given the mock GitHub API has the following issues:
      | number | title        | state | labels                      | assignee | body         |
      | 60     | My claimed   | open  | in-progress,agent:my-agent  | api-user | Release this |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    And the mock GitHub API authenticated user is "api-user"
    When I run "backlog release GH-60 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "GH-60"
    And the JSON output should not have array "labels" containing "agent:my-agent"
    And the JSON output should have "assignee" equal to ""

  @github
  Scenario: Release moves issue to todo status
    Given the mock GitHub API has the following issues:
      | number | title        | state | labels                      | assignee | body         |
      | 61     | My claimed   | open  | in-progress,agent:my-agent  | api-user | Release this |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    And the mock GitHub API authenticated user is "api-user"
    When I run "backlog release GH-61 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "todo"
    And the JSON output should have array "labels" containing "ready"
    And the JSON output should not have array "labels" containing "in-progress"

  @github
  Scenario: Release with comment adds issue comment
    Given the mock GitHub API has the following issues:
      | number | title        | state | labels                      | assignee | body         |
      | 62     | My claimed   | open  | in-progress,agent:my-agent  | api-user | Release this |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    And the mock GitHub API authenticated user is "api-user"
    When I run "backlog release GH-62 --comment='Blocked on external dependency'"
    Then the exit code should be 0
    And stdout should contain "Released"
    And stdout should contain "GH-62"

  @github
  Scenario: Release issue claimed by different agent returns exit code 2
    Given the mock GitHub API has the following issues:
      | number | title            | state | labels                       | assignee   | body              |
      | 63     | Other's claimed  | open  | in-progress,agent:other-agent | other-user | Not mine          |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    When I run "backlog release GH-63"
    Then the exit code should be 2
    And stderr should contain "claimed by different agent"

  @github
  Scenario: Release unclaimed issue fails
    Given the mock GitHub API has the following issues:
      | number | title          | state | labels | assignee | body          |
      | 64     | Unclaimed task | open  | ready  |          | Not claimed   |
    When I run "backlog release GH-64"
    Then the exit code should be 2
    And stderr should contain "not claimed"

  @github
  Scenario: Release non-existent issue returns exit code 3
    When I run "backlog release GH-9999"
    Then the exit code should be 3
    And stderr should contain "not found"

  @github
  Scenario: Claim uses workspace agent_id from config
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
          agent_id: workspace-agent
          agent_label_prefix: agent
          default: true
      """
    And the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running
    And the mock GitHub API has the following issues:
      | number | title          | state | labels | assignee | body              |
      | 70     | Unclaimed task | open  | ready  |          | Task to be claimed |
    When I run "backlog claim GH-70 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have array "labels" containing "agent:workspace-agent"

  @github
  Scenario: Claim uses BACKLOG_AGENT_ID environment variable
    Given the mock GitHub API has the following issues:
      | number | title          | state | labels | assignee | body              |
      | 71     | Unclaimed task | open  | ready  |          | Task to be claimed |
    And the environment variable "BACKLOG_AGENT_ID" is "env-agent"
    When I run "backlog claim GH-71 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have array "labels" containing "agent:env-agent"

  @github
  Scenario: CLI flag agent-id takes precedence over environment variable
    Given the mock GitHub API has the following issues:
      | number | title          | state | labels | assignee | body              |
      | 72     | Unclaimed task | open  | ready  |          | Task to be claimed |
    And the environment variable "BACKLOG_AGENT_ID" is "env-agent"
    When I run "backlog claim GH-72 --agent-id=flag-agent -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have array "labels" containing "agent:flag-agent"
    And the JSON output should not have array "labels" containing "agent:env-agent"
