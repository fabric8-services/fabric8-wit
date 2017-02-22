package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var space = a.Type("Space", func() {
	a.Attribute("type", d.String, "The type of the related resource", func() {
		a.Enum("spaces")
	})
	a.Attribute("id", d.UUID, "ID of the space", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", spaceAttributes)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
	a.Attribute("relationships", spaceRelationships)
})

var spaceRelationships = a.Type("SpaceRelationships", func() {
	a.Attribute("iterations", relationGeneric, "Space can have one or many iterations")
	a.Attribute("areas", relationGeneric, "Space can have one or many areas")
	a.Attribute("workitemlinktypes", relationGeneric, "Space can have one or many work item link types")
})

var spaceAttributes = a.Type("SpaceAttributes", func() {
	a.Attribute("name", d.String, "Name of the space", func() {
		a.Example("foobar")
	})
	a.Attribute("description", d.String, "Description for the space", func() {
		a.Example("This is the foobar collaboration space")
	})
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control (optional during creating)", func() {
		a.Example(23)
	})
	a.Attribute("created-at", d.DateTime, "When the space was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("updated-at", d.DateTime, "When the space was updated", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
})

var spaceListMeta = a.Type("SpaceListMeta", func() {
	a.Attribute("totalCount", d.Integer)
	a.Required("totalCount")
})

var spaceList = JSONList(
	"Space", "Holds the paginated response to a space list request",
	space,
	pagingLinks,
	spaceListMeta)

var spaceSingle = JSONSingle(
	"Space", "Holds a single response to a space request",
	space,
	nil)

var _ = a.Resource("space", func() {
	a.BasePath("/spaces")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:id"),
		)
		a.Description("Retrieve space (as JSONAPI) for the given ID.")
		a.Params(func() {
			a.Param("id", d.String, "ID of the space")
		})
		a.Response(d.OK, func() {
			a.Media(spaceSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List spaces.")
		a.Params(func() {
			a.Param("page[offset]", d.String, "Paging start position")
			a.Param("page[limit]", d.Integer, "Paging size")
		})

		a.Response(d.OK, func() {
			a.Media(spaceList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("Create a space")
		a.Payload(spaceSingle)
		a.Response(d.Created, "/spaces/.*", func() {
			a.Media(spaceSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})

	a.Action("delete", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/:id"),
		)
		a.Description("Delete a space with given id.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Response(d.OK)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
	a.Action("update", func() {
		a.Security("jwt")
		a.Routing(
			a.PATCH("/:id"),
		)
		a.Description("Update the space with given id.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Payload(spaceSingle)
		a.Response(d.OK, func() {
			a.Media(spaceSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})
