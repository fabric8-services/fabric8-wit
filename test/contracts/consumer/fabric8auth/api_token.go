package fabric8auth

import (
	"fmt"
	"log"
	"net/http"
	"testing"

	"github.com/fabric8-services/fabric8-auth/test/contracts/model"
	"github.com/pact-foundation/pact-go/dsl"
)

// AuthAPITokenKeys defines contract of /api/status endpoint
func AuthAPITokenKeys(t *testing.T, pact *dsl.Pact) {

	log.Printf("Invoking AuthAPITokenKeys now\n")

	// Pass in test case
	var test = func() error {
		u := fmt.Sprintf("http://localhost:%d/api/token/keys", pact.Server.Port)
		req, err := http.NewRequest("GET", u, nil)

		req.Header.Set("Accept", "application/json")
		if err != nil {
			return err
		}

		_, err = http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		return err
	}

	// Set up our expected interactions.
	pact.
		AddInteraction().
		Given("Auth service is up and running.").
		UponReceiving("A request to get public keys").
		WithRequest(dsl.Request{
			Method:  "GET",
			Path:    dsl.String("/api/token/keys"),
			Headers: dsl.MapMatcher{"Accept": dsl.String("application/json")},
		}).
		WillRespondWith(dsl.Response{
			Status:  200,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/vnd.publickeys+json")},
			Body:    dsl.Match(model.TokenKeys{}),
		})

	// Verify
	if err := pact.Verify(test); err != nil {
		log.Fatalf("Error on Verify: %v", err)
	}
}
