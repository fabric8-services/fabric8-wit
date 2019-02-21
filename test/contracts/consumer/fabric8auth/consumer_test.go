package fabric8auth_test

import (
	"log"
	"os"
	"testing"

	contracts_test "github.com/fabric8-services/fabric8-wit/test/contracts"
	"github.com/pact-foundation/pact-go/dsl"
	"github.com/stretchr/testify/require"
)

// TestFabric8AuthConsumer runs all consumer side contract tests
// for the fabric8-wit (consumer) to fabric8-auth (provider) contract.
func TestFabric8AuthConsumer(t *testing.T) {
	log.SetOutput(os.Stdout)

	var pactDir = os.Getenv("PACT_DIR")

	var pactConsumer = "fabric8-wit"
	var pactProvider = "fabric8-auth"

	// Create Pact connecting to local Daemon
	pact := &dsl.Pact{
		Consumer:             pactConsumer,
		Provider:             pactProvider,
		PactDir:              pactDir,
		LogDir:               pactDir,
		Host:                 "localhost",
		LogLevel:             "INFO",
		PactFileWriteMode:    "overwrite",
		SpecificationVersion: 2,
	}
	defer pact.Teardown()

	// Test interactions
	t.Run("api status", func(t *testing.T) { AuthAPIStatus(t, pact) })

	t.Run("api user by name", func(t *testing.T) { AuthAPIUserByName(t, pact, TestUserName) })
	t.Run("api user by id", func(t *testing.T) { AuthAPIUserByID(t, pact, TestUserID) })
	t.Run("api user by valid token", func(t *testing.T) { AuthAPIUserByToken(t, pact, TestJWSToken) })
	t.Run("api user with no token", func(t *testing.T) { AuthAPIUserNoToken(t, pact) })
	t.Run("api user with invalid token", func(t *testing.T) { AuthAPIUserInvalidToken(t, pact, TestInvalidJWSToken) })

	t.Run("api token keys", func(t *testing.T) { AuthAPITokenKeys(t, pact) })

	// Write a pact file
	pactFile := contracts_test.PactFile(pactConsumer, pactProvider)
	log.Printf("All tests done, writing a pact file (%s).\n", pactFile)
	err := pact.WritePact()
	require.NoError(t, err)
}

// APIStatusMessage represents a service status message returned by /api/status endpoint.
type APIStatusMessage struct {
	BuildTime           string `json:"buildTime" pact:"example=2018-10-05T10:03:04Z"`
	Commit              string `json:"commit" pact:"example=0f9921980549b2baeb43f6f16cbe794f430f498c"`
	ConfigurationStatus string `json:"configurationStatus" pact:"example=OK"`
	DatabaseStatus      string `json:"databaseStatus" pact:"example=OK"`
	StartTime           string `json:"startTime" pact:"example=2018-10-09T15:04:50Z"`
}

// UserData represents a JSON object containing user's info.
type UserData struct {
	Attributes struct {
		Bio                   string `json:"bio" pact:"example=n/a,regex=^[ a-zA-Z0-9,\\./]*$"`
		Cluster               string `json:"cluster" pact:"example=openshift.developer.osio/"`
		Company               string `json:"company" pact:"example=n/a,regex=^[ a-zA-Z0-9,\\./]*$"`
		CreatedAt             string `json:"created-at" pact:"example=2018-03-16T14:34:31.615511Z"`
		Email                 string `json:"email" pact:"example=developer@email.com"`
		EmailPrivate          bool   `json:"emailPrivate" pact:"example=false,regex=^[(true)(false)]$"`
		EmailVerified         bool   `json:"emailVerified" pact:"example=true,regex=^[(true)(false)]$"`
		FeatureLevel          string `json:"featureLevel" pact:"example=internal"`
		FullName              string `json:"fullName" pact:"example=Osio Developer"`
		IdentityID            string `json:"identityID" pact:"example=00000000-0000-4000-a000-000000000000"`
		ImageURL              string `json:"imageURL" pact:"example=n/a"`
		ProviderType          string `json:"providerType" pact:"example=kc"`
		RegistrationCompleted bool   `json:"registrationCompleted" pact:"example=true,regex=^[(true)(false)]$"`
		UpdatedAt             string `json:"updated-at" pact:"example=2018-05-30T11:05:23.513612Z"`
		URL                   string `json:"url" pact:"example=n/a"`
		UserID                string `json:"userID" pact:"example=5f41b66e-6f84-42b3-ab5f-8d9ef21149b1"`
		Username              string `json:"username" pact:"example=developer"`
	} `json:"attributes"`
	ID    string `json:"id" pact:"example=00000000-0000-4000-a000-000000000000"`
	Links struct {
		Related string `json:"related" pact:"example=http://localhost:8089/api/users/00000000-0000-4000-a000-000000000000"`
		Self    string `json:"self" pact:"example=http://localhost:8089/api/users/00000000-0000-4000-a000-000000000000"`
	} `json:"links"`
	Type string `json:"type" pact:"example=identities"`
}

// User represents a JSON object of a single user.
type User struct {
	Data UserData `json:"data"`
}

type Users struct {
	Data []UserData `json:"data"`
}

// InvalidTokenMessage represents a message returned when the Authorization header is invalid in secured endpoint calls
type InvalidTokenMessage struct {
	Errors []struct {
		Code   string `json:"code" pact:"example=token_validation_failed"`
		Detail string `json:"detail" pact:"example=token is invalid"`
		ID     string `json:"id" pact:"example=76J0ww+6"`
		Status string `json:"status" pact:"example=401"`
		Title  string `json:"title" pact:"example=Unauthorized"`
	} `json:"errors"`
}

// MissingTokenMessage represents a message returned when the Authorization header is missing to secured endpoint calls
type MissingTokenMessage struct {
	Errors []struct {
		Code   string `json:"code" pact:"example=jwt_security_error"`
		Detail string `json:"detail" pact:"example=missing header \"Authorization\""`
		ID     string `json:"id" pact:"example=FRzHbogQ"`
		Status string `json:"status" pact:"example=401"`
		Title  string `json:"title" pact:"example=Unauthorized"`
	} `json:"errors"`
}

// TokenKeys represents JSON message returned by /api/token/keys endpoint
type TokenKeys struct {
	Keys []struct {
		Alg string `json:"alg" pact:"example=RS256"`
		E   string `json:"e" pact:"example=AQAB"`
		Kid string `json:"kid" pact:"example=abcdefghijklmnopqrstuvwxyz-0123456789_ABCDE,regex=^[a-zA-Z0-9_-]{43}$"`
		Kty string `json:"kty" pact:"example=RSA"`
		N   string `json:"n" pact:"example=abcdefghijklmnopqrstuvwxyz-0123456789_ABCDE,regex=^[a-zA-Z0-9_-]+"`
		Use string `json:"use" pact:"example=sig"`
	} `json:"keys"`
}

// JWSRegex is a regular expression for matching JWS tokens
const JWSRegex = "[a-zA-Z0-9\\-_]+?\\.?[a-zA-Z0-9\\-_]+?\\.?([a-zA-Z0-9\\-_]+)?"

// TestInvalidJWSToken   Base64 encoded '{"alg":"RS256","kid":"1111111111111111111111111111111111111111111","typ":"JWT"}somerandombytes'
const TestInvalidJWSToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTEiLCJ0eXAiOiJKV1QifXNvbWVyYW5kb21ieXRlcw"

// TestJWSToken contains Base64 encoded '{"alg":"RS256","kid":"0000000000000000000000000000000000000000000","typ":"JWT"}somerandombytes'
const TestJWSToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAiLCJ0eXAiOiJKV1QifXNvbWVyYW5kb21ieXRlcw"

// UserNameRegex is a regular expression for matching usernames.
const UserNameRegex = "[a-zA-Z\\-0-9]+"

// UserIDRegex is a regular expression for matching user IDs.
const UserIDRegex = "[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}"

//TestUserID contains user id placeholder
const TestUserID = "00000000-0000-4000-a000-000000000000"

//TestUserName contains username placeholder
const TestUserName = "testuser00000000"
