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

// TestWitAPIConsumer runs all user related tests
func TestWitAPIConsumer(t *testing.T) {
	log.SetOutput(os.Stdout)

	var pactDir = os.Getenv("PACT_DIR")
	var pactVersion = os.Getenv("PACT_VERSION")

	var pactConsumer = "Fabric8Wit"
	var pactProvider = "Fabric8Auth"

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
	AuthAPIStatus(t, pact)

	AuthAPIUserByName(t, pact, model.TestUserName)
	AuthAPIUserByID(t, pact, model.TestUserID)
	AuthAPIUserByToken(t, pact, model.TestJWSToken)

	AuthAPITokenKeys(t, pact)

	// Write a pact file
	pactFile := contracts.PactFile(pactConsumer, pactProvider)
	log.Printf("All tests done, writting a pact file (%s).\n", pactFile)
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
		log.Fatalf("Unable to publish pact to a broker (%s):\n%q\n", pactBrokerURL, err)
	}
}
