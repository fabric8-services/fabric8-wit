package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// ALMStatus defines the status of the current running ALM instance
var ALMStatus = a.MediaType("application/vnd.status+json", func() {
	a.Description("The status of the current running instance")
	a.Attributes(func() {
		a.Attribute("commit", d.String, "Commit SHA this build is based on")
		a.Attribute("buildTime", d.String, "The time when built")
		a.Attribute("startTime", d.String, "The time when started")
		a.Attribute("error", d.String, "The error if any")
		a.Required("commit", "buildTime", "startTime")
	})
	a.View("default", func() {
		a.Attribute("commit")
		a.Attribute("buildTime")
		a.Attribute("startTime")
		a.Attribute("error")
	})
})

// workItem is the media type for work items
// Deprecated, but kept around as internal model for now.
var workItem = a.MediaType("application/vnd.workitem+json", func() {
	a.TypeName("WorkItem")
	a.Description("A work item hold field values according to a given field type")
	a.Attribute("id", d.String, "unique id per installation")
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control")
	a.Attribute("type", d.String, "Name of the type of this work item")
	a.Attribute("fields", a.HashOf(d.String, d.Any), "The field values, according to the field type")

	a.Required("id")
	a.Required("version")
	a.Required("type")
	a.Required("fields")

	a.View("default", func() {
		a.Attribute("id")
		a.Attribute("version")
		a.Attribute("type")
		a.Attribute("fields")
	})
})

var pagingLinks = a.Type("pagingLinks", func() {
	a.Attribute("prev", d.String)
	a.Attribute("next", d.String)
	a.Attribute("first", d.String)
	a.Attribute("last", d.String)
})

var meta = a.Type("workItemListResponseMeta", func() {
	a.Attribute("totalCount", d.Integer)

	a.Required("totalCount")
})

// position represents the ID of the workitem above which the to-be-reordered workitem(s) should be placed
var position = a.Type("workItemReorderPosition", func() {
	a.Description("Position represents the ID of the workitem above which the to-be-reordered workitem(s) should be placed")
	a.Attribute("id", d.String, "ID of the workitem above which the to-be-reordered workitem(s) should be placed")
	a.Attribute("direction", d.String, "position of workitem")

	a.Required("id", "direction")
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

// fieldType is the datatype of a single field in a work item type
var fieldType = a.Type("fieldType", func() {
	a.Description("A fieldType describes the values a particular field can hold")
	a.Attribute("kind", d.String, "The constant indicating the kind of type, for example 'string' or 'enum' or 'instant'")
	a.Attribute("componentType", d.String, "The kind of type of the individual elements for a list type. Required for list types. Must be a simple type, not  enum or list")
	a.Attribute("baseType", d.String, "The kind of type of the enumeration values for an enum type. Required for enum types. Must be a simple type, not  enum or list")
	a.Attribute("values", a.ArrayOf(d.Any), "The possible values for an enum type. The values must be of a type convertible to the base type")

	a.Required("kind")
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

// Tracker configuration
var Tracker = a.MediaType("application/vnd.tracker+json", func() {
	a.TypeName("Tracker")
	a.Description("Tracker configuration")
	a.Attribute("id", d.String, "unique id per tracker")
	a.Attribute("url", d.String, "URL of the tracker")
	a.Attribute("type", d.String, "Type of the tracker")

	a.Required("id")
	a.Required("url")
	a.Required("type")

	a.View("default", func() {
		a.Attribute("id")
		a.Attribute("url")
		a.Attribute("type")
	})
})

// TrackerQuery represents the search query with schedule
var TrackerQuery = a.MediaType("application/vnd.trackerquery+json", func() {
	a.TypeName("TrackerQuery")
	a.Description("Tracker query with schedule")
	a.Attribute("id", d.String, "unique id per installation")
	a.Attribute("query", d.String, "Search query")
	a.Attribute("schedule", d.String, "Schedule for fetch and import")
	a.Attribute("trackerID", d.String, "Tracker ID")

	a.Required("id")
	a.Required("query")
	a.Required("schedule")
	a.Required("trackerID")

	a.View("default", func() {
		a.Attribute("id")
		a.Attribute("query")
		a.Attribute("schedule")
		a.Attribute("trackerID")
	})
})
