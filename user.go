package main

import (
	"fmt"
	"log"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
)

// UserController implements the user resource.
type UserController struct {
	*goa.Controller
	db           application.DB
	tokenManager token.Manager
}

// NewUserController creates a user controller.
func NewUserController(service *goa.Service, db application.DB, tokenManager token.Manager) *UserController {
	return &UserController{Controller: service.NewController("UserController"), db: db, tokenManager: tokenManager}
}

// Show returns the authorized user based on the provided Token
func (c *UserController) Show(ctx *app.ShowUserContext) error {
	id, err := c.tokenManager.Locate(ctx)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(err.Error()))
		return ctx.BadRequest(jerrors)
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		identity, err := appl.Identities().Load(ctx, id)
		if err != nil || identity == nil {
			log.Printf("Auth token contains id %s of unknown Identity\n", id)
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(fmt.Sprintf("Auth token contains id %s of unknown Identity\n", id)))
			return ctx.Unauthorized(jerrors)
		}

		var user *account.User
		userID := identity.UserID
		if userID.Valid {
			user, err = appl.Users().Load(ctx.Context, userID.UUID)
			if err != nil {
				jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
				return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
			}
		}

		return ctx.OK(ConvertUser(ctx.RequestData, identity, user))
	})
}
