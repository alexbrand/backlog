Feature: Releasing Tasks
  As an agent using the backlog CLI
  I want to release claimed tasks
  So that other agents can work on them when I'm blocked

  Background:
    Given a backlog with the following tasks:
      | id    | title               | status      | priority | assignee | labels               | agent_id  |
      | task1 | Claimed by me       | in-progress | high     | user     | feature,agent:me     | me        |
      | task2 | Claimed by other    | in-progress | medium   | alex     | bug,agent:other      | other     |
      | task3 | Unclaimed task      | todo        | low      |          | backend              |           |
      | task4 | Done task           | done        | none     |          |                      |           |

  Scenario: Release claimed task succeeds
    Given the environment variable "BACKLOG_AGENT_ID" is "me"
    And task "task1" is claimed by agent "me"
    When I run "backlog release task1"
    Then the exit code should be 0
    And stdout should contain "Released"
    And stdout should contain "task1"

  Scenario: Release removes agent label
    Given the environment variable "BACKLOG_AGENT_ID" is "me"
    And task "task1" is claimed by agent "me"
    When I run "backlog release task1"
    Then the exit code should be 0
    And the task "task1" should not have label "agent:me"

  Scenario: Release moves task to todo
    Given the environment variable "BACKLOG_AGENT_ID" is "me"
    And task "task1" is claimed by agent "me"
    When I run "backlog release task1"
    Then the exit code should be 0
    And the task "task1" should have status "todo"
    And the task "task1" should be in directory "todo"

  Scenario: Release unassigns user
    Given the environment variable "BACKLOG_AGENT_ID" is "me"
    And task "task1" is claimed by agent "me"
    When I run "backlog release task1"
    Then the exit code should be 0
    And the task "task1" should have assignee ""

  Scenario: Release removes lock file
    Given the environment variable "BACKLOG_AGENT_ID" is "me"
    And task "task1" is claimed by agent "me"
    When I run "backlog release task1"
    Then the exit code should be 0
    And no lock file should exist for task "task1"

  Scenario: Release with comment flag
    Given the environment variable "BACKLOG_AGENT_ID" is "me"
    And task "task1" is claimed by agent "me"
    When I run "backlog release task1 --comment='Blocked on external API'"
    Then the exit code should be 0
    And the task "task1" should have comment containing "Blocked on external API"

  Scenario: Release task not claimed by this agent returns exit code 2
    Given the environment variable "BACKLOG_AGENT_ID" is "different-agent"
    And task "task2" is claimed by agent "other"
    When I run "backlog release task2"
    Then the exit code should be 2
    And stderr should contain "claimed by different agent"

  Scenario: Release unclaimed task fails
    Given the environment variable "BACKLOG_AGENT_ID" is "me"
    When I run "backlog release task3"
    Then the exit code should be 2
    And stderr should contain "not claimed"
