package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/label"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
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
		wi, err := appl.WorkItems().LoadByID(ctx, ctx.WiID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		labelIDs := wi.Fields[workitem.SystemLabels].([]interface{})
		ls := make([]label.Label, 0, len(labelIDs))
		for _, lbl := range labelIDs {
			lblStr := lbl.(string)
			id, err := uuid.FromString(lblStr)
			if err != nil {
				log.Error(nil, map[string]interface{}{
					"label_id": lblStr,
					"err":      err,
				}, "error in converting string to UUID")
				return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
			}
			l, err := appl.Labels().Load(ctx, id)
			if err != nil {
				log.Error(nil, map[string]interface{}{
					"label_id": id,
					"err":      err,
				}, "error in loading label")
				return jsonapi.JSONErrorResponse(ctx, err)
			}
			ls = append(ls, *l)
		}

		return ctx.ConditionalEntities(ls, c.config.GetCacheControlLabels, func() error {
			res := &app.LabelList{}
			res.Data = ConvertLabels(appl, ctx.Request, ls)
			res.Meta = &app.WorkItemListResponseMeta{
				TotalCount: len(res.Data),
			}
			return ctx.OK(res)
		})
	})
}
