package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/goadesign/goa"
)

// WorkItemLabelsController implements the work_item_labels resource.
type WorkItemLabelsController struct {
	*goa.Controller
	db     application.DB
	config WorkItemLabelsControllerConfiguration
}

//WorkItemLabelsControllerConfiguration configuration for the WorkItemLabelsController
type WorkItemLabelsControllerConfiguration interface {
	GetCacheControlLabels() string
}

// NewWorkItemLabelsController creates a work_item_labels controller.
func NewWorkItemLabelsController(service *goa.Service, db application.DB, config WorkItemLabelsControllerConfiguration) *WorkItemLabelsController {
	return &WorkItemLabelsController{
		Controller: service.NewController("WorkItemLabelsController"),
		db:         db,
		config:     config,
	}
}

// List runs the list action.
func (c *WorkItemLabelsController) List(ctx *app.ListWorkItemLabelsContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		_, err := appl.WorkItems().LoadByID(ctx, ctx.WiID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		// var ls []*label.Label
		// labelIDs := wi.Fields[workitem.SystemLabels].([]uuid.UUID)
		// for _, lbl := range labelIDs {
		// 	// l, err := appl.Labels().Load(ctx, lbl)
		// 	// ls := append(ls, l)
		// }
		res := &app.LabelList{}
		res.Data = []*app.Label{}
		// res.Meta = &app.CommentListMeta{TotalCount: count}
		// res.Data = ConvertLabels(ctx.Request, ls)
		return ctx.OK(res)
		// return ctx.ConditionalEntities(ls, c.config.GetCacheControlLabels, func() error {
		// res := &app.LabelList{}
		// res.Data = []*app.Label{}
		// res.Meta = &app.CommentListMeta{TotalCount: count}
		// res.Data = ConvertLabels(ctx.Request, comments)
		// return ctx.OK(res)
		// })
	})
}
