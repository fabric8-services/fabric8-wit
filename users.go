package main

import (
	"fmt"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/rest"
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
		return ctx.OK(ConvertUser(ctx.RequestData, result))
	})
}

// ConvertUser converts a complete Identity object into REST representation
func ConvertUser(request *goa.RequestData, account *account.Identity) *app.Identity {
	id := account.ID.String()
	converted := app.Identity{
		Data: &app.IdentityData{
			ID:   &id,
			Type: "identities",
			Attributes: &app.IdentityDataAttributes{
				FullName: &account.FullName,
				ImageURL: &account.ImageURL,
			},
			Links: createUserLinks(request, account.ID),
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
	selfURL := rest.AbsoluteURL(request, app.UsersHref(id))
	return &app.GenericLinks{
		Self: &selfURL,
	}
}
