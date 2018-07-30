package controller

import (
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// WorkItemBoardController implements the work_item_board resource.
type WorkItemBoardController struct {
	*goa.Controller
	db application.DB
}

// APIWorkItemBoards is the type constant used when referring to work item
// board relationships in JSONAPI
var APIWorkItemBoards = "workitemboards"

// APIBoardColumns is the type constant used when referring to work item
// board column relationships in JSONAPI
var APIBoardColumns = "boardcolumns"

// NewWorkItemBoardController creates a work_item_board controller.
func NewWorkItemBoardController(service *goa.Service, db application.DB) *WorkItemBoardController {
	return &WorkItemBoardController{
		Controller: service.NewController("WorkItemBoardController"),
		db:         db,
	}
}

// Show runs the show action.
func (c *WorkItemBoardController) Show(ctx *app.ShowWorkItemBoardContext) error {
	var board *workitem.Board
	err := application.Transactional(c.db, func(appl application.Application) error {
		b, err := appl.Boards().Load(ctx, ctx.BoardID)
		if err != nil {
			return errs.WithStack(err)
		}
		board = b
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	res := &app.WorkItemBoardSingle{
		Data: ConvertBoardFromModel(ctx.Request, *board),
	}
	for _, column := range board.Columns {
		res.Included = append(res.Included, ConvertColumnsFromModel(ctx.Request, column))
	}
	return ctx.OK(res)
}

// ConvertColumnsFromModel converts WorkitemTypeBoard model to a response
// resource object for jsonapi.org specification
func ConvertColumnsFromModel(request *http.Request, column workitem.BoardColumn) *app.WorkItemBoardColumnData {
	return &app.WorkItemBoardColumnData{
		ID:   column.ID,
		Type: APIBoardColumns,
		Attributes: &app.WorkItemBoardColumnAttributes{
			Name:  column.Name,
			Order: &column.Order,
		},
	}
}

// ConvertBoardFromModel converts WorkitemTypeBoard model to a response resource
// object for jsonapi.org specification
func ConvertBoardFromModel(request *http.Request, b workitem.Board) *app.WorkItemBoardData {
	res := &app.WorkItemBoardData{
		ID:   &b.ID,
		Type: APIWorkItemBoards,
		Links: &app.GenericLinks{
			Related: ptr.String(rest.AbsoluteURL(request, app.WorkItemBoardHref(b.ID))),
		},
		Attributes: &app.WorkItemBoardAttributes{
			Context:     b.Context,
			ContextType: b.ContextType,
			Name:        b.Name,
			CreatedAt:   ptr.Time(b.CreatedAt.UTC()),
			UpdatedAt:   ptr.Time(b.UpdatedAt.UTC()),
		},
		Relationships: &app.WorkItemBoardRelationships{
			Columns: &app.RelationGenericList{
				Data: make([]*app.GenericData, len(b.Columns)),
			},
			SpaceTemplate: &app.RelationGeneric{
				Data: &app.GenericData{
					ID:   ptr.String(b.SpaceTemplateID.String()),
					Type: &APISpaceTemplates,
				},
			},
		},
	}

	// iterate over the columns and attach them as an
	// included relationship
	for i, column := range b.Columns {
		res.Relationships.Columns.Data[i] = &app.GenericData{
			ID:   ptr.String(column.ID.String()),
			Type: ptr.String(APIBoardColumns),
		}
	}

	return res
}

// ConvertBoardColumnSimple converts a simple boardcolumn ID into a Generic Relationship
func ConvertBoardColumnSimple(request *http.Request, id interface{}) *app.GenericData {
	i := ""
	switch t := id.(type) {
	case string:
		i = t
	case uuid.UUID:
		i = t.String()
	}
	return &app.GenericData{
		Type: ptr.String(APIBoardColumns),
		ID:   &i,
	}
}
