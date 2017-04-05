package controller

import (
	"fmt"
	"html"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/codebase"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	query "github.com/almighty/almighty-core/query/simple"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// Defines the constants to be used in json api "type" attribute
const (
	APIStringTypeUser         = "identities"
	APIStringTypeWorkItem     = "workitems"
	APIStringTypeWorkItemType = "workitemtypes"
)

// WorkitemController implements the workitem resource.
type WorkitemController struct {
	*goa.Controller
	db     application.DB
	config WorkItemControllerConfig
}

// WorkItemControllerConfig the config interface for the WorkitemController
type WorkItemControllerConfig interface {
	GetCacheControlWorkItems() string
}

// NewWorkitemController creates a workitem controller.
func NewWorkitemController(service *goa.Service, db application.DB, config WorkItemControllerConfig) *WorkitemController {
	if db == nil {
		panic("db must not be nil")
	}
	return &WorkitemController{
		Controller: service.NewController("WorkitemController"),
		db:         db,
		config:     config}
}

// List runs the list action.
// Prev and Next links will be present only when there actually IS a next or previous page.
// Last will always be present. Total Item count needs to be computed from the "Last" link.
func (c *WorkitemController) List(ctx *app.ListWorkitemContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return errors.NewNotFoundError("spaceID", ctx.ID)
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

	offset, limit := computePagingLimts(ctx.PageOffset, ctx.PageLimit)
	return application.Transactional(c.db, func(tx application.Application) error {
		workitems, tc, err := tx.WorkItems().List(ctx.Context, spaceID, exp, &offset, &limit)
		count := int(tc)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Error listing work items"))
		}
		return ctx.ConditionalEntities(workitems, c.config.GetCacheControlWorkItems, func() error {
			response := app.WorkItemList{
				Links: &app.PagingLinks{},
				Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
				Data:  ConvertWorkItems(ctx.RequestData, workitems),
			}
			setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(workitems), offset, limit, count, additionalQuery...)
			addFilterLinks(response.Links, ctx.RequestData)
			return ctx.OK(&response)
		})

	})
}

// Update does PATCH workitem
func (c *WorkitemController) Update(ctx *app.UpdateWorkitemContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return errors.NewNotFoundError("spaceID", ctx.ID)
	}
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
		return ctx.Unauthorized(jerrors)
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		if ctx.Payload == nil || ctx.Payload.Data == nil || ctx.Payload.Data.ID == nil {
			return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("missing data.ID element in request", nil))
		}
		wi, err := appl.WorkItems().Load(ctx, spaceID, *ctx.Payload.Data.ID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, fmt.Sprintf("Failed to load work item with id %v", *ctx.Payload.Data.ID)))
		}
		// Type changes of WI are not allowed which is why we overwrite it the
		// type with the old one after the WI has been converted.
		oldType := wi.Type
		err = ConvertJSONAPIToWorkItem(appl, *ctx.Payload.Data, wi)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		wi.Type = oldType
		wi, err = appl.WorkItems().Save(ctx, spaceID, *wi, *currentUserIdentityID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Error updating work item"))
		}
		wi2 := ConvertWorkItem(ctx.RequestData, *wi)
		resp := &app.WorkItemSingle{
			Data: wi2,
			Links: &app.WorkItemLinks{
				Self: buildAbsoluteURL(ctx.RequestData),
			},
		}

		ctx.ResponseData.Header().Set("Last-Modified", lastModified(*wi))
		return ctx.OK(resp)
	})
}

// Reorder does PATCH workitem
func (c *WorkitemController) Reorder(ctx *app.ReorderWorkitemContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return errors.NewNotFoundError("spaceID", ctx.ID)
	}

	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		var dataArray []*app.WorkItem
		if ctx.Payload == nil || ctx.Payload.Data == nil || ctx.Payload.Position == nil {
			return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("missing payload element in request", nil))
		}

		// Reorder workitems in the array one by one
		for i := 0; i < len(ctx.Payload.Data); i++ {
			wi, err := appl.WorkItems().Load(ctx, spaceID, *ctx.Payload.Data[i].ID)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "failed to reorder work item"))
			}

			err = ConvertJSONAPIToWorkItem(appl, *ctx.Payload.Data[i], wi)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "failed to reorder work item"))
			}
			wi, err = appl.WorkItems().Reorder(ctx, workitem.DirectionType(ctx.Payload.Position.Direction), ctx.Payload.Position.ID, *wi, *currentUserIdentityID)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, err)
			}
			wi2 := ConvertWorkItem(ctx.RequestData, *wi)
			dataArray = append(dataArray, wi2)
		}
		resp := &app.WorkItemReorder{
			Data: dataArray,
		}

		return ctx.OK(resp)
	})
}

// Create does POST workitem
func (c *WorkitemController) Create(ctx *app.CreateWorkitemContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return errors.NewNotFoundError("spaceID", ctx.ID)
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

	// Set the space to the Payload
	if ctx.Payload.Data != nil && ctx.Payload.Data.Relationships != nil {
		// We overwrite or use the space ID in the URL to set the space of this WI
		spaceSelfURL := rest.AbsoluteURL(goa.ContextRequest(ctx), app.SpaceHref(spaceID.String()))
		ctx.Payload.Data.Relationships.Space = app.NewSpaceRelation(spaceID, spaceSelfURL)
	}
	wi := workitem.WorkItem{
		Fields: make(map[string]interface{}),
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		//verify spaceID:
		// To be removed once we have endpoint like - /api/space/{spaceID}/workitems
		spaceInstance, spaceLoadErr := appl.Spaces().Load(ctx, spaceID)
		if spaceLoadErr != nil {
			return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("space", "string").Expected("valid space ID"))
		}
		err := ConvertJSONAPIToWorkItem(appl, *ctx.Payload.Data, &wi)
		// fetch root iteration for this space and assign it to WI if not present already
		if _, ok := wi.Fields[workitem.SystemIteration]; ok == false {
			// no iteration set hence set to root iteration of its space
			rootItr, rootItrErr := appl.Iterations().Root(ctx, spaceInstance.ID)
			if rootItrErr == nil {
				wi.Fields[workitem.SystemIteration] = rootItr.ID.String()
			}
		}
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, fmt.Sprintf("Error creating work item")))
		}
		wi, err := appl.WorkItems().Create(ctx, spaceID, *wit, wi.Fields, *currentUserIdentityID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, fmt.Sprintf("Error creating work item")))
		}
		wi2 := ConvertWorkItem(ctx.RequestData, *wi)
		resp := &app.WorkItemSingle{
			Data: wi2,
			Links: &app.WorkItemLinks{
				Self: buildAbsoluteURL(ctx.RequestData),
			},
		}
		ctx.ResponseData.Header().Set("Last-Modified", lastModified(*wi))
		ctx.ResponseData.Header().Set("Location", app.WorkitemHref(wi2.Relationships.Space.Data.ID.String(), wi2.ID))
		return ctx.Created(resp)
	})
}

// Show does GET workitem
func (c *WorkitemController) Show(ctx *app.ShowWorkitemContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return errors.NewNotFoundError("spaceID", ctx.ID)
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		comments := WorkItemIncludeCommentsAndTotal(ctx, c.db, ctx.WiID)
		wi, err := appl.WorkItems().Load(ctx, spaceID, ctx.WiID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, fmt.Sprintf("Fail to load work item with id %v", ctx.WiID)))
		}
		return ctx.ConditionalEntity(*wi, c.config.GetCacheControlWorkItems, func() error {
			wi2 := ConvertWorkItem(ctx.RequestData, *wi, comments)
			resp := &app.WorkItemSingle{
				Data: wi2,
			}
			return ctx.OK(resp)

		})
	})
}

// Delete does DELETE workitem
func (c *WorkitemController) Delete(ctx *app.DeleteWorkitemContext) error {

	// Temporarly disabled, See https://github.com/almighty/almighty-core/issues/1036
	if true {
		return ctx.MethodNotAllowed()
	}
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return errors.NewNotFoundError("spaceID", ctx.ID)
	}
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
		return ctx.Unauthorized(jerrors)
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		err := appl.WorkItems().Delete(ctx, spaceID, ctx.WiID, *currentUserIdentityID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err, "error deleting work item %s", ctx.WiID))
		}
		if err := appl.WorkItemLinks().DeleteRelatedLinks(ctx, ctx.WiID, *currentUserIdentityID); err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err, "failed to delete work item links related to work item %s", ctx.WiID))
		}
		return ctx.OK([]byte{})
	})
}

// Time is default value if no UpdatedAt field is found
func updatedAt(wi workitem.WorkItem) time.Time {
	var t time.Time
	if ua, ok := wi.Fields[workitem.SystemUpdatedAt]; ok {
		t = ua.(time.Time)
	}
	return t.Truncate(time.Second)
}

func lastModified(wi workitem.WorkItem) string {
	return lastModifiedTime(updatedAt(wi))
}

func lastModifiedTime(t time.Time) string {
	return t.Format(time.RFC1123)
}

func findLastModified(wis []workitem.WorkItem) time.Time {
	var t time.Time
	for _, wi := range wis {
		lm := updatedAt(wi)
		if lm.After(t) {
			t = lm
		}
	}
	return t
}

// ConvertJSONAPIToWorkItem is responsible for converting given WorkItem model object into a
// response resource object by jsonapi.org specifications
func ConvertJSONAPIToWorkItem(appl application.Application, source app.WorkItem, target *workitem.WorkItem) error {
	// construct default values from input WI
	version, err := getVersion(source.Attributes["version"])
	if err != nil {
		return err
	}
	target.Version = version

	if source.Relationships != nil && source.Relationships.Assignees != nil {
		if source.Relationships.Assignees.Data == nil {
			delete(target.Fields, workitem.SystemAssignees)
		} else {
			var ids []string
			for _, d := range source.Relationships.Assignees.Data {
				assigneeUUID, err := uuid.FromString(*d.ID)
				if err != nil {
					return errors.NewBadParameterError("data.relationships.assignees.data.id", *d.ID)
				}
				if ok := appl.Identities().IsValid(context.Background(), assigneeUUID); !ok {
					return errors.NewBadParameterError("data.relationships.assignees.data.id", *d.ID)
				}
				ids = append(ids, assigneeUUID.String())
			}
			target.Fields[workitem.SystemAssignees] = ids
		}
	}
	if source.Relationships != nil && source.Relationships.Iteration != nil {
		if source.Relationships.Iteration.Data == nil {
			delete(target.Fields, workitem.SystemIteration)
		} else {
			d := source.Relationships.Iteration.Data
			iterationUUID, err := uuid.FromString(*d.ID)
			if err != nil {
				return errors.NewBadParameterError("data.relationships.iteration.data.id", *d.ID)
			}
			if _, err = appl.Iterations().Load(context.Background(), iterationUUID); err != nil {
				return errors.NewBadParameterError("data.relationships.iteration.data.id", *d.ID)
			}
			target.Fields[workitem.SystemIteration] = iterationUUID.String()
		}
	}
	if source.Relationships != nil && source.Relationships.Area != nil {
		if source.Relationships.Area.Data == nil {
			delete(target.Fields, workitem.SystemArea)
		} else {
			d := source.Relationships.Area.Data
			areaUUID, err := uuid.FromString(*d.ID)
			if err != nil {
				return errors.NewBadParameterError("data.relationships.area.data.id", *d.ID)
			}
			if _, err = appl.Areas().Load(context.Background(), areaUUID); err != nil {
				return errors.NewBadParameterError("data.relationships.area.data.id", *d.ID)
			}
			target.Fields[workitem.SystemArea] = areaUUID.String()
		}
	}
	if source.Relationships != nil && source.Relationships.BaseType != nil {
		if source.Relationships.BaseType.Data != nil {
			target.Type = source.Relationships.BaseType.Data.ID
		}
	}

	for key, val := range source.Attributes {
		// convert legacy description to markup content
		if key == workitem.SystemDescription {
			if m := rendering.NewMarkupContentFromValue(val); m != nil {
				// if no description existed before, set the new one
				if target.Fields[key] == nil {
					target.Fields[key] = *m
				} else {
					// only update the 'description' field in the existing description
					existingDescription := target.Fields[key].(rendering.MarkupContent)
					existingDescription.Content = (*m).Content
					target.Fields[key] = existingDescription
				}
			}
		} else if key == workitem.SystemDescriptionMarkup {
			markup := val.(string)
			// if no description existed before, set the markup in a new one
			if target.Fields[workitem.SystemDescription] == nil {
				target.Fields[workitem.SystemDescription] = rendering.MarkupContent{Markup: markup}
			} else {
				// only update the 'description' field in the existing description
				existingDescription := target.Fields[workitem.SystemDescription].(rendering.MarkupContent)
				existingDescription.Markup = markup
				target.Fields[workitem.SystemDescription] = existingDescription
			}
		} else if key == workitem.SystemCodebase {
			if m, err := codebase.NewCodebaseContentFromValue(val); err == nil {
				target.Fields[key] = *m
			} else {
				return err
			}
		} else {
			target.Fields[key] = val
		}
	}
	if description, ok := target.Fields[workitem.SystemDescription].(rendering.MarkupContent); ok {
		// verify the description markup
		if !rendering.IsMarkupSupported(description.Markup) {
			return errors.NewBadParameterError("data.relationships.attributes[system.description].markup", description.Markup)
		}
	}
	return nil
}

func getVersion(version interface{}) (int, error) {
	if version != nil {
		v, err := strconv.Atoi(fmt.Sprintf("%v", version))
		if err != nil {
			return -1, errors.NewBadParameterError("data.attributes.version", version)
		}
		return v, nil
	}
	return -1, nil
}

// WorkItemConvertFunc is a open ended function to add additional links/data/relations to a Comment during
// conversion from internal to API
type WorkItemConvertFunc func(*goa.RequestData, *workitem.WorkItem, *app.WorkItem)

// ConvertWorkItems is responsible for converting given []WorkItem model object into a
// response resource object by jsonapi.org specifications
func ConvertWorkItems(request *goa.RequestData, wis []workitem.WorkItem, additional ...WorkItemConvertFunc) []*app.WorkItem {
	ops := []*app.WorkItem{}
	for _, wi := range wis {
		ops = append(ops, ConvertWorkItem(request, wi, additional...))
	}
	return ops
}

// ConvertWorkItem is responsible for converting given WorkItem model object into a
// response resource object by jsonapi.org specifications
func ConvertWorkItem(request *goa.RequestData, wi workitem.WorkItem, additional ...WorkItemConvertFunc) *app.WorkItem {
	// construct default values from input WI
	selfURL := rest.AbsoluteURL(request, app.WorkitemHref(wi.SpaceID.String(), wi.ID))
	sourceLinkTypesURL := rest.AbsoluteURL(request, app.WorkitemtypeHref(wi.SpaceID.String(), wi.Type)+sourceLinkTypesRouteEnd)
	targetLinkTypesURL := rest.AbsoluteURL(request, app.WorkitemtypeHref(wi.SpaceID.String(), wi.Type)+targetLinkTypesRouteEnd)
	spaceSelfURL := rest.AbsoluteURL(request, app.SpaceHref(wi.SpaceID.String()))
	witSelfURL := rest.AbsoluteURL(request, app.WorkitemtypeHref(wi.SpaceID.String(), wi.Type))

	op := &app.WorkItem{
		ID:   &wi.ID,
		Type: APIStringTypeWorkItem,
		Attributes: map[string]interface{}{
			"version": wi.Version,
		},
		Relationships: &app.WorkItemRelationships{
			BaseType: &app.RelationBaseType{
				Data: &app.BaseTypeData{
					ID:   wi.Type,
					Type: APIStringTypeWorkItemType,
				},
				Links: &app.GenericLinks{
					Self: &witSelfURL,
				},
			},
			Space: app.NewSpaceRelation(wi.SpaceID, spaceSelfURL),
		},
		Links: &app.GenericLinksForWorkItem{
			Self:            &selfURL,
			SourceLinkTypes: &sourceLinkTypesURL,
			TargetLinkTypes: &targetLinkTypesURL,
		},
	}

	// Move fields into Relationships or Attributes as needed
	// TODO: Loop based on WorkItemType and match against Field.Type instead of directly to field value
	for name, val := range wi.Fields {
		switch name {
		case workitem.SystemAssignees:
			if val != nil {
				valArr := val.([]interface{})
				op.Relationships.Assignees = &app.RelationGenericList{
					Data: ConvertUsersSimple(request, valArr),
				}
			}
		case workitem.SystemCreator:
			if val != nil {
				valStr := val.(string)
				op.Relationships.Creator = &app.RelationGeneric{
					Data: ConvertUserSimple(request, valStr),
				}
			}
		case workitem.SystemIteration:
			if val != nil {
				valStr := val.(string)
				op.Relationships.Iteration = &app.RelationGeneric{
					Data: ConvertIterationSimple(request, valStr),
				}
			}
		case workitem.SystemArea:
			if val != nil {
				valStr := val.(string)
				op.Relationships.Area = &app.RelationGeneric{
					Data: ConvertAreaSimple(request, valStr),
				}
			}

		case workitem.SystemTitle:
			// 'HTML escape' the title to prevent script injection
			op.Attributes[name] = html.EscapeString(val.(string))
		case workitem.SystemDescription:
			description := rendering.NewMarkupContentFromValue(val)
			if description != nil {
				op.Attributes[name] = (*description).Content
				op.Attributes[workitem.SystemDescriptionMarkup] = (*description).Markup
				// let's include the rendered description while 'HTML escaping' it to prevent script injection
				op.Attributes[workitem.SystemDescriptionRendered] =
					rendering.RenderMarkupToHTML(html.EscapeString((*description).Content), (*description).Markup)
			}
		case workitem.SystemCodebase:
			if val != nil {
				op.Attributes[name] = val
				// TODO: Following format is TBD and hence commented out
				// cb := val.(codebase.CodebaseContent)
				// urlparams := fmt.Sprintf("/codebase/generate?repo=%s&branch=%s&file=%s&line=%d", cb.Repository, cb.Branch, cb.FileName, cb.LineNumber)
				// doitURL := rest.AbsoluteURL(request, url.QueryEscape(urlparams))
				// op.Links.Doit = &doitURL
			}
		default:
			op.Attributes[name] = val
		}
	}
	if op.Relationships.Assignees == nil {
		op.Relationships.Assignees = &app.RelationGenericList{Data: nil}
	}
	if op.Relationships.Iteration == nil {
		op.Relationships.Iteration = &app.RelationGeneric{Data: nil}
	}
	if op.Relationships.Area == nil {
		op.Relationships.Area = &app.RelationGeneric{Data: nil}
	}
	// Always include Comments Link, but optionally use WorkItemIncludeCommentsAndTotal
	WorkItemIncludeComments(request, &wi, op)
	WorkItemIncludeChildren(request, &wi, op)
	for _, add := range additional {
		add(request, &wi, op)
	}
	return op
}

// ListChildren runs the list action.
func (c *WorkitemController) ListChildren(ctx *app.ListChildrenWorkitemContext) error {
	// WorkItemChildrenController_List: start_implement

	// Put your logic here
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.WorkItemLinks().ListWorkItemChildren(ctx, ctx.WiID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		return ctx.ConditionalEntities(result, c.config.GetCacheControlWorkItems, func() error {
			response := app.WorkItemList{
				Data: ConvertWorkItems(ctx.RequestData, result),
			}
			return ctx.OK(&response)
		})
	})
}

// WorkItemIncludeChildren adds relationship about children to workitem (include totalCount)
func WorkItemIncludeChildren(request *goa.RequestData, wi *workitem.WorkItem, wi2 *app.WorkItem) {
	childrenRelated := rest.AbsoluteURL(request, app.WorkitemHref(wi.SpaceID, wi.ID)) + "/children"
	wi2.Relationships.Children = &app.RelationGeneric{
		Links: &app.GenericLinks{
			Related: &childrenRelated,
		},
	}

}
