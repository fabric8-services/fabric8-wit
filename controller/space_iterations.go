package controller

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// SpaceIterationsController implements the space-iterations resource.
type SpaceIterationsController struct {
	*goa.Controller
	db application.DB
}

// NewSpaceIterationsController creates a space-iterations controller.
func NewSpaceIterationsController(service *goa.Service, db application.DB) *SpaceIterationsController {
	return &SpaceIterationsController{Controller: service.NewController("SpaceIterationsController"), db: db}
}

// List runs the list action.
func (c *SpaceIterationsController) List(ctx *app.ListSpaceIterationsContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {

		_, err = appl.Spaces().Load(ctx, spaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		iterations, err := appl.Iterations().List(ctx, spaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
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
