@login
Feature: Login Support

  Scenario: User successfully logs in
    Given I have user/pass "foo" / "bar"
    And they log into the website with user "foo" and password "bar"
    Then the user should be successfully logged in
