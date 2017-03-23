package login

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/almighty/almighty-core/errors"
)

// KeycloakUserProfile represents standard Keycloak User profile payload
type KeycloakUserProfile struct {
	FirstName  *string                        `json:"firstName,omitempty"`
	LastName   *string                        `json:"lastName,omitempty"`
	Email      *string                        `json:"email,omitempty"`
	Attributes *KeycloakUserProfileAttributes `json:"attributes,omitempty"`
}

// KeycloakUserProfileAttributes represents standard Keycloak profile payload Attributes
type KeycloakUserProfileAttributes struct {
	Bio      *string `json:"bio,omitempty"`
	URL      *string `json:"url,omitempty"`
	ImageURL *string `json:"image_url,omitempty"`
}

// NewKeycloakUserProfile creates a new keycloakUserProfile instance.
func NewKeycloakUserProfile(firstName, lastName, email, bio, url, imageURL *string) *KeycloakUserProfile {
	return &KeycloakUserProfile{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Attributes: &KeycloakUserProfileAttributes{
			Bio:      bio,
			URL:      url,
			ImageURL: imageURL,
		},
	}
}

// UserProfileService describes what the services need to be capable of doing.
type UserProfileService interface {
	Update(keycloakUserProfile *KeycloakUserProfile, accessToken string, keycloakProfileURL string) error
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

//Update updates the user profile information in Keycloak
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
		return errors.NewInternalError(err.Error())
	}
	defer resp.Body.Close()
	return nil
}
