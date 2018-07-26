package controller

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/swagger"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

const embeddedSwaggerSpecFile = "swagger.json"

type asseter interface {
	Asset(name string) ([]byte, error)
}

// EndpointsController implements the endpoints resource.
type EndpointsController struct {
	*goa.Controller
	FileHandler asseter
	// endpoints caches the result of parsing the endpoints so that
	// consecutive calls to this API only call the API once
	endpoints *app.Endpoints
	// endpointsLock protects the endpoints resource
	endpointsLock sync.RWMutex
}

type workingFileFetcher struct{}

func (s workingFileFetcher) Asset(fileName string) ([]byte, error) {
	return swagger.Asset(fileName)
}

var _ asseter = workingFileFetcher{}
var _ asseter = (*workingFileFetcher)(nil)

// NewEndpointsController creates an endpoints controller.
func NewEndpointsController(service *goa.Service) *EndpointsController {
	return &EndpointsController{
		Controller:  service.NewController("EndpointsController"),
		FileHandler: workingFileFetcher{},
	}
}

// List runs the list action.
func (c *EndpointsController) List(ctx *app.ListEndpointsContext) error {
	c.endpointsLock.Lock()
	defer c.endpointsLock.Unlock()
	if c.endpoints == nil {
		endpoints, err := getEndpoints(ctx, c.FileHandler)
		if err != nil || endpoints == nil {
			log.Error(ctx, map[string]interface{}{
				"err":  errs.WithStack(err),
				"file": embeddedSwaggerSpecFile,
			}, err.Error())
			return jsonapi.JSONErrorResponse(ctx, errs.WithStack(err))
		}
		c.endpoints = endpoints
	}
	return ctx.OK(&app.EndpointSingle{Data: c.endpoints})
}

// Get a list of all endpoints formatted to json api format.
func getEndpoints(ctx *app.ListEndpointsContext, fileHandler asseter) (*app.Endpoints, error) {
	// Get an unmarshal swagger specification
	swaggerJSON, err := fileHandler.Asset(embeddedSwaggerSpecFile)
	if err != nil {
		// TODO(tinakurian): fix error handling
		return nil, errors.NewNotFoundError("file", embeddedSwaggerSpecFile)
	}

	var result map[string]interface{}
	err = json.Unmarshal(swaggerJSON, &result)
	if err != nil {
		return nil, errs.Wrapf(err, "unable to unmarshal the file with id "+"'"+embeddedSwaggerSpecFile+"'")
	}

	// Get and iterate over paths from swagger specification.
	swaggerPaths, ok := result["paths"]
	if !ok {
		return nil, errors.NewInternalErrorFromString("field `paths` could be found in swagger specification")
	}

	swaggerPathz, ok := swaggerPaths.(map[string]interface{})
	if !ok {
		return nil, errors.NewInternalErrorFromString("unable to assert concrete type map for field `paths` in swagger specification")
	}

	// the namedPaths map stores paths as key and URLs as values
	namedPaths := make(map[string]interface{})
	for path, swaggerPath := range swaggerPathz {

		// Currently not supporting endpoints that contain parameters.
		if !strings.Contains(path, "{") {
			key := path

			pathsObj, ok := swaggerPath.(map[string]interface{})
			if !ok {
				return nil, errors.NewInternalErrorFromString("unable to assert concrete type map for field `paths` in swagger specification")
			}

			// Get the x-tag value. If the tag exists, use it as path name.
			xtagObj, ok := pathsObj["x-tag"]
			if ok {
				xtag, ok := xtagObj.(string)
				if !ok {
					return nil, errors.NewInternalErrorFromString("unable to assert concrete type string for field `x-tag` in swagger specification")
				}

				key = xtag
			}

			// cleanup the key to conform to JSONAPI member names
			key = jsonapi.FormatMemberName(key)

			// Set the related field and link objects for each name.
			namedPaths[key] = map[string]interface{}{
				"links": map[string]string{
					"related": path,
				},
			}
		}
	}

	basePathObj, ok := result["basePath"]
	if !ok {
		return nil, errors.NewInternalErrorFromString("field `basePath` could be found in swagger specification")
	}

	basePath, ok := basePathObj.(string)
	if !ok {
		return nil, errors.NewInternalErrorFromString("unable to assert concrete type string for field `basePath` in swagger specification")
	}

	return &app.Endpoints{
		Type:          "endpoints",
		ID:            uuid.NewV4(),
		Relationships: namedPaths,
		Links: &app.GenericLinks{
			Self: ptr.String(rest.AbsoluteURL(ctx.Request, basePath)),
		},
	}, nil
}
