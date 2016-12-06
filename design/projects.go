package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var project = a.Type("Project", func() {
	a.Attribute("type", d.String, "The type of the related resource", func() {
		a.Enum("projects")
	})
	a.Attribute("id", d.UUID, "ID of the project", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", projectAttributes)
	//a.Attribute("relationships", projectRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type", "id")
})

var projectAttributes = a.Type("ProjectAttributes", func() {
	a.Attribute("name", d.String, "Name of the project", func() {
		a.Example("foobar")
	})
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control (optional during creating)", func() {
		a.Example(23)
	})
	a.Attribute("created-at", d.DateTime, "When the project was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("updated-at", d.DateTime, "When the project was updated", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
})

var projectListMeta = a.Type("ProjectListMeta", func() {
	a.Attribute("totalCount", d.Integer)
	a.Required("totalCount")
})

/*
var projectRelationships = a.Type("ProjectRelations", func() {
})
*/

// workItemList contains paged results for listing work items and paging links
var projectList = JSONList(
	"Project", "Holds the paginated response to a project list request",
	project,
	pagingLinks,
	projectListMeta)

// projectSingle is the media type for work items
var projectSingle = JSONSingle(
	"Project", "Holds a single response to a project request",
	project,
	nil)

var _ = a.Resource("project", func() {
	a.BasePath("/projects")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:id"),
		)
		a.Description("Retrieve project (as JSONAPI) for the given ID.")
		a.Params(func() {
			a.Param("id", d.String, "ID of the project")
		})
		a.Response(d.OK, func() {
			a.Media(projectSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List projects.")
		a.Params(func() {
			a.Param("page[offset]", d.String, "Paging start position")
			a.Param("page[limit]", d.Integer, "Paging size")
		})

		a.Response(d.OK, func() {
			a.Media(projectList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("Create a project")
		a.Payload(projectSingle)
		a.Response(d.Created, "/projects/.*", func() {
			a.Media(projectSingle)
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
		a.Description("Delete a project with given id.")
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
		a.Description("Update the project with given id.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Payload(projectSingle)
		a.Response(d.OK, func() {
			a.Media(projectSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})
