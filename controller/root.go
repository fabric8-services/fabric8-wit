package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
		"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/goadesign/goa"
	"net/http"
	"github.com/satori/go.uuid"
	"path/filepath"
	"encoding/json"
	"strings"
	"io/ioutil"
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
	SWAGGER = "swagger/swagger.json"
	ROOT_CONTROLLER = "RootController"
	BASE_PATH = "basePath"
)

// Root describes a single Root
type Root struct {
	Relationships	map[string]interface{}
	Attributes   	interface{}
	ID              uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	BasePath      	string
}

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
func convertRoot(request *http.Request, root Root) *app.Root {
	selfURL := request.Host + root.BasePath
	l := &app.Root{
		Relationships:root.Relationships,
		Attributes:&root.Attributes,
		Links: &app.GenericLinksForRoot{
			Self: &selfURL,
		},
		ID: &root.ID,
	}
	return l
}

// Get a list of all endpoints formatted to json api format
func getRoot() (Root, error) {

	s, e := filepath.Abs(SWAGGER)
	if e != nil {
		return Root{}, e
	}

	swaggerJSON, err := ioutil.ReadFile(s)
	if err != nil {
		return Root{}, err
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

	root := Root{Relationships: namedPaths, Attributes: map[string]string{}, ID: uuid.NewV4(), BasePath:result[BASE_PATH].(string)}
	return root, err
}