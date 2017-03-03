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
	a.Attribute("type", d.UUID, "ID of the type of this work item")
	a.Attribute("fields", a.HashOf(d.String, d.Any), "The field values, according to the field type")
	a.Attribute("relationships", workItemRelationships)

	a.Required("id")
	a.Required("version")
	a.Required("type")
	a.Required("fields")
	a.Required("relationships")

	a.View("default", func() {
		a.Attribute("id")
		a.Attribute("version")
		a.Attribute("type")
		a.Attribute("fields")
		a.Attribute("relationships")
	})
})

var pagingLinks = a.Type("pagingLinks", func() {
	a.Attribute("prev", d.String)
	a.Attribute("next", d.String)
	a.Attribute("first", d.String)
	a.Attribute("last", d.String)
	a.Attribute("filters", d.String)
})

var meta = a.Type("workItemListResponseMeta", func() {
	a.Attribute("totalCount", d.Integer)

	a.Required("totalCount")
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
