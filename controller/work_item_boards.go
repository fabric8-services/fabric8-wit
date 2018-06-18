package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

// WorkItemBoardsController implements the work_item_boards resource.
type WorkItemBoardsController struct {
	*goa.Controller
	db application.DB
}

// NewWorkItemBoardsController creates a work_item_boards controller.
func NewWorkItemBoardsController(service *goa.Service, db application.DB) *WorkItemBoardsController {
	return &WorkItemBoardsController{
		Controller: service.NewController("WorkItemBoardsController"),
		db:         db,
	}
}

// List runs the list action.
func (c *WorkItemBoardsController) List(ctx *app.ListWorkItemBoardsContext) error {

	var boards []*workitem.Board
	err := application.Transactional(c.db, func(appl application.Application) error {
		list, err := appl.Boards().List(ctx, ctx.SpaceTemplateID)
		if err != nil {
			return errs.WithStack(err)
		}
		boards = list
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	res := &app.WorkItemBoardList{
		Data: make([]*app.WorkItemBoardData, len(boards)),
		Links: &app.WorkItemBoardLinks{
			Self: rest.AbsoluteURL(ctx.Request, app.SpaceTemplateHref(ctx.SpaceTemplateID)) + "/" + APIWorkItemBoards,
		},
	}
	for i, board := range boards {
		res.Data[i] = ConvertBoardFromModel(ctx.Request, *board)
		for _, column := range board.Columns {
			res.Included = append(res.Included, ConvertColumnsFromModel(ctx.Request, column))
		}
	}
	return ctx.OK(res)
}

