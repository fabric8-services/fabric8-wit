package controller

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/errors"
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
		return errors.NewNotFoundError("spaceID", ctx.ID)
	}

	start, limit, err := parseLimit(ctx.Page)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "could not parse paging"))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		iterations, err := appl.Iterations().ListBacklogIterations(ctx.Context, spaceID, start, &limit)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "error listing backlog iterations"))
		}
		itrMap := make(iterationIDMap)
		for _, itr := range iterations {
			itrMap[itr.ID] = itr
		}
		// fetch extra information(counts of WI in each iteration of the space) to be added in response
		wiCounts, err := appl.WorkItems().GetCountsPerIteration(ctx, spaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		res := &app.IterationList{}
		res.Data = ConvertIterations(ctx.RequestData, iterations, updateIterationsWithCounts(wiCounts), parentPathResolver(itrMap))
		return ctx.OK(res)
	})
}
