package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/comment"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/rest"
	"github.com/goadesign/goa"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
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

		err = appl.Comments().Create(ctx, &newComment)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
			return ctx.InternalServerError(jerrors)
		}

		res := &app.CommentSingle{
			Data: ConvertComment(ctx.RequestData, &newComment),
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

		comments, err := appl.Comments().List(ctx, ctx.ID)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
			return ctx.InternalServerError(jerrors)
		}
		res.Data = ConvertComments(ctx.RequestData, comments)

		return ctx.OK(res)
	})
}

// Relations runs the relation action.
// TODO: Should only return Resource Identifier Objects, not complete object (See List)
func (c *WorkItemCommentsController) Relations(ctx *app.RelationsWorkItemCommentsContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		wi, err := appl.WorkItems().Load(ctx, ctx.ID)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
			return ctx.NotFound(jerrors)
		}

		comments, err := appl.Comments().List(ctx, ctx.ID)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
			return ctx.InternalServerError(jerrors)
		}

		res := &app.CommentArray{}
		res.Data = []*app.Comment{}
		res.Data = ConvertCommentsResourceID(ctx.RequestData, comments)
		res.Links = CreateCommentsRelationLinks(ctx.RequestData, wi)

		return ctx.OK(res)
	})
}

// WorkItemIncludeCommentsAndTotal adds relationship about comments to workitem (include totalCount)
func WorkItemIncludeCommentsAndTotal(ctx context.Context, db application.DB, parentID string) WorkItemConvertFunc {
	// TODO: Wrap ctx in a Timeout context?
	count := make(chan int)
	go func() {
		defer close(count)
		application.Transactional(db, func(appl application.Application) error {
			cs, err := appl.Comments().List(ctx, parentID)
			if err != nil {
				count <- 0
				return errors.WithStack(err)
			}
			count <- len(cs)
			return nil
		})
	}()
	return func(request *goa.RequestData, wi *app.WorkItem, wi2 *app.WorkItem2) {
		wi2.Relationships.Comments = CreateCommentsRelation(request, wi)
		wi2.Relationships.Comments.Meta = map[string]interface{}{
			"totalCount": <-count,
		}
	}
}

// WorkItemIncludeComments adds relationship about comments to workitem (include totalCount)
func WorkItemIncludeComments(request *goa.RequestData, wi *app.WorkItem, wi2 *app.WorkItem2) {
	wi2.Relationships.Comments = CreateCommentsRelation(request, wi)
}

// CreateCommentsRelation returns a RelationGeneric object representing the relation for a workitem to comment relation
func CreateCommentsRelation(request *goa.RequestData, wi *app.WorkItem) *app.RelationGeneric {
	return &app.RelationGeneric{
		Links: CreateCommentsRelationLinks(request, wi),
	}
}

// CreateCommentsRelationLinks returns a RelationGeneric object representing the links for a workitem to comment relation
func CreateCommentsRelationLinks(request *goa.RequestData, wi *app.WorkItem) *app.GenericLinks {
	commentsSelf := rest.AbsoluteURL(request, app.WorkitemHref(wi.ID)) + "/relationships/comments"
	commentsRelated := rest.AbsoluteURL(request, app.WorkitemHref(wi.ID)) + "/comments"
	return &app.GenericLinks{
		Self:    &commentsSelf,
		Related: &commentsRelated,
	}
}
