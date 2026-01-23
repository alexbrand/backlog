Feature: Claiming Tasks
  As an agent using the backlog CLI
  I want to claim tasks for exclusive work
  So that multiple agents don't work on the same task simultaneously

  Background:
    Given a backlog with the following tasks:
      | id    | title               | status      | priority | assignee | labels   | agent_id  |
      | task1 | Unclaimed task      | todo        | high     |          | feature  |           |
      | task2 | Already claimed     | in-progress | medium   | alex     | bug      | claude-1  |
      | task3 | Another unclaimed   | todo        | low      |          | backend  |           |
      | task4 | Done task           | done        | none     |          |          |           |

  Scenario: Claim unclaimed task succeeds
    When I run "backlog claim task1"
    Then the exit code should be 0
    And stdout should contain "Claimed"
    And stdout should contain "task1"

  Scenario: Claim adds agent label to task
    Given the environment variable "BACKLOG_AGENT_ID" is "test-agent"
    When I run "backlog claim task1"
    Then the exit code should be 0
    And the task "task1" should have label "agent:test-agent"

  Scenario: Claim moves task to in-progress
    When I run "backlog claim task1"
    Then the exit code should be 0
    And the task "task1" should have status "in-progress"
    And the task "task1" should be in directory "in-progress"

  Scenario: Claim assigns to authenticated user
    When I run "backlog claim task1"
    Then the exit code should be 0
    And the task "task1" should be assigned

  Scenario: Claim already-claimed task by same agent is no-op
    Given the environment variable "BACKLOG_AGENT_ID" is "claude-1"
    When I run "backlog claim task2"
    Then the exit code should be 0
    And stdout should contain "task2"
    And the task "task2" should have status "in-progress"

  Scenario: Claim task claimed by different agent returns exit code 2
    Given the environment variable "BACKLOG_AGENT_ID" is "different-agent"
    When I run "backlog claim task2"
    Then the exit code should be 2
    And stderr should contain "already claimed"

  Scenario: Claim with explicit agent-id flag
    When I run "backlog claim task1 --agent-id=my-custom-agent"
    Then the exit code should be 0
    And the task "task1" should have label "agent:my-custom-agent"

  Scenario: Claim uses BACKLOG_AGENT_ID environment variable
    Given the environment variable "BACKLOG_AGENT_ID" is "env-agent"
    When I run "backlog claim task1"
    Then the exit code should be 0
    And the task "task1" should have label "agent:env-agent"

  Scenario: Claim uses workspace config agent_id
    Given a config file with the following content:
      """
      version: 1
      defaults:
        agent_id: config-agent
      """
    When I run "backlog claim task1"
    Then the exit code should be 0
    And the task "task1" should have label "agent:config-agent"

  Scenario: Claim uses global default agent_id
    Given a config file with the following content:
      """
      version: 1
      defaults:
        agent_id: global-default-agent
      """
    When I run "backlog claim task1"
    Then the exit code should be 0
    And the task "task1" should have label "agent:global-default-agent"

  Scenario: Claim falls back to hostname
    When I run "backlog claim task1"
    Then the exit code should be 0
    And the task "task1" should have agent label

  Scenario: Claim creates lock file in file mode
    When I run "backlog claim task1"
    Then the exit code should be 0
    And a lock file should exist for task "task1"

  Scenario: Claim non-existent task returns exit code 3
    When I run "backlog claim nonexistent-task"
    Then the exit code should be 3
    And stderr should contain "not found"

  Scenario: Claim in JSON format
    When I run "backlog claim task1 -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task1"
    And the JSON output should have "status" equal to "in-progress"
