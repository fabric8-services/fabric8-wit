package consumer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"testing"

	"github.com/fabric8-services/fabric8-wit/test/contracts"
	"github.com/pact-foundation/pact-go/dsl"
)

// APISpacesCreate defines contract of /api/spaces endpoint to create a new space
func APISpacesCreate(t *testing.T, pact *dsl.Pact, spaceName string) {

	log.Printf("Invoking APISpaces now\n")

	// Pass in test case
	var test = func() error {
		u := fmt.Sprintf("http://localhost:%d/api/spaces", pact.Server.Port)

		reqBody, err := json.Marshal(contracts.CreateSpaceRequest{
			Data: contracts.CreateSpaceRequestData{
				Name: spaceName,
				Path: "",
				Attributes: contracts.CreateSpaceRequestAttributes{
					Name:        spaceName,
					Description: "Space created by the contract tests to test new space creation.",
				},
				Type:         "spaces",
				PrivateSpace: false,
			},
		})

		if err != nil {
			return err
		}
		req, err := http.NewRequest("POST", u, bytes.NewBuffer(reqBody))

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", contracts.TestJWSToken))
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
		Given("WIT service is up and running.").
		UponReceiving("A request to create a new space with a given name").
		WithRequest(dsl.Request{
			Method: "POST",
			Path:   dsl.String("/api/spaces"),
			Headers: dsl.MapMatcher{
				"Content-Type": dsl.String("application/json"),
				"Authorization": dsl.Term(
					fmt.Sprintf("Bearer %s", contracts.TestJWSToken),
					fmt.Sprintf("^Bearer %s$", contracts.JWSRegex),
				),
			},
			Body: dsl.Match(contracts.CreateSpaceRequest{}),
		}).
		WillRespondWith(dsl.Response{
			Status:  201,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/vnd.api+json")},
			Body:    dsl.Match(contracts.Space{}),
		})

	// Verify
	if err := pact.Verify(test); err != nil {
		log.Fatalf("Error on Verify: %v", err)
	}
}
