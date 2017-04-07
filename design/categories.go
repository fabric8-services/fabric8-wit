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
		a.MinLength(1)
	})
	a.Required("name")
})

var categoryList = JSONList(
	"category", "Holds the list of Categories",
	category,
	pagingLinks,
	meta)

// relationCategories is the JSONAPI store for the categories
var relationCategories = a.Type("RelationCategories", func() {
	a.Attribute("data", relationCategoriesData)
	a.Attribute("links", genericLinks)
	a.Attribute("meta", a.HashOf(d.String, d.Any))
})

// relationCategoriesData is the JSONAPI data object of the category relationship objects
var relationCategoriesData = a.Type("RelationCategoriesData", func() {
	a.Attribute("type", d.String, func() {
		a.Enum("category")
	})
	a.Attribute("id", d.UUID, "UUID for the category", func() {
		a.Example("6c5610be-30b2-4880-9fec-81e4f8e4fd76")
	})
	a.Attribute("links", genericLinks)
})
