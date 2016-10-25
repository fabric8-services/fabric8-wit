package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/search"
	"github.com/almighty/almighty-core/transaction"
	"github.com/goadesign/goa"
)

// SearchController implements the search resource.
type SearchController struct {
	*goa.Controller
	sRepository search.Repository
	ts          transaction.Support
}

// NewSearchController creates a search controller.
func NewSearchController(service *goa.Service, sRepository search.Repository, ts transaction.Support) *SearchController {
	return &SearchController{Controller: service.NewController("SearchController"), sRepository: sRepository, ts: ts}
}

// Show runs the show action.
func (c *SearchController) Show(ctx *app.ShowSearchContext) error {
	return transaction.Do(c.ts, func() error {
		t, err := c.sRepository.SearchFullText(ctx.Context, ctx.Q)
		if err != nil {
			return err
		}
		return ctx.OK(t)
	})
}
