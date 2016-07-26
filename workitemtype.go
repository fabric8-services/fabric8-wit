package main

import (
	"fmt"
	"strconv"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
	"github.com/goadesign/goa"
)

var wellKnown = map[string]*models.WorkItemType{
	"1": &models.WorkItemType{
		Id:   1,
		Name: "system.workitem",
		Fields: map[string]models.FieldDefinition{
			"system.owner": models.FieldDefinition{Type: models.SimpleType{Kind: models.UserKind}, Required: true},
			"system.state": models.FieldDefinition{Type: models.SimpleType{Kind: models.StringKind}, Required: true},
		}}}

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
	res, _ := loadTypeFromDB(ctx.ID)
	if res != nil {
		converted := convertTypeFromModels(*res)
		return ctx.OK(&converted)
	}
	return ctx.NotFound()
}

func loadTypeFromDB(id string) (*models.WorkItemType, error) {
	if wellKnown[id] == nil {
		return nil, fmt.Errorf("Work item type not found: %s", id)
	}
	return wellKnown[id], nil
}

func convertTypeFromModels(t models.WorkItemType) app.WorkItemType {
	var converted = app.WorkItemType{
		ID:      strconv.FormatUint(t.Id, 10),
		Name:    t.Name,
		Version: 0,
		Fields:  map[string]*app.FieldDefinition{},
	}
	for name, def := range t.Fields {
		converted.Fields[name] = &app.FieldDefinition{
			Required: def.Required,
			Type:     def.Type,
		}
	}
	return converted
}
