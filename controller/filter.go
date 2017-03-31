package controller

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/rest"
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
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Workitem state",
				Query:       "filter[workitemstate]={id}",
				Description: "Filter by workitemstate",
				Type:        "workitemstate",
			},
			Type: "filters",
		},
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Category",
				Query:       "filter[category]={id}",
				Description: "Filter by category",
				Type:        "category",
			},
			Type: "filters",
		},
	)
	result := &app.FilterList{
		Data: arr,
	}

	return ctx.OK(result)
}

func addFilterLinks(links *app.PagingLinks, request *goa.RequestData) {
	filter := rest.AbsoluteURL(request, app.FilterHref())
	links.Filters = &filter
}
