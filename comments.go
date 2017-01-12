package main

import (
	"errors"
	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/comment"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// CommentsController implements the comments resource.
type CommentsController struct {
	*goa.Controller
	db application.DB
}

// NewCommentsController creates a comments controller.
func NewCommentsController(service *goa.Service, db application.DB) *CommentsController {
	return &CommentsController{Controller: service.NewController("CommentsController"), db: db}
}

// Show runs the show action.
func (c *CommentsController) Show(ctx *app.ShowCommentsContext) error {
	id, err := uuid.FromString(ctx.ID)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
		return ctx.BadRequest(jerrors)
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		c, err := appl.Comments().Load(ctx, id)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
			return ctx.NotFound(jerrors)
		}

		res := &app.CommentSingle{}
		res.Data = ConvertComment(
			ctx.RequestData,
			c,
			CommentIncludeParentWorkItem())

		return ctx.OK(res)
	})
}

// Update does PATCH comment
func (c *CommentsController) Update(ctx *app.UpdateCommentsContext) error {
	id, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}
	identity, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		cm, err := appl.Comments().Load(ctx.Context, id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		if identity != cm.CreatedBy.String() {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(errors.New("Not same user")))
		}

		cm.Body = *ctx.Payload.Data.Attributes.Body
		cm, err = appl.Comments().Save(ctx.Context, cm)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		res := &app.CommentSingle{
			Data: ConvertComment(ctx.RequestData, cm, CommentIncludeParentWorkItem()),
		}
		return ctx.OK(res)
	})
}

// CommentConvertFunc is a open ended function to add additional links/data/relations to a Comment during
// convertion from internal to API
type CommentConvertFunc func(*goa.RequestData, *comment.Comment, *app.Comment)

// ConvertComments converts between internal and external REST representation
func ConvertComments(request *goa.RequestData, comments []*comment.Comment, additional ...CommentConvertFunc) []*app.Comment {
	var cs = []*app.Comment{}
	for _, c := range comments {
		cs = append(cs, ConvertComment(request, c, additional...))
	}
	return cs
}

// ConvertCommentsResourceID converts between internal and external REST representation, ResourceIdentificationObject only
func ConvertCommentsResourceID(request *goa.RequestData, comments []*comment.Comment, additional ...CommentConvertFunc) []*app.Comment {
	var cs = []*app.Comment{}
	for _, c := range comments {
		cs = append(cs, ConvertCommentResourceID(request, c, additional...))
	}
	return cs
}

// ConvertCommentResourceID converts between internal and external REST representation, ResourceIdentificationObject only
func ConvertCommentResourceID(request *goa.RequestData, comment *comment.Comment, additional ...CommentConvertFunc) *app.Comment {
	c := &app.Comment{
		Type: "comments",
		ID:   &comment.ID,
	}
	for _, add := range additional {
		add(request, comment, c)
	}
	return c
}

// ConvertComment converts between internal and external REST representation
func ConvertComment(request *goa.RequestData, comment *comment.Comment, additional ...CommentConvertFunc) *app.Comment {
	selfURL := AbsoluteURL(request, app.CommentsHref(comment.ID))
	c := &app.Comment{
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
		Links: &app.GenericLinks{
			Self: &selfURL,
		},
	}
	for _, add := range additional {
		add(request, comment, c)
	}
	return c
}

// HrefFunc generic function to greate a relative Href to a resource
type HrefFunc func(id interface{}) string

// CommentIncludeParentWorkItem includes a "parent" relation to a WorkItem
func CommentIncludeParentWorkItem() CommentConvertFunc {
	return func(request *goa.RequestData, comment *comment.Comment, data *app.Comment) {
		CommentIncludeParent(request, comment, data, app.WorkitemHref, APIStringTypeWorkItem)
	}
}

// CommentIncludeParent adds the "parent" relationship to this Comment
func CommentIncludeParent(request *goa.RequestData, comment *comment.Comment, data *app.Comment, ref HrefFunc, parentType string) {
	parentSelf := AbsoluteURL(request, ref(comment.ParentID))

	data.Relationships.Parent = &app.RelationGeneric{
		Data: &app.GenericData{
			Type: &parentType,
			ID:   &comment.ParentID,
		},
		Links: &app.GenericLinks{
			Self: &parentSelf,
		},
	}
}

// AbsoluteURL prefixes a relative URL with absolute address
func AbsoluteURL(req *goa.RequestData, relative string) string {
	scheme := "http"
	if req.TLS != nil { // isHTTPS
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, req.Host, relative)
}
