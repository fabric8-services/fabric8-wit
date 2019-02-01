package fabric8auth_test

import (
	"fmt"
	"log"
	"testing"

	contracts_test "github.com/fabric8-services/fabric8-wit/test/contracts"
	consumer "github.com/fabric8-services/fabric8-wit/test/contracts/consumer"
	"github.com/pact-foundation/pact-go/dsl"
)

// AuthAPIUserByName defines contract of /api/users?filter[username]=<user_name> endpoint
func AuthAPIUserByName(t *testing.T, pact *dsl.Pact, userName string) {

	log.Println("Invoking AuthAPIUserByName test interaction now")

	// Set up our expected interactions.
	pact.
		AddInteraction().
		Given("User with a given username exists.").
		UponReceiving("A request to get user's information by username").
		WithRequest(dsl.Request{
			Method: "GET",
			Path:   dsl.String("/api/users"),
			Query: dsl.MapMatcher{
				"filter[username]": dsl.Term(
					userName,
					UserNameRegex,
				),
			},
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
		}).
		WillRespondWith(dsl.Response{
			Status:  200,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/vnd.api+json")},
			Body:    dsl.Match(Users{}),
		})

	// Verify
	err := pact.Verify(consumer.SimpleGetInteraction(pact, fmt.Sprintf("/api/users?filter[username]=%s", userName)))
	contracts_test.CheckErrorAndCleanPact(t, pact, err)
}

// AuthAPIUserByID defines contract of /api/users/<user_id> endpoint
func AuthAPIUserByID(t *testing.T, pact *dsl.Pact, userID string) {

	log.Printf("Invoking AuthAPIUserByID test interaction now\n")

	// Set up our expected interactions.
	pact.
		AddInteraction().
		Given("User with a given ID exists.").
		UponReceiving("A request to get user's information by ID").
		WithRequest(dsl.Request{
			Method: "GET",
			Path: dsl.Term(
				fmt.Sprintf("/api/users/%s", userID),
				fmt.Sprintf("/api/users/%s", UserIDRegex),
			),
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
		}).
		WillRespondWith(dsl.Response{
			Status:  200,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/vnd.api+json")},
			Body:    dsl.Match(User{}),
		})

	// Verify
	err := pact.Verify(consumer.SimpleGetInteraction(pact, fmt.Sprintf("/api/users/%s", userID)))
	contracts_test.CheckErrorAndCleanPact(t, pact, err) //workaround for https://github.com/pact-foundation/pact-go/issues/108
}

// AuthAPIUserByToken defines contract of /api/user endpoint with valid auth token
// passed as 'Authorization: Bearer ...' header
func AuthAPIUserByToken(t *testing.T, pact *dsl.Pact, userToken string) {

	log.Printf("Invoking AuthAPIUserByToken test interaction now\n")

	// Pass in test case
	// Set up our expected interactions.
	pact.
		AddInteraction().
		Given("A user exists with the given valid token.").
		UponReceiving("A request to get user's information with valid auth token ").
		WithRequest(dsl.Request{
			Method: "GET",
			Path:   dsl.String("/api/user"),
			Headers: dsl.MapMatcher{
				"Content-Type": dsl.String("application/json"),
				"Authorization": dsl.Term(
					fmt.Sprintf("Bearer %s", userToken),
					fmt.Sprintf("^Bearer %s$", JWSRegex),
				),
			},
		}).
		WillRespondWith(dsl.Response{
			Status:  200,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/vnd.api+json")},
			Body:    dsl.Match(User{}),
		})

	// Verify
	err := pact.Verify(consumer.SimpleGetInteractionWithToken(pact, "/api/user", userToken))
	contracts_test.CheckErrorAndCleanPact(t, pact, err) //workaround for https://github.com/pact-foundation/pact-go/issues/108
}

// AuthAPIUserInvalidToken defines contract of /api/user endpoint with invalid auth token
func AuthAPIUserInvalidToken(t *testing.T, pact *dsl.Pact, invalidToken string) {

	log.Printf("Invoking AuthAPIUserInvalidToken test interaction now\n")

	// Set up our expected interactions.
	pact.
		AddInteraction().
		Given("No user exists with the given token valid.").
		UponReceiving("A request to get user's information with invalid auth token ").
		WithRequest(dsl.Request{
			Method: "GET",
			Path:   dsl.String("/api/user"),
			Headers: dsl.MapMatcher{
				"Content-Type": dsl.String("application/json"),
				"Authorization": dsl.Term(
					fmt.Sprintf("Bearer %s", invalidToken),
					fmt.Sprintf("^Bearer %s$", JWSRegex),
				),
			},
		}).
		WillRespondWith(dsl.Response{
			Status:  401,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/vnd.api+json")},
			Body:    dsl.Match(InvalidTokenMessage{}),
		})

	// Verify
	err := pact.Verify(consumer.SimpleGetInteractionWithToken(pact, "/api/user", invalidToken))
	contracts_test.CheckErrorAndCleanPact(t, pact, err) //workaround for https://github.com/pact-foundation/pact-go/issues/108
}

// AuthAPIUserNoToken defines contract of /api/user endpoint with missing auth token
func AuthAPIUserNoToken(t *testing.T, pact *dsl.Pact) {

	log.Printf("Invoking AuthAPIUserNoToken test interaction now\n")

	// Set up our expected interactions.
	pact.
		AddInteraction().
		Given("Any user exists but no auth token was provided.").
		UponReceiving("A request to get user's information with no auth token ").
		WithRequest(dsl.Request{
			Method: "GET",
			Path:   dsl.String("/api/user"),
			Headers: dsl.MapMatcher{
				"Content-Type": dsl.String("application/json"),
			},
		}).
		WillRespondWith(dsl.Response{
			Status:  401,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/vnd.api+json")},
			Body:    dsl.Match(MissingTokenMessage{}),
		})

	// Verify
	err := pact.Verify(consumer.SimpleGetInteraction(pact, "/api/user"))
	contracts_test.CheckErrorAndCleanPact(t, pact, err) //workaround for https://github.com/pact-foundation/pact-go/issues/108
}
