@story-409 @iterations
Feature: Create iterations in projects

  Scenario: Create iterations

    Given a user with permissions to create iterations in a space,
    And an existing space,
    When the user creates a new iteration,
    Then a new iteration should be created.