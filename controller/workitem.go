package controller

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"time"

	"github.com/fabric8-services/fabric8-wit/workitem/link"

	"github.com/fabric8-services/fabric8-wit/ptr"

	"context"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/notification"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space/authz"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// Defines the constants to be used in json api "type" attribute
const (
	APIStringTypeUser         = "identities"
	APIStringTypeWorkItem     = "workitems"
	APIStringTypeWorkItemType = "workitemtypes"
	none                      = "none"
)

// WorkitemController implements the workitem resource.
type WorkitemController struct {
	*goa.Controller
	db           application.DB
	config       WorkItemControllerConfig
	notification notification.Channel
}

// WorkItemControllerConfig the config interface for the WorkitemController
type WorkItemControllerConfig interface {
	GetCacheControlWorkItems() string
	GetCacheControlWorkItem() string
}

// NewWorkitemController creates a workitem controller.
func NewWorkitemController(service *goa.Service, db application.DB, config WorkItemControllerConfig) *WorkitemController {
	return NewNotifyingWorkitemController(service, db, &notification.DevNullChannel{}, config)
}

// NewNotifyingWorkitemController creates a workitem controller with notification broadcast.
func NewNotifyingWorkitemController(service *goa.Service, db application.DB, notificationChannel notification.Channel, config WorkItemControllerConfig) *WorkitemController {
	n := notificationChannel
	if n == nil {
		n = &notification.DevNullChannel{}
	}
	return &WorkitemController{
		Controller:   service.NewController("WorkitemController"),
		db:           db,
		notification: n,
		config:       config}
}

// Returns true if the user is the work item creator or space collaborator
func authorizeWorkitemEditor(ctx context.Context, db application.DB, spaceID uuid.UUID, creatorID string, editorID string) (bool, error) {
	if editorID == creatorID {
		return true, nil
	}
	authorized, err := authz.Authorize(ctx, spaceID.String())
	if err != nil {
		return false, errors.NewUnauthorizedError(err.Error())
	}
	return authorized, nil
}

// Update does PATCH workitem
func (c *WorkitemController) Update(ctx *app.UpdateWorkitemContext) error {
	if ctx.Payload == nil || ctx.Payload.Data == nil || ctx.Payload.Data.ID == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("missing data.ID element in request", nil))
	}
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	var wi *workitem.WorkItem
	err = application.Transactional(c.db, func(appl application.Application) error {
		wi, err = appl.WorkItems().LoadByID(ctx, *ctx.Payload.Data.ID)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	creator := wi.Fields[workitem.SystemCreator]
	if creator == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.New("work item doesn't have creator")))
	}
	authorized, err := authorizeWorkitemEditor(ctx, c.db, wi.SpaceID, creator.(string), currentUserIdentityID.String())
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	if !authorized {
		return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not authorized to access the space"))
	}
	err = application.Transactional(c.db, func(appl application.Application) error {
		// The Number and Type of a work item are not allowed to be changed
		// which is why we overwrite those values with their old value after the
		// work item was converted.
		oldNumber := wi.Number
		oldType := wi.Type
		err = ConvertJSONAPIToWorkItem(ctx, ctx.Method, appl, *ctx.Payload.Data, wi, wi.Type, wi.SpaceID)
		if err != nil {
			return err
		}
		wi.Number = oldNumber
		wi.Type = oldType
		wi, err = appl.WorkItems().Save(ctx, wi.SpaceID, *wi, *currentUserIdentityID)
		if err != nil {
			return errs.Wrap(err, "Error updating work item")
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	wit, err := c.db.WorkItemTypes().Load(ctx.Context, wi.Type)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err, "failed to load work item type: %s", wi.Type))
	}
	c.notification.Send(ctx, notification.NewWorkItemUpdated(ctx.Payload.Data.ID.String()))
	converted, err := ConvertWorkItem(ctx.Request, *wit, *wi, workItemIncludeHasChildren(ctx, c.db))
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	resp := &app.WorkItemSingle{
		Data: converted,
		Links: &app.WorkItemLinks{
			Self: buildAbsoluteURL(ctx.Request),
		},
	}
	ctx.ResponseData.Header().Set("Last-Modified", lastModified(*wi))
	return ctx.OK(resp)
}

// Show does GET workitem
func (c *WorkitemController) Show(ctx *app.ShowWorkitemContext) error {
	var wi *workitem.WorkItem
	var wit *workitem.WorkItemType
	err := application.Transactional(c.db, func(appl application.Application) error {
		var err error
		wi, err = appl.WorkItems().LoadByID(ctx, ctx.WiID)
		if err != nil {
			return errs.Wrap(err, fmt.Sprintf("Fail to load work item with id %v", ctx.WiID))
		}
		wit, err = appl.WorkItemTypes().Load(ctx.Context, wi.Type)
		if err != nil {
			return errs.Wrapf(err, "failed to load work item type: %s", wi.Type)
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.ConditionalRequest(*wi, c.config.GetCacheControlWorkItem, func() error {
		comments := workItemIncludeCommentsAndTotal(ctx, c.db, ctx.WiID)
		hasChildren := workItemIncludeHasChildren(ctx, c.db)
		wi2, err := ConvertWorkItem(ctx.Request, *wit, *wi, comments, hasChildren)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		resp := &app.WorkItemSingle{
			Data: wi2,
		}
		return ctx.OK(resp)
	})
}

// Delete does DELETE workitem
func (c *WorkitemController) Delete(ctx *app.DeleteWorkitemContext) error {
	// Temporarly disabled, See https://github.com/fabric8-services/fabric8-wit/issues/1036
	if true {
		return ctx.MethodNotAllowed()
	}
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	var wi *workitem.WorkItem
	err = application.Transactional(c.db, func(appl application.Application) error {
		wi, err = appl.WorkItems().LoadByID(ctx, ctx.WiID)
		if err != nil {
			return errs.Wrap(err, fmt.Sprintf("Failed to load work item with id %v", ctx.WiID))
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	authorized, err := authz.Authorize(ctx, wi.SpaceID.String())
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	if !authorized {
		return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not authorized to access the space"))
	}
	err = application.Transactional(c.db, func(appl application.Application) error {
		if err := appl.WorkItems().Delete(ctx, ctx.WiID, *currentUserIdentityID); err != nil {
			return errs.Wrapf(err, "error deleting work item %s", ctx.WiID)
		}
		if err := appl.WorkItemLinks().DeleteRelatedLinks(ctx, ctx.WiID, *currentUserIdentityID); err != nil {
			return errs.Wrapf(err, "failed to delete work item links related to work item %s", ctx.WiID)
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.OK([]byte{})
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
func ConvertJSONAPIToWorkItem(ctx context.Context, method string, appl application.Application, source app.WorkItem, target *workitem.WorkItem, witID uuid.UUID, spaceID uuid.UUID) error {
	// load work item type to perform conversion according to a field type
	wit, err := appl.WorkItemTypes().Load(ctx, witID)
	if err != nil {
		return errs.Wrapf(err, "failed to load work item type: %s", witID)
	}
	_ = wit

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
				if ok := appl.Identities().IsValid(ctx, assigneeUUID); !ok {
					return errors.NewBadParameterError("data.relationships.assignees.data.id", *d.ID)
				}
				ids = append(ids, assigneeUUID.String())
			}
			target.Fields[workitem.SystemAssignees] = ids
		}
	}
	if source.Relationships != nil && source.Relationships.Labels != nil {
		// Pass empty array to remove all lables
		// null is treated as bad param
		if source.Relationships.Labels.Data == nil {
			return errors.NewBadParameterError("data.relationships.labels.data", nil)
		}
		distinctIDs := make(map[string]struct{})
		for _, d := range source.Relationships.Labels.Data {
			labelUUID, err := uuid.FromString(*d.ID)
			if err != nil {
				return errors.NewBadParameterError("data.relationships.labels.data.id", *d.ID)
			}
			if ok := appl.Labels().IsValid(ctx, labelUUID); !ok {
				return errors.NewBadParameterError("data.relationships.labels.data.id", *d.ID)
			}
			if _, ok := distinctIDs[labelUUID.String()]; !ok {
				distinctIDs[labelUUID.String()] = struct{}{}
			}
		}
		ids := make([]string, 0, len(distinctIDs))
		for k := range distinctIDs {
			ids = append(ids, k)
		}
		target.Fields[workitem.SystemLabels] = ids
	}
	if source.Relationships != nil {
		if source.Relationships.Iteration == nil || (source.Relationships.Iteration != nil && source.Relationships.Iteration.Data == nil) {
			log.Debug(ctx, map[string]interface{}{
				"wi_id":    target.ID,
				"space_id": spaceID,
			}, "assigning the work item to the root iteration of the space.")
			rootIteration, err := appl.Iterations().Root(ctx, spaceID)
			if err != nil {
				return errors.NewInternalError(ctx, err)
			}
			if method == http.MethodPost {
				target.Fields[workitem.SystemIteration] = rootIteration.ID.String()
			} else if method == http.MethodPatch {
				if source.Relationships.Iteration != nil && source.Relationships.Iteration.Data == nil {
					target.Fields[workitem.SystemIteration] = rootIteration.ID.String()
				}
			}
		} else if source.Relationships.Iteration != nil && source.Relationships.Iteration.Data != nil {
			d := source.Relationships.Iteration.Data
			iterationUUID, err := uuid.FromString(*d.ID)
			if err != nil {
				return errors.NewBadParameterError("data.relationships.iteration.data.id", *d.ID)
			}
			if err := appl.Iterations().CheckExists(ctx, iterationUUID); err != nil {
				return errors.NewNotFoundError("data.relationships.iteration.data.id", *d.ID)
			}
			target.Fields[workitem.SystemIteration] = iterationUUID.String()
		}
	}
	if source.Relationships != nil {
		if source.Relationships.Area == nil || (source.Relationships.Area != nil && source.Relationships.Area.Data == nil) {
			log.Debug(ctx, map[string]interface{}{
				"wi_id":    target.ID,
				"space_id": spaceID,
			}, "assigning the work item to the root area of the space.")
			err := appl.Spaces().CheckExists(ctx, spaceID)
			if err != nil {
				return errors.NewInternalError(ctx, err)
			}
			log.Debug(ctx, map[string]interface{}{
				"space_id": spaceID,
			}, "Loading root area for the space")
			rootArea, err := appl.Areas().Root(ctx, spaceID)
			if err != nil {
				return err
			}
			if method == http.MethodPost {
				target.Fields[workitem.SystemArea] = rootArea.ID.String()
			} else if method == http.MethodPatch {
				if source.Relationships.Area != nil && source.Relationships.Area.Data == nil {
					target.Fields[workitem.SystemArea] = rootArea.ID.String()
				}
			}
		} else if source.Relationships.Area != nil && source.Relationships.Area.Data != nil {
			d := source.Relationships.Area.Data
			areaUUID, err := uuid.FromString(*d.ID)
			if err != nil {
				return errors.NewBadParameterError("data.relationships.area.data.id", *d.ID)
			}
			if err := appl.Areas().CheckExists(ctx, areaUUID); err != nil {
				cause := errs.Cause(err)
				switch cause.(type) {
				case errors.NotFoundError:
					return errors.NewNotFoundError("data.relationships.area.data.id", *d.ID)
				default:
					return errs.Wrapf(err, "unknown error when verifying the area id %s", *d.ID)
				}
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
		switch key {
		case workitem.SystemDescription:
			// convert legacy description to markup content
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
		case workitem.SystemDescriptionMarkup:
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
		case workitem.SystemCodebase:
			m, err := codebase.NewCodebaseContentFromValue(val)
			if err != nil {
				return errs.Wrapf(err, "failed to create new codebase from value: %+v", val)
			}
			setupCodebase(appl, m, spaceID)
			target.Fields[key] = *m
		default:
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

// setupCodebase is the link between CodebaseContent & Codebase
// setupCodebase creates a codebase and saves it's ID in CodebaseContent
// for future use
func setupCodebase(appl application.Application, cb *codebase.Content, spaceID uuid.UUID) error {
	if cb.CodebaseID == "" {
		newCodeBase := codebase.Codebase{
			SpaceID: spaceID,
			Type:    "git",
			URL:     cb.Repository,
			StackID: ptr.String("java-centos"),
			//TODO: Think of making stackID dynamic value (from analyzer)
		}
		existingCB, err := appl.Codebases().LoadByRepo(context.Background(), spaceID, cb.Repository)
		if existingCB != nil {
			cb.CodebaseID = existingCB.ID.String()
			return nil
		}
		err = appl.Codebases().Create(context.Background(), &newCodeBase)
		if err != nil {
			return errors.NewInternalError(context.Background(), err)
		}
		cb.CodebaseID = newCodeBase.ID.String()
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
type WorkItemConvertFunc func(*http.Request, *workitem.WorkItem, *app.WorkItem) error

// ConvertWorkItems is responsible for converting given []WorkItem model object into a
// response resource object by jsonapi.org specifications
func ConvertWorkItems(request *http.Request, wits []workitem.WorkItemType, wis []workitem.WorkItem, additional ...WorkItemConvertFunc) ([]*app.WorkItem, error) {
	ops := []*app.WorkItem{}
	if len(wits) != len(wis) {
		return nil, errs.Errorf("length mismatch of work items (%d) and work item types (%d)", len(wis), len(wits))
	}
	for i := 0; i < len(wis); i++ {
		wi, err := ConvertWorkItem(request, wits[i], wis[i], additional...)
		if err != nil {
			return nil, errs.Wrapf(err, "failed to convert work item: %s", wis[i].ID)
		}
		ops = append(ops, wi)
	}
	return ops, nil
}

// ConvertWorkItem is responsible for converting given WorkItem model object into a
// response resource object by jsonapi.org specifications
func ConvertWorkItem(request *http.Request, wit workitem.WorkItemType, wi workitem.WorkItem, additional ...WorkItemConvertFunc) (*app.WorkItem, error) {
	// construct default values from input WI
	relatedURL := rest.AbsoluteURL(request, app.WorkitemHref(wi.ID))
	labelsRelated := relatedURL + "/labels"
	workItemLinksRelated := relatedURL + "/links"

	op := &app.WorkItem{
		ID:   &wi.ID,
		Type: APIStringTypeWorkItem,
		Attributes: map[string]interface{}{
			workitem.SystemVersion: wi.Version,
			workitem.SystemNumber:  wi.Number,
		},
		Relationships: &app.WorkItemRelationships{
			BaseType: &app.RelationBaseType{
				Data: &app.BaseTypeData{
					ID:   wi.Type,
					Type: APIStringTypeWorkItemType,
				},
				Links: &app.GenericLinks{
					Self: ptr.String(rest.AbsoluteURL(request, app.WorkitemtypeHref(wi.Type))),
				},
			},
			Space: app.NewSpaceRelation(wi.SpaceID, rest.AbsoluteURL(request, app.SpaceHref(wi.SpaceID.String()))),
			WorkItemLinks: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &workItemLinksRelated,
				},
			},
		},
		Links: &app.GenericLinksForWorkItem{
			Self:    &relatedURL,
			Related: &relatedURL,
		},
	}

	// Move fields into Relationships or Attributes as needed
	// TODO(kwk): Loop based on WorkItemType and match against Field.Type instead of directly to field value
	for name, val := range wi.Fields {
		switch name {
		case workitem.SystemAssignees:
			if val != nil {
				userID := val.([]interface{})
				op.Relationships.Assignees = &app.RelationGenericList{
					Data: ConvertUsersSimple(request, userID),
				}
			}
		case workitem.SystemLabels:
			if val != nil {
				labelIDs := val.([]interface{})
				op.Relationships.Labels = &app.RelationGenericList{
					Data: ConvertLabelsSimple(request, labelIDs),
					Links: &app.GenericLinks{
						Related: &labelsRelated,
					},
				}
			}
		case workitem.SystemCreator:
			if val != nil {
				userID := val.(string)
				op.Relationships.Creator = &app.RelationGeneric{
					Data: ConvertUserSimple(request, userID),
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
				cb := val.(codebase.Content)
				editURL := rest.AbsoluteURL(request, app.CodebaseHref(cb.CodebaseID)+"/edit")
				op.Links.EditCodebase = &editURL
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
	// Always include Comments Link, but optionally use workItemIncludeCommentsAndTotal
	workItemIncludeComments(request, &wi, op)
	workItemIncludeChildren(request, &wi, op)
	workItemIncludeEvents(request, &wi, op)
	for _, add := range additional {
		if err := add(request, &wi, op); err != nil {
			return nil, errs.Wrap(err, "failed to run additional conversion function")
		}
	}
	return op, nil
}

// workItemIncludeHasChildren adds meta information about existing children
func workItemIncludeHasChildren(ctx context.Context, appl application.Application, childLinks ...link.WorkItemLinkList) WorkItemConvertFunc {
	// TODO: Wrap ctx in a Timeout context?
	return func(request *http.Request, wi *workitem.WorkItem, wi2 *app.WorkItem) error {
		var hasChildren bool
		// If we already have information about children inside the child links
		// we can use that before querying the DB.
		if len(childLinks) == 1 {
			for _, l := range childLinks[0] {
				if l.LinkTypeID == link.SystemWorkItemLinkTypeParentChildID && l.SourceID == wi.ID {
					hasChildren = true
				}
			}
		}
		if !hasChildren {
			var err error
			repo := appl.WorkItemLinks()
			if repo != nil {
				hasChildren, err = appl.WorkItemLinks().WorkItemHasChildren(ctx, wi.ID)
				log.Info(ctx, map[string]interface{}{"wi_id": wi.ID}, "Work item has children: %t", hasChildren)
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"wi_id": wi.ID,
						"err":   err,
					}, "unable to find out if work item has children: %s", wi.ID)
					// enforce to have no children
					hasChildren = false
					return errs.Wrapf(err, "failed to determine if work item %s has children", wi.ID)
				}
			}
		}
		if wi2.Relationships.Children == nil {
			wi2.Relationships.Children = &app.RelationGeneric{}
		}
		wi2.Relationships.Children.Meta = map[string]interface{}{
			"hasChildren": hasChildren,
		}
		return nil
	}
}

// includeParentWorkItem adds the parent of given WI to relationships & included object
func includeParentWorkItem(ctx context.Context, ancestors link.AncestorList, childLinks link.WorkItemLinkList) WorkItemConvertFunc {
	return func(request *http.Request, wi *workitem.WorkItem, wi2 *app.WorkItem) error {
		var parentID *uuid.UUID
		// If we have an ancestry we can lookup the parent in no time.
		if ancestors != nil && len(ancestors) != 0 {
			p := ancestors.GetParentOf(wi.ID)
			if p != nil {
				parentID = &p.ID
			}
		}
		// If no parent ID was found in the ancestor list, see if the child
		// link list contains information to use.
		if parentID == nil && childLinks != nil && len(childLinks) != 0 {
			p := childLinks.GetParentIDOf(wi.ID, link.SystemWorkItemLinkTypeParentChildID)
			if p != uuid.Nil {
				parentID = &p
			}
		}
		if wi2.Relationships.Parent == nil {
			wi2.Relationships.Parent = &app.RelationKindUUID{}
		}
		if parentID != nil {
			if wi2.Relationships.Parent.Data == nil {
				wi2.Relationships.Parent.Data = &app.DataKindUUID{}
			}
			wi2.Relationships.Parent.Data.ID = *parentID
			wi2.Relationships.Parent.Data.Type = APIStringTypeWorkItem
		}
		return nil
	}
}

// ListChildren runs the list action.
func (c *WorkitemController) ListChildren(ctx *app.ListChildrenWorkitemContext) error {
	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)
	var result []workitem.WorkItem
	var count int
	var wits []workitem.WorkItemType
	err := application.Transactional(c.db, func(appl application.Application) error {
		var err error
		result, count, err = appl.WorkItemLinks().ListWorkItemChildren(ctx, ctx.WiID, &offset, &limit)
		if err != nil {
			return errs.Wrap(err, "unable to list work item children")
		}
		wits, err = loadWorkItemTypesFromArr(ctx.Context, appl, result)
		if err != nil {
			return errs.Wrap(err, "failed to load the work item types")
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.ConditionalEntities(result, c.config.GetCacheControlWorkItems, func() error {
		var response app.WorkItemList
		application.Transactional(c.db, func(appl application.Application) error {
			hasChildren := workItemIncludeHasChildren(ctx, appl)
			converted, err := ConvertWorkItems(ctx.Request, wits, result, hasChildren)
			if err != nil {
				return errs.WithStack(err)
			}
			response = app.WorkItemList{
				Links: &app.PagingLinks{},
				Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
				Data:  converted,
			}
			return nil
		})
		setPagingLinks(response.Links, buildAbsoluteURL(ctx.Request), len(result), offset, limit, count)
		return ctx.OK(&response)
	})
}

// workItemIncludeChildren adds relationship about children to workitem (include totalCount)
func workItemIncludeChildren(request *http.Request, wi *workitem.WorkItem, wi2 *app.WorkItem) {
	childrenRelated := rest.AbsoluteURL(request, app.WorkitemHref(wi.ID.String())) + "/children"
	if wi2.Relationships.Children == nil {
		wi2.Relationships.Children = &app.RelationGeneric{}
	}
	wi2.Relationships.Children.Links = &app.GenericLinks{
		Related: &childrenRelated,
	}
}

// workItemIncludeEvents adds relationship about events to workitem (include totalCount)
func workItemIncludeEvents(request *http.Request, wi *workitem.WorkItem, wi2 *app.WorkItem) {
	eventsRelated := rest.AbsoluteURL(request, app.WorkitemHref(wi.ID.String())) + "/events"
	if wi2.Relationships.Events == nil {
		wi2.Relationships.Events = &app.RelationGeneric{}
	}
	wi2.Relationships.Events.Links = &app.GenericLinks{
		Related: &eventsRelated,
	}
}

func loadWorkItemTypesFromArr(ctx context.Context, appl application.Application, wis []workitem.WorkItem) ([]workitem.WorkItemType, error) {
	wits := make([]workitem.WorkItemType, len(wis))
	for idx, wi := range wis {
		wit, err := appl.WorkItemTypes().Load(ctx, wi.Type)
		if err != nil {
			return nil, errs.Wrapf(err, "failed to load the work item type: %s", wi.Type)
		}
		wits[idx] = *wit
	}
	return wits, nil
}

func loadWorkItemTypesFromPtrArr(ctx context.Context, appl application.Application, wis []*workitem.WorkItem) ([]workitem.WorkItemType, error) {
	wits := make([]workitem.WorkItemType, len(wis))
	for idx, wi := range wis {
		wit, err := appl.WorkItemTypes().Load(ctx, wi.Type)
		if err != nil {
			return nil, errs.Wrapf(err, "failed to load the work item type: %s", wi.Type)
		}
		wits[idx] = *wit
	}
	return wits, nil
}
