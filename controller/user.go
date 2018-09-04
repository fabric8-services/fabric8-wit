package controller

import (
	"context"
	"net/http"
	"net/url"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/configuration"
	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/fabric8-services/fabric8-wit/goasupport"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/goadesign/goa"
	goaclient "github.com/goadesign/goa/client"
	errs "github.com/pkg/errors"
)

// UserController implements the user resource.
type UserController struct {
	*goa.Controller
	config      UserControllerConfiguration
	httpOptions []configuration.HTTPClientOption
	db          application.DB
	InitTenant  func(context.Context) error
}

// UserControllerConfiguration the configuration for the UserController
type UserControllerConfiguration interface {
	auth.ServiceConfiguration
	GetCacheControlUser() string
}

// NewUserController creates a user controller.
func NewUserController(service *goa.Service, db application.DB, config UserControllerConfiguration, httpOptions ...configuration.HTTPClientOption) *UserController {
	return &UserController{
		Controller:  service.NewController("UserController"),
		config:      config,
		db:          db,
		httpOptions: httpOptions,
	}
}

// Show returns the authorized user based on the provided Token
func (c *UserController) Show(ctx *app.ShowUserContext) error {
	client, err := newAuthClient(ctx, c.config, c.httpOptions...)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	res, err := client.ShowUser(goasupport.ForwardContextRequestID(ctx), authservice.ShowUserPath(), ctx.IfModifiedSince, ctx.IfNoneMatch)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to get user from the auth service")
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "unable to get user from the auth service"))
	}
	defer rest.CloseResponse(res)

	switch res.StatusCode {
	case 200:
	// OK
	case 401:
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(rest.ReadBody(res.Body)))
	default:
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Errorf("status: %s, body: %s", res.Status, rest.ReadBody(res.Body))))
	}

	authUser, _ := client.DecodeUser(res)

	if c.InitTenant != nil {
		go func(ctx context.Context) {
			c.InitTenant(ctx)
		}(ctx)
	}
	return ctx.OK(convertToAppUser(authUser))
}

func convertToAppUser(user *authservice.User) *app.User {
	return &app.User{
		Data: &app.UserData{
			ID:   user.Data.ID,
			Type: user.Data.Type,
			Attributes: &app.UserDataAttributes{
				CreatedAt:             user.Data.Attributes.CreatedAt,
				UpdatedAt:             user.Data.Attributes.UpdatedAt,
				Username:              user.Data.Attributes.Username,
				FullName:              user.Data.Attributes.FullName,
				ImageURL:              user.Data.Attributes.ImageURL,
				Bio:                   user.Data.Attributes.Bio,
				URL:                   user.Data.Attributes.URL,
				UserID:                user.Data.Attributes.UserID,
				IdentityID:            user.Data.Attributes.IdentityID,
				ProviderType:          user.Data.Attributes.ProviderType,
				Email:                 user.Data.Attributes.Email,
				Company:               user.Data.Attributes.Company,
				ContextInformation:    user.Data.Attributes.ContextInformation,
				RegistrationCompleted: user.Data.Attributes.RegistrationCompleted,
			},
			Links: &app.GenericLinks{
				Self:    user.Data.Links.Self,
				Related: user.Data.Links.Related,
				Meta:    user.Data.Links.Meta,
			},
		},
	}
}

// ListSpaces returns the list of spaces in which the user has a role
func (c *UserController) ListSpaces(ctx *app.ListSpacesUserContext) error {
	client, err := newAuthClient(ctx, c.config, c.httpOptions...)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	res, err := client.ListResourcesUser(goasupport.ForwardContextRequestID(ctx), authservice.ListResourcesUserPath(), "spaces")
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to get spaces with a role from the auth service")
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "unable to get spaces with a role from the auth service"))
	}
	defer rest.CloseResponse(res)

	switch res.StatusCode {
	case 200:
	// OK
	case 401:
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(rest.ReadBody(res.Body)))
	default:
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Errorf("status: %s, body: %s", res.Status, rest.ReadBody(res.Body))))
	}

	spaces, err := client.DecodeUserResourcesList(res)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to decode spaces with a role from the auth service")
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "unable to decode spaces with a role from the auth service"))
	}
	result, err := convertToUserSpaces(ctx, ctx.Request, c.db, spaces)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to returns spaces with a role from the auth service")
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "unable to return spaces with a role from the auth service"))
	}
	return ctx.OK(result)
}

func convertToUserSpaces(ctx context.Context, request *http.Request, db application.DB, spaces *authservice.UserResourcesList) (*app.UserSpacesList, error) {
	data := make([]*app.UserSpacesData, len(spaces.Data))
	for i, s := range spaces.Data {
		spaceID, err := uuid.FromString(s.ID)
		if err != nil {
			return nil, errs.Wrapf(err, "unable to fetch space name: ID '%s' is not a UUID.", s.ID)
		}
		space, err := db.Spaces().Load(ctx, spaceID)
		if err != nil {
			return nil, errs.Wrapf(err, "unable to fetch name of space with ID '%s'", s.ID)
		}
		selfLink := rest.AbsoluteURL(request, app.SpaceHref(s.ID))
		data[i] = &app.UserSpacesData{
			ID:   s.ID,
			Type: s.Type,
			Links: &app.GenericLinks{
				Self: &selfLink,
			},
			Attributes: &app.UserSpacesDataAttributes{
				Name: space.Name,
			},
		}
	}
	return &app.UserSpacesList{
		Data: data,
		Meta: &app.UserSpacesListMeta{
			TotalCount: len(spaces.Data),
		},
	}, nil
}

func newAuthClient(ctx context.Context, config UserControllerConfiguration, options ...configuration.HTTPClientOption) (*authservice.Client, error) {
	u, err := url.Parse(config.GetAuthServiceURL())
	if err != nil {
		return nil, err
	}
	httpClient := http.DefaultClient
	// apply options
	for _, opt := range options {
		opt(httpClient)
	}
	c := authservice.New(goaclient.HTTPClientDoer(httpClient))
	c.Host = u.Host
	c.Scheme = u.Scheme
	c.SetJWTSigner(goasupport.NewForwardSigner(ctx))
	return c, nil
}
