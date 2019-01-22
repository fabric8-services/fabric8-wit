package fabric8auth

import (
	"log"
	"os"
	"testing"

	"github.com/fabric8-services/fabric8-auth/test/contracts/model"
	"github.com/fabric8-services/fabric8-wit/test/contracts"
	"github.com/pact-foundation/pact-go/dsl"
	"github.com/pact-foundation/pact-go/types"
)

// TestFabric8AuthConsumer runs all consumer side contract tests
// for the fabric8-wit (consumer) to fabric8-auth (provider) contract.
func TestFabric8AuthConsumer(t *testing.T) {
	log.SetOutput(os.Stdout)

	var pactDir = os.Getenv("PACT_DIR")
	var pactVersion = os.Getenv("PACT_VERSION")

	var pactConsumer = "fabric8-wit"
	var pactProvider = "fabric8-auth"

	var pactBrokerURL = os.Getenv("PACT_BROKER_URL")
	var pactBrokerUsername = os.Getenv("PACT_BROKER_USERNAME")
	var pactBrokerPassword = os.Getenv("PACT_BROKER_PASSWORD")

	// Create Pact connecting to local Daemon
	pact := &dsl.Pact{
		Consumer:             pactConsumer,
		Provider:             pactProvider,
		PactDir:              pactDir,
		Host:                 "localhost",
		LogLevel:             "INFO",
		PactFileWriteMode:    "overwrite",
		SpecificationVersion: 2,
	}
	defer pact.Teardown()

	// Test interactions
	t.Run("api status", func(t *testing.T) { AuthAPIStatus(t, pact) })

	t.Run("api user by name", func(t *testing.T) { AuthAPIUserByName(t, pact, model.TestUserName) })
	t.Run("api user by id", func(t *testing.T) { AuthAPIUserByID(t, pact, model.TestUserID) })
	t.Run("api user by valid token", func(t *testing.T) { AuthAPIUserByToken(t, pact, model.TestJWSToken) })
	t.Run("api user with no token", func(t *testing.T) { AuthAPIUserNoToken(t, pact) })
	t.Run("api user with invalid token", func(t *testing.T) { AuthAPIUserInvalidToken(t, pact, model.TestInvalidJWSToken) })

	t.Run("api token keys", func(t *testing.T) { AuthAPITokenKeys(t, pact) })

	// Write a pact file
	pactFile := contracts.PactFile(pactConsumer, pactProvider)
	log.Printf("All tests done, writing a pact file (%s).\n", pactFile)
	pact.WritePact()

	log.Printf("Publishing pact to a broker (%s)...\n", pactBrokerURL)

	p := dsl.Publisher{}
	err := p.Publish(types.PublishRequest{
		PactURLs:        []string{pactFile},
		PactBroker:      pactBrokerURL,
		BrokerUsername:  pactBrokerUsername,
		BrokerPassword:  pactBrokerPassword,
		ConsumerVersion: pactVersion,
		Tags:            []string{"latest"},
	})

	if err != nil {
		log.Fatalf("Unable to publish pact to a broker (%s):\n%+v\n", pactBrokerURL, err)
	}
}
