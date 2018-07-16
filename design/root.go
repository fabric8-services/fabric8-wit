package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
	)

var root = a.Type("Root", func() {

	a.Description("JSONAPI store for the data of a Root.")
	a.Attribute("relationships", a.HashOf(d.String, d.Any), "Describes relationship between names and links")
	a.Attribute("basePath", d.String, "Base path to all endpoints")
	a.Attribute("attributes", d.Any)
	a.Attribute("id", d.UUID, "ID of root")
	a.Attribute("links", genericLinksForRoot, "Describes the related path")

})

// genericLinksForRoot defines generic relations links that are specific to a root
var genericLinksForRoot = a.Type("GenericLinksForRoot", func() {
	a.Attribute("self", d.String)
})

var rootSingle = JSONSingle(
	"Root", "Holds a single Root",
	root,
	nil)

var _ = a.Resource("root", func() {
	a.BasePath("/root")
	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List all endpoints. ")
		a.Response(d.OK, rootSingle)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
})
