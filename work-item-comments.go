package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/comment"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// WorkItemCommentsController implements the work-item-comments resource.
type WorkItemCommentsController struct {
	*goa.Controller
	db application.DB
}

// NewWorkItemCommentsController creates a work-item-relationships-comments controller.
func NewWorkItemCommentsController(service *goa.Service, db application.DB) *WorkItemCommentsController {
	return &WorkItemCommentsController{Controller: service.NewController("WorkItemRelationshipsCommentsController"), db: db}
}

// Create runs the create action.
func (c *WorkItemCommentsController) Create(ctx *app.CreateWorkItemCommentsContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		_, err := appl.WorkItems().Load(ctx, ctx.ID)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
			return ctx.NotFound(jerrors)
		}

		currentUser, err := login.ContextIdentity(ctx)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
			return ctx.Unauthorized(jerrors)
		}

		currentUserID, err := uuid.FromString(currentUser)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
			return ctx.Unauthorized(jerrors)
		}

		reqComment := ctx.Payload.Data

		newComment := comment.Comment{
			ParentID:  ctx.ID,
			Body:      reqComment.Attributes.Body,
			CreatedBy: currentUserID,
		}

		err = appl.WorkItemComments().Create(ctx, &newComment)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
			return ctx.InternalServerError(jerrors)
		}

		res := &app.CommentSingle{
			Data: toAPI(&newComment),
		}
		return ctx.OK(res)
	})
}

// List runs the list action.
func (c *WorkItemCommentsController) List(ctx *app.ListWorkItemCommentsContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		_, err := appl.WorkItems().Load(ctx, ctx.ID)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
			return ctx.NotFound(jerrors)
		}

		res := &app.CommentArray{}
		res.Data = []*app.Comment{}

		comments, err := appl.WorkItemComments().List(ctx, ctx.ID)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
			return ctx.InternalServerError(jerrors)
		}
		for _, comment := range comments {
			res.Data = append(res.Data, toAPI(comment))
		}

		return ctx.OK(res)
	})
}

func toAPI(comment *comment.Comment) *app.Comment {
	return &app.Comment{
		Type: "comments",
		ID:   &comment.ID,
		Attributes: &app.CommentAttributes{
			Body:      &comment.Body,
			CreatedAt: &comment.CreatedAt,
		},
		Relationships: &app.CommentRelations{
			CreatedBy: &app.CommentCreatedBy{
				Data: &app.IdentityRelationData{
					Type: "identities",
					ID:   &comment.CreatedBy,
				},
			},
		},
	}
}
