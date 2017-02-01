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

	//	a.View("default", func() {
	//		a.Attribute("kind")
	//	})
})

var workItemTypeAttributes = a.Type("WorkItemTypeAttributes", func() {
	a.Description("A work item type describes the values a work item type instance can hold.")
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control")
	a.Attribute("fields", a.HashOf(d.String, fieldDefinition), "Definitions of fields in this work item type")

	a.Required("version")
	a.Required("fields")
})

var workItemTypeData = a.Type("WorkItemTypeData", func() {
	a.Attribute("type", d.String, func() {
		a.Enum("workitemtypes")
	})
	a.Attribute("id", d.String, "The name of the work item type (not optional)", func() {
		a.Example("bug")
	})
	a.Attribute("attributes", workItemTypeAttributes)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
	// NOTICE: for now the id attribute is not optional because it is just the
	// WIT's name. This might change in the future but not at this point.
	a.Required("id")
})

// workItemTypeLinks has `self` as of now according to http://jsonapi.org/format/#fetching-resources
var workItemTypeLinks = a.Type("WorkItemTypeLinks", func() {
	a.Attribute("self", d.String, func() {
		a.Example("http://api.almighty.io/api/workitemtypes/bug")
	})
	a.Required("self")
})

var workItemTypeListMeta = a.Type("WorkItemTypeListMeta", func() {
	a.Attribute("totalCount", d.Integer)
	a.Required("totalCount")
})

// workItemTypeList contains paged results for listing work item types and paging links
var workItemTypeList = JSONList(
	"WorkItemType", "Holds the paginated response to a work item type list request",
	workItemTypeData,
	pagingLinks,
	workItemTypeListMeta)

// workItemTypeSingle is the media type for work item types
var workItemTypeSingle = JSONSingle(
	"WorkItemType", "A work item type describes the values a work item type instance can hold.",
	workItemTypeData,
	workItemTypeLinks)

var _ = a.Resource("workitemtype", func() {
	a.BasePath("/workitemtypes")
	a.Action("show", func() {
		a.Routing(
			a.GET("/:id"),
		)
		a.Description("Retrieve work item type with given ID.")
		a.Params(func() {
			a.Param("id", d.String, "The name of the work item type")
		})
		a.Response(d.OK, func() {
			a.Media(workItemTypeSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("Create work item type.")
		a.Payload(workItemTypeSingle)
		a.Response(d.Created, "/workitemtypes/.*", func() {
			a.Media(workItemTypeSingle)
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
			// TODO: Support same params as in work item list-action?
		})
		a.Response(d.OK, func() {
			a.Media(workItemTypeList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("Create work item type.")
		a.Payload(workItemTypeSingle)
		a.Response(d.Created, "/workitemtypes/.*", func() {
			a.Media(workItemTypeSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
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
