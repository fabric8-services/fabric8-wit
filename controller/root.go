package controller

import (
	"encoding/json"
	"strings"

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

// RootController implements the root resource.
type RootController struct {
	*goa.Controller
	FileHandler asseter
}

type workingFileFetcher struct{}

func (s workingFileFetcher) Asset(fileName string) ([]byte, error) {
	return swagger.Asset(fileName)
}

var _ asseter = workingFileFetcher{}
var _ asseter = (*workingFileFetcher)(nil)

// NewRootController creates a root controller.
func NewRootController(service *goa.Service) *RootController {
	return &RootController{
		Controller:  service.NewController("RootController"),
		FileHandler: workingFileFetcher{},
	}
}

// List runs the list action.
func (c *RootController) List(ctx *app.ListRootContext) error {
	roots, err := getRoot(ctx, c.FileHandler)
	if err != nil || roots == nil {
		return jsonapi.JSONErrorResponse(
			ctx, err)
	}
	return ctx.OK(&app.RootSingle{Data: roots})
}

// Get a list of all endpoints formatted to json api format.
func getRoot(ctx *app.ListRootContext, fileHandler asseter) (*app.Root, error) {
	// Get an unmarshal swagger specification
	result, err := getUnmarshalledSwagger(ctx, fileHandler)
	if err != nil {
		return nil, err
	}

	// Get and iterate over paths from swagger specification.
	swaggerPaths, err := getSwaggerFieldAsMap(ctx, "paths", result)
	if err != nil {
		return nil, err
	}

	// the path map stores paths as key and URLs as values
	namedPaths := make(map[string]interface{})
	for path, swaggerPath := range swaggerPaths {

		// Currently not supporting endpoints that contain parameters.
		if !strings.Contains(path, "{") {
			key := path

			pathsObj, ok := swaggerPath.(map[string]interface{})
			if !ok {
				log.Error(ctx, map[string]interface{}{
					"file": embeddedSwaggerSpecFile,
				}, "Unable to assert correct format for swagger specifiation")

				return nil, errors.NewInternalErrorFromString("Unable to assert correct format for swagger specifiation")
			}

			// Get the x-tag value. If the tag exists, use it as path name.
			xtag, _ := getSwaggerFieldAsString(ctx, "x-tag", pathsObj, true)
			if len(xtag) > 0 {
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

	basePath, err := getSwaggerFieldAsString(ctx, "basePath", result, false)
	if err != nil {
		return nil, err
	}

	return &app.Root{
		Type:          "endpoints",
		ID:            uuid.NewV4(),
		Relationships: namedPaths,
		Links: &app.GenericLinks{
			Self: ptr.String(rest.AbsoluteURL(ctx.Request, basePath)),
		},
	}, nil
}

// getUnmarshalledSwagger gets the swagger specification binary and attempts to
// unmarshal it. Returns the unmarshed specification, or error. Error is
// returned if the specification could not be found, or the specification was
// not able to be unmarshalled.
func getUnmarshalledSwagger(ctx *app.ListRootContext, fileHandler asseter) (map[string]interface{}, error) {
	swaggerJSON, err := fileHandler.Asset(embeddedSwaggerSpecFile)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":  errs.WithStack(err),
			"file": embeddedSwaggerSpecFile,
		}, `file "%s" not found`, embeddedSwaggerSpecFile)
		return nil, errors.NewNotFoundError("file", embeddedSwaggerSpecFile)
	}

	var result map[string]interface{}
	err = json.Unmarshal([]byte(swaggerJSON), &result)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":  errs.WithStack(err),
			"file": embeddedSwaggerSpecFile,
		}, `Unable to unmarshal the file with id "%s"`, embeddedSwaggerSpecFile)
		return nil, errors.NewInternalErrorFromString("Unable to unmarshal the file with id " + "'" + embeddedSwaggerSpecFile + "'")
	}
	return result, nil
}

// getSwaggerFieldAsString gets a field from te swagger specification and
// asserts the type to string. Returns errors if the key does not exist in the
// swagger specification, or the field cannot be type asserted to a string.
func getSwaggerFieldAsString(ctx *app.ListRootContext, field string, json map[string]interface{}, isXtag bool) (string, error) {
	value, ok := json[field]
	if !ok {
		if !isXtag {
			log.Error(ctx, map[string]interface{}{
				"file": embeddedSwaggerSpecFile,
			}, `Field "%s" cannot be found in swagger specification`, field)
		}
		return "", errors.NewInternalErrorFromString(" Field " + "'" + field + "'" + " cannot be found in swagger specification")
	}

	strValue, ok := value.(string)
	if !ok {
		log.Error(ctx, map[string]interface{}{
			"file": embeddedSwaggerSpecFile,
		}, `Unable to assert concrete type string for field "%s" in swagger specifiation.`, field)
		return "", errors.NewInternalErrorFromString("Unable to assert concrete type string for field " + "'" + field + "'" + " in swagger specifiation.")
	}
	return strValue, nil
}

// Gets a field from te swagger specification and asserts the type to
// map. Returns errors if the key does not exist in the swagger
// specification, or the field cannot be type asserted to a map.
func getSwaggerFieldAsMap(ctx *app.ListRootContext, field string, json map[string]interface{}) (map[string]interface{}, error) {
	value, ok := json[field]
	if !ok {
		log.Error(ctx, map[string]interface{}{
			"file": embeddedSwaggerSpecFile,
		}, `Field "%s" cannot be found in swagger specification`, field)
		return nil, errors.NewInternalErrorFromString("Field " + "'" + field + "'" + " cannot be found in swagger specification")
	}

	mapValue, err := value.(map[string]interface{})
	if !err {
		log.Error(ctx, map[string]interface{}{
			"file": embeddedSwaggerSpecFile,
		}, `Unable to assert concrete type map for field "%s" in swagger specifiation`, field)
		return nil, errors.NewInternalErrorFromString("Unable to assert concrete type map for field " + "'" + field + "'" + " in swagger specifiation")
	}
	return mapValue, nil
}
