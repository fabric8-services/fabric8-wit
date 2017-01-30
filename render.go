package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/rendering"
	"github.com/goadesign/goa"
)

// RenderController implements the render resource.
type RenderController struct {
	*goa.Controller
}

// NewRenderController creates a render controller.
func NewRenderController(service *goa.Service) *RenderController {
	return &RenderController{Controller: service.NewController("RenderController")}
}

// Render runs the render action.
func (c *RenderController) Render(ctx *app.RenderRenderContext) error {
	content := ctx.Payload.Content
	markup := ctx.Payload.Markup
	if !rendering.IsMarkupSupported(markup) {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("Unsupported markup type", markup))
	}
	htmlResult := rendering.RenderMarkupToHTML(content, markup)
	res := &app.RenderOutputMediaType{Rendered: htmlResult}
	return ctx.OK(res)
}
