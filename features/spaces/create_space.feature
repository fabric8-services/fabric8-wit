@story-357 @spaces
Feature: Create spaces

  Scenario: Create a new space

    Given a user with permissions to create spaces,
    When the user creates a new space,
    Then a new space should be created.