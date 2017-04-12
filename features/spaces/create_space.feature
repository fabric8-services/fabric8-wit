@story-357 @spaces
Feature: Create spaces

  Scenario: Create a new space

    Given a user with permissions to create spaces,
    When the user creates a new space "Test space",
    Then a new space should be created.

  @undone
  Scenario: Create a duplicate space under the same user/org

    Given a user with permissions to create spaces,
     And a space "Test space" already exists with the same user as owner,
    When the user creates a new space "Test space",
    Then a new space should not be created.

  @undone
  Scenario: Create a duplicate space under a different user/org

    Given a user with permissions to create spaces,
    And a space "Test space" already exists with a different user as owner,
    When the user creates a new space "Test space",
    Then a new space should be created.