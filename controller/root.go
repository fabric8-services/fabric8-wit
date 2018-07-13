package controller

import (
	"encoding/json"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/swagger"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"net/http"
	"strings"
)

const (
	xTag           = "x-tag"
	paths          = "paths"
	links          = "links"
	related        = "related"
	frwdSlash      = "/"
	bracket        = "{"
	underscore     = "_"
	emptyStr       = ""
	swaggerFile    = "swagger.json"
	rootController = "RootController"
	basePath       = "basePath"
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
		Controller: service.NewController(rootController),
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

// ConvertRoot converts from internal to external REST representation
func convertRoot(request *http.Request, root app.Root) *app.Root {
	selfURL := request.Host + *root.BasePath
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

// Get a list of all endpoints formatted to json api format
func getRoot(fileHandler asseter) (app.Root, error) {
	swaggerJSON, err := fileHandler.Asset(swaggerFile)
	if err != nil {
		// TODO(tinakurian): log error
		return app.Root{}, errors.NewNotFoundError("file", swaggerFile)
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(swaggerJSON), &result)

	swaggerPaths := result[paths].(map[string]interface{})
	namedPaths := make(map[string]interface{})
	for path, pathObj := range swaggerPaths {

		if !strings.Contains(path, bracket) {
			key := strings.Replace(path, frwdSlash, underscore, -1)
			key = strings.Replace(key, underscore, emptyStr, 1)
			xtag := pathObj.(map[string]interface{})[xTag]

			// If xtag doesn't exist, result to using first path segment
			// otherwise, use the xtag
			if xtag != nil {
				key = xtag.(string)
			}

			name := map[string]string{
				related: path,
			}

			links := map[string]interface{}{
				links: name,
			}
			namedPaths[key] = links
		}
	}

	basePath := result[basePath].(string)
	id := uuid.NewV4()
	root := app.Root{Relationships: namedPaths, ID: &id, BasePath: &basePath}
	return root, nil
}
