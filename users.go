package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/goadesign/goa"
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
		result, err := appl.Users().Load(ctx.Context, id)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		return ctx.OK(result.ConvertIdentityFromModel())
	})
}
