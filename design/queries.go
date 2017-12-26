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
})

var queryAttributes = a.Type("QueryAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a Query. See also http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("title", d.String, "The title of saved query", nameValidationFunction)
	a.Attribute("created-at", d.DateTime, "When the query was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("updated-at", d.DateTime, "When the query was updated", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("fields", d.String, "Actual query string created by user", func() {
		a.Example("...")
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
		a.Routing(
			a.GET("/:queryID"),
		)
		a.Description("Retrieve query for the given id.")
		a.Params(func() {
			a.Param("queryID", d.UUID, "ID of the query")
		})
		// a.UseTrait("conditional")
		a.Response(d.OK, querySingle)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("list", func() {
		a.Security("jwt")
		a.Routing(
			a.GET(""),
		)
		a.Description("List queries.")
		// a.UseTrait("conditional")
		a.Response(d.OK, queryList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
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
		a.Response(d.Conflict, JSONAPIErrors)
	})
})
