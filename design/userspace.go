package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("userspace", func() {
	a.BasePath("/userspace")

	a.Action("create", func() {
		a.Routing(
			a.POST("/*"),
		)
		a.Description("Data dump endpoint ")
		a.Payload(a.HashOf(d.String, d.Any))
		a.Response(d.Created)
		a.Response(d.InternalServerError)
	})
})
