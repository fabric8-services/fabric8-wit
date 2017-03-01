package controller

import (
	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/goadesign/goa"
)

// FilterController implements the filter resource.
type FilterController struct {
	*goa.Controller
}

// NewFilterController creates a filter controller.
func NewFilterController(service *goa.Service) *FilterController {
	return &FilterController{Controller: service.NewController("FilterController")}
}

// List runs the list action.
func (c *FilterController) List(ctx *app.ListFilterContext) error {
	var arr []*app.Filters
	arr = append(arr, &app.Filters{
		Attributes: &app.FilterAttributes{
			Title:       "Assignee",
			Query:       "filter[assignee]={id}",
			Description: "Filter by assignee",
			Type:        "users",
		},
		Type: "filters",
	},
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Area",
				Query:       "filter[area]={id}",
				Description: "Filter by area",
				Type:        "areas",
			},
			Type: "filters",
		},
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Iteration",
				Query:       "filter[iteration]={id}",
				Description: "Filter by iteration",
				Type:        "iterations",
			},
			Type: "filters",
		},
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Workitem type",
				Query:       "filter[workitemtype]={id}",
				Description: "Filter by workitemtype",
				Type:        "workitemtypes",
			},
			Type: "filters",
		},
	)
	result := &app.FilterList{
		Data: arr,
	}

	return ctx.OK(result)
}

func addFilterLinks(links *app.PagingLinks, path string) {
	filter := fmt.Sprintf("/api/filters")
	links.Filters = &filter
}
