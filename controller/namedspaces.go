package controller

import (
	"context"
	"fmt"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	"github.com/goadesign/goa"
)

// NamedspacesController implements the namedspaces resource.
type NamedspacesController struct {
	*goa.Controller
	db application.DB
}

// NewNamedspacesController creates a namedspaces controller.
func NewNamedspacesController(service *goa.Service, db application.DB) *NamedspacesController {
	return &NamedspacesController{Controller: service.NewController("NamedspacesController"), db: db}
}

// Show runs the show action.
func (c *NamedspacesController) Show(ctx *app.ShowNamedspacesContext) error {
	if ctx.UserName == "" {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound("not found, userName=%v", ctx.UserName))
	}

	if ctx.SpaceName == "" {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound("not found, spaceName=%v", ctx.SpaceName))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		identity, err := loadKeyCloakIdentityByUserName(ctx, appl, ctx.UserName)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound("not found, userName=%v", ctx.UserName))
		}
		s, err := appl.Spaces().LoadByOwnerAndName(ctx.Context, &identity.ID, &ctx.SpaceName)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		resp := app.SpaceSingle{
			Data: ConvertSpace(ctx.RequestData, s),
		}

		return ctx.OK(&resp)
	})

	res := &app.SpaceSingle{}
	return ctx.OK(res)
}

func (c *NamedspacesController) List(ctx *app.ListNamedspacesContext) error {
	offset, limit := computePagingLimts(ctx.PageOffset, ctx.PageLimit)
	if ctx.UserName == "" {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(fmt.Sprintf("not found, userName=%v", ctx.UserName)))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		identity, err := loadKeyCloakIdentityByUserName(ctx, appl, ctx.UserName)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(fmt.Sprintf("not found, userName=%v. %v", ctx.UserName, err.Error())))
		}
		spaces, c, err := appl.Spaces().LoadByOwner(ctx.Context, &identity.ID, &offset, &limit)
		count := int(c)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		response := app.SpaceList{
			Links: &app.PagingLinks{},
			Meta:  &app.SpaceListMeta{TotalCount: count},
			Data:  ConvertSpaces(ctx.RequestData, spaces),
		}
		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(spaces), offset, limit, count)

		return ctx.OK(&response)
	})
}

func loadKeyCloakIdentityByUserName(ctx context.Context, appl application.Application, username string) (*account.Identity, error) {
	identities, err := appl.Identities().Query(account.IdentityFilterByUsername(username))
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"userName": username,
		}, "Fail to locate identity for user")
		return nil, err
	}
	for _, identity := range identities {
		if identity.ProviderType == account.KeycloakIDP {
			return identity, nil
		}
	}
	log.Error(ctx, map[string]interface{}{
		"userName": username,
	}, "Fail to locate Keycloak identity for user")
	return nil, fmt.Errorf("Can't find Keycloak Identity for user %s", username)
}
