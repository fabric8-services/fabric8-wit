package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("pipelines", func() {
	a.BasePath("/pipelines")

	// An auth token is required to call the auth API to get an OpenShift auth token.
	a.Security("jwt")

	a.Action("delete", func() {
		a.Routing(
			a.DELETE(""),
		)
		a.Description("Delete pipelines")
		a.Params(func() {
			a.Param("space", d.String, "Name of the space")
			a.Required("space")
		})
		a.Response(d.OK)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
	})
})
