package provider

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/test/contracts"
	"github.com/pact-foundation/pact-go/dsl"
	"github.com/pact-foundation/pact-go/types"
)

// TestWitAPIProvider verifies the provider
func TestWitAPIProvider(t *testing.T) {

	pactProviderBaseURL := os.Getenv("PACT_PROVIDER_BASE_URL")
	pactConsumer := contracts.PactConsumer()
	pactProvider := contracts.PactProvider()
	pactDir := contracts.PactDir()

	// Create Pact connecting to local Daemon
	pact := &dsl.Pact{
		Consumer:             pactConsumer,
		Provider:             pactProvider,
		PactDir:              pactDir,
		Host:                 "localhost",
		LogLevel:             "INFO",
		SpecificationVersion: 2,
	}
	defer pact.Teardown()

	var providerSetupHost = "localhost" // this should ultimately be part of the provider api (developer mode: on)
	var providerSetupPort = 8888

	// Create user to get userid
	var userName = os.Getenv("OSIO_USERNAME")
	var userPassword = os.Getenv("OSIO_PASSWORD")
	var spaceName = contracts.NewSpaceName()
	var pactProviderAuthBaseURL = os.Getenv("PACT_PROVIDER_AUTH_BASE_URL")

	var initialState = Setup(providerSetupHost, providerSetupPort, map[string]string{
		"pactProviderBaseURL":     pactProviderBaseURL,
		"pactProviderAuthBaseURL": pactProviderAuthBaseURL,
		"userName":                userName,
		"userPassword":            userPassword,
		"spaceName":               spaceName,
	})

	if initialState == nil {
		log.Fatalf("Error returning user")
	}

	pactFile := contracts.PactFile()
	pactContent := contracts.PactFromFile(pactFile)
	providerPactFilePath := fmt.Sprintf("%s/provider-%s-%s.json", pactDir, strings.ToLower(pactConsumer), strings.ToLower(pactProvider))

	//log.Printf("Pact taken from broker:\n%s\n", pactContent)
	pactContent = strings.Replace(pactContent, contracts.TestUserName, initialState.User.Data.Attributes.Username, -1)
	pactContent = strings.Replace(pactContent, contracts.TestUserID, initialState.User.Data.ID, -1)
	pactContent = strings.Replace(pactContent, contracts.TestJWSToken, initialState.UserToken, -1)
	pactContent = strings.Replace(pactContent, contracts.TestSpaceName, initialState.Space.Data.Attributes.Name, -1)
	pactContent = strings.Replace(pactContent, contracts.TestSpaceID, initialState.Space.Data.ID, -1)
	//log.Printf("Pact filtered:\n%s\n", pactContent)

	providerPactFile, err := os.Create(providerPactFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer providerPactFile.Close()

	_, err = providerPactFile.WriteString(pactContent)

	// Verify the Provider with local Pact Files
	pact.VerifyProvider(t, types.VerifyRequest{
		ProviderBaseURL:        pactProviderBaseURL,
		PactURLs:               []string{providerPactFilePath},
		ProviderStatesSetupURL: fmt.Sprintf("http://%s:%d/pact/setup", providerSetupHost, providerSetupPort),
	})

	log.Println("Test Passed!")
}
