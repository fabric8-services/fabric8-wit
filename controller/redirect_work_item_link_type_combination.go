package controller

import (
	"strings"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/space"
	"github.com/goadesign/goa"
)

// RedirectWorkItemLinkTypeCombinationController implements the redirect_work_item_link_type_combination resource.
type RedirectWorkItemLinkTypeCombinationController struct {
	*goa.Controller
}

// NewRedirectWorkItemLinkTypeCombinationController creates a redirect_work_item_link_type_combination controller.
func NewRedirectWorkItemLinkTypeCombinationController(service *goa.Service) *RedirectWorkItemLinkTypeCombinationController {
	return &RedirectWorkItemLinkTypeCombinationController{Controller: service.NewController("RedirectWorkItemLinkTypeCombinationController")}
}

// Create runs the create action.
func (c *RedirectWorkItemLinkTypeCombinationController) Create(ctx *app.CreateRedirectWorkItemLinkTypeCombinationContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemLinkTypeCombinationsURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

// Show runs the show action.
func (c *RedirectWorkItemLinkTypeCombinationController) Show(ctx *app.ShowRedirectWorkItemLinkTypeCombinationContext) error {
	ctx.ResponseData.Header().Set("Location", redirectWorkItemLinkTypeCombinationsURL(ctx.RequestURI))
	return ctx.MovedPermanently()
}

func redirectWorkItemLinkTypeCombinationsURL(url string) string {
	return strings.Replace(url, "/workitemlinktypecombinations", "/spaces/"+space.SystemSpace.String()+"/workitemlinktypecombinations", -1)
}
