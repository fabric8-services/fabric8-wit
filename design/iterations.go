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
	a.Attribute("name", d.String, desc("The iteration name").mandatoryOnCreate().String(), nameValidationFunction)
	a.Attribute("description", d.String, desc("Description of the iteration").String(), func() {
		a.Example("Sprint #42 focusing on UI and build process improvements")
	})
	a.Attribute("created-at", d.DateTime, desc("When the iteration was created").String(), func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("updated-at", d.DateTime, desc("When the iteration was updated").String(), func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("startAt", d.DateTime, desc("When the iteration starts").String(), func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("endAt", d.DateTime, desc("When the iteration ends").String(), func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("state", d.String, desc("State of an iteration").String(), func() {
		a.Enum("new", "start", "close")
	})
	a.Attribute("user_active", d.Boolean, desc("Active flag set by user").String(), func() {
	})
	a.Attribute("active_status", d.Boolean, desc("Active status of iteration calculated using user_active, startAt and endAt").String(), func() {
	})
	a.Attribute("parent_path", d.String, desc("Path string separataed by / having UUIDs of all parent iterations").String(), func() {
		a.Example("/8ab013be-6477-41e2-b206-53593dac6543/300d9835-fcf7-4d2f-a629-1919de091663/42f0dabd-16bf-40a6-a521-888ec2ad7461")
	})
	a.Attribute("resolved_parent_path", d.String, desc("Path string separataed by / having names of all parent iterations").String(), func() {
		a.Example("/beta/Web-App/Sprint 9/Sprint 9.1")
	})
})

var iterationRelationships = a.Type("IterationRelations", func() {
	a.Attribute("space", relationGeneric, desc("This defines the owning space").String())
	a.Attribute("parent", relationGeneric, desc("This defines the parent iteration").String())
	a.Attribute("workitems", relationGeneric, desc("This defines the workitems associated with the iteration").String())
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
			a.GET("/:iterationID"),
		)
		a.Description("Retrieve iteration with given id.")
		a.Params(func() {
			a.Param("iterationID", d.String, "Iteration Identifier")
		})
		a.UseTrait("conditional")
		a.Response(d.OK, iterationSingle)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
	a.Action("create-child", func() {
		a.Security("jwt")
		a.Routing(
			a.POST("/:iterationID"),
		)
		a.Params(func() {
			a.Param("iterationID", d.String, "Iteration Identifier")
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
		a.Response(d.Forbidden, JSONAPIErrors)
		a.Response(d.Conflict, JSONAPIErrors)
	})
	a.Action("update", func() {
		a.Security("jwt")
		a.Routing(
			a.PATCH("/:iterationID"),
		)
		a.Description("update the iteration for the given id.")
		a.Params(func() {
			a.Param("iterationID", d.String, "Iteration Identifier")
		})
		a.Payload(iterationSingle)
		a.Response(d.OK, func() {
			a.Media(iterationSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
	})
	a.Action("delete", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/:iterationID"),
		)
		a.Description("delete the iteration for the given id.")
		a.Params(func() {
			a.Param("iterationID", d.UUID, "ID of an iteration to delete")
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
		a.Response(d.NoContent)
	})
})

// new version of "list" for migration
var _ = a.Resource("space_iterations", func() {
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
		a.UseTrait("conditional")
		a.Response(d.OK, iterationList)
		a.Response(d.NotModified)
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
		a.Response(d.Forbidden, JSONAPIErrors)
	})
})
