package contracts

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	// nop
}

// APIStatusResponse represents a service status message returned by /api/status endpoint.
type APIStatusResponse struct {
	BuildTime string `json:"buildTime" pact:"example=2018-10-05T10:03:04Z"`
	Commit    string `json:"commit" pact:"example=0f9921980549b2baeb43f6f16cbe794f430f498c"`
	StartTime string `json:"startTime" pact:"example=2018-10-09T15:04:50Z"`
}

// UserData represents a JSON object containing user's info.
type UserData struct {
	Attributes struct {
		Bio                   string `json:"bio" pact:"example=n/a,regex=^[ a-zA-Z0-9,\\./]*$"`
		Cluster               string `json:"cluster" pact:"example=openshift.developer.osio/"`
		Company               string `json:"company" pact:"example=n/a,regex=^[ a-zA-Z0-9,\\./]*$"`
		CreatedAt             string `json:"created-at" pact:"example=2018-03-16T14:34:31.615511Z"`
		Email                 string `json:"email" pact:"example=osio-developer@email.com"`
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

// Users represents a JSON object of a collection of users.
type Users struct {
	Data []UserData `json:"data"`
}

// EmptyData represents an empty message returned by API
type EmptyData struct {
	Data []interface{} `json:"data"`
}

// Space represents JSON description of space
type Space struct {
	Data SpaceData `json:"data"`
}

// Spaces represents JSON object of space list
type Spaces struct {
	Data []SpaceData `json:"data"`
}

// SpaceAttributes represents JSON description of space attributes
type SpaceAttributes struct {
	CreatedAt   string `json:"created-at"`
	Description string `json:"description"`
	Name        string `json:"name"`
	UpdatedAt   string `json:"updated-at"`
	Version     int    `json:"version"`
}

// SpaceData represents JSON description of space data
type SpaceData struct {
	Attributes SpaceAttributes `json:"attributes"`
	ID         string          `json:"id"`
	Type       string          `json:"type"`
}

//CreateSpaceRequestAttributes represents attributes of a JSON request message to create a new space
type CreateSpaceRequestAttributes struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

//CreateSpaceRequestData represents payload of a JSON request message to create a new space
type CreateSpaceRequestData struct {
	Name         string                       `json:"name"`
	Path         string                       `json:"path"`
	Attributes   CreateSpaceRequestAttributes `json:"attributes"`
	Type         string                       `json:"type"`
	PrivateSpace bool                         `json:"privateSpace"`
}

//CreateSpaceRequest represents JSON request message to create a new space
type CreateSpaceRequest struct {
	Data CreateSpaceRequestData `json:"data"`
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

//TestSpaceName contains space name placeholder
const TestSpaceName = "testspace11111111"

//TestSpaceID contains user id placeholder
const TestSpaceID = "11111111-0000-4000-a000-000000000000"

type ProviderInitialState struct {
	User      User
	Space     Space
	UserToken string
}

//NewSpaceName returns a name of the new space
func NewSpaceName() string {
	return "test-space"
}

// PactDir returns a path to the directory to store pact files (taken from PACT_DIR env variable)
func PactDir() string {
	return os.Getenv("PACT_DIR")
}

// PactFile returns a path to the generated pact file
func PactFile() string {
	return fmt.Sprintf("%s/%s-%s.json", PactDir(), strings.ToLower(PactConsumer()), strings.ToLower(PactProvider()))
}

// PactConsumer returns a name of the pact consumer (taken from PACT_CONSUMER env variable)
func PactConsumer() string {
	return os.Getenv("PACT_CONSUMER")
}

// PactProvider returns a name of the pact provider (taken from PACT_PROVIDER env variable)
func PactProvider() string {
	return os.Getenv("PACT_PROVIDER")
}

// PactFromFile reads a pact from a given file and returns as string
func PactFromFile(pactFile string) string {
	f, err := ioutil.ReadFile(pactFile)
	if err != nil {
		log.Fatalf("Unable to read pact file: %s", pactFile)
	}
	return string(f)
}

// PactFromBroker reads a pact from a given pact broker and returns as string
func PactFromBroker(pactBrokerURL string, pactBrokerUsername string, pactBrokerPassword string) string {
	var httpClient = &http.Client{
		Timeout: time.Second * 30,
	}
	pactURL := fmt.Sprintf("%s/pacts/provider/%s/consumer/%s/latest", pactBrokerURL, PactProvider(), PactConsumer())
	request, err := http.NewRequest("GET", pactURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", pactBrokerUsername, pactBrokerPassword)))))

	log.Printf("Downloading a pact file from pact broker: %s", pactURL)
	response, err := httpClient.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)

	return string(responseBody)
}
