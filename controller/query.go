package controller

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/fabric8-services/fabric8-wit/ptr"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/query"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/goadesign/goa"
)

// QueryController implements the query resource.
type QueryController struct {
	*goa.Controller
	db     application.DB
	config QueryControllerConfiguration
}

// QueryControllerConfiguration the configuration for the QueryController
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
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	var q query.Query
	err = application.Transactional(c.db, func(appl application.Application) error {
		err = appl.Spaces().CheckExists(ctx, ctx.SpaceID)
		if err != nil {
			return err
		}
		q = query.Query{
			SpaceID: ctx.SpaceID,
			Fields:  ctx.Payload.Data.Attributes.Fields,
			Title:   strings.TrimSpace(ctx.Payload.Data.Attributes.Title),
			Creator: *currentUserIdentityID,
		}
		err = appl.Queries().Create(ctx, &q)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	res := &app.QuerySingle{
		Data: ConvertQuery(ctx.Request, q),
	}
	ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.Request, app.QueryHref(ctx.SpaceID, res.Data.ID)))
	return ctx.Created(res)
}

// ConvertQuery converts from internal to external REST representation
func ConvertQuery(request *http.Request, q query.Query) *app.Query {
	spaceID := q.SpaceID.String()
	relatedURL := rest.AbsoluteURL(request, app.QueryHref(spaceID, q.ID))
	creatorID := q.Creator.String()
	relatedCreatorLink := rest.AbsoluteURL(request, fmt.Sprintf("%s/%s", usersEndpoint, creatorID))
	spaceRelatedURL := rest.AbsoluteURL(request, app.SpaceHref(spaceID))
	appQuery := &app.Query{
		Type: query.APIStringTypeQuery,
		ID:   &q.ID,
		Attributes: &app.QueryAttributes{
			Title:     q.Title,
			Fields:    q.Fields,
			CreatedAt: &q.CreatedAt,
			Version:   &q.Version,
		},
		Links: &app.GenericLinks{
			Self:    &relatedURL,
			Related: &relatedURL,
		},
		Relationships: &app.QueryRelations{
			Creator: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: ptr.String(APIStringTypeUser),
					ID:   &creatorID,
					Links: &app.GenericLinks{
						Related: &relatedCreatorLink,
					},
				},
			},
			Space: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: &space.SpaceType,
					ID:   &spaceID,
				},
				Links: &app.GenericLinks{
					Self:    &spaceRelatedURL,
					Related: &spaceRelatedURL,
				},
			},
		},
	}
	return appQuery
}

// ConvertQueries from internal to external REST representation
func ConvertQueries(request *http.Request, queries []query.Query) []*app.Query {
	var ls = []*app.Query{}
	for _, q := range queries {
		ls = append(ls, ConvertQuery(request, q))
	}
	return ls
}

// List runs the list action.
func (c *QueryController) List(ctx *app.ListQueryContext) error {
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	var queries []query.Query
	err = application.Transactional(c.db, func(appl application.Application) error {
		err = appl.Spaces().CheckExists(ctx, ctx.SpaceID)
		if err != nil {
			return err
		}
		queries, err = appl.Queries().ListByCreator(ctx, ctx.SpaceID, *currentUserIdentityID)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	res := &app.QueryList{}
	res.Data = ConvertQueries(ctx.Request, queries)
	res.Meta = &app.WorkItemListResponseMeta{
		TotalCount: len(res.Data),
	}
	return ctx.OK(res)
}

// Show runs the show action.
func (c *QueryController) Show(ctx *app.ShowQueryContext) error {
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	var q *query.Query
	err = application.Transactional(c.db, func(appl application.Application) error {
		err := appl.Spaces().CheckExists(ctx, ctx.SpaceID)
		if err != nil {
			return err
		}
		q, err = appl.Queries().Load(ctx, ctx.QueryID, ctx.SpaceID)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	if *currentUserIdentityID != q.Creator {
		log.Warn(ctx, map[string]interface{}{
			"query_id":     ctx.QueryID,
			"creator":      q.Creator,
			"current_user": *currentUserIdentityID,
		}, "user is not the query creator")
		return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not the query creator"))
	}
	res := &app.QuerySingle{
		Data: ConvertQuery(ctx.Request, *q),
	}
	return ctx.OK(res)
}

// Update runs the update action.
func (c *QueryController) Update(ctx *app.UpdateQueryContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	if ctx.Payload.Data.Attributes.Version == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("data.attributes.version", nil).Expected("not nil"))
	}
	var q *query.Query
	err = application.Transactional(c.db, func(appl application.Application) error {
		var err error
		q, err = appl.Queries().Load(ctx.Context, ctx.QueryID, ctx.SpaceID)
		if err != nil {
			return err
		}
		if q.Version != *ctx.Payload.Data.Attributes.Version {
			return errors.NewVersionConflictError("version conflict")
		}
		if strings.TrimSpace(ctx.Payload.Data.Attributes.Title) != "" {
			q.Title = strings.TrimSpace(ctx.Payload.Data.Attributes.Title)
		}
		q, err = appl.Queries().Save(ctx, *q)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	result := &app.QuerySingle{
		Data: ConvertQuery(ctx.Request, *q),
	}
	ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.Request, app.QueryHref(ctx.SpaceID, result.Data.ID)))
	return ctx.OK(result)
}

// Delete runs the delete action.
func (c *QueryController) Delete(ctx *app.DeleteQueryContext) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	err = application.Transactional(c.db, func(appl application.Application) error {
		q, err := appl.Queries().Load(ctx.Context, ctx.QueryID, ctx.SpaceID)
		if err != nil {
			return err
		}
		if q.Creator != *currentUser {
			log.Warn(ctx, map[string]interface{}{
				"query_id":     ctx.QueryID,
				"creator":      q.Creator,
				"current_user": *currentUser,
			}, "user is not the query creator")
			return errors.NewForbiddenError("user is not the query creator")
		}
		return appl.Queries().Delete(ctx.Context, ctx.QueryID)
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.NoContent()
}
