@story-348
Feature: Add comments on work items

  Scenario: Add comments to work items

    Given an existing space,
    And a user with permissions to comment on work items,
    And an existing work item exists in the space
    When the user adds a plain text comment to the existing work item,
    Then a new comment should be appended against the work item
    And the creator of the comment must be the said user.

  Scenario: Add comments to work items in closed state

    Given an existing space,
    And a user with permissions to comment on work items,
    And an existing work item exists in the space in a closed state
    When the user adds a plain text comment to the existing work item,
    Then a new comment should be appended against the work item
    And the creator of the comment must be the said user.