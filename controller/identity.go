package controller

import (
	"fmt"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/goadesign/goa"
)

const identitiesEndpoint = "/api/identities"

// IdentityController implements the identity resource.
type IdentityController struct {
	*goa.Controller
	db application.DB
}

// NewIdentityController creates a identity controller.
func NewIdentityController(service *goa.Service, db application.DB) *IdentityController {
	return &IdentityController{Controller: service.NewController("IdentityController"), db: db}
}

// List runs the list action.
func (c *IdentityController) List(ctx *app.ListIdentityContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		modelIdentities, err := appl.Identities().List(ctx.Context)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(fmt.Sprintf("Error listing identities: %s", err.Error())))
			return ctx.InternalServerError(jerrors)
		}
		appIdentities := app.IdentityArray{}
		appIdentities.Data = make([]*app.IdentityData, len(modelIdentities))
		for index, modelIdentity := range modelIdentities {
			appIdentityData := ConvertIdentityFromModel(modelIdentity)
			appIdentities.Data[index] = appIdentityData
		}
		return ctx.OK(&appIdentities)
	})
}

// ConvertIdentityFromModel convert identity from model to app representation
func ConvertIdentityFromModel(m account.Identity) *app.IdentityData {
	id := m.ID.String()
	data := &app.IdentityData{
		ID:   &id,
		Type: "identities",
		Attributes: &app.IdentityDataAttributes{
			Username:     &m.Username,
			ProviderType: &m.ProviderType,
		},
	}
	return data
}
