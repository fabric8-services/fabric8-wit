package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("jenkins", func() {

	a.BasePath("/jenkins")

	a.Action("start", func() {
		a.Security("jwt")
		a.Routing(
			a.GET("start"),
		)
		a.Description("Make sure that jenkins is up and running")
		a.Response(d.TemporaryRedirect)
		a.Response(d.OK, jenkinsState)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})

var jenkinsState = a.MediaType("application/vnd.jenkinsState+json", func() {
	a.TypeName("JenkinsState")
	a.Description("JenkinsState")
	a.Attributes(func() {
		a.Attribute("state", d.String, "Jenkins State: idle|starting|running")
	})
	a.View("default", func() {
		a.Attribute("state", d.String)
		a.Required("state")
	})
})
