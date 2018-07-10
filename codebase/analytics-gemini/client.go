package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/fabric8-services/fabric8-wit/client"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/goasupport"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/rest"

	goaclient "github.com/goadesign/goa/client"
	"github.com/goadesign/goa/middleware"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	errs "github.com/pkg/errors"
)

// response is the struct which will be used to read the response
// from the analytics gemini service
type response struct {
	Summary string `json:"summary"`
	Error   string `json:"error"`
}

// request is the struct used to form the request to the analytics
// gemini service, which needs RepoURL to be scanned
type Request struct {
	GitURL string `json:"git-url"`
}

// NewScanRepoRequest returns a request object when the URL to
// repository to be scanned and email id of the user is given.
// This request object can be passed while registering or deregistering
// to or from the analytics gemini service respectively.
func NewScanRepoRequest(repoURL string) *Request {
	return &Request{
		GitURL: repoURL,
	}
}

// ScanRepoClient struct defines values that can be used to do
// request to the Analytics Gemini service, the Codebase search
// service and also you can mention if the developer mode is enabled
type ScanRepoClient struct {
	// Analytics Gemini Service URL and Client
	geminiURL    string
	geminiClient *http.Client

	// Codebase Search service URL and Client
	codebaseSearchURL    string
	codebaseSearchClient *http.Client

	// specify if the dev mode is enabled
	devMode bool
}

// NewScanRepoClient function returns the Analytics Gemini Service
// client that can be used to make request to register or deregister
// a codebase repository URL with the service. It takes in endpoint
// to the Gemini service and Client. Also it takes the endpoint to
// the Codebase search URL and client to it. You can also specify
// if the mode is developer so that these requests can be avoided.
func NewScanRepoClient(
	geminiURL string,
	geminiClient *http.Client,
	codebaseSearchURL string,
	codebaseSearchClient *http.Client,
	devMode bool,
) *ScanRepoClient {
	return &ScanRepoClient{
		// setting gemini resources
		geminiURL:    geminiURL,
		geminiClient: geminiClient,

		// setting codebase search service resources
		codebaseSearchURL:    codebaseSearchURL,
		codebaseSearchClient: codebaseSearchClient,

		// specify if the developer mode is enabled
		devMode: devMode,
	}
}

// setHeaders takes in the request object and sets headers like the
// content-type for request and response and the bearer token
func (sr *ScanRepoClient) setHeaders(ctx context.Context, req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	token := goajwt.ContextJWT(ctx)
	if token != nil {
		req.Header.Set("Authorization", "Bearer "+token.Raw)
	} else {
		log.Warn(ctx, map[string]interface{}{
			"err": "context has no JWT token",
		}, "")
	}

	req.Header.Set(middleware.RequestIDHeader, middleware.ContextRequestID(ctx))
}

func logErrors(ctx context.Context, d *Request, err error, bespokeError string) {
	log.Error(ctx, map[string]interface{}{
		"repoURL": d.GitURL,
		"err":     err,
	}, bespokeError)
}

// callGemini is a generic method that makes http call to the gemini
// service, here you can define if the call is to register or deregister
// using the 'path' attribute and check if the request was successful
// using the 'finalResponse' attribute
func (sr *ScanRepoClient) callGemini(
	ctx context.Context,
	d *Request,
	path string,
	finalResponse string,
) error {
	dBytes, err := json.Marshal(d)
	if err != nil {
		logErrors(ctx, d, err, "failed to marshal object into json")
		return errs.WithStack(err)
	}
	data := bytes.NewReader(dBytes)

	geminiURL, err := url.Parse(sr.geminiURL)
	if err != nil {
		return errs.WithStack(err)
	}
	geminiURL.Path = path

	req, err := http.NewRequest("POST", geminiURL.String(), data)
	if err != nil {
		logErrors(ctx, d, err, "failed to create request object")
		return errs.WithStack(err)
	}
	sr.setHeaders(ctx, req)

	if log.IsDebug() {
		b, _ := httputil.DumpRequest(req, true)
		log.Debug(ctx, map[string]interface{}{
			"request": string(b),
		}, "request object")
	}

	resp, err := sr.geminiClient.Do(req)
	if err != nil {
		logErrors(ctx, d, err, "failed to talk to analytics gemini service")
		return errs.WithStack(err)
	}
	defer rest.CloseResponse(resp)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logErrors(ctx, d, err, "")
		return errs.WithStack(err)
	}

	r := response{}
	if err := json.Unmarshal(body, &r); err != nil {
		logErrors(ctx, d, err, "failed to unmarshal the response")
		return errs.WithStack(err)
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("unauthorized access: %v", r.Error)
		default:
			return fmt.Errorf("unknown error, got response: %s", string(body))
		}
	}

	if r.Summary != finalResponse {
		return fmt.Errorf("got unknown response: %s", string(body))
	}

	return nil
}

// Register makes call to the analytics Gemini service which registers the
// given codebase URL for scanning of the CVEs in the code.
func (sr *ScanRepoClient) Register(ctx context.Context, d *Request) error {
	if sr.devMode {
		return nil
	}
	return sr.callGemini(ctx, d, "/api/v1/user-repo/scan", "Repository scan initiated")
}

// DeRegister unsubscribes you from the analytics Gemini service so that the
// scanning for CVEs will be disabled for this codebase. Call this method when
// the codebase is deleted from database and everywhere else.
func (sr *ScanRepoClient) DeRegister(ctx context.Context, d *Request) error {
	if sr.devMode {
		return nil
	}
	// first list all the records with this repoURL in the database
	codebases, err := sr.listCodebases(ctx, d)
	if err != nil {
		return err
	}

	// see if we should actually disable the scanning of the codebase
	// with the analytics service, if this returns true we don't disable
	// if it returns false then we disable
	if keepScanningThisCodebase(codebases) {
		return nil
	}

	// now deregister the repo from gemini server for scanning
	return sr.callGemini(ctx, d, "/api/v1/user-repo/drop", "Repository scan unsubscribed")
}

// keepScanningThisCodebase returns true if there is even one codebase
// which has 'cve-scan' set to true. If there are no such codebases
// then it returns false.
func keepScanningThisCodebase(codebases *client.CodebaseList) bool {
	var keepScanning bool

	for _, codebase := range codebases.Data {
		if *codebase.Attributes.CveScan == true {
			keepScanning = true
			break
		}
	}

	return keepScanning
}

// listCodebases makes a search call to the codebase service given
// the URL of the codebase
func (sr *ScanRepoClient) listCodebases(ctx context.Context, d *Request) (*client.CodebaseList, error) {
	u, err := url.Parse(sr.codebaseSearchURL)
	if err != nil {
		return nil, errors.NewInternalError(ctx, fmt.Errorf("malformed codebase service URL %s: %v",
			sr.codebaseSearchURL, err))
	}

	cl := client.New(goaclient.HTTPClientDoer(sr.codebaseSearchClient))
	cl.Host = u.Host
	cl.Scheme = u.Scheme
	cl.SetJWTSigner(goasupport.NewForwardSigner(goasupport.ForwardContextRequestID(ctx)))

	// search all the codebases associated with the repoURL
	path := client.CodebasesSearchPath()
	resp, err := cl.CodebasesSearch(ctx, path, d.GitURL, nil, nil)
	if err != nil {
		return nil, errors.NewInternalError(ctx, fmt.Errorf("could not search codebases: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		formattedErrors, err := cl.DecodeJSONAPIErrors(resp)
		if err != nil {
			return nil, errors.NewInternalError(ctx, fmt.Errorf("could not decode JSON formatted errors returned while listing codebases: %v", err))
		}
		if len(formattedErrors.Errors) > 0 {
			return nil, errors.NewInternalError(ctx, errs.Errorf(formattedErrors.Errors[0].Detail))
		}
		return nil, errors.NewInternalError(ctx, errs.Errorf("unknown error"))
	}
	codebases, err := cl.DecodeCodebaseList(resp)
	if err != nil {
		return nil, errors.NewInternalError(ctx, fmt.Errorf("could not decode the codebase list: %v", err))
	}
	return codebases, nil
}
