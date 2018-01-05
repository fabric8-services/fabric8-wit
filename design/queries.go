package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var query = a.Type("Query", func() {
	a.Description(`JSONAPI store for the data of a Query. See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("queries")
	})
	a.Attribute("id", d.UUID, "ID of query", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", queryAttributes)
	a.Attribute("links", genericLinks)
	a.Attribute("relationships", queryRelationships)
	a.Required("type", "attributes")
})

var queryRelationships = a.Type("QueryRelations", func() {
	a.Attribute("creator", relationGeneric, "This defines the creator of the query")
	a.Attribute("space", relationGeneric, "This defines the space in which query is saved")
})

var queryAttributes = a.Type("QueryAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a Query. See also http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("title", d.String, mandatoryOnCreate("The query title"), nameValidationFunction)
	a.Attribute("created-at", d.DateTime, "When the query was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("fields", d.String, mandatoryOnCreate("Query fields"), func() {
		a.Example(`"{ \"$AND\":[ { \"space\":\"a2d6ab7a-5d35-47b5-8fff-d4ce6285a158\" }, { \"assignee\":\"7ef78c14-f314-4a5a-8512-21640e3d2ef8\" } ] }"`)
	})
	a.Required("title", "fields")
})

var queryList = JSONList(
	"Query", "Holds the list of queries",
	query,
	pagingLinks,
	meta,
)

var querySingle = JSONSingle(
	"Query", "Holds a single query",
	query,
	nil,
)

var _ = a.Resource("query", func() {
	a.Parent("space")
	a.BasePath("/queries")

	a.Action("show", func() {
		a.Security("jwt")
		a.Routing(
			a.GET("/:queryID"),
		)
		a.Description("Retrieve query for the given id.")
		a.Params(func() {
			a.Param("queryID", d.UUID, "ID of the query")
		})
		a.UseTrait("conditional")
		a.Response(d.OK, querySingle)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
	})

	a.Action("list", func() {
		a.Security("jwt")
		a.Routing(
			a.GET(""),
		)
		a.Description("List queries.")
		a.UseTrait("conditional")
		a.Response(d.OK, queryList)
		a.Response(d.NotModified)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("create query with id, title and fields.")
		a.Payload(querySingle)
		a.Response(d.Created, "/queries/.*", func() {
			a.Media(querySingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Conflict, JSONAPIErrors)
	})

	a.Action("delete", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/:queryID"),
		)
		a.Description("Delete a query with the given ID.")
		a.Params(func() {
			a.Param("queryID", d.UUID, "ID of the query to delete")
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
		a.Response(d.NoContent)
	})
})
