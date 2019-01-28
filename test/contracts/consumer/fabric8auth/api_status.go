package fabric8auth_test

import (
	"log"
	"testing"

	"github.com/fabric8-services/fabric8-auth/test/contracts/model"
	"github.com/fabric8-services/fabric8-wit/test/contracts"
	"github.com/fabric8-services/fabric8-wit/test/contracts/consumer"
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
	err := pact.Verify(consumer_test.SimpleGetInteraction(pact, "/api/status"))
	contracts_test.CheckErrorAndCleanPact(t, pact, err) //workaround for https://github.com/pact-foundation/pact-go/issues/108
}
