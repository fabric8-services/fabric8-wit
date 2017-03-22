package controller

import (
	"strings"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/space"
	"github.com/goadesign/goa"
)

// RedirectWorkitemtypeController implements the redirect_workitemtype resource.
type RedirectWorkitemtypeController struct {
	*goa.Controller
}

// NewRedirectWorkitemtypeController creates a redirect_workitemtype controller.
func NewRedirectWorkitemtypeController(service *goa.Service) *RedirectWorkitemtypeController {
	return &RedirectWorkitemtypeController{Controller: service.NewController("RedirectWorkitemtypeController")}
}

// Create runs the create action.
func (c *RedirectWorkitemtypeController) Create(ctx *app.CreateRedirectWorkitemtypeContext) error {
	return ctx.MovedPermanently()
}

// List runs the list action.
func (c *RedirectWorkitemtypeController) List(ctx *app.ListRedirectWorkitemtypeContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemTypesURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

// ListSourceLinkTypes runs the list-source-link-types action.
func (c *RedirectWorkitemtypeController) ListSourceLinkTypes(ctx *app.ListSourceLinkTypesRedirectWorkitemtypeContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemTypesURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

// ListTargetLinkTypes runs the list-target-link-types action.
func (c *RedirectWorkitemtypeController) ListTargetLinkTypes(ctx *app.ListTargetLinkTypesRedirectWorkitemtypeContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemTypesURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

// Show runs the show action.
func (c *RedirectWorkitemtypeController) Show(ctx *app.ShowRedirectWorkitemtypeContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemTypesURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

func redirectWorkItemTypesURL(url string) string {
	return strings.Replace(url, "/workitemtypes", "/spaces/"+space.SystemSpace.String()+"/workitemtypes", -1)
}
