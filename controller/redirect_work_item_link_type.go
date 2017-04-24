package controller

import (
	"strings"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/space"
	"github.com/goadesign/goa"
)

// RedirectWorkItemLinkTypeController implements the redirect_work_item_link_type resource.
type RedirectWorkItemLinkTypeController struct {
	*goa.Controller
}

// NewRedirectWorkItemLinkTypeController creates a redirect_work_item_link_type controller.
func NewRedirectWorkItemLinkTypeController(service *goa.Service) *RedirectWorkItemLinkTypeController {
	return &RedirectWorkItemLinkTypeController{Controller: service.NewController("RedirectWorkItemLinkTypeController")}
}

// Create runs the create action.
func (c *RedirectWorkItemLinkTypeController) Create(ctx *app.CreateRedirectWorkItemLinkTypeContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemLinkTypesURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

// Delete runs the delete action.
func (c *RedirectWorkItemLinkTypeController) Delete(ctx *app.DeleteRedirectWorkItemLinkTypeContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemLinkTypesURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

// List runs the list action.
func (c *RedirectWorkItemLinkTypeController) List(ctx *app.ListRedirectWorkItemLinkTypeContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemLinkTypesURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

// Show runs the show action.
func (c *RedirectWorkItemLinkTypeController) Show(ctx *app.ShowRedirectWorkItemLinkTypeContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemLinkTypesURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

// Update runs the update action.
func (c *RedirectWorkItemLinkTypeController) Update(ctx *app.UpdateRedirectWorkItemLinkTypeContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemLinkTypesURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

func redirectWorkItemLinkTypesURL(url string) string {
	return strings.Replace(url, "/workitemlinktypes", "/spaces/"+space.SystemSpace.String()+"/workitemlinktypes", -1)
}
