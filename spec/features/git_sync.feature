Feature: Git Sync
  As an agent using the backlog CLI with git-based coordination
  I want mutations to automatically commit to git
  So that multiple agents on different machines can coordinate through a shared git repository

  Background:
    Given a git repository is initialized
    And a backlog with the following tasks:
      | id    | title               | status | priority | assignee | labels  | agent_id |
      | task1 | Unclaimed task      | todo   | high     |          | feature |          |
      | task2 | Another task        | todo   | medium   |          | bug     |          |
    And git_sync is enabled in the config

  Scenario: Mutation auto-commits when git_sync enabled
    When I run "backlog move task1 in-progress"
    Then the exit code should be 0
    And a git commit should exist with message containing "move: task1"

  Scenario: Add task creates git commit
    When I run "backlog add 'New feature task' --priority=high"
    Then the exit code should be 0
    And a git commit should exist with message containing "add:"

  Scenario: Edit task creates git commit
    When I run "backlog edit task1 --priority=urgent"
    Then the exit code should be 0
    And a git commit should exist with message containing "edit: task1"

  Scenario: Claim task creates git commit with agent info
    Given the environment variable "BACKLOG_AGENT_ID" is "test-agent"
    When I run "backlog claim task1"
    Then the exit code should be 0
    And a git commit should exist with message containing "claim: task1"
    And a git commit should exist with message containing "[agent:test-agent]"

  Scenario: Release task creates git commit
    Given task "task1" is claimed by agent "test-agent"
    And the environment variable "BACKLOG_AGENT_ID" is "test-agent"
    When I run "backlog release task1"
    Then the exit code should be 0
    And a git commit should exist with message containing "release: task1"

  Scenario: Comment creates git commit
    When I run "backlog comment task1 'Working on this'"
    Then the exit code should be 0
    And a git commit should exist with message containing "comment: task1"

  Scenario: Commit message format is correct
    When I run "backlog move task1 in-progress"
    Then the exit code should be 0
    And the last git commit message should match pattern "^(add|edit|move|claim|release|comment): .+"

  Scenario: Sync pulls and pushes
    Given a remote git repository
    When I run "backlog sync"
    Then the exit code should be 0
    And the local repository should be in sync with remote

  Scenario: Sync with --force overwrites local changes
    Given a remote git repository
    And the remote has different content than local
    When I run "backlog sync --force"
    Then the exit code should be 0
    And the local repository should match the remote

  Scenario: Failed push returns exit code 2
    Given a remote git repository
    And the remote has been updated by another agent
    When I run "backlog move task1 in-progress"
    Then the exit code should be 2
    And stderr should contain "conflict"

  Scenario: No commit when git_sync is disabled
    Given git_sync is disabled in the config
    When I run "backlog move task1 in-progress"
    Then the exit code should be 0
    And no new git commits should exist

  Scenario: Uncommitted changes prevent operations when git_sync enabled
    Given there are uncommitted changes in the repository
    When I run "backlog move task1 in-progress"
    Then the exit code should be 1
    And stderr should contain "uncommitted changes"
