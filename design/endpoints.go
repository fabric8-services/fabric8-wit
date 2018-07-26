package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var endpoint = a.Type("Endpoints", func() {
	a.Description("JSONAPI store for the data of all endpoints.")
	a.Attribute("relationships", a.HashOf(d.String, d.Any), "Describes relationship between names and links")
	a.Attribute("type", d.String, func() {
		a.Enum("endpoints")
	})
	a.Attribute("id", d.UUID, "ID of endpoints (this is a newly generated UUID upon every call)")
	a.Attribute("links", genericLinks)
	a.Required("type", "links", "id", "relationships")
})

var endpointsSingle = JSONSingle(
	"Endpoint", "Contains endpoints",
	endpoint,
	nil)

var _ = a.Resource("endpoints", func() {
	a.BasePath("/endpoints")
	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List all endpoints. ")
		a.Response(d.OK, endpointsSingle)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
})
