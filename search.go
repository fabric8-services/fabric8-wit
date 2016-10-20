package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/goadesign/goa"
)

// SearchController implements the search resource.
type SearchController struct {
	*goa.Controller
}

// NewSearchController creates a search controller.
func NewSearchController(service *goa.Service) *SearchController {
	return &SearchController{Controller: service.NewController("SearchController")}
}

// Show runs the show action.
func (c *SearchController) Show(ctx *app.ShowSearchContext) error {
	// SearchController_Show: start_implement

	// Put your logic here

	// SearchController_Show: end_implement
	res := app.WorkItemCollection{}
	return ctx.OK(res)
}
