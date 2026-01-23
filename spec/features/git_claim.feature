Feature: Git-Based Claims
  As an agent using the backlog CLI with git-based locking
  I want claims to be coordinated through git commits and pushes
  So that distributed agents can safely coordinate without file locks

  Background:
    Given a git repository is initialized
    And a remote git repository
    And a backlog with the following tasks:
      | id    | title               | status | priority | assignee | labels  | agent_id |
      | task1 | Unclaimed task      | todo   | high     |          | feature |          |
      | task2 | Another task        | todo   | medium   |          | bug     |          |
    And lock_mode is "git" in the config
    And git_sync is enabled in the config

  Scenario: Claim with lock_mode git commits and pushes
    Given the environment variable "BACKLOG_AGENT_ID" is "git-agent"
    When I run "backlog claim task1"
    Then the exit code should be 0
    And a git commit should exist with message containing "claim: task1"
    And a git commit should exist with message containing "[agent:git-agent]"
    And the remote should have the latest commit
    And no lock file should exist for task "task1"

  Scenario: Claim with lock_mode git pulls before claiming
    Given the environment variable "BACKLOG_AGENT_ID" is "git-agent"
    And the remote has a new commit
    When I run "backlog claim task1"
    Then the exit code should be 0
    And the local repository should include the remote commit

  Scenario: Concurrent git claims - push conflict returns exit code 2
    Given the environment variable "BACKLOG_AGENT_ID" is "agent-a"
    And another agent has claimed task "task1" and pushed while we were working
    When I run "backlog claim task1"
    Then the exit code should be 2
    And stderr should contain "conflict"

  Scenario: Release with lock_mode git commits and pushes
    Given task "task1" is claimed by agent "git-agent"
    And the environment variable "BACKLOG_AGENT_ID" is "git-agent"
    When I run "backlog release task1"
    Then the exit code should be 0
    And a git commit should exist with message containing "release: task1"
    And the remote should have the latest commit

  Scenario: Git claim removes any stale file locks
    Given task "task1" has a stale lock file
    And the environment variable "BACKLOG_AGENT_ID" is "git-agent"
    When I run "backlog claim task1"
    Then the exit code should be 0
    And no lock file should exist for task "task1"
    And the task "task1" should have label "agent:git-agent"

  Scenario: Git claim fails gracefully when remote is unreachable
    Given the remote repository is unreachable
    And the environment variable "BACKLOG_AGENT_ID" is "git-agent"
    When I run "backlog claim task1"
    Then the exit code should be 1
    And stderr should contain "remote"

  Scenario: Git release fails gracefully when remote is unreachable
    Given task "task1" is claimed by agent "git-agent"
    And the environment variable "BACKLOG_AGENT_ID" is "git-agent"
    And the remote repository is unreachable
    When I run "backlog release task1"
    Then the exit code should be 1
    And stderr should contain "remote"
