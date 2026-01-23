Feature: Linear Claims
  As an agent using the backlog CLI with Linear backend
  I want to claim and release tasks stored as Linear issues
  So that multiple agents don't work on the same issue simultaneously

  # Note: These scenarios test the Linear backend's claim/release operations.
  # All scenarios require a mock Linear API server for testing without real credentials.

  Background:
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: linear
        agent_id: test-agent
      workspaces:
        linear:
          backend: linear
          team: ENG
          api_key_env: LINEAR_API_KEY
          agent_label_prefix: agent
          default: true
      """
    And the environment variable "LINEAR_API_KEY" is "lin_api_valid_test_key"
    And a mock Linear API server is running

  @linear
  Scenario: Claim adds agent label to issue
    Given the mock Linear API has the following issues:
      | identifier | title          | state | labels | assignee | team |
      | ENG-50     | Unclaimed task | Todo  |        |          | ENG  |
    And the environment variable "BACKLOG_AGENT_ID" is "claude-1"
    When I run "backlog claim ENG-50 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "ENG-50"
    And the JSON output should have array "labels" containing "agent:claude-1"

  @linear
  Scenario: Claim assigns issue to authenticated user
    Given the mock Linear API has the following issues:
      | identifier | title          | state | labels | assignee | team |
      | ENG-51     | Unclaimed task | Todo  |        |          | ENG  |
    And the mock Linear API authenticated user is "api-user"
    When I run "backlog claim ENG-51 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "assignee" equal to "api-user"

  @linear
  Scenario: Claim moves issue to in-progress status
    Given the mock Linear API has the following issues:
      | identifier | title          | state | labels | assignee | team |
      | ENG-52     | Unclaimed task | Todo  |        |          | ENG  |
    When I run "backlog claim ENG-52 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "in-progress"

  @linear
  Scenario: Claim with explicit agent-id flag
    Given the mock Linear API has the following issues:
      | identifier | title          | state | labels | assignee | team |
      | ENG-53     | Unclaimed task | Todo  |        |          | ENG  |
    When I run "backlog claim ENG-53 --agent-id=custom-agent -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have array "labels" containing "agent:custom-agent"

  @linear
  Scenario: Claim already-claimed issue by same agent is no-op
    Given the mock Linear API has the following issues:
      | identifier | title        | state       | labels          | assignee | team |
      | ENG-54     | Already mine | In Progress | agent:my-agent  | api-user | ENG  |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    When I run "backlog claim ENG-54 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "ENG-54"
    And the JSON output should have "status" equal to "in-progress"

  @linear
  Scenario: Claim issue already claimed by different agent returns exit code 2
    Given the mock Linear API has the following issues:
      | identifier | title            | state       | labels            | assignee   | team |
      | ENG-55     | Claimed by other | In Progress | agent:other-agent | other-user | ENG  |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    When I run "backlog claim ENG-55"
    Then the exit code should be 2
    And stderr should contain "already claimed"

  @linear
  Scenario: Claim issue already claimed by different agent returns JSON error
    Given the mock Linear API has the following issues:
      | identifier | title            | state       | labels            | assignee   | team |
      | ENG-56     | Claimed by other | In Progress | agent:other-agent | other-user | ENG  |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    When I run "backlog claim ENG-56 -f json"
    Then the exit code should be 2
    And the JSON output should be valid
    And the JSON output should have "error" as an object
    And the JSON output should have "error.code" equal to "CONFLICT"
    And the JSON output should have "error.message" containing "already claimed"

  @linear
  Scenario: Claim non-existent issue returns exit code 3
    When I run "backlog claim ENG-9999"
    Then the exit code should be 3
    And stderr should contain "not found"

  @linear
  Scenario: Claim non-existent issue returns JSON error
    When I run "backlog claim ENG-9999 -f json"
    Then the exit code should be 3
    And the JSON output should be valid
    And the JSON output should have "error" as an object
    And the JSON output should have "error.code" equal to "NOT_FOUND"

  @linear
  Scenario: Release removes agent label
    Given the mock Linear API has the following issues:
      | identifier | title       | state       | labels          | assignee | team |
      | ENG-60     | My claimed  | In Progress | agent:my-agent  | api-user | ENG  |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    And the mock Linear API authenticated user is "api-user"
    When I run "backlog release ENG-60 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "ENG-60"
    And the JSON output should not have array "labels" containing "agent:my-agent"

  @linear
  Scenario: Release moves issue to todo status
    Given the mock Linear API has the following issues:
      | identifier | title       | state       | labels          | assignee | team |
      | ENG-61     | My claimed  | In Progress | agent:my-agent  | api-user | ENG  |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    And the mock Linear API authenticated user is "api-user"
    When I run "backlog release ENG-61 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "todo"

  @linear
  Scenario: Release unassigns user
    Given the mock Linear API has the following issues:
      | identifier | title       | state       | labels          | assignee | team |
      | ENG-62     | My claimed  | In Progress | agent:my-agent  | api-user | ENG  |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    And the mock Linear API authenticated user is "api-user"
    When I run "backlog release ENG-62 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "assignee" equal to ""

  @linear
  Scenario: Release with comment adds issue comment
    Given the mock Linear API has the following issues:
      | identifier | title       | state       | labels          | assignee | team |
      | ENG-63     | My claimed  | In Progress | agent:my-agent  | api-user | ENG  |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    And the mock Linear API authenticated user is "api-user"
    When I run "backlog release ENG-63 --comment='Blocked on external dependency'"
    Then the exit code should be 0
    And stdout should contain "Released"
    And stdout should contain "ENG-63"

  @linear
  Scenario: Release issue claimed by different agent returns exit code 2
    Given the mock Linear API has the following issues:
      | identifier | title           | state       | labels            | assignee   | team |
      | ENG-64     | Other's claimed | In Progress | agent:other-agent | other-user | ENG  |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    When I run "backlog release ENG-64"
    Then the exit code should be 2
    And stderr should contain "claimed by different agent"

  @linear
  Scenario: Release unclaimed issue fails
    Given the mock Linear API has the following issues:
      | identifier | title          | state | labels | assignee | team |
      | ENG-65     | Unclaimed task | Todo  |        |          | ENG  |
    When I run "backlog release ENG-65"
    Then the exit code should be 2
    And stderr should contain "not claimed"

  @linear
  Scenario: Release non-existent issue returns exit code 3
    When I run "backlog release ENG-9999"
    Then the exit code should be 3
    And stderr should contain "not found"

  @linear
  Scenario: Claim uses workspace agent_id from config
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: linear
      workspaces:
        linear:
          backend: linear
          team: ENG
          api_key_env: LINEAR_API_KEY
          agent_id: workspace-agent
          agent_label_prefix: agent
          default: true
      """
    And the environment variable "LINEAR_API_KEY" is "lin_api_valid_test_key"
    And a mock Linear API server is running
    And the mock Linear API has the following issues:
      | identifier | title          | state | labels | assignee | team |
      | ENG-70     | Unclaimed task | Todo  |        |          | ENG  |
    When I run "backlog claim ENG-70 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have array "labels" containing "agent:workspace-agent"

  @linear
  Scenario: Claim uses BACKLOG_AGENT_ID environment variable
    Given the mock Linear API has the following issues:
      | identifier | title          | state | labels | assignee | team |
      | ENG-71     | Unclaimed task | Todo  |        |          | ENG  |
    And the environment variable "BACKLOG_AGENT_ID" is "env-agent"
    When I run "backlog claim ENG-71 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have array "labels" containing "agent:env-agent"

  @linear
  Scenario: CLI flag agent-id takes precedence over environment variable
    Given the mock Linear API has the following issues:
      | identifier | title          | state | labels | assignee | team |
      | ENG-72     | Unclaimed task | Todo  |        |          | ENG  |
    And the environment variable "BACKLOG_AGENT_ID" is "env-agent"
    When I run "backlog claim ENG-72 --agent-id=flag-agent -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have array "labels" containing "agent:flag-agent"
    And the JSON output should not have array "labels" containing "agent:env-agent"
