package controller

import (
	"strings"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/space"
	"github.com/goadesign/goa"
)

// RedirectWorkitemController implements the redirect_workitem resource.
type RedirectWorkitemController struct {
	*goa.Controller
}

// NewRedirectWorkitemController creates a redirect_workitem controller.
func NewRedirectWorkitemController(service *goa.Service) *RedirectWorkitemController {
	return &RedirectWorkitemController{Controller: service.NewController("RedirectWorkitemController")}
}

// Create runs the create action.
func (c *RedirectWorkitemController) Create(ctx *app.CreateRedirectWorkitemContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemsURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

// Delete runs the delete action.
func (c *RedirectWorkitemController) Delete(ctx *app.DeleteRedirectWorkitemContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemsURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

// List runs the list action.
func (c *RedirectWorkitemController) List(ctx *app.ListRedirectWorkitemContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemsURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

// Reorder runs the reorder action.
func (c *RedirectWorkitemController) Reorder(ctx *app.ReorderRedirectWorkitemContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemsURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

// Show runs the show action.
func (c *RedirectWorkitemController) Show(ctx *app.ShowRedirectWorkitemContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemsURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

// Update runs the update action.
func (c *RedirectWorkitemController) Update(ctx *app.UpdateRedirectWorkitemContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemsURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

func redirectWorkItemsURL(url string) string {
	return strings.Replace(url, "/workitems", "/spaces/"+space.SystemSpace.String()+"/workitems", -1)
}
