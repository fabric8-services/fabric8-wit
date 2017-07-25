package controller

import (
	"fmt"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/workitem/typegroup"
	"github.com/goadesign/goa"
)

// WorkItemTypeGroupController implements the work_item_type_group resource.
type WorkItemTypeGroupController struct {
	*goa.Controller
	db application.DB
}

// NewWorkItemTypeGroupController creates a work_item_type_group controller.
func NewWorkItemTypeGroupController(service *goa.Service, db application.DB) *WorkItemTypeGroupController {
	return &WorkItemTypeGroupController{
		Controller: service.NewController("WorkItemTypeGroupController"),
		db:         db,
	}
}

// List runs the list action.
func (c *WorkItemTypeGroupController) List(ctx *app.ListWorkItemTypeGroupContext) error {
	// WorkItemTypeGroupController_List: start_implement

	// Put your logic here

	// WorkItemTypeGroupController_List: end_implement
	res := &app.WorkItemTypeGroupSigleSingle{}
	res.Data = &app.WorkItemTypeGroupData{
		Attributes: &app.WorkItemTypeGroupAttributes{
			Hierarchy: []*app.WorkItemTypeGroup{
				ConvertTypeGroup(ctx.RequestData, typegroup.Portfolio0),
				ConvertTypeGroup(ctx.RequestData, typegroup.Portfolio1),
				ConvertTypeGroup(ctx.RequestData, typegroup.Requirements0),
			},
		},
		Included: []*app.WorkItemTypeData{},
	}
	IncludeWorkItemType(c, ctx, res.Data)
	return ctx.OK(res)
}

func IncludeWorkItemType(c *WorkItemTypeGroupController, ctx *app.ListWorkItemTypeGroupContext, res *app.WorkItemTypeGroupData) {
	// witr := workitem.NewWorkItemTypeRepository()
	for _, node := range res.Attributes.Hierarchy {
		for _, witID := range node.WitCollection {
			err := application.Transactional(c.db, func(appl application.Application) error {
				t, err := appl.WorkItemTypes().LoadByID(ctx, witID)
				if err != nil {
					return err
				}
				converted := ConvertWorkItemTypeFromModel(ctx.RequestData, t)
				res.Included = append(res.Included, &converted)
				return nil
			})
			if err != nil {
				fmt.Println("logging err")
			}
		}
	}
}

func ConvertTypeGroup(request *goa.RequestData, tg typegroup.WorkItemTypeGroup) *app.WorkItemTypeGroup {
	return &app.WorkItemTypeGroup{
		Group:         tg.Group,
		Level:         tg.Level,
		Name:          tg.Name,
		Sublevel:      tg.Sublevel,
		WitCollection: tg.WorkItemTypeCollection,
	}
}
