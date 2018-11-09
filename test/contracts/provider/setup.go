package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fabric8-services/fabric8-wit/test/contracts"
	"github.com/google/uuid"
	"github.com/pmacik/loginusers-go/config"
	"github.com/pmacik/loginusers-go/loginusers"
)

// State represents JSON request for 'state setup' from Pact
type State struct {
	// Consumer name
	Consumer string `json:"consumer"`
	// State
	State string `json:"state"`
	// States
	States []string `json:"states"`
}

type createUserAttributes struct {
	Bio       string `json:"bio"`
	Cluster   string `json:"cluster"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	RhdUserID string `json:"rhd_user_id"`
}

type createUserData struct {
	createUserAttributes `json:"attributes"`
	Type                 string `json:"type" pact:"example=identities"`
}

type createUserRequest struct {
	createUserData `json:"data"`
}

// Setup starts a setup service for a provider - should be replaced by a provider setup endpoint
func Setup(setupHost string, setupPort int, params map[string]string) *contracts.ProviderInitialState {
	log.SetOutput(os.Stdout)

	userName := params["userName"]
	userPassword := params["userPassword"]

	spaceName := params["spaceName"]

	pactProviderBaseURL := params["pactProviderBaseURL"]
	pactProviderAuthBaseURL := params["pactProviderAuthBaseURL"]

	// Create test user in Auth and retun user info (such as id)
	log.Printf("Making sure user %s is created...", userName)
	var user = createUser(pactProviderAuthBaseURL, userName)
	if user == nil {
		log.Fatalf("Error creating/getting user")
		return nil
	}

	loginUsersConfig := config.DefaultConfig()
	loginUsersConfig.Auth.ServerAddress = pactProviderAuthBaseURL
	// Login user to get tokens
	userTokens, err := loginusers.OAuth2(userName, userPassword, loginUsersConfig)
	if err != nil {
		log.Fatalf("Unable to login user: %s", err)
		return nil
	}
	log.Printf("Provider setup with user ID: %s", user.Data.ID)

	//Create userspace
	space := createSpace(pactProviderBaseURL, spaceName, user, userTokens.AccessToken)
	if space == nil {
		log.Fatalf("Error creating/getting space")
		return nil
	}
	go setupEndpoint(setupHost, setupPort)

	return &contracts.ProviderInitialState{
		User:      *user,
		Space:     *space,
		UserToken: userTokens.AccessToken,
	}
}

func setupEndpoint(setupHost string, setupPort int) {
	http.HandleFunc("/pact/setup", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatalf(">>> ERROR: Unable to read request body.\n %q", err)
			return
		}
		//log.Printf("\nBody: %s\n", body)
		//log.Printf("\nHeaders: %s\n", r.Header)

		var providerState State
		json.Unmarshal(body, &providerState)

		switch providerState.State {
		case "Space with a given name does not exist.",
			"Space with a given ID exists.",
			"WIT service is up and running.":
			log.Printf(">>>> %s\n", providerState.State)
		default:
			errorMessage(w, fmt.Sprintf("State '%s' not impemented.", providerState.State))
			return
		}
		fmt.Fprintf(w, "Provider states has ben set up.\n")
	})

	var setupURL = fmt.Sprintf("%s:%d", setupHost, setupPort)
	log.Printf(">>> Starting ProviderSetup and listening at %s\n", setupURL)
	log.Fatal(http.ListenAndServe(setupURL, nil))
}

func errorMessage(w http.ResponseWriter, errorMessage string) {
	w.WriteHeader(500)
	fmt.Fprintf(w, `{"error": "%s"}`, errorMessage)
}

func createUser(providerAuthBaseURL string, userName string) *contracts.User {

	var httpClient = &http.Client{
		Timeout: time.Second * 10,
	}

	log.Println("Getting the auth service account token")
	authServiceAccountToken := serviceAccountToken(providerAuthBaseURL)
	// log.Printf("Auth Service Token: %s", authServiceAccountToken)

	rhdUserUUID, _ := uuid.NewUUID()
	message := &createUserRequest{
		createUserData: createUserData{
			createUserAttributes: createUserAttributes{
				Bio:       "Contract testing user account",
				Cluster:   "localhost",
				Email:     fmt.Sprintf("%s@email.com", userName),
				Username:  userName,
				RhdUserID: rhdUserUUID.String(),
			},
		},
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Fatalf("createUser: Error marshalling JSON object:\n%q", err)
	}

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/api/users", providerAuthBaseURL), bytes.NewBuffer(messageBytes))
	if err != nil {
		log.Fatalf("createUser: Error creating HTTP request:\n%q", err)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authServiceAccountToken))

	log.Println("Sending a request to create a user")
	response, err := httpClient.Do(request)
	if err != nil {
		log.Fatalf("createUser: Error sending HTTP request:\n%q", err)
	}
	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)

	if response.StatusCode != 200 {
		if response.StatusCode == 409 { //user already exists
			log.Printf("User %s already exists, getting user info.", userName)
			response2, err := http.Get(fmt.Sprintf("%s/api/users?filter[username]=%s", providerAuthBaseURL, userName))
			if err != nil {
				log.Fatalf("userExists: Error creating HTTP request:\n%q", err)
			}
			defer response2.Body.Close()

			responseBody, err := ioutil.ReadAll(response2.Body)
			// log.Printf("User info:\n%s\n", responseBody)
			if response2.StatusCode != 200 {
				log.Fatalf("userExists: Something went wrong with reading response body: %s", responseBody)
			}
			var users contracts.Users
			err = json.Unmarshal(responseBody, &users)
			if err != nil {
				log.Fatalf("userExists: Unable to unmarshal response body: %s", err)
			}
			var user = &contracts.User{
				Data: users.Data[0],
			}
			log.Printf("User found with ID: %s", user.Data.ID)
			return user
		}
		log.Fatalf("createUser: Something went wrong with reading response body: %s", responseBody)
	}

	var user contracts.User
	err = json.Unmarshal(responseBody, &user)
	if err != nil {
		log.Fatalf("createUser: Unable to unmarshal response body: %s", err)
	}
	log.Printf("User created with ID: %s", user.Data.ID)
	return &user
}

// ServiceAccountTokenRequest represents a request JSON body
type ServiceAccountTokenRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// ServiceAccountTokenResponse represents a response JSON body
type ServiceAccountTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func serviceAccountToken(providerBaseURL string) string {
	var httpClient = &http.Client{
		Timeout: time.Second * 10,
	}
	authClientID := "f867ec72-3171-4b8f-8eec-90a32eab6e0b"
	authClienSecret := "secret"

	message, err := json.Marshal(&ServiceAccountTokenRequest{
		GrantType:    "client_credentials",
		ClientID:     authClientID,
		ClientSecret: authClienSecret,
	})

	// log.Printf("Message: %s", string(message))

	if err != nil {
		log.Fatalf("serviceAccountToken: Error marshalling json object: %q\n", err)
	}
	request, err := http.NewRequest("POST", fmt.Sprintf("%s/api/token", providerBaseURL), bytes.NewBuffer(message))
	request.Header.Add("Content-Type", "application/json")

	response, err := httpClient.Do(request)
	if err != nil {
		log.Fatalf("serviceAccountToken: Error sending HTTP request: %q\n", err)
	}
	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)

	if response.StatusCode != 200 {
		log.Fatalf("serviceAccountToken: Something went wrong with reading response body: %s", responseBody)
	}

	var tokenResponse ServiceAccountTokenResponse
	err = json.Unmarshal(responseBody, &tokenResponse)
	if err != nil {
		log.Fatalf("serviceAccountToken: Unable to unmarshal response body: %s", err)
	}
	return tokenResponse.AccessToken
}

func createSpace(providerBaseURL string, spaceName string, user *contracts.User, userToken string) *contracts.Space {

	var httpClient = &http.Client{
		Timeout: time.Second * 10,
	}

	message := &contracts.CreateSpaceRequest{
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
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Fatalf("createSpace: Error marshalling JSON object:\n%q", err)
		return nil
	}

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/api/spaces", providerBaseURL), bytes.NewBuffer(messageBytes))
	if err != nil {
		log.Fatalf("createSpace: Error creating HTTP request:\n%q", err)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", userToken))

	log.Println("Sending a request to create a space")
	response, err := httpClient.Do(request)
	if err != nil {
		log.Fatalf("createSpace: Error sending HTTP request:\n%q", err)
	}
	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)

	if response.StatusCode != 201 {
		if response.StatusCode == 409 { //space already exists
			userName := user.Data.Attributes.Username
			log.Printf("Space %s already exists, getting space info.", spaceName)
			response2, err := http.Get(fmt.Sprintf("%s/api/namedspaces/%s", providerBaseURL, userName))
			if err != nil {
				log.Fatalf("spaceExists: Error creating HTTP request:\n%q", err)
			}
			defer response2.Body.Close()
			log.Println("2")
			responseBody, err := ioutil.ReadAll(response2.Body)
			// log.Printf("User info:\n%s\n", responseBody)
			if response2.StatusCode != 200 {
				log.Fatalf("spaceExists: Something went wrong with reading response body: %s", responseBody)
			}
			log.Println("3")
			var spaces contracts.Spaces
			err = json.Unmarshal(responseBody, &spaces)
			if err != nil {
				log.Fatalf("spaceExists: Unable to unmarshal response body: %s", err)
			}
			log.Println("4")
			for _, space := range spaces.Data {
				log.Println("5")
				if space.Attributes.Name == spaceName {
					log.Printf("Space found with ID: %s", space.ID)
					return &contracts.Space{
						Data: space,
					}
				}
			}
		}
		log.Fatalf("createSpace: Something went wrong with reading response body: %s", responseBody)
		return nil
	}

	var space contracts.Space
	err = json.Unmarshal(responseBody, &space)
	if err != nil {
		log.Fatalf("createSpace: Unable to unmarshal response body: %s", err)
	}
	log.Printf("Space created with ID: %s", user.Data.ID)
	return &space
}
