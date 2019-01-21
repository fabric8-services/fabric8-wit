package fabric8auth

import (
	"log"
	"testing"

	"github.com/fabric8-services/fabric8-auth/test/contracts/model"
	"github.com/pact-foundation/pact-go/dsl"
)

// AuthAPITokenKeys defines contract of /api/status endpoint
func AuthAPITokenKeys(t *testing.T, pact *dsl.Pact) {

	log.Printf("Invoking AuthAPITokenKeys now\n")

	// Set up our expected interactions.
	pact.
		AddInteraction().
		Given("Auth service is up and running.").
		UponReceiving("A request to get public keys").
		WithRequest(dsl.Request{
			Method: "GET",
			Path:   dsl.String("/api/token/keys"),
		}).
		WillRespondWith(dsl.Response{
			Status:  200,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/vnd.publickeys+json")},
			Body:    dsl.Match(model.TokenKeys{}),
		})

	// Verify
	if err := pact.Verify(SimpleGetInteraction(pact, "/api/token/keys")); err != nil {
		log.Fatalf("Error on Verify: %+v", err)
	}
}
