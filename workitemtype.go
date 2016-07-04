package main

import (
	"strconv"
	
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
	"github.com/goadesign/goa"
)

var wellKnown = map[string]*models.WorkItemType{
	"1": &models.WorkItemType{
		Id:      1,
		Name:    "system.workitem",
		Version: 1,
		Fields: models.FieldTypes{
			"system.owner": models.SimpleType{Kind: models.User},
			"system.state": models.SimpleType{Kind: models.String}}}}

// WorkitemtypeController implements the workitemtype resource.
type WorkitemtypeController struct {
	*goa.Controller
}

// NewWorkitemtypeController creates a workitemtype controller.
func NewWorkitemtypeController(service *goa.Service) *WorkitemtypeController {
	return &WorkitemtypeController{Controller: service.NewController("WorkitemtypeController")}
}

// Show runs the show action.
func (c *WorkitemtypeController) Show(ctx *app.ShowWorkitemtypeContext) error {
	res := loadTypeFromDB(ctx.ID)
	if res != nil {
		converted:=convertTypeFromModels(*res)
		return ctx.OK(&converted)
	}
	return ctx.NotFound()
}

func loadTypeFromDB(id string) *models.WorkItemType {
	return wellKnown[id]
}

func convertTypeFromModels(t models.WorkItemType) app.WorkItemType {
	var converted= app.WorkItemType{
		ID: strconv.FormatUint(t.Id, 10),
		Name: t.Name,
		Version: 0,
	}
	return converted
}
