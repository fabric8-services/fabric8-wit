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
		a.UseTrait("conditional")
		a.Response(d.OK, userList)
		a.Response(d.NotModified)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.TemporaryRedirect)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("add-many", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("Add users to the list of space collaborators.")
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.TemporaryRedirect)
	})

	a.Action("add", func() {
		a.Security("jwt")
		a.Routing(
			a.POST("/:identityID"),
		)
		a.Description("Add a user to the list of space collaborators.")
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.TemporaryRedirect)
	})

	a.Action("remove-many", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE(""),
		)
		a.Description("Remove users form the list of space collaborators.")
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.TemporaryRedirect)
	})

	a.Action("remove", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/:identityID"),
		)
		a.Description("Remove a user from the list of space collaborators.")
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.TemporaryRedirect)
	})
})
