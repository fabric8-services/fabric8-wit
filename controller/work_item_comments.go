package controller

import (
	"context"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/comment"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// WorkItemCommentsController implements the work-item-comments resource.
type WorkItemCommentsController struct {
	*goa.Controller
	db     application.DB
	config WorkItemCommentsControllerConfiguration
}

//WorkItemCommentsControllerConfiguration configuration for the WorkItemCommentsController
type WorkItemCommentsControllerConfiguration interface {
	GetCacheControlComments() string
}

// NewWorkItemCommentsController creates a work-item-relationships-comments controller.
func NewWorkItemCommentsController(service *goa.Service, db application.DB, config WorkItemCommentsControllerConfiguration) *WorkItemCommentsController {
	return &WorkItemCommentsController{
		Controller: service.NewController("WorkItemRelationshipsCommentsController"),
		db:         db,
		config:     config,
	}
}

// Create runs the create action.
func (c *WorkItemCommentsController) Create(ctx *app.CreateWorkItemCommentsContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		_, err := appl.WorkItems().LoadByID(ctx, ctx.WiID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		currentUserIdentityID, err := login.ContextIdentity(ctx)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
		}

		reqComment := ctx.Payload.Data
		markup := rendering.NilSafeGetMarkup(reqComment.Attributes.Markup)
		newComment := comment.Comment{
			ParentID:  ctx.WiID.String(),
			Body:      reqComment.Attributes.Body,
			Markup:    markup,
			CreatedBy: *currentUserIdentityID,
		}

		err = appl.Comments().Create(ctx, &newComment, *currentUserIdentityID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
		}

		res := &app.CommentSingle{
			Data: ConvertComment(ctx.RequestData, newComment),
		}
		return ctx.OK(res)
	})
}

// List runs the list action.
func (c *WorkItemCommentsController) List(ctx *app.ListWorkItemCommentsContext) error {
	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)
	return application.Transactional(c.db, func(appl application.Application) error {
		wi, err := appl.WorkItems().LoadByID(ctx, ctx.WiID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		comments, tc, err := appl.Comments().List(ctx, wi.ID.String(), &offset, &limit)
		count := int(tc)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
		}
		return ctx.ConditionalEntities(comments, c.config.GetCacheControlComments, func() error {
			res := &app.CommentList{}
			res.Data = []*app.Comment{}
			res.Meta = &app.CommentListMeta{TotalCount: count}
			res.Data = ConvertComments(ctx.RequestData, comments)
			res.Links = &app.PagingLinks{}
			setPagingLinks(res.Links, buildAbsoluteURL(ctx.RequestData), len(comments), offset, limit, count)
			return ctx.OK(res)
		})
	})
}

// Relations runs the relation action.
// TODO: Should only return Resource Identifier Objects, not complete object (See List)
func (c *WorkItemCommentsController) Relations(ctx *app.RelationsWorkItemCommentsContext) error {
	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)
	return application.Transactional(c.db, func(appl application.Application) error {
		wi, err := appl.WorkItems().LoadByID(ctx, ctx.WiID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		comments, tc, err := appl.Comments().List(ctx, wi.ID.String(), &offset, &limit)
		count := int(tc)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
		}
		_ = wi
		_ = comments
		res := &app.CommentRelationshipList{}
		res.Meta = &app.CommentListMeta{TotalCount: count}
		res.Data = ConvertCommentsResourceID(ctx.RequestData, comments)
		res.Links = CreateCommentsRelationLinks(ctx.RequestData, wi)
		return ctx.OK(res)
	})
}

// workItemIncludeCommentsAndTotal adds relationship about comments to workitem (include totalCount)
func workItemIncludeCommentsAndTotal(ctx context.Context, db application.DB, parentID uuid.UUID) WorkItemConvertFunc {
	// TODO: Wrap ctx in a Timeout context?
	count := make(chan int)
	go func() {
		defer close(count)
		application.Transactional(db, func(appl application.Application) error {
			cs, err := appl.Comments().Count(ctx, parentID.String())
			if err != nil {
				count <- 0
				return errs.WithStack(err)
			}
			count <- cs
			return nil
		})
	}()
	return func(request *goa.RequestData, wi *workitem.WorkItem, wi2 *app.WorkItem) {
		wi2.Relationships.Comments = CreateCommentsRelation(request, wi)
		wi2.Relationships.Comments.Meta = map[string]interface{}{
			"totalCount": <-count,
		}
	}
}

// workItemIncludeComments adds relationship about comments to workitem (include totalCount)
func workItemIncludeComments(request *goa.RequestData, wi *workitem.WorkItem, wi2 *app.WorkItem) {
	wi2.Relationships.Comments = CreateCommentsRelation(request, wi)
}

// CreateCommentsRelation returns a RelationGeneric object representing the relation for a workitem to comment relation
func CreateCommentsRelation(request *goa.RequestData, wi *workitem.WorkItem) *app.RelationGeneric {
	return &app.RelationGeneric{
		Links: CreateCommentsRelationLinks(request, wi),
	}
}

// CreateCommentsRelationLinks returns a RelationGeneric object representing the links for a workitem to comment relation
func CreateCommentsRelationLinks(request *goa.RequestData, wi *workitem.WorkItem) *app.GenericLinks {
	commentsSelf := rest.AbsoluteURL(request, app.WorkitemHref(wi.SpaceID, wi.ID)) + "/relationships/comments"
	commentsRelated := rest.AbsoluteURL(request, app.WorkitemHref(wi.SpaceID, wi.ID)) + "/comments"
	return &app.GenericLinks{
		Self:    &commentsSelf,
		Related: &commentsRelated,
	}
}
