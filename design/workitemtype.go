package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// fieldType is the datatype of a single field in a work item type
var fieldType = a.Type("fieldType", func() {
	a.Description("A fieldType describes the values a particular field can hold")
	a.Attribute("kind", d.String, "The constant indicating the kind of type, for example 'string' or 'enum' or 'instant'")
	a.Attribute("componentType", d.String, "The kind of type of the individual elements for a list type. Required for list types. Must be a simple type, not  enum or list")
	a.Attribute("baseType", d.String, "The kind of type of the enumeration values for an enum type. Required for enum types. Must be a simple type, not  enum or list")
	a.Attribute("values", a.ArrayOf(d.Any), "The possible values for an enum type. The values must be of a type convertible to the base type")

	a.Required("kind")
})

// fieldDefinition defines the possible values for a field in a work item type
var fieldDefinition = a.Type("fieldDefinition", func() {
	a.Description("A fieldDescription aggregates a fieldType and additional field metadata")
	a.Attribute("required", d.Boolean)
	a.Attribute("type", fieldType)

	a.Required("required")
	a.Required("type")

	a.View("default", func() {
		a.Attribute("kind")
	})
})

// workItemType is the media type representing a work item type.
var workItemType = a.MediaType("application/vnd.workitemtype+json", func() {
	a.TypeName("WorkItemType")
	a.Description("A work item type describes the values a work item type instance can hold.")
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control")
	a.Attribute("name", d.String, "User Readable Name of this item type")
	a.Attribute("fields", a.HashOf(d.String, fieldDefinition), "Definitions of fields in this work item type")

	a.Required("version")
	a.Required("name")
	a.Required("fields")

	a.View("default", func() {
		a.Attribute("version")
		a.Attribute("name")
		a.Attribute("fields")
	})
	a.View("link", func() {
		a.Attribute("name")
	})

})

var _ = a.Resource("workitemtype", func() {

	a.BasePath("/workitemtypes")

	a.Action("show", func() {

		a.Routing(
			a.GET("/:name"),
		)
		a.Description("Retrieve work item type with given name.")
		a.Params(func() {
			a.Param("name", d.String, "name")
		})
		a.Response(d.OK, func() {
			a.Media(workItemType)
		})
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("Create work item type.")
		a.Payload(CreateWorkItemTypePayload)
		a.Response(d.Created, "/workitemtypes/.*", func() {
			a.Media(workItemType)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List work item types.")
		a.Params(func() {
			a.Param("page", d.String, "Paging in the format <start>,<limit>")
		})
		a.Response(d.OK, func() {
			a.Media(a.CollectionOf(workItemType))
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("list-source-link-types", func() {
		a.Routing(
			a.GET("/:name/source-link-types"),
		)
		a.Params(func() {
			a.Param("name", d.String, "name")
		})
		a.Description(`Retrieve work item link types where the
given work item type can be used in the source of the link.`)
		a.Response(d.OK, func() {
			a.Media(workItemLinkTypeList)
		})
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("list-target-link-types", func() {
		a.Routing(
			a.GET("/:name/target-link-types"),
		)
		a.Params(func() {
			a.Param("name", d.String, "name")
		})
		a.Description(`Retrieve work item link types where the
given work item type can be used in the target of the link.`)
		a.Response(d.OK, func() {
			a.Media(workItemLinkTypeList)
		})
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})
