package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/codebase"
	"github.com/almighty/almighty-core/codebase/che"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/rest"

	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
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

// CodebaseController implements the codebase resource.
type CodebaseController struct {
	*goa.Controller
	db     application.DB
	config codebaseConfiguration
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
		res.Data = ConvertCodebase(ctx.RequestData, c)

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
	cheClient := che.NewStarterClient(c.config.GetCheStarterURL(), c.config.GetOpenshiftTenantMasterURL(), getNamespace(ctx))
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
		openLink := rest.AbsoluteURL(ctx.RequestData, fmt.Sprintf(app.CodebaseHref(cb.ID)+"/open/%v", workspace.ID))
		existingWorkspaces = append(existingWorkspaces, &app.Workspace{
			Attributes: &app.WorkspaceAttributes{
				Name:        &workspace.ID,
				Description: &workspace.Status,
			},
			Links: &app.WorkspaceLinks{
				Open: &openLink,
			},
		})
	}

	createLink := rest.AbsoluteURL(ctx.RequestData, app.CodebaseHref(cb.ID)+"/create")
	resp := &app.WorkspaceList{
		Data: existingWorkspaces,
		Links: &app.WorkspaceEditLinks{
			Create: &createLink,
		},
	}

	return ctx.OK(resp)
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
	cheClient := che.NewStarterClient(c.config.GetCheStarterURL(), c.config.GetOpenshiftTenantMasterURL(), getNamespace(ctx))
	workspace := che.WorkspaceRequest{
		Branch:     "master",
		Name:       "test2",
		StackID:    "java-default",
		Repository: cb.URL,
	}
	workspaceResp, err := cheClient.CreateWorkspace(ctx, workspace)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"codebase_id": cb.ID,
			"err":         err,
		}, "unable to create workspaces")
		if werr, ok := err.(*che.WorkspaceError); ok {
			fmt.Println(werr.String())
		}
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}

	resp := &app.WorkspaceOpen{
		Links: &app.WorkspaceOpenLinks{
			Open: &workspaceResp.HRef,
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
	cheClient := che.NewStarterClient(c.config.GetCheStarterURL(), c.config.GetOpenshiftTenantMasterURL(), getNamespace(ctx))
	workspace := che.WorkspaceRequest{
		ID: ctx.WorkspaceID,
	}
	workspaceResp, err := cheClient.CreateWorkspace(ctx, workspace)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"codebase_id": cb.ID,
			"err":         err,
		}, "unable to open workspaces")
		if werr, ok := err.(*che.WorkspaceError); ok {
			fmt.Println(werr.String())
		}
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}

	resp := &app.WorkspaceOpen{
		Links: &app.WorkspaceOpenLinks{
			Open: &workspaceResp.HRef,
		},
	}
	return ctx.OK(resp)
}

// CodebaseConvertFunc is a open ended function to add additional links/data/relations to a Codebase during
// convertion from internal to API
type CodebaseConvertFunc func(*goa.RequestData, *codebase.Codebase, *app.Codebase)

// ConvertCodebases converts between internal and external REST representation
func ConvertCodebases(request *goa.RequestData, codebases []*codebase.Codebase, additional ...CodebaseConvertFunc) []*app.Codebase {
	var is = []*app.Codebase{}
	for _, i := range codebases {
		is = append(is, ConvertCodebase(request, i, additional...))
	}
	return is
}

// ConvertCodebase converts between internal and external REST representation
func ConvertCodebase(request *goa.RequestData, codebase *codebase.Codebase, additional ...CodebaseConvertFunc) *app.Codebase {
	codebaseType := APIStringTypeCodebase
	spaceType := APIStringTypeSpace

	spaceID := codebase.SpaceID.String()

	selfURL := rest.AbsoluteURL(request, app.CodebaseHref(codebase.ID))
	editURL := rest.AbsoluteURL(request, app.CodebaseHref(codebase.ID)+"/edit")
	spaceSelfURL := rest.AbsoluteURL(request, app.SpaceHref(spaceID))

	i := &app.Codebase{
		Type: codebaseType,
		ID:   &codebase.ID,
		Attributes: &app.CodebaseAttributes{
			CreatedAt: &codebase.CreatedAt,
			Type:      &codebase.Type,
			URL:       &codebase.URL,
		},
		Relationships: &app.CodebaseRelations{
			Space: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: &spaceType,
					ID:   &spaceID,
				},
				Links: &app.GenericLinks{
					Self: &spaceSelfURL,
				},
			},
		},
		Links: &app.CodebaseLinks{
			Self: &selfURL,
			Edit: &editURL,
		},
	}
	for _, add := range additional {
		add(request, codebase, i)
	}
	return i
}

// TODO: We need to dynamically get the real che namespace name from the tenant namespace from
// somewhere more sensible then the token/generate/guess route.
func getNamespace(ctx context.Context) string {
	token := goajwt.ContextJWT(ctx)
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		email := claims["email"].(string)
		return strings.Replace(strings.Split(email, "@")[0], ".", "-", -1) + "-che"
	}
	return ""
}
