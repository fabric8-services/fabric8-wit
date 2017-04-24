package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	errs "github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/workitem"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/jinzhu/gorm"
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
			log.Error(ctx, map[string]interface{}{
				"identity_id": identity.ID,
				"user_id":     identity.UserID.Valid,
				"err":         err,
			}, "failed to update keycloak account")
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		// The keycloak API doesn't support PATCH, hence the entire info needs
		// to be sent over for User profile updation in Keycloak. So the POST request to KC needs
		// to have everything - whatever we are updating, and whatever are not.
		keycloakUserProfile := copyExistingKeycloakUserProfileInfo(keycloakUserExistingInfo)

		updatedEmail := ctx.Payload.Data.Attributes.Email
		if updatedEmail != nil {
			isUnique, err := isEmailUnique(appl, *updatedEmail, *user)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, errors.Wrap(err, fmt.Sprintf("error updating idenitity with id %s and user with id %s", identity.ID, identity.UserID.UUID)))
			}
			if !isUnique {
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInvalidRequest(fmt.Sprintf("email address: %s is already in use", *updatedEmail)))
				return ctx.Conflict(jerrors)
			}
			user.Email = *updatedEmail
			keycloakUserProfile.Email = updatedEmail
		}

		updatedUserName := ctx.Payload.Data.Attributes.Username
		if updatedUserName != nil && *updatedUserName != identity.Username {
			if identity.RegistrationCompleted {
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInvalidRequest(fmt.Sprintf("username cannot be updated more than once for idenitity id %s ", *id)))
				return ctx.Forbidden(jerrors)
			}
			isUnique, err := isUsernameUnique(appl, *updatedUserName, *identity)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, errors.Wrap(err, fmt.Sprintf("error updating idenitity with id %s and user with id %s", identity.ID, identity.UserID.UUID)))
			}
			if !isUnique {
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInvalidRequest(fmt.Sprintf("username : %s is already in use", *updatedUserName)))
				return ctx.Conflict(jerrors)
			}
			identity.Username = *updatedUserName
			identity.RegistrationCompleted = true
			keycloakUserProfile.Username = updatedUserName
		}

		updatedRegistratedCompleted := ctx.Payload.Data.Attributes.RegistrationCompleted
		if updatedRegistratedCompleted != nil {
			if *updatedRegistratedCompleted == false {
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInvalidRequest(fmt.Sprintf("invalid value assigned to registration_completed")))
				return ctx.BadRequest(jerrors)
			}
			identity.RegistrationCompleted = true
		}

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

		updatedCompany := ctx.Payload.Data.Attributes.Company
		if updatedCompany != nil {
			user.Company = *updatedCompany
			(*keycloakUserProfile.Attributes)[login.CompanyAttributeName] = []string{*updatedCompany}
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

		// The update of the keycloak needs to be attempted first because if that fails,
		// we should't update the platform db since that would leave things in an
		// inconsistent state.
		err = c.userProfileService.Update(keycloakUserProfile, tokenString, accountAPIEndpoint)

		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"user_name": keycloakUserProfile.Username,
				"email":     keycloakUserProfile.Email,
				"err":       err,
			}, "failed to update keycloak account")

			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(err)

			// We have mapped keycloak's 500 InternalServerError to our errors.BadParameterError
			// because this scenario is directly associated with attempts to update
			// duplicate email and/or username.
			switch err.(type) {
			default:
				return ctx.BadRequest(jerrors)
			case errs.BadParameterError:
				return ctx.Conflict(jerrors)
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

func isUsernameUnique(appl application.Application, username string, identity account.Identity) (bool, error) {
	usersWithSameUserName, err := appl.Identities().Query(account.IdentityFilterByUsername(username), account.IdentityFilterByProviderType(account.KeycloakIDP))
	if err != nil {
		log.Error(context.Background(), map[string]interface{}{
			"user_name": username,
			"err":       err,
		}, "error fetching users with username filter")
		return false, err
	}
	for _, u := range usersWithSameUserName {
		if u.UserID.UUID != identity.UserID.UUID {
			return false, nil
		}
	}
	return true, nil
}

func isEmailUnique(appl application.Application, email string, user account.User) (bool, error) {
	usersWithSameEmail, err := appl.Users().Query(account.UserFilterByEmail(email))
	if err != nil {
		log.Error(context.Background(), map[string]interface{}{
			"email": email,
			"err":   err,
		}, "error fetching identities with email filter")
		return false, err
	}
	for _, u := range usersWithSameEmail {
		if u.ID != user.ID {
			return false, nil
		}
	}
	return true, nil
}

// List runs the list action.
func (c *UsersController) List(ctx *app.ListUsersContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		var err error
		var identities []*account.Identity
		var users []*account.User
		var result *app.UserArray
		identityFilters := []func(*gorm.DB) *gorm.DB{}
		userFilters := []func(*gorm.DB) *gorm.DB{}

		var appIdentities []*app.IdentityData

		/*
			There are 2 database tables we fetch the data from : identities , users
			First, we filter on the attributes of identities table - providerType , username
			After that we use the above result to cummulatively filter on users  - email , company
		*/

		/*** Start filtering on Identities table ****/

		if ctx.FilterUsername != nil {
			identityFilters = append(identityFilters, account.IdentityFilterByUsername(*ctx.FilterUsername))
		}
		if ctx.FilterRegistrationCompleted != nil {
			identityFilters = append(identityFilters, account.IdentityFilterByRegistrationCompleted(*ctx.FilterRegistrationCompleted))
		}
		// Add more filters when needed , here. ..

		if len(identityFilters) != 0 {
			identityFilters = append(identityFilters, account.IdentityFilterByProviderType(account.KeycloakIDP))
			identityFilters = append(identityFilters, account.IdentityWithUser())

			// From a data model perspective, we are querying by identity ( and not user )
			identities, err = appl.Identities().Query(identityFilters...)

			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, errors.Wrap(err, "error fetching identities with filter(s)"))
			}

			// cumulatively filter out those not matcing the user-based filters.
			for _, identity := range identities {
				// this is where you keep trying all other filters one by one for 'user' fields like email.
				if ctx.FilterEmail == nil || identity.User.Email == *ctx.FilterEmail {

					// if one or more 'User' filters are present, check if it's satified, if Not, proceed with ConvertUser

					appIdentity := ConvertUser(ctx.RequestData, identity, &identity.User)
					appIdentities = append(appIdentities, appIdentity.Data)
				}
			}
			result = &app.UserArray{Data: appIdentities}

		} else {

			/*** Start filtering on Users table ****/

			if ctx.FilterEmail != nil {
				userFilters = append(userFilters, account.UserFilterByEmail(*ctx.FilterEmail))
			}
			// .. Add other filters in future when needed into the userFilters slice in the above manner.

			if len(userFilters) != 0 {
				users, err = appl.Users().Query(userFilters...)
			} else {
				// Not breaking the existing API - If no filters were passed, we fall back on the good old 'list everything'.
				// FIXME We should remove this when fabric8io/fabric8-planner#1538 is fixed
				users, err = appl.Users().List(ctx.Context)
			}

			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, errors.Wrap(err, "error fetching users"))
			}
			result, err = LoadKeyCloakIdentities(appl, ctx.RequestData, users)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, errors.Wrap(err, "error fetching keycloak identities"))
			}
		}
		if result == nil {
			result = &app.UserArray{Data: make([]*app.IdentityData, 0)}
		}
		return ctx.OK(result)
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
	registrationCompleted := identity.RegistrationCompleted
	providerType := identity.ProviderType
	var imageURL string
	var bio string
	var userURL string
	var email string
	var company string
	var contextInformation workitem.Fields

	if user != nil {
		fullName = user.FullName
		imageURL = user.ImageURL
		bio = user.Bio
		userURL = user.URL
		email = user.Email
		company = user.Company
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
				Username:              &userName,
				FullName:              &fullName,
				ImageURL:              &imageURL,
				Bio:                   &bio,
				URL:                   &userURL,
				ProviderType:          &providerType,
				Email:                 &email,
				Company:               &company,
				ContextInformation:    workitem.Fields{},
				RegistrationCompleted: &registrationCompleted,
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
