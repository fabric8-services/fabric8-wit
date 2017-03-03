package controller

import (
	"fmt"
	"html"
	"strconv"

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
	db application.DB
}

// NewWorkitemController creates a workitem controller.
func NewWorkitemController(service *goa.Service, db application.DB) *WorkitemController {
	if db == nil {
		panic("db must not be nil")
	}
	return &WorkitemController{Controller: service.NewController("WorkitemController"), db: db}
}

// List runs the list action.
// Prev and Next links will be present only when there actually IS a next or previous page.
// Last will always be present. Total Item count needs to be computed from the "Last" link.
func (c *WorkitemController) List(ctx *app.ListWorkitemContext) error {
	var additionalQuery []string
	exp, err := query.Parse(ctx.Filter)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("could not parse filter", err))
	}
	if ctx.FilterAssignee != nil {
		assignee := ctx.FilterAssignee
		exp = criteria.And(exp, criteria.Equals(criteria.Field("system.assignees"), criteria.Literal([]string{*assignee})))
		additionalQuery = append(additionalQuery, "filter[assignee]="+*assignee)
	}
	if ctx.FilterIteration != nil {
		iteration := ctx.FilterIteration
		exp = criteria.And(exp, criteria.Equals(criteria.Field(workitem.SystemIteration), criteria.Literal(string(*iteration))))
		additionalQuery = append(additionalQuery, "filter[iteration]="+*iteration)
	}
	if ctx.FilterWorkitemtype != nil {
		wit := ctx.FilterWorkitemtype
		exp = criteria.And(exp, criteria.Equals(criteria.Field("Type"), criteria.Literal([]uuid.UUID{*wit})))
		additionalQuery = append(additionalQuery, "filter[workitemtype]="+wit.String())
	}
	if ctx.FilterArea != nil {
		area := ctx.FilterArea
		exp = criteria.And(exp, criteria.Equals(criteria.Field(workitem.SystemArea), criteria.Literal(string(*area))))
		additionalQuery = append(additionalQuery, "filter[area]="+*area)
	}

	offset, limit := computePagingLimts(ctx.PageOffset, ctx.PageLimit)
	return application.Transactional(c.db, func(tx application.Application) error {
		result, tc, err := tx.WorkItems().List(ctx.Context, exp, &offset, &limit)
		count := int(tc)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Error listing work items"))
		}
		response := app.WorkItem2List{
			Links: &app.PagingLinks{},
			Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
			Data:  ConvertWorkItems(ctx.RequestData, result),
		}
		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(result), offset, limit, count, additionalQuery...)
		addFilterLinks(response.Links, ctx.RequestData)
		return ctx.OK(&response)
	})
}

// Update does PATCH workitem
func (c *WorkitemController) Update(ctx *app.UpdateWorkitemContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		if ctx.Payload == nil || ctx.Payload.Data == nil || ctx.Payload.Data.ID == nil {
			return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("missing data.ID element in request", nil))
		}
		wi, err := appl.WorkItems().Load(ctx, *ctx.Payload.Data.ID)
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
		wi, err = appl.WorkItems().Save(ctx, *wi)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Error updating work item"))
		}
		wi2 := ConvertWorkItem(ctx.RequestData, wi)
		resp := &app.WorkItem2Single{
			Data: wi2,
			Links: &app.WorkItemLinks{
				Self: buildAbsoluteURL(ctx.RequestData),
			},
		}
		return ctx.OK(resp)
	})
}

// Create does POST workitem
func (c *WorkitemController) Create(ctx *app.CreateWorkitemContext) error {
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
		err := ConvertJSONAPIToWorkItem(appl, *ctx.Payload.Data, &wi)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, fmt.Sprintf("Error creating work item")))
		}

		wi, err := appl.WorkItems().Create(ctx, *wit, wi.Fields, *currentUserIdentityID)
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
		ctx.ResponseData.Header().Set("Location", app.WorkitemHref(wi2.ID))
		return ctx.Created(resp)
	})
}

// Show does GET workitem
func (c *WorkitemController) Show(ctx *app.ShowWorkitemContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		comments := WorkItemIncludeCommentsAndTotal(ctx, c.db, ctx.ID)
		wi, err := appl.WorkItems().Load(ctx, ctx.ID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, fmt.Sprintf("Fail to load work item with id %v", ctx.ID)))
		}
		wi2 := ConvertWorkItem(ctx.RequestData, wi, comments)
		resp := &app.WorkItem2Single{
			Data: wi2,
		}
		return ctx.OK(resp)
	})
}

// Delete does DELETE workitem
func (c *WorkitemController) Delete(ctx *app.DeleteWorkitemContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		err := appl.WorkItems().Delete(ctx, ctx.ID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err, "error deleting work item %s", ctx.ID))
		}
		if err := appl.WorkItemLinks().DeleteRelatedLinks(ctx, ctx.ID); err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err, "failed to delete work item links related to work item %s", ctx.ID))
		}
		return ctx.OK([]byte{})
	})
}

// ConvertJSONAPIToWorkItem is responsible for converting given WorkItem model object into a
// response resource object by jsonapi.org specifications
func ConvertJSONAPIToWorkItem(appl application.Application, source app.WorkItem2, target *app.WorkItem) error {
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
				target.Fields[key] = *m
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
type WorkItemConvertFunc func(*goa.RequestData, *app.WorkItem, *app.WorkItem2)

// ConvertWorkItems is responsible for converting given []WorkItem model object into a
// response resource object by jsonapi.org specifications
func ConvertWorkItems(request *goa.RequestData, wis []*app.WorkItem, additional ...WorkItemConvertFunc) []*app.WorkItem2 {
	ops := []*app.WorkItem2{}
	for _, wi := range wis {
		ops = append(ops, ConvertWorkItem(request, wi, additional...))
	}
	return ops
}

// ConvertWorkItem is responsible for converting given WorkItem model object into a
// response resource object by jsonapi.org specifications
func ConvertWorkItem(request *goa.RequestData, wi *app.WorkItem, additional ...WorkItemConvertFunc) *app.WorkItem2 {
	// construct default values from input WI
	selfURL := rest.AbsoluteURL(request, app.WorkitemHref(wi.ID))
	sourceLinkTypesURL := rest.AbsoluteURL(request, app.WorkitemtypeHref(wi.Type)+sourceLinkTypesRouteEnd)
	targetLinkTypesURL := rest.AbsoluteURL(request, app.WorkitemtypeHref(wi.Type)+targetLinkTypesRouteEnd)
	op := &app.WorkItem2{
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
			},
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
	WorkItemIncludeComments(request, wi, op)
	for _, add := range additional {
		add(request, wi, op)
	}
	return op
}
