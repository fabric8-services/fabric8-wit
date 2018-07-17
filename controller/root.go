package controller

import (
	"encoding/json"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/swagger"
	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"net/http"
	"strings"
)

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
	roots, err := getRoot(c.FileHandler)
	if err != nil || roots == nil {
		return jsonapi.JSONErrorResponse(
			ctx, err)
	}

	res := &app.RootSingle{}
	res.Data = convertRoot(ctx.Request, *roots)
	return ctx.OK(res)
}

// ConvertRoot converts from internal to external REST representation.
func convertRoot(request *http.Request, root app.Root) *app.Root {
	selfURL := rest.AbsoluteURL(request, *root.BasePath)
	l := &app.Root{
		Relationships: root.Relationships,
		Attributes:    root.Attributes,
		Links: &app.GenericLinksForRoot{
			Self: &selfURL,
		},
		ID: root.ID,
	}
	return l
}

// Get a list of all endpoints formatted to json api format.
func getRoot(fileHandler asseter) (*app.Root, error) {
	// Get an unmarshal swagger specification
	result, err := getUnmarshalledSwagger(fileHandler)
	if err != nil {
		return nil, err
	}

	// Get and iterate over paths from swagger specification.
	swaggerPaths, err := getSwaggerFieldAsMap("paths", result)
	if err != nil {
		return nil, err
	}

	namedPaths := make(map[string]interface{})
	for path, swaggerPath := range swaggerPaths {

		// Currently not supporting endpoints that contain parameters.
		if !strings.Contains(path, "{") {

			// Use the segments in the path to construct a meaningful name to use for the path.
			key := strings.Replace(path, "/", "_", -1)
			key = strings.Replace(key, "_", "", 1)

			pathsObj, err := swaggerPath.(map[string]interface{})
			if !err {
				return nil, errorLogReturn("Unable to assert correct format for swagger specifiation")
			}

			// Get the x-tag value. If the tag exists, use it as path name.
			xtag, _ := getSwaggerFieldAsString("x-tag", pathsObj)
			if len(xtag) > 0 {
				key = xtag
			}

			// Set the related field, link objects for each name
			namedPaths[key] = map[string]interface{}{
				"links": map[string]string{
					"related": path,
				},
			}
		}
	}

	id := uuid.NewV4()
	basePath, err := getSwaggerFieldAsString("basePath", result)
	if err != nil {
		return nil, err
	}
	return &app.Root{Relationships: namedPaths, ID: &id, BasePath: &basePath}, nil
}

// Gets the swagger specification binary and attempts to unmarshal it.
// Returns the unmarshed specification, or error. Error is returned if
// the specification could not be found, or the specification was not
// able to be unmarshalled.
func getUnmarshalledSwagger(fileHandler asseter) (map[string]interface{}, error) {
	swaggerJSON, err := fileHandler.Asset("swagger.json")
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"file": "swagger.json",
		}, "The file with id 'swagger.json' not found")
		return nil, errors.NewNotFoundError("file", "swagger.json")
	}

	var result map[string]interface{}
	err = json.Unmarshal([]byte(swaggerJSON), &result)
	if err != nil {
		return nil, errorLogReturn("Unable to unmarshal json swagger specification")
	}
	return result, nil
}

// Logs an error with a custom message. Once the error is logged, the
// error is returned.
func errorLogReturn(message string) error {
	log.Error(nil, map[string]interface{}{
		"file": "swagger.json",
	}, message)
	return errors.NewInternalErrorFromString(message)
}

// Gets a field from te swagger specification and asserts the type to
// string. Returns errors if the key does not exist in the swagger, or
// the field cannot be type asserted to a string.
func getSwaggerFieldAsString(field string, json map[string]interface{}) (string, error) {
	value, ok := json[field]
	if !ok {
		return "", errorLogReturn(" Field " + "'" + field + "'" + " cannot be found in swagger specification")
	}

	strValue, ok := value.(string)
	if !ok {
		return "", errorLogReturn("Unable to assert concrete type string for field " + "'" + field + "'" + " in swagger specifiation.")
	}
	return strValue, nil
}

// Gets a field from te swagger specification and asserts the type to
// map. Returns errors if the key does not exist in the swagger, or
// the field cannot be type asserted to a map.
func getSwaggerFieldAsMap(field string, json map[string]interface{}) (map[string]interface{}, error) {
	value, ok := json[field]
	if !ok {
		return nil, errorLogReturn(" Field " + "'" + field + "'" + " cannot be found in swagger specification")
	}

	mapValue, err := value.(map[string]interface{})
	if !err {
		return nil, errorLogReturn("Unable to assert concrete type map for field " + "'" + field + "'" + " in swagger specifiation")
	}
	return mapValue, nil
}
