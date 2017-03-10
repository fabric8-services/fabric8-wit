package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("collabspaces", func() {
	a.BasePath("/collabspaces")

	a.Action("list", func() {
		a.Routing(
			a.GET("/:userName"),
		)
		a.Description("Retrieve a list of spaces (as JSONAPI) where the given user name is a collaborator.")
		a.Params(func() {
			a.Param("userName", d.String, "User name of the owner of the space")
			a.Param("page[offset]", d.String, "Paging start position")
			a.Param("page[limit]", d.Integer, "Paging size")
		})
		a.Response(d.OK, func() {
			a.Media(spaceList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
})
