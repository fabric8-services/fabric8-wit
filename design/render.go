package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// MarkupRenderingInputType is the media type representing a rendering input.
var MarkupRenderingInputType = a.Type("MarkupRenderingInputType", func() {
	a.Description("A render type describes the values a render type instance can hold.")
	a.Attribute("markup", d.String, "The name of the markup language used to input the content")
	a.Attribute("content", d.String, "The content to be rendered")
	a.Required("markup")
	a.Required("content")

})

// MarkupRenderingMediaType is the media type representing a rendering result.
var MarkupRenderingMediaType = a.MediaType("application/vnd.markuprendering+json", func() {
	a.TypeName("RenderOutputMediaType")
	a.Description("A render type describes the values a render type instance can hold.")
	a.Attribute("rendered", d.String, "The result of the rendering of a given content in a given markup language")
	a.Required("rendered")
	a.View("default", func() {
		a.Attribute("rendered")
	})
})

var _ = a.Resource("render", func() {
	a.BasePath("/render")
	a.Action("render", func() {
		a.Description("Render some content using the markup language")
		a.Routing(a.POST(""))
		a.Payload(MarkupRenderingInputType)
		a.Response(d.OK, MarkupRenderingMediaType)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})
