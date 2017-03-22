package controller

import (
	"net/http"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/workitem"

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

	offset, limit := computePagingLimts(ctx.PageOffset, ctx.PageLimit)

	exp := criteria.Equals(criteria.Field(workitem.SystemState), criteria.Literal(workitem.SystemStateNew))
	exp = criteria.Or(exp, criteria.Equals(criteria.Field(workitem.SystemState), criteria.Literal(workitem.SystemStateOpen)))
	exp = criteria.Or(exp, criteria.Equals(criteria.Field(workitem.SystemState), criteria.Literal(workitem.SystemStateInProgress)))
	exp = criteria.Or(exp, criteria.Equals(criteria.Field(workitem.SystemState), criteria.Literal(workitem.SystemStateResolved)))

	// Update filter by adding child iterations if any
	err = application.Transactional(c.db, func(appl application.Application) error {
		iterations, err := appl.Iterations().RootIterations(ctx.Context, spaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Unable to fetch children"))
		}

		var expItrs criteria.Expression
		if len(iterations) >= 1 {
			it := iterations[0]
			expItrs = criteria.Equals(criteria.Field(workitem.SystemIteration), criteria.Literal(it.ID.String()))
			for _, itr := range iterations[1:] {
				itrIDStr := itr.ID.String()
				expItrs = criteria.Or(expItrs, criteria.Equals(criteria.Field(workitem.SystemIteration), criteria.Literal(itrIDStr)))
			}
		} else {
			expItrs = criteria.Equals(criteria.Field(workitem.SystemIteration), criteria.Literal(""))
		}
		exp = criteria.And(exp, expItrs)
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		result, tc, err := appl.WorkItems().List(ctx.Context, spaceID, exp, &offset, &limit)
		count := int(tc)
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
			Data:  ConvertWorkItems(ctx.RequestData, result),
			Links: &app.PagingLinks{},
			Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
		}

		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(result), offset, limit, count)

		ctx.ResponseData.Header().Set("Last-Modified", lastModifiedTime(lastMod))
		return ctx.OK(&response)
	})
}
