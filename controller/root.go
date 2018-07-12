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
	xTag = "x-tag"
	paths = "paths"
	links = "links"
	related = "related"
	frwdSlash = "/"
	bracket = "{"
	underscore = "_"
	emptyStr = ""
	swaggerFile = "swagger.json"
	rootController = "RootController"
	basePath = "basePath"
)

// RootController implements the root resource.
type RootController struct {
	*goa.Controller
}

// NewRootController creates a root controller.
func NewRootController(service *goa.Service) *RootController {
	return &RootController{
		Controller: service.NewController(rootController),
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
	swaggerJSON, err := swagger.Asset(swaggerFile)
	if err != nil {
		return app.Root{}, err
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
	return root, err
}