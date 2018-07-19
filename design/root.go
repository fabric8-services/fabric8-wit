package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var root = a.Type("Root", func() {
	a.Description("JSONAPI store for the data of a Root.")
	a.Attribute("relationships", a.HashOf(d.String, d.Any), "Describes relationship between names and links")
	a.Attribute("type", d.String, func() {
		a.Enum("endpoints")
	})
	a.Attribute("id", d.UUID, "ID of root (this is a newly generated UUID upon every call)")
	a.Attribute("links", genericLinks)
	a.Required("type", "links", "id", "relationships")
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
