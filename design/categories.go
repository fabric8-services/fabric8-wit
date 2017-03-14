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
	a.Attribute("relationships", categoryRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

var categoryAttributes = a.Type("categoryAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a filter. +See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("name", d.String, "The Category name", func() {
		a.Example("Requirements")
	})
	a.Required("name")
})

var categoryRelationships = a.Type("CategoryRelations", func() {
	a.Attribute("space", relationGeneric, "This defines the owning space")
	a.Attribute("workitemtypes", relationGeneric, "This defines the workitemtypes associated with the category")
})
