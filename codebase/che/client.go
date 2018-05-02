package che

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/goadesign/goa/middleware"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	errs "github.com/pkg/errors"
)

// Client the interface for remote operations on Che
type Client interface {
	CreateWorkspace(ctx context.Context, workspace WorkspaceRequest) (*WorkspaceResponse, error)
	ListWorkspaces(ctx context.Context, repository string) ([]*WorkspaceResponse, error)
	DeleteWorkspace(ctx context.Context, workspaceName string) error
	StartExistingWorkspace(ctx context.Context, workspaceName string) (*WorkspaceResponse, error)
	GetCheServerState(ctx context.Context) (*ServerStateResponse, error)
	StartCheServer(ctx context.Context) (*ServerStateResponse, error)
}

// NewStarterClient is a helper function to create a new CheStarter client
// Uses http.DefaultClient
func NewStarterClient(cheStarterURL, openshiftMasterURL string, namespace string, client *http.Client) Client {
	return &StarterClient{cheStarterURL: cheStarterURL, openshiftMasterURL: openshiftMasterURL, namespace: namespace, client: client}
}

// StarterClient describes the REST interface between Platform and Che Starter
type StarterClient struct {
	cheStarterURL      string
	openshiftMasterURL string
	namespace          string
	client             *http.Client
}

func (cs *StarterClient) targetURL(resource string) string {
	return fmt.Sprintf("%v/%v?masterUrl=%v&namespace=%v", cs.cheStarterURL, resource, cs.openshiftMasterURL, cs.namespace)
}

func (cs *StarterClient) setHeaders(ctx context.Context, req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+goajwt.ContextJWT(ctx).Raw)
	req.Header.Set(middleware.RequestIDHeader, middleware.ContextRequestID(ctx))
}

// ListWorkspaces lists the available workspaces for a given user
func (cs *StarterClient) ListWorkspaces(ctx context.Context, repository string) ([]*WorkspaceResponse, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(cs.targetURL("workspace")+"&repository=%v", repository), nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"repository": repository,
			"err":        err,
		}, "failed to create request object")
		return nil, errs.WithStack(err)
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
		return nil, errs.WithStack(err)
	}

	defer rest.CloseResponse(resp)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		workspaceErr := StarterError{}
		err = json.NewDecoder(resp.Body).Decode(&workspaceErr)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"repository": repository,
				"err":        err,
			}, "failed to decode error response from list workspace for repository")
			return nil, errs.WithStack(err)
		}
		log.Error(ctx, map[string]interface{}{
			"repository": repository,
			"err":        workspaceErr.String(),
		}, "failed to execute list workspace for repository")
		return nil, errs.WithStack(workspaceErr)
	}

	workspaceResp := []*WorkspaceResponse{}
	err = json.NewDecoder(resp.Body).Decode(&workspaceResp)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"repository": repository,
			"err":        err,
		}, "failed to decode response from list workspace for repository")
		return nil, errs.WithStack(err)
	}
	return workspaceResp, nil
}

// CreateWorkspace creates a new Che Workspace based on a repository
func (cs *StarterClient) CreateWorkspace(ctx context.Context, workspace WorkspaceRequest) (*WorkspaceResponse, error) {
	body, err := json.Marshal(&workspace)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"workspace_id":       workspace.Name,
			"workspace_stack_id": workspace.StackID,
			"workspace":          workspace,
			"err":                err,
		}, "failed to create request object")
		return nil, errs.WithStack(err)
	}

	req, err := http.NewRequest("POST", cs.targetURL("workspace"), bytes.NewReader(body))
	if err != nil {
		return nil, errs.WithStack(err)
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
		return nil, errs.WithStack(err)
	}

	defer rest.CloseResponse(resp)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		workspaceErr := StarterError{}
		err = json.NewDecoder(resp.Body).Decode(&workspaceErr)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"workspace_id":       workspace.Name,
				"workspace_stack_id": workspace.StackID,
				"workspace":          workspace,
				"err":                err,
			}, "failed to decode error response from create workspace for repository")
			return nil, errs.WithStack(err)
		}
		log.Error(ctx, map[string]interface{}{
			"workspace_id":       workspace.Name,
			"workspace_stack_id": workspace.StackID,
			"workspace":          workspace,
			"err":                workspaceErr.String(),
		}, "failed to execute create workspace for repository")
		return nil, errs.WithStack(workspaceErr)
	}

	workspaceResp := WorkspaceResponse{}
	err = json.NewDecoder(resp.Body).Decode(&workspaceResp)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"workspace_id":       workspace.Name,
			"workspace_stack_id": workspace.StackID,
			"workspace":          workspace,
			"err":                err,
		}, "failed to decode response from create workspace for repository")
		return nil, errs.WithStack(err)
	}
	return &workspaceResp, nil
}

// DeleteWorkspace deletes a Che Workspace by its name
func (cs *StarterClient) DeleteWorkspace(ctx context.Context, workspaceName string) error {
	req, err := http.NewRequest("DELETE", cs.targetURL(fmt.Sprintf("workspace/%s", workspaceName)), nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"name":      workspaceName,
			"masterURL": cs.cheStarterURL,
			"namespace": cs.namespace,
			"err":       err,
		}, "failed to create request object")
		return errs.WithStack(err)
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
			"name":      workspaceName,
			"masterURL": cs.cheStarterURL,
			"namespace": cs.namespace,
			"err":       err,
		}, "failed to delete workspace")
		return errs.WithStack(err)
	}

	defer rest.CloseResponse(resp)

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		workspaceErr := StarterError{}
		err = json.NewDecoder(resp.Body).Decode(&workspaceErr)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"name":      workspaceName,
				"masterURL": cs.cheStarterURL,
				"namespace": cs.namespace,
				"err":       err,
			}, "failed to decode error response from list workspace for repository")
			return errs.WithStack(err)
		}
		log.Error(ctx, map[string]interface{}{
			"name":      workspaceName,
			"masterURL": cs.cheStarterURL,
			"namespace": cs.namespace,
			"err":       workspaceErr.String(),
		}, "failed to delete workspace")
		return errs.WithStack(workspaceErr)
	}

	return nil
}

// StartExistingWorkspace starts an existing Che Workspace based on a repository
func (cs *StarterClient) StartExistingWorkspace(ctx context.Context, workspaceName string) (*WorkspaceResponse, error) {
	log.Debug(ctx, map[string]interface{}{
		"workspace_id": workspaceName,
	}, "starting an existing workspace")

	req, err := http.NewRequest("PATCH", cs.targetURL(fmt.Sprintf("workspace/%s", workspaceName)), nil)
	if err != nil {
		return nil, errs.WithStack(err)
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
		return nil, errs.WithStack(err)
	}

	defer rest.CloseResponse(resp)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		workspaceErr := StarterError{}
		err = json.NewDecoder(resp.Body).Decode(&workspaceErr)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"workspace_id": workspaceName,
				"err":          err,
			}, "failed to decode error response from starting an existing workspace for repository")
			return nil, errs.WithStack(err)
		}
		log.Error(ctx, map[string]interface{}{
			"workspace_id": workspaceName,
			"err":          workspaceErr.String(),
		}, "failed to execute start existing workspace for repository")
		return nil, errs.WithStack(workspaceErr)
	}

	workspaceResp := WorkspaceResponse{}
	err = json.NewDecoder(resp.Body).Decode(&workspaceResp)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"workspace_id": workspaceName,
			"err":          err,
		}, "failed to decode response from starting an existing workspace for repository")
		return nil, errs.WithStack(err)
	}
	return &workspaceResp, nil
}

// GetCheServerState get che server state.
func (cs *StarterClient) GetCheServerState(ctx context.Context) (*ServerStateResponse, error) {
	req, err := http.NewRequest("GET", cs.targetURL("server"), nil)

	if err != nil {
		return nil, errs.WithStack(err)
	}

	cs.setHeaders(ctx, req)

	if log.IsDebug() {
		b, _ := httputil.DumpRequest(req, true)
		log.Debug(ctx, map[string]interface{}{
			"request": string(b),
		}, "dump of the request to get the che server state")
	}

	resp, err := cs.client.Do(req)

	if err != nil {
		return nil, errs.WithStack(err)
	}

	defer rest.CloseResponse(resp)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		statusErr := StarterError{}
		err = json.NewDecoder(resp.Body).Decode(&statusErr)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err": err,
			}, "failed to decode error response from get che server state")
			return nil, errs.WithStack(err)
		}
		log.Error(ctx, map[string]interface{}{
			"err": statusErr.String(),
		}, "failed to execute get che server state")
		return nil, errs.WithStack(statusErr)
	}

	cheServerStateResponse := ServerStateResponse{}
	err = json.NewDecoder(resp.Body).Decode(&cheServerStateResponse)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "failed to decode response from getting che server state")
		return nil, errs.WithStack(err)
	}
	return &cheServerStateResponse, nil
}

// StartCheServer start che server if not running.
func (cs *StarterClient) StartCheServer(ctx context.Context) (*ServerStateResponse, error) {
	req, err := http.NewRequest("PATCH", cs.targetURL("server"), nil)

	if err != nil {
		return nil, errs.WithStack(err)
	}

	cs.setHeaders(ctx, req)

	if log.IsDebug() {
		b, _ := httputil.DumpRequest(req, true)
		log.Debug(ctx, map[string]interface{}{
			"request": string(b),
		}, "dump of the request to star the che server")
	}

	resp, err := cs.client.Do(req)

	if err != nil {
		return nil, errs.WithStack(err)
	}

	defer rest.CloseResponse(resp)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		statusErr := StarterError{}
		err = json.NewDecoder(resp.Body).Decode(&statusErr)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err": err,
			}, "failed to decode error response from start che server")
			return nil, errs.WithStack(err)
		}
		log.Error(ctx, map[string]interface{}{
			"err": statusErr.String(),
		}, "failed to execute start che server")
		return nil, errs.WithStack(statusErr)
	}

	cheServerStateResponse := ServerStateResponse{}
	err = json.NewDecoder(resp.Body).Decode(&cheServerStateResponse)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "failed to decode response from getting che server state endpoint")
		return nil, errs.WithStack(err)
	}
	return &cheServerStateResponse, nil
}

// WorkspaceRequest represents a create workspace request body
type WorkspaceRequest struct {
	Branch      string `json:"branch,omitempty"`
	Description string `json:"description,omitempty"`
	Name        string `json:"config.name,omitempty"`
	Repository  string `json:"repo,omitempty"`
	StackID     string `json:"stackId,omitempty"`
}

// WorkspaceResponse represents a create workspace response body
type WorkspaceResponse struct {
	ID          string          `json:"id,omitempty"`
	Description string          `json:"description,omitempty"`
	Config      WorkspaceConfig `json:"config,omitempty"`
	Status      string          `json:"status,omitempty"`
	Links       []WorkspaceLink `json:"links,omitempty"`
}

// WorkspaceConfig represents the workspace config
type WorkspaceConfig struct {
	Name     string             `json:"name"`
	Projects []WorkspaceProject `json:"projects,omitempty"`
}

// WorkspaceProject represents workspace project
type WorkspaceProject struct {
	Source ProjectSource `json:"source,omitempty"`
}

// ProjectSource represents project source of workspace
type ProjectSource struct {
	Location   string                  `json:"location,omitempty"`
	Parameters ProjectSourceParameters `json:"parameters,omitempty"`
}

// ProjectSourceParameters represent project source parameters e.g. branch
type ProjectSourceParameters struct {
	Branch string `json:"branch,omitempty"`
}

// GetHrefByRel return the 'href' of 'rel' of WorkspaceLink
// {
//   "href": "https://che.prod-preview.openshift.io/user/wksp-0dae",
//   "rel": "ide url",
//   "method": "GET"
// }
func (w WorkspaceResponse) GetHrefByRelOfWorkspaceLink(rel string) string {
	for _, l := range w.Links {
		if l.Rel == rel {
			return l.Href
		}
	}
	return ""
}

// Following const define commonly used WorkspaceLink rel
const (
	IdeUrlRel   = "ide url"
	SelfLinkRel = "self link"
)

// WorkspaceLink represents a URL for the location of a workspace
type WorkspaceLink struct {
	Href   string `json:"href"`
	Method string `json:"method"`
	Rel    string `json:"rel"`
}

// StarterError represent an error comming from the che-starter service
type StarterError struct {
	Status    int    `json:"status"`
	ErrorMsg  string `json:"error"`
	Message   string `json:"message"`
	Timestamp string `json:"timeStamp"`
	Trace     string `json:"trace"`
}

func (err StarterError) Error() string {
	return err.ErrorMsg
}

func (err StarterError) String() string {
	return fmt.Sprintf("Status %v Error %v Message %v Trace\n%v", err.Status, err.ErrorMsg, err.ErrorMsg, err.Trace)
}

// ServerStateResponse represents a get che state response body
type ServerStateResponse struct {
	Running     bool `json:"running"`
	MultiTenant bool `json:"multiTenant"`
}
