package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var trackerData = a.Type("TrackerData", func() {
	a.Attribute("attributes", trackerAttributes)
	a.Attribute("type", d.String, func() {
		a.Enum("trackers")
	})
	a.Attribute("id", d.String, "ID of the tracker record", func() {
		a.Example("42")
	})
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

var trackerLinks = a.Type("TrackerLinks", func() {
	a.Attribute("self", d.String, func() {
		a.Example("http://api.almighty.io/api/trackers/42")
	})
	a.Required("self")
})

// Defines a Media Type for Single Tracker Object
var TrackerObject = JSONSingle("TrackerObject", "Single Tracker Payload", trackerData, trackerLinks)

var trackerAttributes = a.Type("TrackerAttributes", func() {
	a.Attribute("url", d.String, "URL of the tracker", func() {
		a.Example("https://api.github.com/")
		a.MinLength(1)
	})
	a.Attribute("type", d.String, "Type of the tracker", func() {
		a.Example("github")
		a.Pattern("^[\\p{L}]+$")
		a.MinLength(1)
	})
	a.Required("url", "type")
})

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
		a.Response(d.OK, func() {
			a.Media(a.CollectionOf(Tracker))
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("show", func() {
		a.Routing(
			a.GET("/:id"),
		)
		a.Description("Retrieve tracker configuration for the given id.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Response(d.OK, func() {
			a.Media(Tracker)
		})
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
		a.Payload(TrackerObject)
		a.Response(d.Created, "/trackers/.*", func() {
			a.Media(TrackerObject)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
	a.Action("delete", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/:id"),
		)
		a.Description("Delete tracker configuration.")
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
			a.PUT("/:id"),
		)
		a.Description("Update tracker configuration.")
		a.Payload(UpdateTrackerAlternatePayload)
		a.Response(d.OK, func() {
			a.Media(Tracker)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})

})
