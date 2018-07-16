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
	errs "github.com/pkg/errors"
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

	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	res := &app.RootSingle{}
	res.Data = convertRoot(ctx.Request, roots)
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
func getRoot(fileHandler asseter) (app.Root, error) {
	swaggerJSON, err := fileHandler.Asset("swagger.json")
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":  err,
			"file": "swagger.json",
		}, "file with id 'swagger.json' not found")

		return app.Root{}, errors.NewNotFoundError("file", "swagger.json")
	}

	var result map[string]interface{}
	err = json.Unmarshal([]byte(swaggerJSON), &result)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":  err,
			"file": "swagger.json",
		}, "unable to unmarshal json")
		return app.Root{}, errs.Wrap(err, "Could not unmarshal Swagger.json")
	}

	// Get and iterate over paths from swagger specification.
	swaggerPaths := result["paths"].(map[string]interface{})
	namedPaths := make(map[string]interface{})
	for path, pathObj := range swaggerPaths {

		// Currently not supporting endpoints that contain parameters.
		if !strings.Contains(path, "{") {

			// Use the segments in the path to construct a meaningful name to use for the path.
			key := strings.Replace(path, "/", "_", -1)
			key = strings.Replace(key, "_", "", 1)
			xtag, ok := pathObj.(map[string]interface{})["x-tag"]
			if !ok {
				return app.Root{}, errs.Wrap(err, "Invalid path format in swagger specification")
			}

			// If the tag exists, use it as path name.
			if xtag != nil {
				xtag, ok := xtag.(string)
				if !ok {
					return app.Root{}, errs.Wrap(err, "Invalid x-tag value in swagger specification metadata")
				}

				key = xtag
			}

			// Set the related field to the path.
			name := map[string]string{
				"related": path,
			}

			// Set the links object to contain the related field.
			links := map[string]interface{}{
				"links": name,
			}

			// Set the name to contain the links object.
			namedPaths[key] = links
		}
	}

	id := uuid.NewV4()
	basePath, ok := result["basePath"].(string)
	if !ok {
		return app.Root{}, errs.Wrap(err, "Invalid basePath value in swagger specification metadata")
	}

	return app.Root{Relationships: namedPaths, ID: &id, BasePath: &basePath}, nil
}
