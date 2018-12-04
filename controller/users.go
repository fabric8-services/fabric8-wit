package controller

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"

	idpackage "github.com/fabric8-services/fabric8-common/id"
	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/token"

	"github.com/fabric8-services/fabric8-wit/rest/proxy"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

const (
	usersEndpoint   = "/api/users"
	serviceNameAuth = "fabric8-auth"
)

// UsersController implements the users resource.
type UsersController struct {
	*goa.Controller
	db     application.DB
	config UsersControllerConfiguration
}

// UsersControllerConfiguration the configuration for the UsersController
type UsersControllerConfiguration interface {
	auth.ServiceConfiguration
	GetCacheControlUsers() string
	GetCacheControlUser() string
	GetKeycloakAccountEndpoint(*http.Request) (string, error)
}

// NewUsersController creates a users controller.
func NewUsersController(service *goa.Service, db application.DB, config UsersControllerConfiguration) *UsersController {
	return &UsersController{
		Controller: service.NewController("UsersController"),
		db:         db,
		config:     config,
	}
}
func randString(length int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	randomStr := make([]byte, length)
	for i := range randomStr {
		randomStr[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(randomStr)
}

// Obfuscate runs the obfuscate action to invalidate the sensitive data associated
// with an user and her associated identity.
func (c *UsersController) Obfuscate(ctx *app.ObfuscateUsersContext) error {
	isSvcAccount, err := isServiceAccount(ctx, serviceNameAuth)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "failed to determine if account is a service account")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err))

	}
	if !isSvcAccount {
		log.Error(ctx, map[string]interface{}{
			"identity_id": ctx.ID,
		}, "account used to call obfuscate API is not a service account")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(errs.New("account used to call obfuscate API is not a service account")))
	}
	u, err := uuid.FromString(ctx.ID)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "failed to convert user id in valid uuid")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest(errs.New("invalid user id")))

	}
	err = application.Transactional(c.db, func(appl application.Application) error {
		obfStr := randString(12)

		// Obfuscate User
		users, err := appl.Users().Query(account.UserFilterByID(u))
		if err != nil || len(users) != 1 {
			log.Error(ctx, map[string]interface{}{
				"user_id": u,
				"err":     err,
			}, "unable to load the user")
			return errs.WithStack(err)
		}
		user := users[0]
		user.Email = obfStr + "@mail.com"
		user.FullName = obfStr
		user.ImageURL = obfStr
		user.Bio = obfStr
		user.URL = obfStr
		user.ContextInformation = nil
		user.Company = obfStr
		if err := appl.Users().Save(ctx.Context, &user); err != nil {
			log.Error(ctx, map[string]interface{}{
				"user_id": u,
				"err":     err,
			}, "unable to obfuscate the user")
			return errs.WithStack(err)
		}

		log.Debug(ctx, map[string]interface{}{
			"user_id": u,
		}, "User obfuscated!")

		// Obfuscate associated identity
		identities, err := appl.Identities().Query(account.IdentityFilterByUserID(u))
		if err != nil || len(identities) == 0 {
			log.Error(ctx, map[string]interface{}{
				"user_id": u,
				"err":     err,
			}, "unable to retrieve the identity associated to this user id")
			return errs.WithStack(err)
		}
		for _, identity := range identities {
			identity.Username = obfStr
			identity.ProfileURL = &obfStr

			if err := appl.Identities().Save(ctx.Context, &identity); err != nil {
				log.Error(ctx, map[string]interface{}{
					"user_id": u,
					"err":     err,
				}, "unable to obfuscate the identity")
				return errs.WithStack(err)
			}

			log.Debug(ctx, map[string]interface{}{
				"user_id": u,
			}, "Identity obfuscated!")

		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.OK([]byte{})
}

// Show runs the show action.
func (c *UsersController) Show(ctx *app.ShowUsersContext) error {
	return proxy.RouteHTTP(ctx, c.config.GetAuthShortServiceHostName())
}

// CreateUserAsServiceAccount updates a user when requested using a service account token
func (c *UsersController) CreateUserAsServiceAccount(ctx *app.CreateUserAsServiceAccountUsersContext) error {

	isSvcAccount, err := isServiceAccount(ctx, serviceNameAuth)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "failed to determine if account is a service account")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err))

	}
	if !isSvcAccount {
		log.Error(ctx, map[string]interface{}{
			"identity_id": ctx.ID,
		}, "account used to call create api is not a service account")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(errs.New("a non-service account tried to create a user.")))
	}

	return c.createUserInDB(ctx)
}

func (c *UsersController) createUserInDB(ctx *app.CreateUserAsServiceAccountUsersContext) error {

	userID, err := uuid.FromString(ctx.Payload.Data.Attributes.UserID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest(errs.New("invalid user id")))
	}

	id, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest(errs.New("incorrect identity id")))
	}

	returnResponse := application.Transactional(c.db, func(appl application.Application) error {

		var user *account.User
		var identity *account.Identity

		// Mandatory attributes

		user = &account.User{
			ID:    userID,
			Email: ctx.Payload.Data.Attributes.Email,
		}
		identity = &account.Identity{
			ID:           id,
			Username:     ctx.Payload.Data.Attributes.Username,
			ProviderType: ctx.Payload.Data.Attributes.ProviderType,
		}
		// associate foreign key
		identity.UserID = idpackage.NullUUID{UUID: user.ID, Valid: true}

		// Optional Attributes

		updatedRegistratedCompleted := ctx.Payload.Data.Attributes.RegistrationCompleted
		if updatedRegistratedCompleted != nil {
			identity.RegistrationCompleted = true
		}

		updatedBio := ctx.Payload.Data.Attributes.Bio
		if updatedBio != nil {
			user.Bio = *updatedBio
		}

		updatedFullName := ctx.Payload.Data.Attributes.FullName
		if updatedFullName != nil {
			user.FullName = *updatedFullName
		}

		updatedImageURL := ctx.Payload.Data.Attributes.ImageURL
		if updatedImageURL != nil {
			user.ImageURL = *updatedImageURL
		}

		updateURL := ctx.Payload.Data.Attributes.URL
		if updateURL != nil {
			user.URL = *updateURL
		}

		updatedCompany := ctx.Payload.Data.Attributes.Company
		if updatedCompany != nil {
			user.Company = *updatedCompany
		}

		updatedContextInformation := ctx.Payload.Data.Attributes.ContextInformation
		if updatedContextInformation != nil {
			if user.ContextInformation == nil {
				user.ContextInformation = account.ContextInformation{}
			}
			for fieldName, fieldValue := range updatedContextInformation {
				user.ContextInformation[fieldName] = fieldValue
			}
		}

		err = appl.Users().Create(ctx, user)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		err = appl.Identities().Create(ctx, identity)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		return ctx.OK([]byte{})
	})

	return returnResponse
}

// UpdateUserAsServiceAccount updates a user when requested using a service account token
func (c *UsersController) UpdateUserAsServiceAccount(ctx *app.UpdateUserAsServiceAccountUsersContext) error {

	isSvcAccount, err := isServiceAccount(ctx, serviceNameAuth)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":         err,
			"identity_id": ctx.ID,
		}, "failed to determine if account is a service account")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err))
	}
	if !isSvcAccount {
		log.Error(ctx, map[string]interface{}{
			"identity_id": ctx.ID,
		}, "failed to determine if account is a service account")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(errs.New("a non-service account tried to updated a user.")))
	}

	idString := ctx.ID
	id, err := uuid.FromString(idString)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest(errs.New("incorrect identity")))
	}
	return c.updateUserInDB(&id, ctx)
}

func isServiceAccount(ctx context.Context, serviceName string) (bool, error) {
	tokenManager, err := token.ReadManagerFromContext(ctx)
	if err != nil {
		return false, err
	}
	return (*tokenManager).IsServiceAccount(ctx, serviceName), nil
}

func (c *UsersController) updateUserInDB(id *uuid.UUID, ctx *app.UpdateUserAsServiceAccountUsersContext) error {

	// We'll refactor the old Users update API to be redirected to Auth service in the near future.
	// Hence, not spending time in refactoring that to consume this function.

	err := application.Transactional(c.db, func(appl application.Application) error {
		identity, err := appl.Identities().Load(ctx, *id)
		if err != nil || identity == nil {
			log.Error(ctx, map[string]interface{}{
				"identity_id": id,
				"err":         err,
			}, "id %s is unknown or error running query", *id)
			return err
		}

		var user *account.User
		if identity.UserID.Valid {
			user, err = appl.Users().Load(ctx.Context, identity.UserID.UUID)
			if err != nil {
				return errs.Wrap(err, fmt.Sprintf("Can't load user with id %s", identity.UserID.UUID))
			}
		}

		updatedEmail := ctx.Payload.Data.Attributes.Email
		if updatedEmail != nil && *updatedEmail != user.Email {
			user.Email = *updatedEmail
		}

		updatedUserName := ctx.Payload.Data.Attributes.Username
		if updatedUserName != nil {
			identity.Username = *updatedUserName
		}

		updatedRegistratedCompleted := ctx.Payload.Data.Attributes.RegistrationCompleted
		if updatedRegistratedCompleted != nil {
			identity.RegistrationCompleted = true
		}

		updatedBio := ctx.Payload.Data.Attributes.Bio
		if updatedBio != nil {
			user.Bio = *updatedBio
		}
		updatedFullName := ctx.Payload.Data.Attributes.FullName
		if updatedFullName != nil {
			user.FullName = *updatedFullName
		}
		updatedImageURL := ctx.Payload.Data.Attributes.ImageURL
		if updatedImageURL != nil {
			user.ImageURL = *updatedImageURL
		}
		updateURL := ctx.Payload.Data.Attributes.URL
		if updateURL != nil {
			user.URL = *updateURL
		}

		updatedCompany := ctx.Payload.Data.Attributes.Company
		if updatedCompany != nil {
			user.Company = *updatedCompany
		}

		updatedContextInformation := ctx.Payload.Data.Attributes.ContextInformation
		if updatedContextInformation != nil {
			// if user.ContextInformation , we get to PATCH the ContextInformation field,
			// instead of over-writing it altogether. Note: The PATCH-ing is only for the
			// 1st level of JSON.
			if user.ContextInformation == nil {
				user.ContextInformation = account.ContextInformation{}
			}
			for fieldName, fieldValue := range updatedContextInformation {
				// Save it as is, for short-term.
				user.ContextInformation[fieldName] = fieldValue
			}
		}

		err = appl.Users().Save(ctx, user)
		if err != nil {
			return err
		}

		err = appl.Identities().Save(ctx, identity)
		if err != nil {
			return err
		}
		return ctx.OK([]byte{})
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return nil
}

// Update updates the authorized user based on the provided Token
func (c *UsersController) Update(ctx *app.UpdateUsersContext) error {
	return proxy.RouteHTTP(ctx, c.config.GetAuthShortServiceHostName())
}

// List runs the list action.
func (c *UsersController) List(ctx *app.ListUsersContext) error {
	return proxy.RouteHTTP(ctx, c.config.GetAuthShortServiceHostName())
}

// ConvertUsersSimple converts a array of simple Identity IDs into a Generic Reletionship List
func ConvertUsersSimple(request *http.Request, identityIDs []interface{}) []*app.GenericData {
	ops := []*app.GenericData{}
	for _, identityID := range identityIDs {
		data, _ := ConvertUserSimple(request, identityID)
		ops = append(ops, data)
	}
	return ops
}

// ConvertUserSimple converts a simple Identity ID into a Generic Reletionship
func ConvertUserSimple(request *http.Request, identityID interface{}) (*app.GenericData, *app.GenericLinks) {
	t := "users"
	i := fmt.Sprint(identityID)
	data := &app.GenericData{
		Type: &t,
		ID:   &i,
	}
	relatedURL := rest.AbsoluteURL(request, app.UsersHref(i))
	links := &app.GenericLinks{
		Self:    &relatedURL,
		Related: &relatedURL,
	}
	return data, links
}
