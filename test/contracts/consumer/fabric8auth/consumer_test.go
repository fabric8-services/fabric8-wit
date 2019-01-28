package fabric8auth_test

import (
	"log"
	"os"
	"testing"

	"github.com/fabric8-services/fabric8-auth/test/contracts/model"
	"github.com/pact-foundation/pact-go/dsl"
	"github.com/stretchr/testify/require"
)

// TestFabric8AuthConsumer runs all consumer side contract tests
// for the fabric8-wit (consumer) to fabric8-auth (provider) contract.
func TestFabric8AuthConsumer() {
	log.SetOutput(os.Stdout)

	var pactDir = os.Getenv("PACT_DIR")

	var pactConsumer = "fabric8-wit"
	var pactProvider = "fabric8-auth"

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
	err := pact.WritePact()
	require.NoError(t, err)
}
