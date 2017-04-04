package login

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/rest"
)

const ImageURLAttributeName = "ImageURL"
const BioAttributeName = "Bio"
const URLAttributeName = "URL"

// KeycloakUserProfile represents standard Keycloak User profile api request payload
type KeycloakUserProfile struct {
	ID         *string                        `json:"id,omitempty"`
	CreatedAt  int64                          `json:"createdTimestamp,omitempty"`
	Username   *string                        `json:"id,omitempty"`
	FirstName  *string                        `json:"firstName,omitempty"`
	LastName   *string                        `json:"lastName,omitempty"`
	Email      *string                        `json:"email,omitempty"`
	Attributes *KeycloakUserProfileAttributes `json:"attributes,omitempty"`
}

// KeycloakUserProfileAttributes represents standard Keycloak profile payload Attributes
type KeycloakUserProfileAttributes map[string][]string

//KeycloakUserProfileResponse represents the user profile api response from keycloak
type KeycloakUserProfileResponse struct {
	ID                         *string                        `json:"id"`
	CreatedTimestamp           *int64                         `json:"createdTimestamp"`
	Username                   *string                        `json:"username"`
	Enabled                    *bool                          `json:"enabled"`
	Totp                       *bool                          `json:"totp"`
	EmailVerified              *bool                          `json:"emailVerified"`
	FirstName                  *string                        `json:"firstName"`
	LastName                   *string                        `json:"lastName"`
	Email                      *string                        `json:"email"`
	Attributes                 *KeycloakUserProfileAttributes `json:"attributes"`
	DisableableCredentialTypes []*string                      `json:"disableableCredentialTypes"`
	RequiredActions            []interface{}                  `json:"requiredActions"`
}

// NewKeycloakUserProfile creates a new keycloakUserProfile instance.
func NewKeycloakUserProfile(firstName *string, lastName *string, email *string, attributes *KeycloakUserProfileAttributes) *KeycloakUserProfile {
	return &KeycloakUserProfile{
		FirstName:  firstName,
		LastName:   lastName,
		Email:      email,
		Attributes: attributes,
	}
}

// UserProfileService describes what the services need to be capable of doing.
type UserProfileService interface {
	Update(keycloakUserProfile *KeycloakUserProfile, accessToken string, keycloakProfileURL string) error
	Get(accessToken string, keycloakProfileURL string) (*KeycloakUserProfileResponse, error)
}

// KeycloakUserProfileClient describes the interface between platform and Keycloak User profile service.
type KeycloakUserProfileClient struct {
	client *http.Client
}

// NewKeycloakUserProfileClient creates a new KeycloakUserProfileClient
func NewKeycloakUserProfileClient() *KeycloakUserProfileClient {
	return &KeycloakUserProfileClient{
		client: http.DefaultClient,
	}
}

// Update updates the user profile information in Keycloak
func (userProfileClient *KeycloakUserProfileClient) Update(keycloakUserProfile *KeycloakUserProfile, accessToken string, keycloakProfileURL string) error {
	body, err := json.Marshal(keycloakUserProfile)
	if err != nil {
		return errors.NewInternalError(err.Error())
	}

	req, err := http.NewRequest("POST", keycloakProfileURL, bytes.NewReader(body))
	if err != nil {
		return errors.NewInternalError(err.Error())
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := userProfileClient.client.Do(req)

	if err != nil {
		log.Error(context.Background(), map[string]interface{}{
			"keycloak_user_profile_url": keycloakProfileURL,
			"err": err,
		}, "Unable to update Keycloak user profile")
		return errors.NewInternalError(err.Error())
	} else if resp != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		log.Error(context.Background(), map[string]interface{}{
			"response_status":           resp.Status,
			"response_body":             rest.ReadBody(resp.Body),
			"keycloak_user_profile_url": keycloakProfileURL,
		}, "Unable to update Keycloak user profile")
		return errors.NewInternalError(fmt.Sprintf("Received a non-200 response %s while updating keycloak user profile %s", resp.Status, keycloakProfileURL))
	}
	return nil
}

//Get gets the user profile information from Keycloak
func (userProfileClient *KeycloakUserProfileClient) Get(accessToken string, keycloakProfileURL string) (*KeycloakUserProfileResponse, error) {

	keycloakUserProfileResponse := KeycloakUserProfileResponse{}

	req, err := http.NewRequest("GET", keycloakProfileURL, nil)
	if err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json, text/plain, */*")

	resp, err := userProfileClient.client.Do(req)

	if err != nil {
		log.Error(context.Background(), map[string]interface{}{
			"keycloak_user_profile_url": keycloakProfileURL,
			"err": err,
		}, "Unable to fetch Keycloak user profile")
		return nil, errors.NewInternalError(err.Error())
	} else if resp != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		log.Error(context.Background(), map[string]interface{}{
			"response_status":           resp.Status,
			"response_body":             rest.ReadBody(resp.Body),
			"keycloak_user_profile_url": keycloakProfileURL,
		}, "Unable to fetch Keycloak user profile")
		return nil, errors.NewInternalError(fmt.Sprintf("Received a non-200 response %s while fetching keycloak user profile %s", resp.Status, keycloakProfileURL))
	}

	err = json.NewDecoder(resp.Body).Decode(&keycloakUserProfileResponse)
	return &keycloakUserProfileResponse, err
}
