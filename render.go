package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

const (
	RenderingType = "rendering"
	RenderedValue = "value"
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
	content := ctx.Payload.Data.Attributes[workitem.ContentKey].(string)
	markup := ctx.Payload.Data.Attributes[workitem.MarkupKey].(string)
	if !rendering.IsMarkupSupported(markup) {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("Unsupported markup type", markup))
	}
	htmlResult := rendering.RenderMarkupToHTML(content, markup)
	resultAttributes := make(map[string]interface{})
	resultAttributes[RenderedValue] = htmlResult
	res := &app.MarkupRenderingSingle{Data: &app.MarkupRendering{
		ID:         uuid.NewV4().String(),
		Type:       RenderingType,
		Attributes: resultAttributes}}
	return ctx.OK(res)
}
