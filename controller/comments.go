package controller

import (
	"html"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/comment"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/rest"
	"github.com/goadesign/goa"
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
	return application.Transactional(c.db, func(appl application.Application) error {
		c, err := appl.Comments().Load(ctx, ctx.CommentID)
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
	identityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		cm, err := appl.Comments().Load(ctx.Context, ctx.CommentID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		if *identityID != cm.CreatedBy {
			// need to use the goa.NewErrorClass() func as there is no native support for 403 in goa
			// and it is not planned to be supported yet: https://github.com/goadesign/goa/pull/1030
			return jsonapi.JSONErrorResponse(ctx, goa.NewErrorClass("forbidden", 403)("User is not the comment author"))
		}

		cm.Body = *ctx.Payload.Data.Attributes.Body
		cm.Markup = rendering.NilSafeGetMarkup(ctx.Payload.Data.Attributes.Markup)
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

// Delete does DELETE comment
func (c *CommentsController) Delete(ctx *app.DeleteCommentsContext) error {
	identityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		cm, err := appl.Comments().Load(ctx.Context, ctx.CommentID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		if *identityID != cm.CreatedBy {
			// need to use the goa.NewErrorClass() func as there is no native support for 403 in goa
			// and it is not planned to be supported yet: https://github.com/goadesign/goa/pull/1030
			return jsonapi.JSONErrorResponse(ctx, goa.NewErrorClass("forbidden", 403)("User is not the comment author"))
		}

		err = appl.Comments().Delete(ctx.Context, cm.ID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.OK([]byte{})
	})
}

// CommentConvertFunc is a open ended function to add additional links/data/relations to a Comment during
// conversion from internal to API
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
	selfURL := rest.AbsoluteURL(request, app.CommentsHref(comment.ID))
	markup := rendering.NilSafeGetMarkup(&comment.Markup)
	bodyRendered := rendering.RenderMarkupToHTML(html.EscapeString(comment.Body), comment.Markup)
	c := &app.Comment{
		Type: "comments",
		ID:   &comment.ID,
		Attributes: &app.CommentAttributes{
			Body:         &comment.Body,
			BodyRendered: &bodyRendered,
			Markup:       &markup,
			CreatedAt:    &comment.CreatedAt,
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
	parentSelf := rest.AbsoluteURL(request, ref(comment.ParentID))

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
