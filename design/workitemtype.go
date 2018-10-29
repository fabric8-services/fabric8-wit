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
	a.Attribute("defaultValue", d.Any, "Optional default value (if any)")
	a.Required("kind")
})

// fieldDefinition defines the possible values for a field in a work item type
var fieldDefinition = a.Type("fieldDefinition", func() {
	a.Description("A fieldDefinition aggregates a fieldType and additional field metadata")
	a.Attribute("required", d.Boolean)
	a.Attribute("type", fieldType)
	a.Attribute("label", d.String, "A label for the field that is shown in the UI", func() {
		a.Example("Iteration")
		a.MinLength(1)
	})
	a.Attribute("description", d.String, "A description for the field", func() {
		a.Example("The iteration field tells to which iteration a work item belongs.")
		a.MinLength(1)
	})
	a.Required("required", "type", "label", "description")
})

var workItemTypeAttributes = a.Type("WorkItemTypeAttributes", func() {
	a.Description("A work item type describes the values a work item type instance can hold.")
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control")
	a.Attribute("created-at", d.DateTime, "timestamp of entity creation")
	a.Attribute("updated-at", d.DateTime, "timestamp of last entity update")
	a.Attribute("name", d.String, "The human readable name of the work item type", nameValidationFunction)
	a.Attribute("description", d.String, "A human readable description for the work item type", func() {
		a.Example(`A user story encapsulates the action of one function making it possible for software developers to create a vertical slice of their work.`)
	})
	a.Attribute("can-construct", d.Boolean, "Whether or not this work item type is supposed to be used for creating work items directly.")
	a.Attribute("fields", a.HashOf(d.String, fieldDefinition), "Definitions of fields in this work item type", func() {
		a.Example(map[string]interface{}{
			"system_administrator": map[string]interface{}{
				"type": map[string]interface{}{
					"kind": "string",
				},
				"required": true,
			},
		})
		a.MinLength(1)
	})

	// TODO: Maybe this needs to be abandoned at some point
	a.Attribute("extendedTypeName", d.UUID, "If newly created type extends any existing type (This is never present in any response and is only optional when creating.)")

	a.Attribute("icon", d.String, "CSS class string for an icon to use. See http://fontawesome.io/icons/ or http://www.patternfly.org/styles/icons/#_ for examples.", func() {
		a.Example("fa-bug")
		a.MinLength(1)
	})

	//a.Required("version")
	a.Required("fields")
	a.Required("name")
	a.Required("icon")
})

var workItemTypeRelationships = a.Type("WorkItemTypeRelationships", func() {
	a.Attribute("space", relationSpaces, "(OBSOLETE) This defines the owning space of this work item type.")
	a.Attribute("space_template", spaceTemplateRelation, "This defines the owning space template of this work item type.")
	a.Attribute("guidedChildTypes", relationGenericList, "List of work item types that shall be proposed when creating a child of a work item of this type.")
})

var workItemTypeData = a.Type("WorkItemTypeData", func() {
	a.Attribute("type", d.String, func() {
		a.Enum("workitemtypes")
	})
	a.Attribute("id", d.UUID, "ID of work item type (optional during creation)")
	a.Attribute("attributes", workItemTypeAttributes)
	a.Attribute("links", genericLinks)
	a.Attribute("relationships", workItemTypeRelationships)
	a.Required("type", "attributes", "relationships")
})

// workItemTypeLinks has `self` as of now according to http://jsonapi.org/format/#fetching-resources
var workItemTypeLinks = a.Type("WorkItemTypeLinks", func() {
	a.Attribute("self", d.String, func() {
		a.Example("http://api.openshift.io/api/workitemtypes/bug")
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

// workItemTypeSingle is the media type for for a single work item type
var workItemTypeSingle = JSONSingle(
	"WorkItemType", "A work item type describes the values a work item type instance can hold.",
	workItemTypeData,
	workItemTypeLinks)

var _ = a.Resource("workitemtype", func() {
	a.BasePath("/workitemtypes")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:witID"),
		)
		a.Description("Retrieve work item type with given ID.")
		a.Params(func() {
			a.Param("witID", d.UUID, "ID of the work item type")
		})
		a.UseTrait("conditional")
		a.Response(d.OK, workItemTypeSingle)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})

var _ = a.Resource("workitemtypes", func() {
	a.Parent("space_template")
	a.BasePath("/workitemtypes")

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List work item types.")
		a.UseTrait("conditional")
		a.Response(d.OK, workItemTypeList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})
