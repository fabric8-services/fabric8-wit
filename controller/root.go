package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/swagger"
	"github.com/goadesign/goa"
	"net/http"
	"github.com/satori/go.uuid"
	"encoding/json"
	"strings"
	)

const (
	X_TAG = "x-tag"
	PATHS = "paths"
	LINKS = "links"
	RELATED = "related"
	FRWD_SLASH = "/"
	BRACKET = "{"
	UNDERSCORE = "_"
	EMPTY_STR = ""
	SWAGGER = "swagger.json"
	ROOT_CONTROLLER = "RootController"
	BASE_PATH = "basePath"
)

// RootController implements the root resource.
type RootController struct {
	*goa.Controller
}

// NewRootController creates a root controller.
func NewRootController(service *goa.Service) *RootController {
	return &RootController{
		Controller: service.NewController(ROOT_CONTROLLER),
	}
}

// List runs the list action.
func (c *RootController) List(ctx *app.ListRootContext) error {
	roots, err := getRoot()

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
		Relationships:root.Relationships,
		Attributes:root.Attributes,
		Links: &app.GenericLinksForRoot{
			Self: &selfURL,
		},
		ID: root.ID,
	}
	return l
}

// Get a list of all endpoints formatted to json api format
func getRoot() (app.Root, error) {

	swaggerJSON, err := swagger.Asset(SWAGGER)
	if err != nil {
		return app.Root{}, err
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(swaggerJSON), &result)

	swaggerPaths := result[PATHS].(map[string]interface{})
	namedPaths := make(map[string]interface{})
	for path, pathObj := range swaggerPaths {

		if !strings.Contains(path, BRACKET) {
			key := strings.Replace(path, FRWD_SLASH, UNDERSCORE, -1)
			key = strings.Replace(key, UNDERSCORE, EMPTY_STR, 1)
			xtag := pathObj.(map[string]interface{})[X_TAG]

			// If xtag doesn't exist, result to using first path segment
			// otherwise, use the xtag
			if xtag != nil {
				key = xtag.(string)
			}

			name := map[string]string{
				RELATED: path,
			}

			links := map[string]interface{}{
				LINKS: name,
			}
			namedPaths[key] = links
		}
	}

	basePath := result[BASE_PATH].(string)
	id := uuid.NewV4()
	root := app.Root{Relationships: namedPaths, ID: &id, BasePath: &basePath}
	return root, err
}