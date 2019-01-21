package fabric8auth

import (
	"log"
	"testing"

	"github.com/fabric8-services/fabric8-auth/test/contracts/model"
	"github.com/pact-foundation/pact-go/dsl"
)

// AuthAPIStatus defines contract of /api/status endpoint
func AuthAPIStatus(t *testing.T, pact *dsl.Pact) {

	log.Printf("Invoking AuthAPIStatus now\n")

	// Set up our expected interactions.
	pact.
		AddInteraction().
		Given("Auth service is up and running.").
		UponReceiving("A request to get status").
		WithRequest(dsl.Request{
			Method:  "GET",
			Path:    dsl.String("/api/status"),
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
		}).
		WillRespondWith(dsl.Response{
			Status:  200,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/vnd.status+json")},
			Body:    dsl.Match(model.APIStatusMessage{}),
		})

	// Verify

	if err := pact.Verify(SimpleGetInteraction(pact, "/api/status")); err != nil {
		log.Fatalf("Error on Verify: %+v", err)
	}
}
