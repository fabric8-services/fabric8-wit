package controller

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/query"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/search"
	"github.com/goadesign/goa"
)

// QueryController implements the query resource.
type QueryController struct {
	*goa.Controller
	db     application.DB
	config QueryControllerConfiguration
}

// QueryControllerConfiguration the configuration for the LabelController
type QueryControllerConfiguration interface {
	GetCacheControlQueries() string
	GetCacheControlQuery() string
}

// NewQueryController creates a query controller.
func NewQueryController(service *goa.Service, db application.DB, config QueryControllerConfiguration) *QueryController {
	return &QueryController{
		Controller: service.NewController("QueryController"),
		db:         db,
		config:     config,
	}
}

// Create runs the create action.
func (c *QueryController) Create(ctx *app.CreateQueryContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		q := query.Query{
			SpaceID: ctx.SpaceID,
			Fields:  ctx.Payload.Data.Attributes.Fields,
			Title:   strings.TrimSpace(ctx.Payload.Data.Attributes.Title),
		}
		// Parse fields to make sure that query is valid
		_, _, err := search.ParseFilterString(ctx, q.Fields)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		fmt.Println("Done parsing the Q: ", err)
		err = appl.Queries().Create(ctx, &q)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		res := &app.QuerySingle{
			Data: ConvertQuery(appl, ctx.Request, q),
		}
		ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.Request, app.LabelHref(ctx.SpaceID, res.Data.ID)))
		return ctx.Created(res)
	})
}

// ConvertQuery converts from internal to external REST representation
func ConvertQuery(appl application.Application, request *http.Request, q query.Query) *app.Query {
	queryType := query.APIStringTypeQuery
	spaceID := q.SpaceID.String()
	relatedURL := rest.AbsoluteURL(request, app.LabelHref(spaceID, q.ID))
	// spaceRelatedURL := rest.AbsoluteURL(request, app.SpaceHref(spaceID))
	appQuery := &app.Query{
		Type: queryType,
		ID:   &q.ID,
		Attributes: &app.QueryAttributes{
			Title:     q.Title,
			Fields:    q.Fields,
			CreatedAt: &q.CreatedAt,
			UpdatedAt: &q.UpdatedAt,
		},
		Links: &app.GenericLinks{
			Self:    &relatedURL,
			Related: &relatedURL,
		},
	}
	return appQuery
}

// List runs the list action.
func (c *QueryController) List(ctx *app.ListQueryContext) error {
	// QueryController_List: start_implement

	// Put your logic here

	// QueryController_List: end_implement
	res := &app.QueryList{}
	return ctx.OK(res)
}

// Show runs the show action.
func (c *QueryController) Show(ctx *app.ShowQueryContext) error {
	// QueryController_Show: start_implement

	// Put your logic here

	// QueryController_Show: end_implement
	res := &app.QuerySingle{}
	return ctx.OK(res)
}
