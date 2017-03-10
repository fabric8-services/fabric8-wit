package controller

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/goadesign/goa"
)

// CollabspacesController implements the collabspaces resource.
type CollabspacesController struct {
	*goa.Controller
	db application.DB
}

// NewCollabspacesController creates a collabspaces controller.
func NewCollabspacesController(service *goa.Service, db application.DB) *CollabspacesController {
	return &CollabspacesController{Controller: service.NewController("CollabspacesController"), db: db}
}

// List runs the list action.
func (c *CollabspacesController) List(ctx *app.ListCollabspacesContext) error {
	offset, limit := computePagingLimts(ctx.PageOffset, ctx.PageLimit)
	if ctx.UserName == "" {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound("not found, userName=%v", ctx.UserName))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		identity, err := loadKeyCloakIdentityByUserName(ctx, appl, ctx.UserName)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound("not found, userName=%v", ctx.UserName))
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
