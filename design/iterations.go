package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var iteration = a.Type("Iteration", func() {
	a.Description(`JSONAPI store for the data of a iteration.  See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("iterations")
	})
	a.Attribute("id", d.UUID, "ID of iteration", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", iterationAttributes)
	a.Attribute("relationships", iterationRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

var iterationAttributes = a.Type("IterationAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a iteration. +See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("name", d.String, "The iteration name", func() {
		a.Example("Sprint #24")
	})
	a.Attribute("description", d.String, "Description of the iteration ", func() {
		a.Example("this is the description of iteration")
	})
	a.Attribute("startAt", d.DateTime, "When the iteration starts", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("endAt", d.DateTime, "When the iteration starts", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
})

var iterationRelationships = a.Type("IterationRelations", func() {
	a.Attribute("space", relationGeneric, "This defines the owning space")
	a.Attribute("parent", relationGeneric, "This defines the parent iteration")
	a.Attribute("workitems", relationGeneric, "This defines the workitems associated with the iteration")
})

var iterationList = JSONList(
	"Iteration", "Holds the list of iterations",
	iteration,
	pagingLinks,
	meta)

var iterationSingle = JSONSingle(
	"Iteration", "Holds the list of iterations",
	iteration,
	nil)

// new version of "list" for migration
var _ = a.Resource("iteration", func() {
	a.BasePath("/iterations")
	a.Action("show", func() {
		a.Routing(
			a.GET("/:id"),
		)
		a.Description("Retrieve iteration with given id.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Response(d.OK, func() {
			a.Media(iterationSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
	a.Action("create-child", func() {
		a.Security("jwt")
		a.Routing(
			a.POST("/:id"),
		)
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Description("create child iteration.")
		a.Payload(iterationSingle)
		a.Response(d.Created, "/iterations/.*", func() {
			a.Media(iterationSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})

// new version of "list" for migration
var _ = a.Resource("space-iterations", func() {
	a.Parent("space")

	a.Action("list", func() {
		a.Routing(
			a.GET("iterations"),
		)
		a.Description("List iterations.")
		/*
			a.Params(func() {
				a.Param("filter", d.String, "a query language expression restricting the set of found work items")
				a.Param("page[offset]", d.String, "Paging start position")
				a.Param("page[limit]", d.Integer, "Paging size")
			})
		*/
		a.Response(d.OK, func() {
			a.Media(iterationList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST("iterations"),
		)
		a.Description("Create iteration.")
		a.Payload(iterationSingle)
		a.Response(d.Created, "/iterations/.*", func() {
			a.Media(iterationSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})
