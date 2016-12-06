package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var projectData = a.Type("ProjectData", func() {
	a.Attribute("type", d.String, "The type of the related resource", func() {
		a.Enum("projects")
	})
	a.Attribute("id", d.String, "ID of the project", func() {
		a.Example("1234")
	})
	a.Attribute("attributes", projectAttributes)

	a.Required("type", "id", "attributes")
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
	a.Required("name", "version", "created-at", "updated-at")
})

var projectResponse = a.MediaType("application/vnd.project+json", func() {
	a.UseTrait("jsonapi-media-type")
	a.TypeName("ProjectResponse")

	a.Attributes(func() {
		a.Attribute("data", projectData)
		a.Required("data")
	})
	a.View("default", func() {
		a.Attribute("data")
		a.Required("data")
	})
})

var projectMeta = a.Type("projectMeta", func() {
	a.Attribute("totalCount", d.Integer)
	a.Required("totalCount")
})

var projectListResponse = a.MediaType("application/vnd.project-list-response+json", func() {
	a.UseTrait("jsonapi-media-type")
	a.TypeName("ProjectListResponse")
	a.Description(`An array of projects`)
	a.Attributes(func() {
		a.Attribute("links", pagingLinks)

		a.Attribute("meta", projectMeta)
		a.Attribute("data", a.ArrayOf(projectData))
		a.Required("data", "meta", "links")
	})
	a.View("default", func() {
		a.Attribute("data")
		a.Attribute("meta")
		a.Attribute("links")
		a.Required("data", "meta", "links")
	})
})

var projectCreateAttributes = a.Type("ProjectCreateAttributes", func() {
	a.Attribute("name", d.String, "Name of the project", func() {
		a.Example("foobar")
	})
	a.Required("name")
})

var projectCreatePayload = a.Type("ProjectCreatePayload", func() {
	a.Attribute("data", func() {
		a.Attribute("type", d.String, "The type of the related resource", func() {
			a.Enum("projects")
		})
		a.Attribute("attributes", projectCreateAttributes)
		a.Required("type", "attributes")
	})
})

var projectUpdateAttributes = a.Type("ProjectUpdateAttributes", func() {
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control (optional during creating)", func() {
		a.Example(0)
	})
	a.Attribute("name", d.String, "Name of the project", func() {
		a.Example("foobar")
	})
	a.Required("version")
})

var projectUpdatePayload = a.Type("ProjectUpdatePayload", func() {
	a.Attribute("data", func() {
		a.Attribute("type", d.String, "The type of the related resource", func() {
			a.Enum("projects")
		})
		a.Attribute("attributes", projectUpdateAttributes)
		a.Required("type", "attributes")
	})
})

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
			a.Media(projectResponse)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List work item link categories.")
		a.Params(func() {
			a.Param("page[offset]", d.String, "Paging start position")
			a.Param("page[limit]", d.Integer, "Paging size")
		})

		a.Response(d.OK, func() {
			a.Media(projectListResponse)
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
		a.Payload(projectCreatePayload)
		a.Response(d.Created, "/projects/.*", func() {
			a.Media(projectResponse)
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
		a.Description("Delete work item link category with given id.")
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
		a.Payload(projectUpdatePayload)
		a.Response(d.OK, func() {
			a.Media(projectResponse)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})
