package design

import (
	. "github.com/goadesign/goa/design"
	. "github.com/goadesign/goa/design/apidsl"
)

// ALMVersion defines therunning ALM Version MediaType
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

var WorkItem = MediaType("application/vnd.workitem+json", func() {
	TypeName("WorkItem")
	Description("ALM Work Item")
	Attribute("id", String, "unique id per installation")
	Attribute("version", Integer, "Version for optimistic concurrency control")
	Attribute("name", String, "User Readable Name of this item")
	Attribute("type", String, "Id of the type of this work item")
	Attribute("fields", HashOf(String, Any))

	Required("id")
	Required("version")
	Required("name")
	Required("type")
	Required("fields")

	View("default", func() {
		Attribute("id")
		Attribute("version")
		Attribute("name")
		Attribute("type")
		Attribute("fields")
	})
})

var FieldDefinition = Type("fieldDefinition", func() {
	Attribute("required", Boolean)
	Attribute("type", Any)

	Required("required")
	Required("type")

	View("default", func() {
		Attribute("kind")
	})

})

var WorkItemType = MediaType("application/vnd.workitemtype+json", func() {
	TypeName("WorkItemType")
	Description("ALM Work Item Type")
	Attribute("id", String, "unique id per installation")
	Attribute("version", Integer, "Version for optimistic concurrency control")
	Attribute("name", String, "User Readable Name of this item")
	Attribute("fields", HashOf(String, FieldDefinition), "Definitions of fields in this work item")

	Required("id")
	Required("version")
	Required("name")
	Required("fields")

	View("default", func() {
		Attribute("id")
		Attribute("version")
		Attribute("name")
		Attribute("fields")
	})
	View("link", func() {
		Attribute("id")
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
	Attribute("version", Integer, "Version for optimistic concurrency control")
	Attribute("query", String, "Search query")
	Attribute("schedule", String, "Schedule for fetch and import")
	Attribute("tracker", Integer, "Tracker ID")

	Required("id")
	Required("version")
	Required("query")
	Required("schedule")
	Required("tracker")

	View("default", func() {
		Attribute("id")
		Attribute("version")
		Attribute("query")
		Attribute("schedule")
		Attribute("tracker")
	})
})
