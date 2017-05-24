package controller

import (
	"testing"

	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

func TestCopyExistingKeycloakUserProfileInfo(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	LastName := "lastname"
	Email := "s@s.com"
	URL := "http://noURL"

	keycloakUserProfile := &login.KeycloakUserProfile{
		LastName: &LastName,
		Email:    &Email,
		Attributes: &login.KeycloakUserProfileAttributes{
			login.URLAttributeName: {URL},
		},
	}

	Username := "user1"           // this isnt being updated
	FirstName := "firstname"      // this isnt being updated
	oldLastName := "oldlast name" // will be updated
	oldEmail := "old@email"       // will be updated
	Bio := "No more john doe"     // will  not be updated.

	existingProfile := &login.KeycloakUserProfileResponse{
		Username:  &Username,
		FirstName: &FirstName,
		LastName:  &oldLastName,
		Email:     &oldEmail,
		Attributes: &login.KeycloakUserProfileAttributes{
			login.BioAttributeName: {Bio},
			login.URLAttributeName: {URL},
		},
	}

	mergedProfile := mergeKeycloakUserProfileInfo(keycloakUserProfile, existingProfile)

	// ensure existing properties stays as is
	assert.Equal(t, *mergedProfile.Username, Username)
	assert.Equal(t, *mergedProfile.FirstName, FirstName)

	// ensure last name is updated
	assert.Equal(t, *mergedProfile.LastName, LastName)

	// ensure URL is updated
	retrievedURL := (*mergedProfile.Attributes)[login.URLAttributeName]
	assert.Equal(t, retrievedURL[0], URL)

	// ensure existing attributes dont get changed
	retrievedBio := (*mergedProfile.Attributes)[login.BioAttributeName]
	assert.Equal(t, retrievedBio[0], Bio)

}
