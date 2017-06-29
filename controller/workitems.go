package controller

import (
	"fmt"
	"strconv"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	query "github.com/fabric8-services/fabric8-wit/query/simple"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space/authz"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// WorkitemsController implements the workitems resource.
type WorkitemsController struct {
	*goa.Controller
	db     application.DB
	config WorkItemControllerConfig
}

// NewWorkitemsController creates a workitems controller.
func NewWorkitemsController(service *goa.Service, db application.DB, config WorkItemControllerConfig) *WorkitemsController {
	return &WorkitemsController{
		Controller: service.NewController("WorkitemsController"),
		db:         db,
		config:     config}
}

// Create does POST workitem
func (c *WorkitemsController) Create(ctx *app.CreateWorkitemsContext) error {
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	var wit *uuid.UUID
	if ctx.Payload.Data != nil && ctx.Payload.Data.Relationships != nil &&
		ctx.Payload.Data.Relationships.BaseType != nil && ctx.Payload.Data.Relationships.BaseType.Data != nil {
		wit = &ctx.Payload.Data.Relationships.BaseType.Data.ID
	}
	if wit == nil { // TODO Figure out path source etc. Should be a required relation
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("Data.Relationships.BaseType.Data.ID", err))
	}

	// Set the space to the Payload
	if ctx.Payload.Data != nil && ctx.Payload.Data.Relationships != nil {
		// We overwrite or use the space ID in the URL to set the space of this WI
		spaceSelfURL := rest.AbsoluteURL(goa.ContextRequest(ctx), app.SpaceHref(ctx.SpaceID.String()))
		ctx.Payload.Data.Relationships.Space = app.NewSpaceRelation(ctx.SpaceID, spaceSelfURL)
	}
	wi := workitem.WorkItem{
		Fields: make(map[string]interface{}),
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		//verify spaceID:
		// To be removed once we have endpoint like - /api/space/{spaceID}/workitems
		_, spaceLoadErr := appl.Spaces().Load(ctx, ctx.SpaceID)
		if spaceLoadErr != nil {
			return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("space", "string").Expected("valid space ID"))
		}

		err := ConvertJSONAPIToWorkItem(ctx, appl, *ctx.Payload.Data, &wi, ctx.SpaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, fmt.Sprintf("Error creating work item")))
		}

		wi, err := appl.WorkItems().Create(ctx, ctx.SpaceID, *wit, wi.Fields, *currentUserIdentityID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, fmt.Sprintf("Error creating work item")))
		}
		hasChildren := workItemIncludeHasChildren(appl, ctx)
		wi2 := ConvertWorkItem(ctx.RequestData, *wi, hasChildren)
		resp := &app.WorkItemSingle{
			Data: wi2,
			Links: &app.WorkItemLinks{
				Self: buildAbsoluteURL(ctx.RequestData),
			},
		}
		ctx.ResponseData.Header().Set("Last-Modified", lastModified(*wi))
		ctx.ResponseData.Header().Set("Location", app.WorkitemHref(wi2.ID))
		return ctx.Created(resp)
	})
}

// List runs the list action.
// Prev and Next links will be present only when there actually IS a next or previous page.
// Last will always be present. Total Item count needs to be computed from the "Last" link.
func (c *WorkitemsController) List(ctx *app.ListWorkitemsContext) error {
	var additionalQuery []string
	exp, err := query.Parse(ctx.Filter)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("could not parse filter", err))
	}
	if ctx.FilterAssignee != nil {
		if *ctx.FilterAssignee == none {
			exp = criteria.And(exp, criteria.IsNull("system.assignees"))
			additionalQuery = append(additionalQuery, "filter[assignee]=none")

		} else {
			exp = criteria.And(exp, criteria.Equals(criteria.Field("system.assignees"), criteria.Literal([]string{*ctx.FilterAssignee})))
			additionalQuery = append(additionalQuery, "filter[assignee]="+*ctx.FilterAssignee)
		}
	}
	if ctx.FilterIteration != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field(workitem.SystemIteration), criteria.Literal(string(*ctx.FilterIteration))))
		additionalQuery = append(additionalQuery, "filter[iteration]="+*ctx.FilterIteration)
		// Update filter by adding child iterations if any
		application.Transactional(c.db, func(tx application.Application) error {
			iterationUUID, errConversion := uuid.FromString(*ctx.FilterIteration)
			if errConversion != nil {
				return jsonapi.JSONErrorResponse(ctx, errs.Wrap(errConversion, "Invalid iteration ID"))
			}
			childrens, err := tx.Iterations().LoadChildren(ctx.Context, iterationUUID)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Unable to fetch children"))
			}
			for _, child := range childrens {
				childIDStr := child.ID.String()
				exp = criteria.Or(exp, criteria.Equals(criteria.Field(workitem.SystemIteration), criteria.Literal(childIDStr)))
				additionalQuery = append(additionalQuery, "filter[iteration]="+childIDStr)
			}
			return nil
		})
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
	if ctx.FilterParentexists != nil {
		// no need to build expression: it is taken care in wi.List call
		// we need additionalQuery to make sticky filters in URL links
		additionalQuery = append(additionalQuery, "filter[parentexists]="+strconv.FormatBool(*ctx.FilterParentexists))
	}

	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)
	return application.Transactional(c.db, func(tx application.Application) error {
		workitems, tc, err := tx.WorkItems().List(ctx.Context, ctx.SpaceID, exp, ctx.FilterParentexists, &offset, &limit)
		count := int(tc)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Error listing work items"))
		}
		return ctx.ConditionalEntities(workitems, c.config.GetCacheControlWorkItems, func() error {
			hasChildren := workItemIncludeHasChildren(tx, ctx)
			response := app.WorkItemList{
				Links: &app.PagingLinks{},
				Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
				Data:  ConvertWorkItems(ctx.RequestData, workitems, hasChildren),
			}
			setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(workitems), offset, limit, count, additionalQuery...)
			addFilterLinks(response.Links, ctx.RequestData)
			return ctx.OK(&response)
		})

	})
}

// Reorder does PATCH workitem
func (c *WorkitemsController) Reorder(ctx *app.ReorderWorkitemsContext) error {
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	authorized, err := authz.Authorize(ctx, ctx.SpaceID.String())
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	if !authorized {
		return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not authorized to access the space"))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		var dataArray []*app.WorkItem
		if ctx.Payload == nil || ctx.Payload.Data == nil || ctx.Payload.Position == nil {
			return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("missing payload element in request", nil))
		}

		// Reorder workitems in the array one by one
		for i := 0; i < len(ctx.Payload.Data); i++ {
			wi, err := appl.WorkItems().LoadByID(ctx, *ctx.Payload.Data[i].ID)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "failed to reorder work item"))
			}

			err = ConvertJSONAPIToWorkItem(ctx, appl, *ctx.Payload.Data[i], wi, ctx.SpaceID)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "failed to reorder work item"))
			}
			wi, err = appl.WorkItems().Reorder(ctx, workitem.DirectionType(ctx.Payload.Position.Direction), ctx.Payload.Position.ID, *wi, *currentUserIdentityID)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, err)
			}
			hasChildren := workItemIncludeHasChildren(appl, ctx)
			wi2 := ConvertWorkItem(ctx.RequestData, *wi, hasChildren)
			dataArray = append(dataArray, wi2)
		}
		log.Debug(ctx, nil, "Reordered items: %d", len(dataArray))
		resp := &app.WorkItemReorder{
			Data: dataArray,
		}

		return ctx.OK(resp)
	})
}
