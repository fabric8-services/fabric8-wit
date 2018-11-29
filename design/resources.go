package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("status", func() {

	a.DefaultMedia(WITStatus)
	a.BasePath("/status")

	a.Action("show", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("Show the status of the current running instance")
		a.Response(d.OK)
		a.Response(d.ServiceUnavailable, WITStatus)
	})

})

var nameValidationFunction = func() {
	a.MaxLength(63) // maximum name length is 63 characters
	a.MinLength(1)  // minimum name length is 1 characters
	a.Pattern("^[^_|-].*")
	a.Example("name for the object")
}
