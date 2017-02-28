package controller

import (
	"fmt"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/util"
	"github.com/goadesign/goa"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// UsersController implements the users resource.
type UsersController struct {
	*goa.Controller
	db application.DB
}

// NewUsersController creates a users controller.
func NewUsersController(service *goa.Service, db application.DB) *UsersController {
	return &UsersController{Controller: service.NewController("UsersController"), db: db}
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
	if user != nil {
		fullName = user.FullName
		imageURL = user.ImageURL
		bio = user.Bio
		userURL = user.URL
		email = user.Email
	}
	converted := app.Identity{
		Data: &app.IdentityData{
			ID:   &id,
			Type: "identities",
			Attributes: &app.IdentityDataAttributes{
				Username:     &userName,
				FullName:     &fullName,
				ImageURL:     &imageURL,
				Bio:          &bio,
				URL:          &userURL,
				ProviderType: &providerType,
				Email:        &email,
			},
			Links: createUserLinks(request, uuid),
		},
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
	selfURL := util.AbsoluteURL(request, app.UsersHref(id))
	return &app.GenericLinks{
		Self: &selfURL,
	}
}
