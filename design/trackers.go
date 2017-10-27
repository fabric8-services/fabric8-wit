package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var tracker = a.Type("Tracker", func() {
	a.Description(`JSONAPI store for the data of a Tracker. See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("trackers")
	})
	a.Attribute("id", d.UUID, "ID of tracker", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", trackerAttributes)
	a.Attribute("relationships", trackerRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

var trackerAttributes = a.Type("TrackerAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a Tracker. See also http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("created-at", d.DateTime, "When the label was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("updated-at", d.DateTime, "When the label was updated", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("URL", d.String, "URL of the tracker", func() {
		a.Example("#ffa7cb")
	})
	a.Attribute("Type", d.String, "Type of the tracker", func() {
		a.Enum("github", "jira")
	})
	a.Required("URL", "Type")
})

var trackerRelationships = a.Type("TrackerRelations", func() {
})

var trackerList = JSONList(
	"Tracker", "Holds the list of Trackers",
	tracker,
	pagingLinks,
	meta)

var trackerSingle = JSONSingle(
	"Tracker", "Holds a single Tracker",
	tracker,
	nil)

var _ = a.Resource("tracker", func() {
	a.BasePath("/trackers")

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List all tracker configurations.")
		a.Params(func() {
			a.Param("filter", d.String, "a query language expression restricting the set of found items")
			a.Param("page", d.String, "Paging in the format <start>,<limit>")
		})
		a.Response(d.OK, trackerList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.TemporaryRedirect)
	})

	a.Action("show", func() {
		a.Routing(
			a.GET("/:id"),
		)
		a.Description("Retrieve tracker configuration for the given id.")
		a.Params(func() {
			a.Param("id", d.UUID, "id")
		})
		a.Response(d.OK, trackerSingle)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("Add new tracker configuration.")
		a.Payload(trackerSingle)
		a.Response(d.Created, "/trackers/.*", func() {
			a.Media(trackerSingle)
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
			a.DELETE("/:id"),
		)
		a.Description("Delete tracker configuration.")
		a.Params(func() {
			a.Param("id", d.UUID, "id")
		})
		a.Response(d.NoContent)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
	})
	a.Action("update", func() {
		a.Security("jwt")
		a.Routing(
			a.PUT("/:id"),
		)
		a.Description("Update tracker configuration.")
		a.Payload(trackerSingle)
		a.Response(d.OK, func() {
			a.Media(trackerSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
	})

})
