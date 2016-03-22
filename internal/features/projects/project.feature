@project
Feature: Project Support

  Scenario: User successfully creates Project
    Given I have user/pass "foo" / "bar"
    And they log into the website with user "foo" and password "bar"
    Then the user should be able to create new project
