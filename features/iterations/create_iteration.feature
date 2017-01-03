@story-409 @iterations
Feature: Create iterations in projects

  Scenario: Create iterations

    Given a user with permissions to create iterations in a space,
    And an existing space,
    When the user creates a new iteration with start date "2017-01-01" and end date "2017-01-31"
    Then a new iteration should be created.

  Scenario: Create backdated iteration

    Given a user with permissions to create iterations in a space,
    And an existing space,
    When the user creates a new iteration with start date "2016-12-01" and end date "2016-12-31"
    Then a new iteration should be created.