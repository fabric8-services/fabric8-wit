package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var area = a.Type("Area", func() {
	a.Description(`JSONAPI store for the data of a Area.  See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("areas")
	})
	a.Attribute("id", d.UUID, "ID of area", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", areaAttributes)
	a.Attribute("relationships", areaRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

var areaAttributes = a.Type("AreaAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a Area. +See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("name", d.String, "The Area name", func() {
		a.Example("Area for Build related stuff")
	})
})

var areaRelationships = a.Type("AreaRelations", func() {
	a.Attribute("space", relationGeneric, "This defines the owning space")
	a.Attribute("parent", relationGeneric, "This defines the parents' hierarchy for areas")
	a.Attribute("workitems", relationGeneric, "This defines the workitems associated with the Area")
})

var areaList = JSONList(
	"area", "Holds the list of Areas",
	area,
	pagingLinks,
	meta)

var areaSingle = JSONSingle(
	"area", "Holds the list of Areas",
	area,
	nil)

// new version of "list" for migration
var _ = a.Resource("area", func() {
	a.BasePath("/areas")
	a.Action("show", func() {
		a.Routing(
			a.GET("/:id"),
		)
		a.Description("Retrieve area with given id.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Response(d.OK, func() {
			a.Media(areaSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
	a.Action("create-child", func() {
		//a.Security("jwt")
		a.Routing(
			a.POST("/:id"),
		)
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Description("create child area.")
		a.Payload(areaSingle)
		a.Response(d.Created, "/areas/.*", func() {
			a.Media(areaSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})

// new version of "list" for migration
var _ = a.Resource("space-areas", func() {
	a.Parent("space")

	a.Action("list", func() {
		a.Routing(
			a.GET("Areas"),
		)
		a.Description("List Areas.")
		a.Response(d.OK, func() {
			a.Media(areaList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
	a.Action("create", func() {
		//a.Security("jwt")
		a.Routing(
			a.POST("Areas"),
		)
		a.Description("Create Area.")
		a.Payload(areaSingle)
		a.Response(d.Created, "/Areas/.*", func() {
			a.Media(areaSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})
