package consumer

import (
	"log"
	"os"
	"testing"

	"github.com/fabric8-services/fabric8-wit/test/contracts"
	"github.com/pact-foundation/pact-go/dsl"
)

// TestWitAPIConsumer runs all user related tests
func TestWitAPIConsumer(t *testing.T) {

	log.SetOutput(os.Stdout)

	var pactDir = os.Getenv("PACT_DIR")
	var pactConsumer = os.Getenv("PACT_CONSUMER")
	var pactProvider = os.Getenv("PACT_PROVIDER")

	//var pactBrokerURL = os.Getenv("PACT_BROKER_URL")
	//var pactBrokerUsername = os.Getenv("PACT_BROKER_USERNAME")
	//var pactBrokerPassword = os.Getenv("PACT_BROKER_PASSWORD")

	//var userName = os.Getenv("OSIO_USERNAME")

	// Create Pact connecting to local Daemon
	pact := &dsl.Pact{
		Consumer:             pactConsumer,
		Provider:             pactProvider,
		PactDir:              pactDir,
		Host:                 "localhost",
		LogLevel:             "DEBUG",
		PactFileWriteMode:    "overwrite",
		SpecificationVersion: 2,
	}
	defer pact.Teardown()

	// Test interactions
	APIStatus(t, pact)
	APISpacesCreate(t, pact, contracts.TestSpaceName)

	// Write a pact file
	pactFile := contracts.PactFile()
	log.Printf("All tests done, writting a pact file to %s.\n", pactFile)
	pact.WritePact()

	/*log.Printf("Publishing pact to a broker %s\n", pactBrokerURL)

	p := dsl.Publisher{}
	err := p.Publish(types.PublishRequest{
		PactURLs:        []string{},
		PactBroker:      pactBrokerURL,
		BrokerUsername:  pactBrokerUsername,
		BrokerPassword:  pactBrokerPassword,
		ConsumerVersion: pactVersion,
		Tags:            []string{"latest"},
	})

	if err != nil {
		log.Fatalf("Unable to publish pact to a broker %s:\n%q\n", pactBrokerURL, err)
	}*/
}
