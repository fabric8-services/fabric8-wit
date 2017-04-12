@story-296
Feature: Project backlog management

  Scenario: Add work items to backlog

    Given an existing space,
    And a user with permissions to add items to backlog,
    When the user adds an item to the backlog with title and description,
    Then a new work item with a space-unique ID should be created in the backlog
    And the creator of the work item must be the said user.