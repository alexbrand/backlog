Feature: Linear Status Mapping
  As a user of the backlog CLI
  I want Linear workflow states to be mapped to canonical statuses
  So that I can work with consistent status values across backends

  # Note: These scenarios test the Linear backend's status mapping functionality.
  # All scenarios require a mock Linear API server for testing without real credentials.
  #
  # Default Linear state mapping:
  #   backlog     -> Backlog state
  #   todo        -> Todo state
  #   in-progress -> In Progress state
  #   review      -> In Review state
  #   done        -> Done state

  @linear
  Scenario: Linear states map to canonical statuses - Backlog
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
          default: true
      """
    And the environment variable "LINEAR_API_KEY" is "lin_api_valid_test_key"
    And a mock Linear API server is running
    And the mock Linear API has the following issues:
      | identifier | title           | state   | priority | team |
      | ENG-1      | Backlog issue   | Backlog | medium   | ENG  |
    When I run "backlog show ENG-1 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "backlog"

  @linear
  Scenario: Linear states map to canonical statuses - Todo
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
          default: true
      """
    And the environment variable "LINEAR_API_KEY" is "lin_api_valid_test_key"
    And a mock Linear API server is running
    And the mock Linear API has the following issues:
      | identifier | title        | state | priority | team |
      | ENG-2      | Todo issue   | Todo  | medium   | ENG  |
    When I run "backlog show ENG-2 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "todo"

  @linear
  Scenario: Linear states map to canonical statuses - In Progress
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
          default: true
      """
    And the environment variable "LINEAR_API_KEY" is "lin_api_valid_test_key"
    And a mock Linear API server is running
    And the mock Linear API has the following issues:
      | identifier | title              | state       | priority | team |
      | ENG-3      | In progress issue  | In Progress | medium   | ENG  |
    When I run "backlog show ENG-3 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "in-progress"

  @linear
  Scenario: Linear states map to canonical statuses - In Review
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
          default: true
      """
    And the environment variable "LINEAR_API_KEY" is "lin_api_valid_test_key"
    And a mock Linear API server is running
    And the mock Linear API has the following issues:
      | identifier | title         | state     | priority | team |
      | ENG-4      | Review issue  | In Review | medium   | ENG  |
    When I run "backlog show ENG-4 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "status" equal to "review"

  @linear
  Scenario: Linear states map to canonical statuses - Done
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
          default: true
      """
    And the environment variable "LINEAR_API_KEY" is "lin_api_valid_test_key"
    And a mock Linear API server is running
    And the mock Linear API has the following issues:
      | identifier | title       | state | priority | team |
      | ENG-5      | Done issue  | Done  | medium   | ENG  |
    When I run "backlog list --status=all -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks[0].status" equal to "done"

  @linear
  Scenario: Custom state mapping in config
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
          default: true
          status_map:
            backlog:
              state: Triage
            todo:
              state: Ready
            in-progress:
              state: Started
            review:
              state: Review
            done:
              state: Completed
      """
    And the environment variable "LINEAR_API_KEY" is "lin_api_valid_test_key"
    And a mock Linear API server is running
    And the mock Linear API has the following issues:
      | identifier | title          | state     | priority | team |
      | ENG-10     | Triage issue   | Triage    | medium   | ENG  |
      | ENG-11     | Ready issue    | Ready     | medium   | ENG  |
      | ENG-12     | Started issue  | Started   | medium   | ENG  |
      | ENG-13     | Review issue   | Review    | medium   | ENG  |
      | ENG-14     | Complete issue | Completed | medium   | ENG  |
    When I run "backlog list --status=all -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks[0].status" equal to "backlog"
    And the JSON output should have "tasks[1].status" equal to "todo"
    And the JSON output should have "tasks[2].status" equal to "in-progress"
    And the JSON output should have "tasks[3].status" equal to "review"
    And the JSON output should have "tasks[4].status" equal to "done"
