package contracts_test

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/logutils"
	"github.com/pact-foundation/pact-go/dsl"
	"github.com/pact-foundation/pact-go/types"
	"github.com/stretchr/testify/require"
)

// PactDir returns a path to the directory to store pact files (taken from PACT_DIR env variable)
func PactDir() string {
	return os.Getenv("PACT_DIR")
}

// PactFile returns a path to the generated pact file
func PactFile(pactConsumer string, pactProvider string) string {
	return fmt.Sprintf("%s/%s-%s.json", PactDir(), strings.ToLower(pactConsumer), strings.ToLower(pactProvider))
}

// CheckErrorAndCleanPact is a workaround for the https://github.com/pact-foundation/pact-go/issues/108 issue
// by manually clearing the interactions from pact and the mock service.
func CheckErrorAndCleanPact(t *testing.T, pact *dsl.Pact, err1 error) {
	if err1 != nil {
		pact.Interactions = make([]*dsl.Interaction, 0)
		mockServer := &dsl.MockService{
			BaseURL:  fmt.Sprintf("http://%s:%d", pact.Host, pact.Server.Port),
			Consumer: pact.Consumer,
			Provider: pact.Provider,
		}
		err2 := mockServer.DeleteInteractions()
		require.NoError(t, err2)
	}
	require.NoError(t, err1)
}

// PublishPactFileToBroker publishes given Pact files to a given Pact broker.
func PublishPactFileToBroker(pactFiles []string, pactBrokerURL string, pactBrokerUsername string, pactBrokerPassword string, pactVersion string, tags []string) {
	log.SetOutput(&logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel("INFO"),
		Writer:   os.Stderr,
	})
	log.Printf("Publishing pact files %s to a broker (%s)...\n", pactFiles, pactBrokerURL)
	p := dsl.Publisher{}
	err := p.Publish(types.PublishRequest{
		PactURLs:        pactFiles,
		PactBroker:      pactBrokerURL,
		BrokerUsername:  pactBrokerUsername,
		BrokerPassword:  pactBrokerPassword,
		ConsumerVersion: pactVersion,
		Tags:            tags,
	})

	if err != nil {
		log.Fatalf("Unable to publish pact to a broker (%s):\n%+v\n", pactBrokerURL, err)
	}
}
