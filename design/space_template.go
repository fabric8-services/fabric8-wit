package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("space_template", func() {
	a.BasePath("/spacetemplates")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:spaceTemplateID"),
		)
		a.Description("Retrieve space template with given ID")
		a.Params(func() {
			a.Param("spaceTemplateID", d.UUID, "id of the space template to fetch")
		})
		a.Response(d.MethodNotAllowed)
		a.Response(d.TemporaryRedirect)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})
