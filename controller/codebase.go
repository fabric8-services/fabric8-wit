package controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/codebase/che"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
)

const (
	// APIStringTypeCodebase contains the JSON API type for codebases
	APIStringTypeCodebase = "codebases"
	// APIStringTypeWorkspace contains the JSON API type for worksapces
	APIStringTypeWorkspace = "workspaces"
)

// CodebaseConfiguration contains the configuraiton required by this Controller
type codebaseConfiguration interface {
	GetOpenshiftTenantMasterURL() string
	GetCheStarterURL() string
}

// CodebaseCheClientProvider the function that provides a `cheClient`
type CodebaseCheClientProvider func(ctx context.Context, ns string) (che.Client, error)

// CodebaseController implements the codebase resource.
type CodebaseController struct {
	*goa.Controller
	db           application.DB
	config       codebaseConfiguration
	ShowTenant   account.CodebaseInitTenantProvider
	NewCheClient CodebaseCheClientProvider
}

// NewCodebaseController creates a codebase controller.
func NewCodebaseController(service *goa.Service, db application.DB, config codebaseConfiguration) *CodebaseController {
	return &CodebaseController{Controller: service.NewController("CodebaseController"), db: db, config: config}
}

// Show runs the show action.
func (c *CodebaseController) Show(ctx *app.ShowCodebaseContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		c, err := appl.Codebases().Load(ctx, ctx.CodebaseID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}

		res := &app.CodebaseSingle{}
		res.Data = ConvertCodebase(ctx.Request, *c)

		return ctx.OK(res)
	})
}

// Edit runs the edit action.
func (c *CodebaseController) Edit(ctx *app.EditCodebaseContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}

	var cb *codebase.Codebase

	err = application.Transactional(c.db, func(appl application.Application) error {
		cb, err = appl.Codebases().Load(ctx, ctx.CodebaseID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	ns, err := c.getCheNamespace(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	cheClient, err := c.NewCheClient(ctx, ns)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	workspaces, err := cheClient.ListWorkspaces(ctx, cb.URL)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"codebase_id": cb.ID,
			"err":         err,
		}, "unable fetch list of workspaces")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}

	var existingWorkspaces []*app.Workspace
	for _, workspace := range workspaces {
		openLink := rest.AbsoluteURL(ctx.Request, fmt.Sprintf(app.CodebaseHref(cb.ID)+"/open/%v", workspace.Config.Name))
		existingWorkspaces = append(existingWorkspaces, &app.Workspace{
			Attributes: &app.WorkspaceAttributes{
				Name:        &workspace.Config.Name,
				Description: &workspace.Status,
			},
			Type: "workspaces",
			Links: &app.WorkspaceLinks{
				Open: &openLink,
			},
		})
	}

	createLink := rest.AbsoluteURL(ctx.Request, app.CodebaseHref(cb.ID)+"/create")
	resp := &app.WorkspaceList{
		Data: existingWorkspaces,
		Links: &app.WorkspaceEditLinks{
			Create: &createLink,
		},
	}

	return ctx.OK(resp)
}

// Delete deletes the given codebase if the user is authenticated and authorized
func (c *CodebaseController) Delete(ctx *app.DeleteCodebaseContext) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	var cb *codebase.Codebase
	var cbSpace *space.Space
	err = application.Transactional(c.db, func(appl application.Application) error {
		var err error
		cb, err = appl.Codebases().Load(ctx.Context, ctx.CodebaseID)
		if err != nil {
			return err
		}
		cbSpace, err = appl.Spaces().Load(ctx.Context, cb.SpaceID)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	if !uuid.Equal(*currentUser, cbSpace.OwnerID) {
		log.Warn(ctx, map[string]interface{}{
			"codebase_id":  ctx.CodebaseID,
			"space_id":     cbSpace.ID,
			"space_owner":  cbSpace.OwnerID,
			"current_user": *currentUser,
		}, "user is not the space owner")
		return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not the space owner"))
	}
	// attempt to remotely delete the Che workspaces
	ns, err := c.getCheNamespace(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	cheClient, err := c.NewCheClient(ctx, ns)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	workspaces, err := cheClient.ListWorkspaces(ctx, cb.URL)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	log.Info(ctx, nil, "Found %d workspaces to delete", len(workspaces))
	for _, workspace := range workspaces {
		for _, link := range workspace.Links {
			if strings.ToLower(link.Method) == "delete" {
				log.Info(ctx,
					map[string]interface{}{"codebase_url": cb.URL,
						"che_namespace": ns,
						"workspace":     workspace.Config.Name,
					}, "About to delete Che workspace")
				err = cheClient.DeleteWorkspace(ctx.Context, workspace.Config.Name)
				if err != nil {
					log.Error(ctx,
						map[string]interface{}{
							"codebase_url":  cb.URL,
							"che_namespace": ns,
							"workspace":     workspace.Config.Name},
						"failed to delete Che workspace: %s", err.Error())
				}
			}
		}
	}

	// delete the local codebase data
	err = application.Transactional(c.db, func(appl application.Application) error {
		return appl.Codebases().Delete(ctx, ctx.CodebaseID)
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.NoContent()
}

// Create runs the create action.
func (c *CodebaseController) Create(ctx *app.CreateCodebaseContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	var cb *codebase.Codebase
	err = application.Transactional(c.db, func(appl application.Application) error {
		cb, err = appl.Codebases().Load(ctx, ctx.CodebaseID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	ns, err := c.getCheNamespace(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	cheClient, err := c.NewCheClient(ctx, ns)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}

	stackID := "java-centos"
	if cb.StackID != nil && *cb.StackID != "" {
		stackID = *cb.StackID
	}
	workspace := che.WorkspaceRequest{
		Branch:     "master",
		StackID:    stackID,
		Repository: cb.URL,
	}
	workspaceResp, err := cheClient.CreateWorkspace(ctx, workspace)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"codebase_id": cb.ID,
			"stack_id":    stackID,
			"err":         err,
		}, "unable to create workspaces")
		if werr, ok := err.(*che.StarterError); ok {
			log.Error(ctx, map[string]interface{}{
				"codebase_id": cb.ID,
				"stack_id":    stackID,
				"err":         err,
			}, "unable to create workspaces: %s", werr.String())
		}
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}

	err = application.Transactional(c.db, func(appl application.Application) error {
		cb.LastUsedWorkspace = workspaceResp.Config.Name
		_, err = appl.Codebases().Save(ctx, cb)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	ideURL := workspaceResp.GetIDEURL()
	resp := &app.WorkspaceOpen{
		Links: &app.WorkspaceOpenLinks{
			Open: &ideURL,
		},
	}
	return ctx.OK(resp)
}

// Open runs the open action.
func (c *CodebaseController) Open(ctx *app.OpenCodebaseContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	var cb *codebase.Codebase
	err = application.Transactional(c.db, func(appl application.Application) error {
		cb, err = appl.Codebases().Load(ctx, ctx.CodebaseID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	ns, err := c.getCheNamespace(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	cheClient, err := c.NewCheClient(ctx, ns)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	workspaceResp, err := cheClient.StartExistingWorkspace(ctx, ctx.WorkspaceID)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"codebase_id": cb.ID,
			"stack_id":    cb.StackID,
			"err":         err,
		}, "unable to open workspaces")
		if werr, ok := err.(*che.StarterError); ok {
			log.Error(ctx, map[string]interface{}{
				"codebase_id": cb.ID,
				"stack_id":    cb.StackID,
				"err":         err,
			}, "unable to open workspaces: %s", werr.String())
		}
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}

	err = application.Transactional(c.db, func(appl application.Application) error {
		cb.LastUsedWorkspace = ctx.WorkspaceID
		_, err = appl.Codebases().Save(ctx, cb)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		return nil
	})

	ideURL := workspaceResp.GetIDEURL()
	resp := &app.WorkspaceOpen{
		Links: &app.WorkspaceOpenLinks{
			Open: &ideURL,
		},
	}
	return ctx.OK(resp)
}

// CodebaseConvertFunc is a open ended function to add additional links/data/relations to a Codebase during
// convertion from internal to API
type CodebaseConvertFunc func(*http.Request, codebase.Codebase, *app.Codebase)

// ConvertCodebases converts between internal and external REST representation
func ConvertCodebases(request *http.Request, codebases []codebase.Codebase, options ...CodebaseConvertFunc) []*app.Codebase {
	result := make([]*app.Codebase, len(codebases))
	for i, c := range codebases {
		result[i] = ConvertCodebase(request, c, options...)
	}
	return result
}

// ConvertCodebase converts between internal and external REST representation
func ConvertCodebase(request *http.Request, codebase codebase.Codebase, options ...CodebaseConvertFunc) *app.Codebase {
	codebaseType := APIStringTypeCodebase
	spaceType := APIStringTypeSpace
	spaceID := codebase.SpaceID.String()
	relatedURL := rest.AbsoluteURL(request, app.CodebaseHref(codebase.ID))
	editURL := rest.AbsoluteURL(request, app.CodebaseHref(codebase.ID)+"/edit")
	spaceRelatedURL := rest.AbsoluteURL(request, app.SpaceHref(spaceID))

	result := &app.Codebase{
		Type: codebaseType,
		ID:   &codebase.ID,
		Attributes: &app.CodebaseAttributes{
			CreatedAt:         &codebase.CreatedAt,
			Type:              &codebase.Type,
			URL:               &codebase.URL,
			StackID:           codebase.StackID,
			LastUsedWorkspace: &codebase.LastUsedWorkspace,
		},
		Relationships: &app.CodebaseRelations{
			Space: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: &spaceType,
					ID:   &spaceID,
				},
				Links: &app.GenericLinks{
					Self:    &spaceRelatedURL,
					Related: &spaceRelatedURL,
				},
			},
		},
		Links: &app.CodebaseLinks{
			Self:    &relatedURL,
			Related: &relatedURL,
			Edit:    &editURL,
		},
	}
	for _, option := range options {
		option(request, codebase, result)
	}
	return result
}

// CheState gets che server state.
func (c *CodebaseController) CheState(ctx *app.CheStateCodebaseContext) error {
	ns, err := c.getCheNamespace(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	cheClient, err := c.NewCheClient(ctx, ns)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	cheState, err := cheClient.GetCheServerState(ctx)

	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get che server state")
		if werr, ok := err.(*che.StarterError); ok {
			log.Error(ctx, map[string]interface{}{
				"err": err,
			}, "unable to get che server state: %s", werr.String())
		}
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}

	isRunning := cheState.Running
	isMultiTenant := cheState.MultiTenant
	state := app.CheServerState{
		Running:     &isRunning,
		MultiTenant: &isMultiTenant,
	}
	return ctx.OK(&state)
}

// CheStart starts server if not running.
func (c *CodebaseController) CheStart(ctx *app.CheStartCodebaseContext) error {
	ns, err := c.getCheNamespace(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	cheClient, err := c.NewCheClient(ctx, ns)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	cheState, err := cheClient.StartCheServer(ctx)

	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to start che server")
		if werr, ok := err.(*che.StarterError); ok {
			log.Error(ctx, map[string]interface{}{
				"err": err,
			}, "unable to start che server: %s", werr.String())
		}
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}

	isRunning := cheState.Running
	isMultiTenant := cheState.MultiTenant
	state := app.CheServerState{
		Running:     &isRunning,
		MultiTenant: &isMultiTenant,
	}

	if isRunning {
		return ctx.OK(&state)
	}
	return ctx.Accepted(&state)
}

func (c *CodebaseController) getCheNamespace(ctx context.Context) (string, error) {
	t, err := c.ShowTenant(ctx)
	if err != nil {
		return "", err
	}
	if t.Data != nil && t.Data.Attributes != nil && t.Data.Attributes.Namespaces != nil {
		for _, ns := range t.Data.Attributes.Namespaces {
			if ns.Type != nil && *ns.Type == "che" && ns.Name != nil {
				return *ns.Name, nil
			}
		}
	}
	log.Error(ctx, map[string]interface{}{
		"data": t.Data,
	}, "unable to locate che namespace")

	return "", fmt.Errorf("unable to resolve user service che namespace")
}

// NewDefaultCheClient returns the default function to initialize a new Che client with a "regular" http client
func NewDefaultCheClient(config codebaseConfiguration) CodebaseCheClientProvider {
	return func(ctx context.Context, ns string) (che.Client, error) {
		cheClient := che.NewStarterClient(config.GetCheStarterURL(), config.GetOpenshiftTenantMasterURL(), ns, http.DefaultClient)
		return cheClient, nil
	}
}
