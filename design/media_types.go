package design

import (
	. "github.com/goadesign/goa/design"
	. "github.com/goadesign/goa/design/apidsl"
)

// ALMVersion defines the running ALM Version MediaType
var ALMVersion = MediaType("application/vnd.version+json", func() {
	Description("The current running version")
	Attributes(func() {
		Attribute("commit", String, "Commit SHA this build is based on")
		Attribute("build_time", String, "The date when build")
		Required("commit", "build_time")
	})
	View("default", func() {
		Attribute("commit")
		Attribute("build_time")
	})
})

// AuthToken represents an authentication JWT Token
var AuthToken = MediaType("application/vnd.authtoken+json", func() {
	TypeName("AuthToken")
	Description("JWT Token")
	Attributes(func() {
		Attribute("token", String, "JWT Token")
		Required("token")
	})
	View("default", func() {
		Attribute("token")
	})
})

// workItem is the media type for work items
var workItem = MediaType("application/vnd.workitem+json", func() {
	TypeName("WorkItem")
	Description("A work item hold field values according to a given field type")
	Attribute("id", String, "unique id per installation")
	Attribute("version", Integer, "Version for optimistic concurrency control")
	Attribute("type", String, "Name of the type of this work item")
	Attribute("fields", HashOf(String, Any), "The field values, according to the field type")

	Required("id")
	Required("version")
	Required("type")
	Required("fields")

	View("default", func() {
		Attribute("id")
		Attribute("version")
		Attribute("type")
		Attribute("fields")
	})
})

// fieldDefinition defines the possible values for a field in a work item type
var fieldDefinition = Type("fieldDefinition", func() {
	Description("A fieldDescription aggregates a fieldType and additional field metadata")
	Attribute("required", Boolean)
	Attribute("type", fieldType)

	Required("required")
	Required("type")

	View("default", func() {
		Attribute("kind")
	})
})

// fieldType is the datatype of a single field in a work item tepy
var fieldType = Type("fieldType", func() {
	Description("A fieldType describes the values a particular field can hold")
	Attribute("kind", String, "The constant indicating the kind of type, for example 'string' or 'enum' or 'instant'")
	Attribute("componentType", String, "The kind of type of the individual elements for a list type. Required for list types. Must be a simple type, not  enum or list")
	Attribute("baseType", String, "The kind of type of the enumeration values for an enum type. Required for enum types. Must be a simple type, not  enum or list")
	Attribute("values", ArrayOf(Any), "The possible values for an enum type. The values must be of a type convertible to the base type")

	Required("kind")
})

// workItemType is the media type representing a work item type.
var workItemType = MediaType("application/vnd.workitemtype+json", func() {
	TypeName("WorkItemType")
	Description("A work item type describes the values a work item type instance can hold.")
	Attribute("version", Integer, "Version for optimistic concurrency control")
	Attribute("name", String, "User Readable Name of this item type")
	Attribute("fields", HashOf(String, fieldDefinition), "Definitions of fields in this work item type")

	Required("version")
	Required("name")
	Required("fields")

	View("default", func() {
		Attribute("version")
		Attribute("name")
		Attribute("fields")
	})
	View("link", func() {
		Attribute("name")
	})

})
