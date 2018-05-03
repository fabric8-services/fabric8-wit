package controller

import (
	"context"
	"fmt"
	"html"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/comment"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/notification"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space/authz"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// CommentsController implements the comments resource.
type CommentsController struct {
	*goa.Controller
	db           application.DB
	notification notification.Channel
	config       CommentsControllerConfiguration
}

// CommentsControllerConfiguration the configuration for CommentsController
type CommentsControllerConfiguration interface {
	GetCacheControlComments() string
	GetCacheControlComment() string
}

// NewCommentsController creates a comments controller.
func NewCommentsController(service *goa.Service, db application.DB, config CommentsControllerConfiguration) *CommentsController {
	return NewNotifyingCommentsController(service, db, &notification.DevNullChannel{}, config)
}

// NewNotifyingCommentsController creates a comments controller with notification broadcast.
func NewNotifyingCommentsController(service *goa.Service, db application.DB, notificationChannel notification.Channel, config CommentsControllerConfiguration) *CommentsController {
	n := notificationChannel
	if n == nil {
		n = &notification.DevNullChannel{}
	}
	return &CommentsController{
		Controller:   service.NewController("CommentsController"),
		db:           db,
		notification: n,
		config:       config,
	}
}

// Show runs the show action.
func (c *CommentsController) Show(ctx *app.ShowCommentsContext) error {
	var cmt *comment.Comment
	err := application.Transactional(c.db, func(appl application.Application) error {
		var err error
		cmt, err = appl.Comments().Load(ctx, ctx.CommentID)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.ConditionalRequest(*cmt, c.config.GetCacheControlComment, func() error {
		res := &app.CommentSingle{}
		// This code should change if others type of parents than WI are allowed
		includeParentWorkItem := CommentIncludeParentWorkItem(ctx, cmt)
		res.Data = ConvertComment(
			ctx.Request,
			*cmt,
			includeParentWorkItem)
		return ctx.OK(res)
	})
}

// Update does PATCH comment
func (c *CommentsController) Update(ctx *app.UpdateCommentsContext) error {
	identityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	cm, wi, userIsCreator, err := c.loadComment(ctx.Context, ctx.CommentID, *identityID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	// User is allowed to update if user is creator of the comment OR user is a space collaborator
	if !userIsCreator {
		authorized, err := authz.Authorize(ctx, wi.SpaceID.String())
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
		}
		if !authorized {
			return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not a space collaborator"))
		}
	}
	err = c.performUpdate(ctx, cm, identityID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	// This code should change if others type of parents than WI are allowed
	res := &app.CommentSingle{
		Data: ConvertComment(ctx.Request, *cm, CommentIncludeParentWorkItem(ctx, cm)),
	}
	c.notification.Send(ctx, notification.NewCommentUpdated(cm.ID.String()))
	return ctx.OK(res)
}

func (c *CommentsController) loadComment(ctx context.Context, commentID, identityID uuid.UUID) (cm *comment.Comment, wi *workitem.WorkItem, userIsCreator bool, err error) {
	// Following transaction verifies if a user is allowed to update or not
	err = application.Transactional(c.db, func(appl application.Application) error {
		cm, err = appl.Comments().Load(ctx, commentID)
		if err != nil {
			return err
		}
		if identityID == cm.Creator {
			userIsCreator = true
			return err
		}
		wi, err = appl.WorkItems().LoadByID(ctx, cm.ParentID)
		if err != nil {
			return err
		}
		return nil
	})
	return // using names returned value
}

func (c *CommentsController) performUpdate(ctx *app.UpdateCommentsContext, cm *comment.Comment, identityID *uuid.UUID) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		cm.Body = *ctx.Payload.Data.Attributes.Body
		cm.Markup = rendering.NilSafeGetMarkup(ctx.Payload.Data.Attributes.Markup)
		err := appl.Comments().Save(ctx.Context, cm, *identityID)
		return err
	})
}

// Delete does DELETE comment
func (c *CommentsController) Delete(ctx *app.DeleteCommentsContext) error {
	identityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	cm, wi, userIsCreator, err := c.loadComment(ctx.Context, ctx.CommentID, *identityID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	// User is allowed to delete if user is creator of the comment OR user is a space collaborator
	if !userIsCreator {
		authorized, err := authz.Authorize(ctx, wi.SpaceID.String())
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
		}
		if !authorized {
			return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not a space collaborator"))
		}
	}
	err = application.Transactional(c.db, func(appl application.Application) error {
		return appl.Comments().Delete(ctx.Context, cm.ID, *identityID)
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.OK([]byte{})
}

// CommentConvertFunc is a open ended function to add additional links/data/relations to a Comment during
// conversion from internal to API
type CommentConvertFunc func(*http.Request, *comment.Comment, *app.Comment)

// ConvertComments converts between internal and external REST representation
func ConvertComments(request *http.Request, comments []comment.Comment, additional ...CommentConvertFunc) []*app.Comment {
	var cs = []*app.Comment{}
	for _, c := range comments {
		cs = append(cs, ConvertComment(request, c, additional...))
	}
	return cs
}

// ConvertCommentsResourceID converts between internal and external REST representation, ResourceIdentificationObject only
func ConvertCommentsResourceID(request *http.Request, comments []comment.Comment, additional ...CommentConvertFunc) []*app.Comment {
	var cs = []*app.Comment{}
	for _, c := range comments {
		cs = append(cs, ConvertCommentResourceID(request, c, additional...))
	}
	return cs
}

// ConvertCommentResourceID converts between internal and external REST representation, ResourceIdentificationObject only
func ConvertCommentResourceID(request *http.Request, comment comment.Comment, additional ...CommentConvertFunc) *app.Comment {
	c := &app.Comment{
		Type: "comments",
		ID:   &comment.ID,
	}
	for _, add := range additional {
		add(request, &comment, c)
	}
	return c
}

// ConvertComment converts between internal and external REST representation
func ConvertComment(request *http.Request, comment comment.Comment, additional ...CommentConvertFunc) *app.Comment {
	relatedURL := rest.AbsoluteURL(request, app.CommentsHref(comment.ID))
	relatedCreatorLink := rest.AbsoluteURL(request, fmt.Sprintf("%s/%s", usersEndpoint, comment.Creator))
	c := &app.Comment{
		Type: "comments",
		ID:   &comment.ID,
		Attributes: &app.CommentAttributes{
			Body:         &comment.Body,
			BodyRendered: ptr.String(rendering.RenderMarkupToHTML(html.EscapeString(comment.Body), comment.Markup)),
			Markup:       ptr.String(comment.Markup.NilSafeGetMarkup().String()),
			CreatedAt:    &comment.CreatedAt,
			UpdatedAt:    &comment.UpdatedAt,
		},
		Relationships: &app.CommentRelations{
			Creator: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: ptr.String(APIStringTypeUser),
					ID:   ptr.String(comment.Creator.String()),
					Links: &app.GenericLinks{
						Related: &relatedCreatorLink,
					},
				},
			},
			CreatedBy: &app.CommentCreatedBy{ // Keep old API style until all cients are updated
				Data: &app.IdentityRelationData{
					Type: APIStringTypeUser,
					ID:   &comment.Creator,
				},
				Links: &app.GenericLinks{
					Related: &relatedCreatorLink,
				},
			},
		},
		Links: &app.GenericLinks{
			Self:    &relatedURL,
			Related: &relatedURL,
		},
	}
	for _, add := range additional {
		add(request, &comment, c)
	}
	return c
}

// HrefFunc generic function to greate a relative Href to a resource
type HrefFunc func(id interface{}) string

// CommentIncludeParentWorkItem includes a "parent" relation to a WorkItem
func CommentIncludeParentWorkItem(ctx context.Context, c *comment.Comment) CommentConvertFunc {
	return func(request *http.Request, comment *comment.Comment, data *app.Comment) {
		HrefFunc := func(obj interface{}) string {
			return fmt.Sprintf(app.WorkitemHref("%v"), obj)
		}
		CommentIncludeParent(request, comment, data, HrefFunc, APIStringTypeWorkItem)
	}
}

// CommentIncludeParent adds the "parent" relationship to this Comment
func CommentIncludeParent(request *http.Request, comment *comment.Comment, data *app.Comment, href HrefFunc, parentType string) {
	data.Relationships.Parent = &app.RelationGeneric{
		Data: &app.GenericData{
			Type: &parentType,
			ID:   ptr.String(comment.ParentID.String()),
		},
		Links: &app.GenericLinks{
			Self: ptr.String(rest.AbsoluteURL(request, href(comment.ParentID))),
		},
	}
}
