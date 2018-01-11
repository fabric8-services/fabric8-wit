package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("features", func() {
	a.BasePath("/api/features")

	a.Action("show", func() {
		a.Security("jwt")
		a.Routing(
			a.GET("/:featureName"),
		)
		a.Params(func() {
			a.Param("featureName", d.String, "featureName")
		})
		a.Description("Show feature details.")
		a.Response(d.OK)
		a.Response(d.BadRequest)
		a.Response(d.NotFound)
		a.Response(d.InternalServerError)
		a.Response(d.Unauthorized)
	})

	a.Action("list", func() {
		a.Security("jwt")
		a.Routing(
			a.GET(""),
		)
		a.Description("Show a list of features enabled.")
		a.Response(d.OK)
		a.Response(d.BadRequest)
		a.Response(d.NotFound)
		a.Response(d.InternalServerError)
		a.Response(d.Unauthorized)
	})
})
