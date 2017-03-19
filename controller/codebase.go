package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/codebase"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/rest"
	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

const (
	// APIStringTypeCodebase contains the JSON API type for codebases
	APIStringTypeCodebase = "codebases"
	// APIStringTypeWorkspace contains the JSON API type for worksapces
	APIStringTypeWorkspace = "workspaces"
)

// CodebaseController implements the codebase resource.
type CodebaseController struct {
	*goa.Controller
	db application.DB
}

// NewCodebaseController creates a codebase controller.
func NewCodebaseController(service *goa.Service, db application.DB) *CodebaseController {
	return &CodebaseController{Controller: service.NewController("CodebaseController"), db: db}
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

// Edit runs the show action.
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
	che := NewCheStarterClient(getClusterURL(), getNamespace(ctx))
	workspaces, err := che.ListWorkspaces(ctx, cb.URL)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"codebaseID": cb.ID,
			"err":        err,
		}, "unable fetch list of workspaces")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}

	var existingWorkspaces []*app.Workspace
	for _, workspace := range workspaces {
		openLink := rest.AbsoluteURL(ctx.RequestData, fmt.Sprintf(app.CodebaseHref(cb.ID)+"/open/%v", workspace.ID))
		existingWorkspaces = append(existingWorkspaces, &app.Workspace{
			Attributes: &app.WorkspaceAttributes{
				Name:        &workspace.Name,
				Description: &workspace.Description,
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
	che := NewCheStarterClient(getClusterURL(), getNamespace(ctx))
	workspace := WorkspaceRequest{
		Branch:     "master",
		Name:       "test2",
		StackID:    "java-default",
		Repository: cb.URL,
	}
	workspaceResp, err := che.CreateWorkspace(ctx, workspace)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"codebaseID": cb.ID,
			"err":        err,
		}, "unable to create workspaces")
		if werr, ok := err.(*workspaceError); ok {
			fmt.Println(werr.String())
		}
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}

	resp := &app.WorkspaceOpen{
		Links: &app.WorkspaceOpenLinks{
			Open: &workspaceResp.WorkspaceIDEURL,
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
	che := NewCheStarterClient(getClusterURL(), getNamespace(ctx))
	workspace := WorkspaceRequest{
		ID: ctx.WorkspaceID,
	}
	workspaceResp, err := che.CreateWorkspace(ctx, workspace)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"codebaseID": cb.ID,
			"err":        err,
		}, "unable to open workspaces")
		if werr, ok := err.(*workspaceError); ok {
			fmt.Println(werr.String())
		}
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}

	resp := &app.WorkspaceOpen{
		Links: &app.WorkspaceOpenLinks{
			Open: &workspaceResp.WorkspaceIDEURL,
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
		return strings.Split(email, "@")[0] + "-dsaas-che"
	}
	return ""
}

func getClusterURL() *url.URL {
	clusterURL, _ := url.Parse("https://tsrv.devshift.net:8443")
	return clusterURL
}

// NewCheStarterClient is a helper function to create a new CheStarter client
// Uses http.DefaultClient
func NewCheStarterClient(masterURL *url.URL, namespace string) *CheStarterClient {
	return &CheStarterClient{masterURL: masterURL, namespace: namespace, client: http.DefaultClient}
}

// CheStarterClient describes the REST interface between Platform and Che Starter
type CheStarterClient struct {
	masterURL *url.URL
	namespace string
	client    *http.Client
}

func (cs *CheStarterClient) targetURL(resource string) string {
	return fmt.Sprintf("http://che-starter.prod-preview.openshift.io/%v?masterUrl=%v&namespace=%v", resource, cs.masterURL.String(), cs.namespace)
}

func (cs *CheStarterClient) setHeaders(ctx context.Context, req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+goajwt.ContextJWT(ctx).Raw)
	req.Header.Set(middleware.RequestIDHeader, middleware.ContextRequestID(ctx))
}

// ListWorkspaces lists the available workspaces for a given user
func (cs *CheStarterClient) ListWorkspaces(ctx context.Context, repository string) ([]WorkspaceResponse, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(cs.targetURL("workspace")+"&repository=%v", repository), nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"repository": repository,
			"err":        err,
		}, "failed to create request object")
		return nil, err
	}
	cs.setHeaders(ctx, req)

	if log.IsDebug() {
		b, _ := httputil.DumpRequest(req, true)
		log.Debug(ctx, map[string]interface{}{
			"request": string(b),
		}, "request object")
	}

	resp, err := cs.client.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"repository": repository,
			"err":        err,
		}, "failed to list workspace for repository")
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		workspaceErr := workspaceError{}
		err = json.NewDecoder(resp.Body).Decode(&workspaceErr)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"repository": repository,
				"err":        err,
			}, "failed to decode error response from list workspace for repository")
			return nil, err
		}
		log.Error(ctx, map[string]interface{}{
			"repository": repository,
			"err":        workspaceErr.String(),
		}, "failed to execute list workspace for repository")
		return nil, &workspaceErr
	}

	workspaceResp := []WorkspaceResponse{}
	err = json.NewDecoder(resp.Body).Decode(&workspaceResp)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"repository": repository,
			"err":        err,
		}, "failed to decode response from list workspace for repository")
		return nil, err
	}
	return workspaceResp, nil
}

// CreateWorkspace creates a new Che Workspace based on a repository
func (cs *CheStarterClient) CreateWorkspace(ctx context.Context, workspace WorkspaceRequest) (*WorkspaceResponse, error) {
	body, err := json.Marshal(&workspace)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"workspaceID": workspace.ID,
			"err":         err,
		}, "failed to create request object")
		return nil, err
	}

	req, err := http.NewRequest("POST", cs.targetURL("workspace"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	cs.setHeaders(ctx, req)

	if log.IsDebug() {
		b, _ := httputil.DumpRequest(req, true)
		log.Debug(ctx, map[string]interface{}{
			"request": string(b),
		}, "request object")
	}

	resp, err := cs.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		workspaceErr := workspaceError{}
		err = json.NewDecoder(resp.Body).Decode(&workspaceErr)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"workspaceID": workspace.ID,
				"err":         err,
			}, "failed to decode error response from create workspace for repository")
			return nil, err
		}
		log.Error(ctx, map[string]interface{}{
			"workspaceID": workspace.ID,
			"err":         workspaceErr.String(),
		}, "failed to execute create workspace for repository")
		return nil, &workspaceErr
	}

	workspaceResp := WorkspaceResponse{}
	err = json.NewDecoder(resp.Body).Decode(&workspaceResp)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"workspaceID": workspace.ID,
			"err":         err,
		}, "failed to decode response from create workspace for repository")
		return nil, err
	}
	return &workspaceResp, nil
}

// WorkspaceRequest represents a create workspace request body
type WorkspaceRequest struct {
	ID         string `json:"id,omitempty"`
	Branch     string `json:"branch,omitempty"`
	Name       string `json:"name,omitempty"`
	Repository string `json:"repo,omitempty"`
	StackID    string `json:"stack,omitempty"`
}

// WorkspaceResponse represents a create workspace response body
type WorkspaceResponse struct {
	ID              string `json:"id"`
	Branch          string `json:"branch"`
	Description     string `json:"description"`
	Location        string `json:"location"`
	Login           string `json:"login"`
	Name            string `json:"name"`
	Repository      string `json:"repository"`
	Status          string `json:"status"`
	WorkspaceIDEURL string `json:"workspaceIdeUrl"`
}

type workspaceError struct {
	Status    int    `json:"status"`
	ErrorMsg  string `json:"error"`
	Message   string `json:"message"`
	Timestamp string `json:"timeStamp"`
	Trace     string `json:"trace"`
}

func (err *workspaceError) Error() string {
	return err.ErrorMsg
}

func (err *workspaceError) String() string {
	return fmt.Sprintf("Status %v Error %v Message %v Trace\n%v", err.Status, err.ErrorMsg, err.ErrorMsg, err.Trace)
}
