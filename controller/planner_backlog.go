package controller

import (
	"net/http"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// PlannerBacklogController implements the planner_backlog resource.
type PlannerBacklogController struct {
	*goa.Controller
	db application.DB
}

// NewPlannerBacklogController creates a planner_backlog controller.
func NewPlannerBacklogController(service *goa.Service, db application.DB) *PlannerBacklogController {
	return &PlannerBacklogController{Controller: service.NewController("PlannerBacklogController"), db: db}
}

func (c *PlannerBacklogController) List(ctx *app.ListPlannerBacklogContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	start, limit, err := parseLimit(ctx.Page)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "could not parse paging"))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.WorkItems().Backlog(ctx.Context, spaceID, start, &limit)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "error listing backlog items"))
		}

		lastMod := findLastModified(result)

		if ifMod, ok := ctx.RequestData.Header["If-Modified-Since"]; ok {
			ifModSince, err := http.ParseTime(ifMod[0])
			if err == nil {
				if lastMod.Before(ifModSince) || lastMod.Equal(ifModSince) {
					return ctx.NotModified()
				}
			}
		}
		response := app.WorkItem2List{
			Data: ConvertWorkItems(ctx.RequestData, result),
		}

		ctx.ResponseData.Header().Set("Last-Modified", lastModifiedTime(lastMod))
		return ctx.OK(&response)
	})
}
