package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var category = a.Type("categories", func() {
	a.Description(`JSONAPI store for the data of a filter. See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("categories")
	})
	a.Attribute("id", d.UUID, "ID of category", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", categoryAttributes)
	a.Required("type", "attributes")
})

var categoryAttributes = a.Type("categoryAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a filter. +See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("name", d.String, "The Category name", func() {
		a.Example("Requirements")
	})
	a.Required("name")
})

var categoryList = JSONList(
	"category", "Holds the list of categories",
	category,
	pagingLinks,
	meta)

var _ = a.Resource("category", func() {
	a.BasePath("/categories")
	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List categories")
		a.Response(d.OK, func() {
			a.Media(categoryList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})
