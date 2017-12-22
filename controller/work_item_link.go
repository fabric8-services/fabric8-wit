package controller

import (
	"context"
	"net/http"
	"reflect"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/space/authz"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

// WorkItemLinkController implements the work-item-link resource.
type WorkItemLinkController struct {
	*goa.Controller
	db     application.DB
	config WorkItemLinkControllerConfig
}

// WorkItemLinkControllerConfig the config interface for the WorkitemLinkController
type WorkItemLinkControllerConfig interface {
	GetCacheControlWorkItemLinks() string
	GetCacheControlWorkItemLink() string
}

// NewWorkItemLinkController creates a work-item-link controller.
func NewWorkItemLinkController(service *goa.Service, db application.DB, config WorkItemLinkControllerConfig) *WorkItemLinkController {
	return &WorkItemLinkController{
		Controller: service.NewController("WorkItemLinkController"),
		db:         db,
		config:     config,
	}
}

// Instead of using app.WorkItemLinkHref directly, we store a function object
// inside the WorkItemLinkContext in order to generate proper links nomatter
// from which controller the context is being used. For example one could use
// app.WorkItemRelationshipsLinksHref.
type hrefLinkFunc func(obj interface{}) string

// workItemLinkContext bundles objects that are needed by most of the functions.
// It can easily be extended.
type workItemLinkContext struct {
	Request               *http.Request
	ResponseWriter        http.ResponseWriter
	Application           application.Application
	Context               context.Context
	Service               *goa.Service
	CurrentUserIdentityID *uuid.UUID
	DB                    application.DB
	LinkFunc              hrefLinkFunc
}

// newWorkItemLinkContext returns a new workItemLinkContext
func newWorkItemLinkContext(ctx context.Context, service *goa.Service, appl application.Application, db application.DB, request *http.Request, responseWriter http.ResponseWriter, linkFunc hrefLinkFunc, currentUserIdentityID *uuid.UUID) *workItemLinkContext {
	return &workItemLinkContext{
		Request:               request,
		ResponseWriter:        responseWriter,
		Application:           appl,
		Context:               ctx,
		Service:               service,
		CurrentUserIdentityID: currentUserIdentityID,
		DB:       db,
		LinkFunc: linkFunc,
	}
}

// getTypesOfLinks returns an array of distinct work item link types for the
// given work item links
func getTypesOfLinks(ctx context.Context, appl application.Application, req *http.Request, linksDataArr []*app.WorkItemLinkData) ([]*app.WorkItemLinkTypeData, error) {
	// Build our "set" of distinct type IDs
	idMap := map[uuid.UUID]struct{}{}
	idArr := []uuid.UUID{}
	for _, linkData := range linksDataArr {
		id := linkData.Relationships.LinkType.Data.ID
		if _, ok := idMap[id]; !ok {
			idMap[id] = struct{}{}
			idArr = append(idArr, id)
		}
	}

	// Now include the optional link type data in the work item link "included" array
	linkTypeModels := []link.WorkItemLinkType{}
	for _, typeID := range idArr {
		linkTypeModel, err := appl.WorkItemLinkTypes().Load(ctx, typeID)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		linkTypeModels = append(linkTypeModels, *linkTypeModel)
	}
	appLinkTypes, err := ConvertLinkTypesFromModels(req, linkTypeModels)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	return appLinkTypes.Data, nil
}

// getWorkItemsOfLinks returns an array of distinct work items as they appear as
// source or target in the given work item links.
func getWorkItemsOfLinks(ctx context.Context, appl application.Application, req *http.Request, linksDataArr []*app.WorkItemLinkData) ([]*app.WorkItem, error) {
	// Build our "set" of distinct work item IDs
	idMap := map[uuid.UUID]struct{}{}
	idArr := []uuid.UUID{}
	for _, linkData := range linksDataArr {
		src := linkData.Relationships.Source.Data.ID
		tgt := linkData.Relationships.Target.Data.ID
		if _, ok := idMap[src]; !ok {
			idMap[src] = struct{}{}
			idArr = append(idArr, src)
		}
		if _, ok := idMap[tgt]; !ok {
			idMap[tgt] = struct{}{}
			idArr = append(idArr, tgt)
		}
	}

	// Fetch all work items specified in the array
	res := []*app.WorkItem{}
	for _, id := range idArr {
		wi, err := appl.WorkItems().LoadByID(ctx, id)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		res = append(res, ConvertWorkItem(req, *wi))
	}
	return res, nil
}

// enrichLinkSingle includes related resources in the link's "included" array
func enrichLinkSingle(ctx context.Context, appl application.Application, req *http.Request, appLinks *app.WorkItemLinkSingle) error {
	// Include link type
	//modelLinkType, err := ctx.Application.WorkItemLinkTypes().Load(ctx.Context, appLinks.Data.Relationships.LinkType.Data.ID)
	modelLinkType, err := appl.WorkItemLinkTypes().Load(ctx, appLinks.Data.Relationships.LinkType.Data.ID)
	if err != nil {
		return errs.WithStack(err)
	}
	appLinkType := ConvertWorkItemLinkTypeFromModel(req, *modelLinkType)
	appLinks.Included = append(appLinks.Included, appLinkType.Data)

	// Include source work item
	sourceWi, err := appl.WorkItems().LoadByID(ctx, appLinks.Data.Relationships.Source.Data.ID)
	if err != nil {
		return errs.WithStack(err)
	}
	appLinks.Included = append(appLinks.Included, ConvertWorkItem(req, *sourceWi))

	// Include target work item
	targetWi, err := appl.WorkItems().LoadByID(ctx, appLinks.Data.Relationships.Target.Data.ID)
	if err != nil {
		return errs.WithStack(err)
	}
	appLinks.Included = append(appLinks.Included, ConvertWorkItem(req, *targetWi))

	return nil
}

// enrichLinkList includes related resources in the linkArr's "included" element
func enrichLinkList(ctx context.Context, appl application.Application, req *http.Request, linkArr *app.WorkItemLinkList) error {
	// include link types
	typeDataArr, err := getTypesOfLinks(ctx, appl, req, linkArr.Data)
	if err != nil {
		return errs.WithStack(err)
	}
	// Convert slice of objects to slice of interface (see https://golang.org/doc/faq#convert_slice_of_interface)
	interfaceArr := make([]interface{}, len(typeDataArr))
	for i, v := range typeDataArr {
		interfaceArr[i] = v
	}
	linkArr.Included = append(linkArr.Included, interfaceArr...)

	// TODO(kwk): Include WIs from source and target
	workItemDataArr, err := getWorkItemsOfLinks(ctx, appl, req, linkArr.Data)
	if err != nil {
		return errs.WithStack(err)
	}
	// Convert slice of objects to slice of interface (see https://golang.org/doc/faq#convert_slice_of_interface)
	interfaceArr = make([]interface{}, len(workItemDataArr))
	for i, v := range workItemDataArr {
		interfaceArr[i] = v
	}
	linkArr.Included = append(linkArr.Included, interfaceArr...)

	return nil
}

type createWorkItemLinkFuncs interface {
	BadRequest(r *app.JSONAPIErrors) error
	Created(r *app.WorkItemLinkSingle) error
	InternalServerError(r *app.JSONAPIErrors) error
	Unauthorized(r *app.JSONAPIErrors) error
}

func createWorkItemLink(ctx *workItemLinkContext, httpFuncs createWorkItemLinkFuncs, payload *app.CreateWorkItemLinkPayload) error {
	// Convert payload from app to model representation
	in := app.WorkItemLinkSingle{
		Data: payload.Data,
	}
	modelLink, err := ConvertLinkToModel(in)
	if err != nil {
		return jsonapi.JSONErrorResponse(httpFuncs, err)
	}
	createdModelLink, err := ctx.Application.WorkItemLinks().Create(ctx.Context, modelLink.SourceID, modelLink.TargetID, modelLink.LinkTypeID, *ctx.CurrentUserIdentityID)
	if err != nil {
		switch reflect.TypeOf(err) {
		case reflect.TypeOf(&goa.ErrorResponse{}):
			return jsonapi.JSONErrorResponse(httpFuncs, goa.ErrBadRequest(err.Error()))
		default:
			return jsonapi.JSONErrorResponse(httpFuncs, err)
		}
	}
	// convert from model to rest representation
	createdAppLink := ConvertLinkFromModel(ctx.Request, *createdModelLink)
	if err := enrichLinkSingle(ctx.Context, ctx.Application, ctx.Request, &createdAppLink); err != nil {
		return jsonapi.JSONErrorResponse(httpFuncs, err)
	}
	ctx.ResponseWriter.Header().Set("Location", app.WorkItemLinkHref(createdAppLink.Data.ID))
	return httpFuncs.Created(&createdAppLink)
}

// Create runs the create action.
func (c *WorkItemLinkController) Create(ctx *app.CreateWorkItemLinkContext) error {
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		linkCtx := newWorkItemLinkContext(ctx.Context, ctx.Service, appl, c.db, ctx.Request, ctx.ResponseWriter, app.WorkItemLinkHref, currentUserIdentityID)
		return createWorkItemLink(linkCtx, ctx, ctx.Payload)
	})
}

func (c *WorkItemLinkController) checkIfUserIsSpaceCollaboratorOrWorkItemCreator(ctx context.Context, linkID uuid.UUID, currentIdentityID uuid.UUID) (bool, error) {
	var authorized bool
	var sourceSpaceID *uuid.UUID
	var targetSpaceID *uuid.UUID
	err := application.Transactional(c.db, func(appl application.Application) error {
		link, err := appl.WorkItemLinks().Load(ctx, linkID)
		if err != nil {
			return err
		}
		authorized, sourceSpaceID, err = c.checkWorkItemCreatorOrSpaceOwner(ctx, appl, link.SourceID, currentIdentityID)
		if err != nil {
			return err
		}
		if !authorized {
			authorized, targetSpaceID, err = c.checkWorkItemCreatorOrSpaceOwner(ctx, appl, link.TargetID, currentIdentityID)
		}
		return err
	})
	if err != nil {
		return false, err
	}
	// Check if the user is a space collaborator
	if !authorized {
		authorized, err = authz.Authorize(ctx, sourceSpaceID.String())
		if err != nil {
			return false, err
		}
		return authz.Authorize(ctx, targetSpaceID.String())
	}
	return authorized, nil
}

func (c *WorkItemLinkController) checkWorkItemCreatorOrSpaceOwner(ctx context.Context, appl application.Application, workItemID uuid.UUID, currentIdentityID uuid.UUID) (bool, *uuid.UUID, error) {
	wi, err := appl.WorkItems().LoadByID(ctx, workItemID)
	if err != nil {
		return false, nil, err
	}
	creator := wi.Fields[workitem.SystemCreator]
	if currentIdentityID.String() == creator {
		return true, nil, nil
	}
	space, err := appl.Spaces().Load(ctx, wi.SpaceID)
	return currentIdentityID == space.OwnerID, &wi.SpaceID, nil
}

type deleteWorkItemLinkFuncs interface {
	OK(resp []byte) error
	BadRequest(r *app.JSONAPIErrors) error
	NotFound(r *app.JSONAPIErrors) error
	Unauthorized(r *app.JSONAPIErrors) error
	InternalServerError(r *app.JSONAPIErrors) error
}

func deleteWorkItemLink(ctx *workItemLinkContext, httpFuncs deleteWorkItemLinkFuncs, linkID uuid.UUID) error {
	err := ctx.Application.WorkItemLinks().Delete(ctx.Context, linkID, *ctx.CurrentUserIdentityID)
	if err != nil {
		return jsonapi.JSONErrorResponse(httpFuncs, err)
	}
	return httpFuncs.OK([]byte{})
}

//
// Delete runs the delete action
func (c *WorkItemLinkController) Delete(ctx *app.DeleteWorkItemLinkContext) error {
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	authorized, err := c.checkIfUserIsSpaceCollaboratorOrWorkItemCreator(ctx, ctx.LinkID, *currentUserIdentityID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	if !authorized {
		return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not authorized to delete the link"))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		linkCtx := newWorkItemLinkContext(ctx.Context, ctx.Service, appl, c.db, ctx.Request, ctx.ResponseWriter, app.WorkItemLinkHref, currentUserIdentityID)
		return deleteWorkItemLink(linkCtx, ctx, ctx.LinkID)
	})
}

// Show runs the show action.
func (c *WorkItemLinkController) Show(ctx *app.ShowWorkItemLinkContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		modelLink, err := appl.WorkItemLinks().Load(ctx.Context, ctx.LinkID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalRequest(*modelLink, c.config.GetCacheControlWorkItemLink, func() error {
			// convert to rest representation
			appLink := ConvertLinkFromModel(ctx.Request, *modelLink)
			if err := enrichLinkSingle(ctx.Context, appl, ctx.Request, &appLink); err != nil {
				return jsonapi.JSONErrorResponse(ctx, err)
			}
			return ctx.OK(&appLink)
		})
	})
}

// ConvertLinkFromModel converts a work item from model to REST representation
func ConvertLinkFromModel(request *http.Request, t link.WorkItemLink) app.WorkItemLinkSingle {
	linkSelfURL := rest.AbsoluteURL(request, app.WorkItemLinkHref(t.ID.String()))
	linkTypeRelatedURL := rest.AbsoluteURL(request, app.WorkItemLinkTypeHref(space.SystemSpace, t.LinkTypeID.String()))

	sourceRelatedURL := rest.AbsoluteURL(request, app.WorkitemHref(t.SourceID.String()))
	targetRelatedURL := rest.AbsoluteURL(request, app.WorkitemHref(t.TargetID.String()))

	var converted = app.WorkItemLinkSingle{
		Data: &app.WorkItemLinkData{
			Type: link.EndpointWorkItemLinks,
			ID:   &t.ID,
			Attributes: &app.WorkItemLinkAttributes{
				CreatedAt: &t.CreatedAt,
				UpdatedAt: &t.UpdatedAt,
				Version:   &t.Version,
			},
			Links: &app.GenericLinks{
				Self: &linkSelfURL,
			},
			Relationships: &app.WorkItemLinkRelationships{
				LinkType: &app.RelationWorkItemLinkType{
					Data: &app.RelationWorkItemLinkTypeData{
						Type: link.EndpointWorkItemLinkTypes,
						ID:   t.LinkTypeID,
					},
					Links: &app.GenericLinks{
						Self: &linkTypeRelatedURL,
					},
				},
				Source: &app.RelationWorkItem{
					Data: &app.RelationWorkItemData{
						Type: link.EndpointWorkItems,
						ID:   t.SourceID,
					},
					Links: &app.GenericLinks{
						Related: &sourceRelatedURL,
					},
				},
				Target: &app.RelationWorkItem{
					Data: &app.RelationWorkItemData{
						Type: link.EndpointWorkItems,
						ID:   t.TargetID,
					},
					Links: &app.GenericLinks{
						Related: &targetRelatedURL,
					},
				},
			},
		},
	}
	return converted
}

// ConvertLinkToModel converts the incoming app representation of a work item link to the model layout.
// Values are only overwrriten if they are set in "in", otherwise the values in "out" remain.
// NOTE: Only the LinkTypeID, SourceID, and TargetID fields will be set.
//       You need to preload the elements after calling this function.
func ConvertLinkToModel(appLink app.WorkItemLinkSingle) (*link.WorkItemLink, error) {
	modelLink := link.WorkItemLink{}
	attrs := appLink.Data.Attributes
	rel := appLink.Data.Relationships
	if appLink.Data.ID != nil {
		modelLink.ID = *appLink.Data.ID
	}

	if attrs != nil && attrs.Version != nil {
		modelLink.Version = *attrs.Version
	}

	if rel != nil && rel.LinkType != nil && rel.LinkType.Data != nil {
		modelLink.LinkTypeID = rel.LinkType.Data.ID
	}

	if rel != nil && rel.Source != nil && rel.Source.Data != nil {
		d := rel.Source.Data
		// The the work item id MUST NOT be empty
		if d.ID == uuid.Nil {
			return nil, errors.NewBadParameterError("data.relationships.source.data.id", d.ID)
		}
		modelLink.SourceID = d.ID
	}

	if rel != nil && rel.Target != nil && rel.Target.Data != nil {
		d := rel.Target.Data
		// If the the target type is not nil, it MUST be "workitems"
		// The the work item id MUST NOT be empty
		if d.ID == uuid.Nil {
			return nil, errors.NewBadParameterError("data.relationships.target.data.id", d.ID)
		}
		modelLink.TargetID = d.ID
	}

	return &modelLink, nil
}
