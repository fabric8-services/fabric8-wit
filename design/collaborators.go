package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("collaborators", func() {
	a.Parent("space")
	a.BasePath("/collaborators")

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List collaborators for the given space ID.")
		a.Params(func() {
			a.Param("page[offset]", d.String, "Paging start position")
			a.Param("page[limit]", d.Integer, "Paging size")
		})
		a.Response(d.OK, userList)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("add", func() {
		a.Security("jwt")
		a.Routing(
			a.POST("/:identityID"),
		)
		a.Description("Add a user to the list of space collaborators.")
		a.Response(d.OK)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})

	a.Action("remove", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/:identityID"),
		)
		a.Description("Remove a user from the list of space collaborators.")
		a.Response(d.OK)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})
