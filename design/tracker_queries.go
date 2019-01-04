package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var trackerquery = a.Type("TrackerQuery", func() {
	a.Description(`JSONAPI store for the data of a Tracker query. See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("trackerquery")
	})
	a.Attribute("id", d.UUID, "ID of tracker query", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", trackerQueryAttributes)
	a.Attribute("relationships", trackerQueryRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

var trackerQueryAttributes = a.Type("TrackerQueryAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a Tracker Query. See also http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("created-at", d.DateTime, "When the tracker query was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("updated-at", d.DateTime, "When the tracker query was updated", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("query", d.String, "search query", func() {
		a.Example("is:open is:issue")
	})
	a.Attribute("schedule", d.String, "Schedule to fetch and import. Expression Format -> [Seconds] [Minutes] [Hours] [Day of month] [Month] [Day of week]. See also -> https://godoc.org/github.com/robfig/cron", func() {
		a.Example("0 0/15 * * * *")
	})
	a.Required("query", "schedule")
})

var trackerQueryRelationships = a.Type("TrackerQueryRelations", func() {
	a.Attribute("tracker", relationKindUUID, "This defines the related tracker")
	a.Attribute("space", relationSpaces, "This defines the owning space")
	a.Attribute("workItemType", relationBaseType, "Defines what work item type to use when instantiating work items using this tracker query.")
})

var trackerQueryList = JSONList(
	"TrackerQuery", "Holds the list of Tracker Queries",
	trackerquery,
	pagingLinks,
	meta)

var trackerQuerySingle = JSONSingle(
	"TrackerQuery", "Holds a single Tracker query",
	trackerquery,
	nil)

var _ = a.Resource("trackerquery", func() {
	a.BasePath("/trackerqueries")

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List all tracker queries.")
		a.Params(func() {
			a.Param("filter", d.String, "a query language expression restricting the set of found items")
			a.Param("page", d.String, "Paging in the format <start>,<limit>")
		})
		a.Response(d.OK, trackerQueryList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.TemporaryRedirect)
	})

	a.Action("show", func() {
		a.Routing(
			a.GET("/:id"),
		)
		a.Description("Retrieve tracker queries for the given id.")
		a.Params(func() {
			a.Param("id", d.UUID, "id")
		})
		a.Response(d.OK, trackerQuerySingle)
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
		a.Description("Add new tracker query configuration.")
		a.Payload(trackerQuerySingle)
		a.Response(d.Created, "/trackerqueries/.*", func() {
			a.Media(trackerQuerySingle)
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
		a.Description("Delete tracker query.")
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
})
