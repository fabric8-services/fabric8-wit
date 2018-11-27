package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/area"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/client"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/goasupport"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"

	"github.com/goadesign/goa"
	goaclient "github.com/goadesign/goa/client"
	goauuid "github.com/goadesign/goa/uuid"
	errs "github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

const (
	// APIStringTypeCodebase contains the JSON API type for codebases
	APIStringTypeSpace = "spaces"
)

// SpaceConfiguration represents space configuratoin
type SpaceConfiguration interface {
	GetCacheControlSpaces() string
	GetCacheControlSpace() string
}

// SpaceController implements the space resource.
type SpaceController struct {
	*goa.Controller
	db                application.DB
	config            SpaceConfiguration
	resourceManager   auth.ResourceManager
	DeploymentsClient *http.Client
	CodebaseClient    *http.Client
}

// NewSpaceController creates a space controller.
func NewSpaceController(
	service *goa.Service,
	db application.DB,
	config SpaceConfiguration,
	resourceManager auth.ResourceManager) *SpaceController {

	return &SpaceController{
		Controller:        service.NewController("SpaceController"),
		db:                db,
		config:            config,
		resourceManager:   resourceManager,
		DeploymentsClient: http.DefaultClient,
		CodebaseClient:    http.DefaultClient,
	}
}

// Create runs the create action.
func (c *SpaceController) Create(ctx *app.CreateSpaceContext) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}

	err = validateCreateSpace(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	reqSpace := ctx.Payload.Data
	spaceName := *reqSpace.Attributes.Name
	spaceID := uuid.NewV4()
	if reqSpace.ID != nil {
		spaceID = *reqSpace.ID
	}

	var rSpace *space.Space
	err = application.Transactional(c.db, func(appl application.Application) error {
		newSpace := space.Space{
			ID:      spaceID,
			Name:    spaceName,
			OwnerID: *currentUser,
			// Default to Agile space template
			SpaceTemplateID: spacetemplate.SystemAgileTemplateID,
		}
		if reqSpace.Attributes.Description != nil {
			newSpace.Description = *reqSpace.Attributes.Description
		}
		// if given, use space template from relationship
		if reqSpace.Relationships != nil && reqSpace.Relationships.SpaceTemplate != nil && reqSpace.Relationships.SpaceTemplate.Data != nil {
			stID := reqSpace.Relationships.SpaceTemplate.Data.ID
			if err := appl.SpaceTemplates().CheckExists(ctx, stID); err != nil {
				return errs.Wrapf(err, "space template not found: %s", stID)
			}
			newSpace.SpaceTemplateID = stID
		}

		rSpace, err = appl.Spaces().Create(ctx, &newSpace)
		if err != nil {
			return errs.Wrapf(err, "failed to create space: %s", newSpace.Name)
		}
		/*
			Should we create the new area
			- over the wire(service) something like app.NewCreateSpaceAreasContext(..), OR
			- as part of a db transaction ?

			The argument 'for' creating it at a transaction level is :
			You absolutely need both space creation + area creation
			to happen in a single transaction as per requirements.
		*/

		newArea := area.Area{
			SpaceID: rSpace.ID,
			Name:    rSpace.Name,
		}
		err = appl.Areas().Create(ctx, &newArea)
		if err != nil {
			return errs.Wrapf(err, "failed to create area: %s", rSpace.Name)
		}

		// Similar to above, we create a root iteration for this new space
		newIteration := iteration.Iteration{
			SpaceID: rSpace.ID,
			Name:    rSpace.Name,
		}
		err = appl.Iterations().Create(ctx, &newIteration)
		if err != nil {
			return errs.Wrapf(err, "failed to create iteration for space: %s", rSpace.Name)
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	// Create keycloak resource for this space
	err = c.resourceManager.CreateSpace(ctx, ctx.Request, spaceID.String())
	if err != nil {
		// Unable to create a space resource. Can't proceed. Roll back space creation and return an error.
		c.rollBackSpaceCreation(ctx, spaceID)
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	spaceData, err := ConvertSpaceFromModel(ctx.Request, *rSpace, IncludeBacklogTotalCount(ctx.Context, c.db))
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	res := &app.SpaceSingle{
		Data: spaceData,
	}
	ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.Request, app.SpaceHref(res.Data.ID)))
	return ctx.Created(res)
}

func (c *SpaceController) rollBackSpaceCreation(ctx context.Context, spaceID uuid.UUID) error {
	err := application.Transactional(c.db, func(appl application.Application) error {
		return appl.Spaces().Delete(ctx, spaceID)
	})
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":      err,
			"space_id": spaceID,
		}, "unable to roll back space creation")
	}
	return err
}

// Delete runs the delete action.
func (c *SpaceController) Delete(ctx *app.DeleteSpaceContext) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}

	spaceID, err := goauuid.FromString(ctx.SpaceID.String())
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": ctx.SpaceID,
			"err":      err,
		}, "could not convert the UUID of type github.com/satori/go.uuid to github.com/goadesign/goa/uuid")
		return jsonapi.JSONErrorResponse(
			ctx, errors.NewInternalError(ctx, errs.Wrap(err, "could not delete space")),
		)
	}

	// extract config in it's generic form to be utilized elsewhere
	config, ok := c.config.(*configuration.Registry)
	if !ok {
		log.Error(ctx, map[string]interface{}{
			"space_id": spaceID,
		}, "no configuation found in SpaceController object")
		return jsonapi.JSONErrorResponse(
			ctx, errors.NewInternalErrorFromString("could not delete space"),
		)
	}

	// delete all the codebases associated with this space
	err = deleteCodebases(c.CodebaseClient, config, ctx.Context, spaceID)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": spaceID,
			"err":      err,
		}, "could not delete codebases")
		return jsonapi.JSONErrorResponse(
			ctx, errors.NewInternalError(ctx, errs.Wrapf(err, "failed to delete codebases associated with space %s", spaceID)))
	}

	// now delete the OpenShift resources associated with this space on an
	// OpenShift cluster, unless otherwise specified
	if ctx.SkipCluster == nil || !*ctx.SkipCluster {
		err = deleteOpenShiftResource(c.DeploymentsClient, config, ctx.Context, spaceID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"space_id": spaceID,
				"err":      err,
			}, "could not delete OpenShift resources")
			return jsonapi.JSONErrorResponse(
				ctx, errors.NewInternalError(ctx, errs.Wrapf(
					err, "could not delete Openshift resources associated with space %s", spaceID),
				),
			)
		}
	}

	err = application.Transactional(c.db, func(appl application.Application) error {
		s, err := appl.Spaces().Load(ctx.Context, ctx.SpaceID)
		if err != nil {
			return err
		}
		if !uuid.Equal(*currentUser, s.OwnerID) {
			log.Warn(ctx, map[string]interface{}{
				"space_id":     ctx.SpaceID,
				"space_owner":  s.OwnerID,
				"current_user": *currentUser,
			}, "user is not the space owner")
			return errors.NewForbiddenError("user is not the space owner")
		}
		return appl.Spaces().Delete(ctx.Context, ctx.SpaceID)
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	err = c.resourceManager.DeleteSpace(ctx, ctx.Request, ctx.SpaceID.String())
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	return ctx.OK([]byte{})
}

// deleteCodebases deletes all the codebases that are associated with this space
func deleteCodebases(
	httpClient *http.Client,
	config *configuration.Registry,
	ctx context.Context,
	spaceID goauuid.UUID) error {

	u, err := url.Parse(config.GetCodebaseServiceURL())
	if err != nil {
		return errors.NewInternalError(ctx,
			fmt.Errorf("malformed codebase service URL %s: %v",
				config.GetCodebaseServiceURL(), err))
	}

	cl := client.New(goaclient.HTTPClientDoer(httpClient))
	cl.Host = u.Host
	cl.Scheme = u.Scheme
	cl.SetJWTSigner(goasupport.NewForwardSigner(goasupport.ForwardContextRequestID(ctx)))

	// list all the codebases associated with the space
	path := client.ListSpaceCodebasesPath(spaceID)
	resp, err := cl.ListSpaceCodebases(ctx, path, nil, nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": spaceID,
			"path":     path,
			"err":      err,
		}, "failed to list codebases")
		return errs.Wrapf(err, "could not list codebases for space: %s", spaceID)
	}
	defer resp.Body.Close()

	if 200 < resp.StatusCode && resp.StatusCode >= 300 {
		formattedErrors, err := cl.DecodeJSONAPIErrors(resp)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err":      err,
				"response": resp,
			}, "failed to decode JSON formatted errors returned while listing codebases")
			return errors.NewInternalError(ctx,
				fmt.Errorf("could not decode JSON formatted errors returned while listing codebases: %v", err))
		}
		if len(formattedErrors.Errors) > 0 {
			return errors.NewInternalError(ctx, errs.Errorf(formattedErrors.Errors[0].Detail))
		}
		return errors.NewInternalError(ctx, errs.Errorf("unknown error"))
	}
	codebases, err := cl.DecodeCodebaseList(resp)
	if err != nil {
		return errors.NewInternalError(ctx,
			fmt.Errorf("could not decode the codebase list: %v", err))
	}

	// iterate on all the codebases and delete them
	var errorsList []error
	for _, cb := range codebases.Data {
		path = client.DeleteCodebasePath(*cb.ID)
		resp, err := cl.DeleteCodebase(ctx, path)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"error":    err,
				"response": resp,
			}, "failed to delete codebase %s", cb.ID)
			errorsList = append(errorsList,
				errs.Wrapf(err, "could not delete codebase %s", cb.ID))
			continue
		}
		if 200 < resp.StatusCode && resp.StatusCode >= 300 {
			formattedErrors, err := cl.DecodeJSONAPIErrors(resp)
			if err != nil {
				log.Error(ctx, map[string]interface{}{
					"error":    err,
					"response": resp,
				}, "failed to decode JSON formatted errors returned while deleting codebase %s", cb.ID)
				errorsList = append(errorsList,
					errs.Wrapf(err, "could not decode JSON formatted errors returned while deleting codebase %s", cb.ID))
				continue
			}
			if len(formattedErrors.Errors) > 0 {
				errorsList = append(errorsList, errs.Errorf(formattedErrors.Errors[0].Detail))
			}
		}
	}
	if len(errorsList) != 0 {
		var errString string
		for _, err = range errorsList {
			errString += fmt.Sprintf("%s\n", err)
		}
		return errors.NewInternalErrorFromString(errString)
	}

	return nil
}

// deleteOpenShiftResource deletes all the openshift resources present in the
// OpenShift online cluster corresponding to the given spaceID
// TODO: fix all the errors, return appropriate errors
func deleteOpenShiftResource(
	httpClient *http.Client,
	config *configuration.Registry,
	ctx context.Context,
	spaceID goauuid.UUID) error {

	u, err := url.Parse(config.GetDeploymentsServiceURL())
	if err != nil {
		return errors.NewInternalError(ctx,
			fmt.Errorf("malformed deployments service URL %s: %v",
				config.GetDeploymentsServiceURL(), err))
	}

	cl := client.New(goaclient.HTTPClientDoer(httpClient))
	cl.Host = u.Host
	cl.Scheme = u.Scheme
	cl.SetJWTSigner(goasupport.NewForwardSigner(goasupport.ForwardContextRequestID(ctx)))

	// get all the apps and envs
	path := client.ShowSpaceDeploymentsPath(spaceID)
	resp, err := cl.ShowSpaceDeployments(ctx, path)
	if err != nil {
		return errors.NewInternalError(ctx,
			fmt.Errorf("could not get deployments: %v", err))
	}
	defer resp.Body.Close()

	if 200 < resp.StatusCode && resp.StatusCode >= 300 {
		formattedErrors, err := cl.DecodeJSONAPIErrors(resp)
		if err != nil {
			return errors.NewInternalError(ctx,
				fmt.Errorf("could not decode JSON formatted errors returned while listing deployments: %v", err))
		}
		for _, e := range formattedErrors.Errors {
			log.Info(ctx, map[string]interface{}{
				"status_code":     resp.StatusCode,
				"formatted_error": *e,
			}, "deleting openshift resources failed")

		}
		if len(formattedErrors.Errors) > 0 {
			return errors.NewInternalError(ctx, errs.Errorf(formattedErrors.Errors[0].Detail))
		}
		return errors.NewInternalError(ctx, errs.Errorf("unknown error"))
	}
	space, err := cl.DecodeSimpleSpaceSingle(resp)
	if err != nil {
		return errors.NewInternalError(ctx,
			fmt.Errorf("could not decode deployments: %v", err))
	}

	// iterate over all the applications
	var errorsList []error
	for _, app := range space.Data.Attributes.Applications {
		for _, env := range app.Attributes.Deployments {
			path = client.DeleteDeploymentDeploymentsPath(
				spaceID,
				app.Attributes.Name,
				env.Attributes.Name,
			)
			resp, err = cl.DeleteDeploymentDeployments(ctx, path)
			if err != nil {
				errorsList = append(errorsList,
					errs.Wrapf(err, "could not delete deployment for space=%s, app=%s, env=%s", spaceID, app.Attributes.Name, env.Attributes.Name))
				continue
			}
			if 200 < resp.StatusCode && resp.StatusCode >= 300 {
				formattedErrors, err := cl.DecodeJSONAPIErrors(resp)
				if err != nil {
					errorsList = append(errorsList,
						errs.Wrapf(err, "could not decode JSON formatted errors returned while deleting deployment for space=%s, app=%s, env=%s", spaceID, app.Attributes.Name, env.Attributes.Name))
					continue
				}
				if len(formattedErrors.Errors) > 0 {
					errorsList = append(errorsList, errs.Errorf(formattedErrors.Errors[0].Detail))
				}
			}
		}
	}

	if len(errorsList) != 0 {
		var errString string
		for _, err = range errorsList {
			errString += fmt.Sprintf("%s\n", err)
		}
		return errors.NewInternalErrorFromString(errString)
	}

	//delete pipelines from the space
	log.Debug(ctx, map[string]interface{}{"label": spaceID}, "deleting pipelines in")
	resp, err = cl.DeletePipelines(ctx, client.DeletePipelinesPath(spaceID))

	if err != nil {
		log.Error(ctx, nil, fmt.Sprintf("error occurred while deleting pipelines from space : %v", spaceID))
	}

	return nil
}

// List runs the list action.
func (c *SpaceController) List(ctx *app.ListSpaceContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)

	var response app.SpaceList
	txnErr := application.Transactional(c.db, func(appl application.Application) error {
		spaces, cnt, err := appl.Spaces().List(ctx.Context, &offset, &limit)
		if err != nil {
			return err
		}
		entityErr := ctx.ConditionalEntities(spaces, c.config.GetCacheControlSpaces, func() error {
			count := int(cnt)
			spaceData, err := ConvertSpacesFromModel(ctx.Request, spaces, IncludeBacklogTotalCount(ctx.Context, c.db))
			if err != nil {
				return err
			}
			response = app.SpaceList{
				Links: &app.PagingLinks{},
				Meta:  &app.SpaceListMeta{TotalCount: count},
				Data:  spaceData,
			}
			setPagingLinks(response.Links, buildAbsoluteURL(ctx.Request), len(spaces), offset, limit, count)
			return nil
		})
		if entityErr != nil {
			return entityErr
		}

		return nil
	})
	if txnErr != nil {
		return jsonapi.JSONErrorResponse(ctx, txnErr)
	}
	return ctx.OK(&response)
}

// Show runs the show action.
func (c *SpaceController) Show(ctx *app.ShowSpaceContext) error {
	var s *space.Space
	err := application.Transactional(c.db, func(appl application.Application) error {
		var err error
		s, err = appl.Spaces().Load(ctx.Context, ctx.SpaceID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err":      err,
				"space_id": ctx.SpaceID,
			}, "unable to load the space by ID")
		}
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.ConditionalRequest(*s, c.config.GetCacheControlSpace, func() error {
		spaceData, err := ConvertSpaceFromModel(ctx.Request, *s, IncludeBacklogTotalCount(ctx.Context, c.db))
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err":      err,
				"space_id": ctx.SpaceID,
			}, "unable to convert the space object")
			return err
		}
		result := &app.SpaceSingle{
			Data: spaceData,
		}
		return ctx.OK(result)
	})
}

// Update runs the update action.
func (c *SpaceController) Update(ctx *app.UpdateSpaceContext) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	err = validateUpdateSpace(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	var s *space.Space
	err = application.Transactional(c.db, func(appl application.Application) error {
		var err error
		s, err = appl.Spaces().Load(ctx.Context, ctx.SpaceID)
		if err != nil {
			return err
		}

		if !uuid.Equal(*currentUser, s.OwnerID) {
			log.Error(ctx, map[string]interface{}{"currentUser": *currentUser, "owner": s.OwnerID}, "Current user is not owner")
			return goa.NewErrorClass("forbidden", 403)("User is not the space owner")
		}

		s.Version = *ctx.Payload.Data.Attributes.Version
		if ctx.Payload.Data.Attributes.Name != nil {
			s.Name = *ctx.Payload.Data.Attributes.Name
		}
		if ctx.Payload.Data.Attributes.Description != nil {
			s.Description = *ctx.Payload.Data.Attributes.Description
		}

		s, err = appl.Spaces().Save(ctx.Context, s)
		return err

	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	spaceData, err := ConvertSpaceFromModel(ctx.Request, *s, IncludeBacklogTotalCount(ctx.Context, c.db))
	if err != nil {
		return err
	}
	response := app.SpaceSingle{
		Data: spaceData,
	}
	return ctx.OK(&response)
}

func validateCreateSpace(ctx *app.CreateSpaceContext) error {
	if ctx.Payload.Data == nil {
		return errors.NewBadParameterError("data", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes == nil {
		return errors.NewBadParameterError("data.attributes", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Name == nil {
		return errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil")
	}
	// // TODO(kwk): Comment back in once space template is official
	// if ctx.Payload.Data.Relationships == nil {
	// 	return errors.NewBadParameterError("data.relationships", nil).Expected("not nil")
	// }
	// if ctx.Payload.Data.Relationships.SpaceTemplate == nil {
	// 	return errors.NewBadParameterError("data.relationships.spacetemplate", nil).Expected("not nil")
	// }
	// if ctx.Payload.Data.Relationships.SpaceTemplate.Data == nil {
	// 	return errors.NewBadParameterError("data.relationships.spacetemplate.data", nil).Expected("not nil")
	// }
	return nil
}

func validateUpdateSpace(ctx *app.UpdateSpaceContext) error {
	if ctx.Payload.Data == nil {
		return errors.NewBadParameterError("data", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes == nil {
		return errors.NewBadParameterError("data.attributes", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Name == nil {
		return errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Version == nil {
		return errors.NewBadParameterError("data.attributes.version", nil).Expected("not nil")
	}
	return nil
}

// ConvertSpaceToModel converts an `app.Space` to a `space.Space`
func ConvertSpaceToModel(appSpace app.Space) space.Space {
	modelSpace := space.Space{}

	if appSpace.ID != nil {
		modelSpace.ID = *appSpace.ID
	}
	if appSpace.Attributes != nil {
		if appSpace.Attributes.CreatedAt != nil {
			modelSpace.CreatedAt = *appSpace.Attributes.CreatedAt
		}
		if appSpace.Attributes.UpdatedAt != nil {
			modelSpace.UpdatedAt = *appSpace.Attributes.UpdatedAt
		}
		if appSpace.Attributes.Version != nil {
			modelSpace.Version = *appSpace.Attributes.Version
		}
		if appSpace.Attributes.Name != nil {
			modelSpace.Name = *appSpace.Attributes.Name
		}
		if appSpace.Attributes.Description != nil {
			modelSpace.Description = *appSpace.Attributes.Description
		}
	}
	if appSpace.Relationships != nil && appSpace.Relationships.OwnedBy != nil &&
		appSpace.Relationships.OwnedBy.Data != nil && appSpace.Relationships.OwnedBy.Data.ID != nil {
		modelSpace.OwnerID = *appSpace.Relationships.OwnedBy.Data.ID
	}
	if appSpace.Relationships != nil && appSpace.Relationships.SpaceTemplate != nil &&
		appSpace.Relationships.SpaceTemplate.Data != nil {
		modelSpace.SpaceTemplateID = appSpace.Relationships.SpaceTemplate.Data.ID
	}
	return modelSpace
}

// SpaceConvertFunc is a open ended function to add additional links/data/relations to a Space during
// conversion from internal to API
type SpaceConvertFunc func(*http.Request, *space.Space, *app.Space) error

// IncludeBacklog returns a SpaceConvertFunc that includes the a link to the backlog
// along with the total count of items in the backlog of the current space
func IncludeBacklogTotalCount(ctx context.Context, db application.DB) SpaceConvertFunc {
	return func(req *http.Request, modelSpace *space.Space, appSpace *app.Space) error {
		count, err := countBacklogItems(ctx, db, modelSpace.ID)
		if err != nil {
			return errs.Wrap(err, "unable to count backlog items")
		}
		appSpace.Links.Backlog.Meta = &app.BacklogLinkMeta{TotalCount: count} // TODO (xcoulon) remove that part
		appSpace.Relationships.Backlog.Meta = map[string]interface{}{"totalCount": count}
		return nil
	}
}

// ConvertSpacesFromModel converts between internal and external REST representation
func ConvertSpacesFromModel(request *http.Request, spaces []space.Space, additional ...SpaceConvertFunc) ([]*app.Space, error) {
	var result = make([]*app.Space, len(spaces))
	for i, p := range spaces {
		spaceData, err := ConvertSpaceFromModel(request, p, additional...)
		if err != nil {
			return nil, err
		}
		result[i] = spaceData
	}
	return result, nil
}

// ConvertSpaceFromModel converts between internal and external REST representation
func ConvertSpaceFromModel(request *http.Request, sp space.Space, options ...SpaceConvertFunc) (*app.Space, error) {
	selfURL := rest.AbsoluteURL(request, app.SpaceHref(sp.ID))
	spaceIDStr := sp.ID.String()
	relatedIterations := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/iterations", spaceIDStr))
	relatedAreas := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/areas", spaceIDStr))
	relatedBacklog := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/backlog", spaceIDStr))
	relatedCodebases := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/codebases", spaceIDStr))
	relatedWorkItems := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/workitems", spaceIDStr))
	relatedWorkItemTypes := rest.AbsoluteURL(request, app.SpaceTemplateHref(sp.SpaceTemplateID)+"/workitemtypes")
	relatedWorkItemLinkTypes := rest.AbsoluteURL(request, app.SpaceTemplateHref(sp.SpaceTemplateID)+"/workitemlinktypes")
	relatedOwners := rest.AbsoluteURL(request, app.UsersHref(sp.OwnerID.String()))
	relatedCollaborators := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/collaborators", spaceIDStr))
	relatedFilters := rest.AbsoluteURL(request, "/api/filters")
	relatedLabels := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/labels", spaceIDStr))
	relatedWorkitemTypeGroups := rest.AbsoluteURL(request, app.SpaceTemplateHref(sp.SpaceTemplateID)+"/workitemtypegroups")
	relatedSpaceTemplateURL := rest.AbsoluteURL(request, app.SpaceTemplateHref(sp.SpaceTemplateID))

	s := &app.Space{
		ID:   &sp.ID,
		Type: APIStringTypeSpace,
		Attributes: &app.SpaceAttributes{
			Name:        &sp.Name,
			Description: &sp.Description,
			CreatedAt:   &sp.CreatedAt,
			UpdatedAt:   &sp.UpdatedAt,
			Version:     &sp.Version,
		},
		Links: &app.GenericLinksForSpace{
			Self:    &selfURL,
			Related: &selfURL, //TODO (xcoulon): remove this link
			Backlog: &app.BacklogGenericLink{ //TODO (xcoulon): remove this link
				Self: &relatedBacklog,
			},
			Workitemtypes:     &relatedWorkItemTypes,     //TODO (xcoulon): remove this link
			Workitemlinktypes: &relatedWorkItemLinkTypes, //TODO (xcoulon): remove this link
			Filters:           &relatedFilters,           //TODO (xcoulon): remove this link
		},
		Relationships: &app.SpaceRelationships{
			Areas: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedAreas,
				},
			},
			Backlog: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedBacklog,
				},
			},
			Codebases: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedCodebases,
				},
			},
			Collaborators: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedCollaborators,
				},
			},
			Filters: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedFilters,
				},
			},
			OwnedBy: &app.SpaceOwnedBy{
				Data: &app.IdentityRelationData{
					Type: "identities",
					ID:   &sp.OwnerID,
				},
				Links: &app.GenericLinks{
					Related: &relatedOwners,
				},
			},
			Iterations: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedIterations,
				},
			},
			Labels: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedLabels,
				},
			},
			Workitems: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedWorkItems,
				},
			},
			Workitemtypes: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedWorkItemTypes,
				},
			},
			Workitemlinktypes: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedWorkItemLinkTypes,
				},
			},
			Workitemtypegroups: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedWorkitemTypeGroups,
				},
			},
			SpaceTemplate: app.NewSpaceTemplateRelation(sp.SpaceTemplateID, relatedSpaceTemplateURL),
		},
	}
	// apply options (ie, if extra content needs to be provided in the response element)
	for _, option := range options {
		err := option(request, &sp, s)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}
