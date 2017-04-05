package controller

import (
	"fmt"
	"strings"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/workitem"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/pkg/errors"

	uuid "github.com/satori/go.uuid"
)

type usersConfiguration interface {
	// add configuration specific to keycloak user profile api url
	GetKeycloakAccountEndpoint(*goa.RequestData) (string, error)
}

// UsersController implements the users resource.
type UsersController struct {
	*goa.Controller
	db                 application.DB
	configuration      usersConfiguration
	userProfileService login.UserProfileService
}

// NewUsersController creates a users controller.
func NewUsersController(service *goa.Service, db application.DB, configuration usersConfiguration, userProfileService login.UserProfileService) *UsersController {
	return &UsersController{Controller: service.NewController("UsersController"), db: db, configuration: configuration, userProfileService: userProfileService}
}

// Show runs the show action.
func (c *UsersController) Show(ctx *app.ShowUsersContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		id, err := uuid.FromString(ctx.ID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		identity, err := appl.Identities().Load(ctx.Context, id)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		var user *account.User
		userID := identity.UserID
		if userID.Valid {
			user, err = appl.Users().Load(ctx.Context, userID.UUID)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, errors.Wrap(err, fmt.Sprintf("User ID %s not valid", userID.UUID)))
			}
		}
		return ctx.OK(ConvertUser(ctx.RequestData, identity, user))
	})
}

func copyExistingKeycloakUserProfileInfo(existingProfile *login.KeycloakUserProfileResponse) *login.KeycloakUserProfile {
	keycloakUserProfile := &login.KeycloakUserProfile{}
	keycloakUserProfile.Attributes = &login.KeycloakUserProfileAttributes{}

	if existingProfile.FirstName != nil {
		keycloakUserProfile.FirstName = existingProfile.FirstName
	}
	if existingProfile.LastName != nil {
		keycloakUserProfile.LastName = existingProfile.LastName
	}
	if existingProfile.Email != nil {
		keycloakUserProfile.Email = existingProfile.Email
	}
	if existingProfile.Attributes != nil {
		// If there are existing attributes, we overwite only those
		// handled by the Users service in platform.
		keycloakUserProfile.Attributes = existingProfile.Attributes
	}
	if existingProfile.Username != nil {
		keycloakUserProfile.Username = existingProfile.Username
	}
	return keycloakUserProfile
}

// Update updates the authorized user based on the provided Token
func (c *UsersController) Update(ctx *app.UpdateUsersContext) error {

	id, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		identity, err := appl.Identities().Load(ctx, *id)
		if err != nil || identity == nil {
			log.Error(ctx, map[string]interface{}{
				"identity_id": id,
			}, "auth token contains id %s of unknown Identity", *id)
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(fmt.Sprintf("Auth token contains id %s of unknown Identity\n", *id)))
			return ctx.Unauthorized(jerrors)
		}

		var user *account.User
		if identity.UserID.Valid {
			user, err = appl.Users().Load(ctx.Context, identity.UserID.UUID)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, errors.Wrap(err, fmt.Sprintf("Can't load user with id %s", identity.UserID.UUID)))
			}
		}

		// prepare for updating keycloak user profile
		tokenString := goajwt.ContextJWT(ctx).Raw

		accountAPIEndpoint, err := c.configuration.GetKeycloakAccountEndpoint(ctx.RequestData)
		keycloakUserExistingInfo, err := c.userProfileService.Get(tokenString, accountAPIEndpoint)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		// The keycloak API doesn't support PATCH, hence the entire info needs
		// to be sent over for User profile updation in Keycloak. So the POST request to KC needs
		// to have everything - whatever we are updating, and whatever are not.
		keycloakUserProfile := copyExistingKeycloakUserProfileInfo(keycloakUserExistingInfo)

		// Disabling updation of email till we figure out how to do the same in Keycloak Error-free.
		//
		/*
			updatedEmail := ctx.Payload.Data.Attributes.Email
			if updatedEmail != nil {
				user.Email = *updatedEmail
				keycloakUserProfile.Email = updatedEmail
			}
		*/

		updatedBio := ctx.Payload.Data.Attributes.Bio
		if updatedBio != nil {
			user.Bio = *updatedBio
			(*keycloakUserProfile.Attributes)[login.BioAttributeName] = []string{*updatedBio}
		}
		updatedFullName := ctx.Payload.Data.Attributes.FullName
		if updatedFullName != nil {
			*updatedFullName = standardizeSpaces(*updatedFullName)
			user.FullName = *updatedFullName

			// In KC, we store as first name and last name.
			nameComponents := strings.Split(*updatedFullName, " ")
			firstName := nameComponents[0]
			lastName := ""
			if len(nameComponents) > 1 {
				lastName = strings.Join(nameComponents[1:], " ")
			}

			keycloakUserProfile.FirstName = &firstName
			keycloakUserProfile.LastName = &lastName
		}
		updatedImageURL := ctx.Payload.Data.Attributes.ImageURL
		if updatedImageURL != nil {
			user.ImageURL = *updatedImageURL
			(*keycloakUserProfile.Attributes)[login.ImageURLAttributeName] = []string{*updatedImageURL}

		}
		updateURL := ctx.Payload.Data.Attributes.URL
		if updateURL != nil {
			user.URL = *updateURL
			(*keycloakUserProfile.Attributes)[login.URLAttributeName] = []string{*updateURL}
		}

		// If none of the 'extra' attributes were present, we better make that section nil
		// so that the Attributes section is omitted in the payload sent to KC

		if updatedBio == nil && updatedImageURL == nil && updateURL == nil {
			keycloakUserProfile.Attributes = nil
		}

		updatedContextInformation := ctx.Payload.Data.Attributes.ContextInformation
		if updatedContextInformation != nil {
			// if user.ContextInformation , we get to PATCH the ContextInformation field,
			// instead of over-writing it altogether. Note: The PATCH-ing is only for the
			// 1st level of JSON.
			if user.ContextInformation == nil {
				user.ContextInformation = workitem.Fields{}
			}
			for fieldName, fieldValue := range updatedContextInformation {
				// Save it as is, for short-term.
				user.ContextInformation[fieldName] = fieldValue
			}
		}

		err = appl.Users().Save(ctx, user)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		err = appl.Identities().Save(ctx, identity)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		c.userProfileService.Update(keycloakUserProfile, tokenString, accountAPIEndpoint)
		return ctx.OK(ConvertUser(ctx.RequestData, identity, user))
	})
}

// List runs the list action.
func (c *UsersController) List(ctx *app.ListUsersContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		var err error
		var users []*account.User
		var result *app.UserArray
		users, err = appl.Users().List(ctx.Context)
		if err == nil {
			result, err = LoadKeyCloakIdentities(appl, ctx.RequestData, users)
			if err == nil {
				return ctx.OK(result)
			}
		}
		return jsonapi.JSONErrorResponse(ctx, errors.Wrap(err, "Error listing users"))
	})
}

// LoadKeyCloakIdentities loads keycloak identies for the users and converts the users into REST representation
func LoadKeyCloakIdentities(appl application.Application, request *goa.RequestData, users []*account.User) (*app.UserArray, error) {
	data := make([]*app.IdentityData, len(users))
	for i, user := range users {
		identity, err := loadKeyCloakIdentity(appl, user)
		if err != nil {
			return nil, err
		}
		appIdentity := ConvertUser(request, identity, user)
		data[i] = appIdentity.Data
	}
	return &app.UserArray{Data: data}, nil
}

func loadKeyCloakIdentity(appl application.Application, user *account.User) (*account.Identity, error) {
	identities, err := appl.Identities().Query(account.IdentityFilterByUserID(user.ID))
	if err != nil {
		return nil, err
	}
	for _, identity := range identities {
		if identity.ProviderType == account.KeycloakIDP {
			return identity, nil
		}
	}
	return nil, fmt.Errorf("Can't find Keycloak Identity for user %s", user.Email)
}

// ConvertUser converts a complete Identity object into REST representation
func ConvertUser(request *goa.RequestData, identity *account.Identity, user *account.User) *app.Identity {
	uuid := identity.ID
	id := uuid.String()
	fullName := identity.Username
	userName := identity.Username
	providerType := identity.ProviderType
	var imageURL string
	var bio string
	var userURL string
	var email string
	var contextInformation workitem.Fields

	if user != nil {
		fullName = user.FullName
		imageURL = user.ImageURL
		bio = user.Bio
		userURL = user.URL
		email = user.Email
		contextInformation = user.ContextInformation
	}

	// The following will be used for ContextInformation.
	// The simplest way to represent is to have all fields
	// as a SimpleType. During conversion from 'model' to 'app',
	// the value would be returned 'as is'.

	simpleFieldDefinition := workitem.FieldDefinition{
		Type: workitem.SimpleType{Kind: workitem.KindString},
	}

	converted := app.Identity{
		Data: &app.IdentityData{
			ID:   &id,
			Type: "identities",
			Attributes: &app.IdentityDataAttributes{
				Username:           &userName,
				FullName:           &fullName,
				ImageURL:           &imageURL,
				Bio:                &bio,
				URL:                &userURL,
				ProviderType:       &providerType,
				Email:              &email,
				ContextInformation: workitem.Fields{},
			},
			Links: createUserLinks(request, uuid),
		},
	}
	for name, value := range contextInformation {
		if value == nil {
			// this can be used to unset a key in contextInformation
			continue
		}
		convertedValue, err := simpleFieldDefinition.ConvertFromModel(name, value)
		if err != nil {
			log.Error(nil, map[string]interface{}{
				"err": err,
			}, "Unable to convert user context field %s ", name)
			converted.Data.Attributes.ContextInformation[name] = nil
		}
		converted.Data.Attributes.ContextInformation[name] = convertedValue
	}
	return &converted
}

// ConvertUsersSimple converts a array of simple Identity IDs into a Generic Reletionship List
func ConvertUsersSimple(request *goa.RequestData, ids []interface{}) []*app.GenericData {
	ops := []*app.GenericData{}
	for _, id := range ids {
		ops = append(ops, ConvertUserSimple(request, id))
	}
	return ops
}

// ConvertUserSimple converts a simple Identity ID into a Generic Reletionship
func ConvertUserSimple(request *goa.RequestData, id interface{}) *app.GenericData {
	t := "identities"
	i := fmt.Sprint(id)
	return &app.GenericData{
		Type:  &t,
		ID:    &i,
		Links: createUserLinks(request, id),
	}
}

func createUserLinks(request *goa.RequestData, id interface{}) *app.GenericLinks {
	selfURL := rest.AbsoluteURL(request, app.UsersHref(id))
	return &app.GenericLinks{
		Self: &selfURL,
	}
}

func standardizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
