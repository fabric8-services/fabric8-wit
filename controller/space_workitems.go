package controller

import (
	"fmt"
	"net/http"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	query "github.com/almighty/almighty-core/query/simple"
	"github.com/almighty/almighty-core/space"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// SpaceWorkitemsController implements the space_workitems resource.
type SpaceWorkitemsController struct {
	*goa.Controller
	db application.DB
}

// NewSpaceWorkitemsController creates a space_workitems controller.
func NewSpaceWorkitemsController(service *goa.Service, db application.DB) *SpaceWorkitemsController {
	return &SpaceWorkitemsController{Controller: service.NewController("SpaceWorkitemsController"), db: db}
}

// Create runs the create action.
func (c *SpaceWorkitemsController) Create(ctx *app.CreateSpaceWorkitemsContext) error {
	//TODO: It confusing if whether the Space data will come from the ctx or payload
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
		return ctx.Unauthorized(jerrors)
	}
	var wit *uuid.UUID
	if ctx.Payload.Data != nil && ctx.Payload.Data.Relationships != nil &&
		ctx.Payload.Data.Relationships.BaseType != nil && ctx.Payload.Data.Relationships.BaseType.Data != nil {
		wit = &ctx.Payload.Data.Relationships.BaseType.Data.ID
	}
	if wit == nil { // TODO Figure out path source etc. Should be a required relation
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("Data.Relationships.BaseType.Data.ID", err))
	}
	wi := app.WorkItem{
		Fields: make(map[string]interface{}),
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		//verify spaceID:
		// To be removed once we have endpoint like - /api/space/{spaceID}/workitems
		if spaceID != space.SystemSpace {
			_, spaceLoadErr := appl.Spaces().Load(ctx, spaceID)
			if spaceLoadErr != nil {
				return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("space", "string").Expected("valid space ID"))
			}
		}
		err := ConvertJSONAPIToWorkItem(appl, *ctx.Payload.Data, &wi)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, fmt.Sprintf("Error creating work item")))
		}
		wi, err := appl.WorkItems().Create(ctx, spaceID, *wit, wi.Fields, *currentUserIdentityID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, fmt.Sprintf("Error creating work item")))
		}
		wi2 := ConvertWorkItem(ctx.RequestData, wi)
		resp := &app.WorkItem2Single{
			Data: wi2,
			Links: &app.WorkItemLinks{
				Self: buildAbsoluteURL(ctx.RequestData),
			},
		}
		ctx.ResponseData.Header().Set("Last-Modified", lastModified(wi))
		ctx.ResponseData.Header().Set("Location", app.WorkitemHref(wi2.ID))
		return ctx.Created(resp)
	})
}

// List runs the list action.
func (c *SpaceWorkitemsController) List(ctx *app.ListSpaceWorkitemsContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	var additionalQuery []string
	exp, err := query.Parse(ctx.Filter)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("could not parse filter", err))
	}
	if ctx.FilterAssignee != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field("system.assignees"), criteria.Literal([]string{*ctx.FilterAssignee})))
		additionalQuery = append(additionalQuery, "filter[assignee]="+*ctx.FilterAssignee)
	}
	if ctx.FilterIteration != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field(workitem.SystemIteration), criteria.Literal(string(*ctx.FilterIteration))))
		additionalQuery = append(additionalQuery, "filter[iteration]="+*ctx.FilterIteration)
	}
	if ctx.FilterWorkitemtype != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field("Type"), criteria.Literal([]uuid.UUID{*ctx.FilterWorkitemtype})))
		additionalQuery = append(additionalQuery, "filter[workitemtype]="+ctx.FilterWorkitemtype.String())
	}
	if ctx.FilterArea != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field(workitem.SystemArea), criteria.Literal(string(*ctx.FilterArea))))
		additionalQuery = append(additionalQuery, "filter[area]="+*ctx.FilterArea)
	}
	if ctx.FilterWorkitemstate != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field(workitem.SystemState), criteria.Literal(string(*ctx.FilterWorkitemstate))))
		additionalQuery = append(additionalQuery, "filter[workitemstate]="+*ctx.FilterWorkitemstate)
	}

	offset, limit := computePagingLimts(ctx.PageOffset, ctx.PageLimit)
	return application.Transactional(c.db, func(tx application.Application) error {
		result, tc, err := tx.WorkItems().List(ctx.Context, spaceID, exp, &offset, &limit)
		count := int(tc)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Error listing work items"))
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
			Links: &app.PagingLinks{},
			Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
			Data:  ConvertWorkItems(ctx.RequestData, result),
		}
		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(result), offset, limit, count, additionalQuery...)
		addFilterLinks(response.Links, ctx.RequestData)

		ctx.ResponseData.Header().Set("Last-Modified", lastModifiedTime(lastMod))
		return ctx.OK(&response)
	})

}
