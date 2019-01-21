package fabric8auth

import (
	"fmt"
	"net/http"

	"github.com/pact-foundation/pact-go/dsl"
)

// SimpleGetInteraction executes a simple GET request against Pact Mock server
// to verify the interaction
func SimpleGetInteraction(pact *dsl.Pact, endpoint string) func() error {
	var test = func() error {
		u := fmt.Sprintf("http://localhost:%d%s", pact.Server.Port, endpoint)
		req, err := http.NewRequest("GET", u, nil)

		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		_, err = http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		return err
	}
	return test
}

// SimpleGetInteractionWithToken executes a simple GET request (incl. the "Authorization: Bearer <token>" header)
// against Pact Mock server to verify the interaction
func SimpleGetInteractionWithToken(pact *dsl.Pact, endpoint string, authToken string) func() error {
	var test = func() error {
		u := fmt.Sprintf("http://localhost:%d%s", pact.Server.Port, endpoint)
		req, err := http.NewRequest("GET", u, nil)

		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

		_, err = http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		return err
	}
	return test
}
