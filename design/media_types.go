package design

import (
	. "github.com/goadesign/goa/design"
	. "github.com/goadesign/goa/design/apidsl"
)

// ALMStatus defines the status of the current running ALM instance
var ALMStatus = MediaType("application/vnd.status+json", func() {
	Description("The status of the current running instance")
	Attributes(func() {
		Attribute("commit", String, "Commit SHA this build is based on")
		Attribute("buildTime", String, "The time when built")
		Attribute("startTime", String, "The time when started")
		Attribute("error", String, "The error if any")
		Required("commit", "buildTime", "startTime")
	})
	View("default", func() {
		Attribute("commit")
		Attribute("buildTime")
		Attribute("startTime")
		Attribute("error")
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

// Tracker configuration
var Tracker = MediaType("application/vnd.tracker+json", func() {
	TypeName("Tracker")
	Description("Tracker configuration")
	Attribute("id", String, "unique id per tracker")
	Attribute("url", String, "URL of the tracker")
	Attribute("type", String, "Type of the tracker")

	Required("id")
	Required("url")
	Required("type")

	View("default", func() {
		Attribute("id")
		Attribute("url")
		Attribute("type")
	})
})

// TrackerQuery represents the search query with schedule
var TrackerQuery = MediaType("application/vnd.trackerquery+json", func() {
	TypeName("TrackerQuery")
	Description("Tracker query with schedule")
	Attribute("id", String, "unique id per installation")
	Attribute("query", String, "Search query")
	Attribute("schedule", String, "Schedule for fetch and import")
	Attribute("trackerID", String, "Tracker ID")

	Required("id")
	Required("query")
	Required("schedule")
	Required("trackerID")

	View("default", func() {
		Attribute("id")
		Attribute("query")
		Attribute("schedule")
		Attribute("trackerID")
	})
})

var User = MediaType("application/vnd.user+json", func() {
	TypeName("User")
	Description("ALM User")
	Attribute("fullName", String, "The users full name")
	Attribute("imageURL", String, "The avatar image for the user")

	View("default", func() {
		Attribute("fullName")
		Attribute("imageURL")
	})
})
