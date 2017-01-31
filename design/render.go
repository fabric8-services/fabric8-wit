package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// MarkupRenderingPayloadData is the media type representing a rendering input.
var markupRenderingPayloadData = a.Type("MarkupRenderingPayloadData", func() {
	a.Description("A render type describes the values a render type instance can hold.")
	a.Attribute("id", d.String, "an ID to conform to the JSON-API spec, even though it is meaningless in the case of the rendering endpoint. Can be null", func() {
		a.Example("42")
	})
	a.Attribute("type", d.String, func() {
		a.Enum("rendering")
	})
	a.Attribute("attributes", a.HashOf(d.String, d.Any), func() {
		a.Example(map[string]interface{}{"markup": "Markdown", "content": "# foo"})
	})
	a.Required("id")
	a.Required("type")
	a.Required("attributes")
})

// MarkupRenderingPayload wraps the data in a JSONAPI compliant request
var markupRenderingPayload = a.Type("MarkupRenderingPayload", func() {
	a.Attribute("data", markupRenderingPayloadData)
	a.Required("data")
})

// MarkupRenderingMediaType is the media type representing a rendering result.
var markupRenderingMediaTypeData = a.Type("MarkupRendering", func() {
	a.Description("A MarkupRendering type describes the values a render type instance can hold.")
	a.Attribute("id", d.String, "an ID to conform to the JSON-API spec, even though it is meaningless in the case of the rendering endpoint. Can be null", func() {
		a.Example("42")
	})
	a.Attribute("type", d.String, func() {
		a.Enum("rendering")
	})
	a.Attribute("attributes", a.HashOf(d.String, d.Any), func() {
		a.Example(map[string]interface{}{"renderedContent": "<h1>foo</h1>"})
	})
	a.Required("id")
	a.Required("type")
	a.Required("attributes")
})

// workItemLinkCategory is the media type for work item link categories
var markupRenderingMediaType = JSONSingle(
	"MarkupRendering",
	`MarkupRenderingMediaType contains the  
		rendering of the 'content' provided in the request, using
		the markup language specified by the 'markup' value.`,
	markupRenderingMediaTypeData,
	nil,
)

var _ = a.Resource("render", func() {
	a.BasePath("/render")
	a.Security("jwt")
	a.Action("render", func() {
		a.Description("Render some content using the markup language")
		a.Routing(a.POST(""))
		a.Payload(markupRenderingPayload)
		a.Response(d.OK, markupRenderingMediaType)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})
