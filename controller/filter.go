package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/goadesign/goa"
)

// FilterController implements the filter resource.
type FilterController struct {
	*goa.Controller
	config FilterControllerConfiguration
}

// FilterControllerConfiguration the configuration for the FilterController.
type FilterControllerConfiguration interface {
	GetCacheControlFilters() string
}

// NewFilterController creates a filter controller.
func NewFilterController(service *goa.Service, config FilterControllerConfiguration) *FilterController {
	return &FilterController{
		Controller: service.NewController("FilterController"),
		config:     config,
	}
}

const (
	filterKeyAssignee     = "assignee"
	filterKeyCreator      = "creator"
	filterKeyArea         = "area"
	filterKeyIteration    = "iteration"
	filterKeyWorkItemType = "workitemtype"
	filterKeyState        = "state"
	filterKeyLabel        = "label"
	filterKeyTitle        = "title"
)

// List runs the list action.
func (c *FilterController) List(ctx *app.ListFilterContext) error {
	var arr []*app.Filters
	arr = append(arr,
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Assignee",
				Description: "Filter by assignee",
				Type:        "users",
				Query:       fmt.Sprintf("filter[%s]={id}", filterKeyAssignee),
				Key:         filterKeyAssignee,
			},
			Type: "filters",
		},
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Creator",
				Description: "Filter by creator",
				Type:        "users",
				Query:       fmt.Sprintf("filter[%s]={id}", filterKeyCreator),
				Key:         filterKeyCreator,
			},
			Type: "filters",
		},
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Area",
				Query:       fmt.Sprintf("filter[%s]={id}", filterKeyArea),
				Key:         filterKeyArea,
				Description: "Filter by area",
				Type:        "areas",
			},
			Type: "filters",
		},
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Iteration",
				Query:       fmt.Sprintf("filter[%s]={id}", filterKeyIteration),
				Key:         filterKeyIteration,
				Description: "Filter by iteration",
				Type:        "iterations",
			},
			Type: "filters",
		},
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Workitem type",
				Query:       fmt.Sprintf("filter[%s]={id}", filterKeyWorkItemType),
				Key:         filterKeyWorkItemType,
				Description: "Filter by workitemtype",
				Type:        "workitemtypes",
			},
			Type: "filters",
		},
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "State",
				Query:       fmt.Sprintf("filter[%s]={id}", filterKeyState),
				Key:         filterKeyState,
				Description: "Filter by state",
				Type:        "state",
			},
			Type: "filters",
		},
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Label",
				Query:       fmt.Sprintf("filter[%s]={id}", filterKeyLabel),
				Key:         filterKeyLabel,
				Description: "Filter by label",
				Type:        "labels",
			},
			Type: "filters",
		},
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Title",
				Query:       fmt.Sprintf("filter[%s]={id}", filterKeyTitle),
				Key:         filterKeyTitle,
				Description: "Filter by title",
				Type:        "title", // not really used anywhere
			},
			Type: "filters",
		},
	)
	result := &app.FilterList{
		Data: arr,
	}
	// compute an ETag based on the type and query of each filter
	filterEtagData := make([]app.ConditionalRequestEntity, len(result.Data))
	for i, filter := range result.Data {
		filterEtagData[i] = FilterEtagData{
			Type:  filter.Attributes.Type,
			Query: filter.Attributes.Query,
		}
	}
	ctx.ResponseData.Header().Set(app.ETag, app.GenerateEntitiesTag(filterEtagData))
	// set now as the last modified date
	ctx.ResponseData.Header().Set(app.LastModified, app.ToHTTPTime(time.Now()))
	// cache-control
	ctx.ResponseData.Header().Set(app.CacheControl, c.config.GetCacheControlFilters())
	return ctx.OK(result)
}

func addFilterLinks(links *app.PagingLinks, request *http.Request) {
	filter := rest.AbsoluteURL(request, app.FilterHref())
	links.Filters = &filter
}

// FilterEtagData structure that carries the data to generate an ETag.
type FilterEtagData struct {
	Type  string
	Query string
}

// GetETagData returns the field values to compute the ETag.
func (f FilterEtagData) GetETagData() []interface{} {
	return []interface{}{f.Type, f.Query}
}

// GetLastModified returns the field values to compute the '`Last-Modified` response header.
func (f FilterEtagData) GetLastModified() time.Time {
	return time.Now()
}
