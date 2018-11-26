package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var event = a.Type("Event", func() {
	a.Description(`JSONAPI store for the data of a event.  See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("events")
	})
	a.Attribute("id", d.UUID, "ID of event. NOTE: this is not the ID of the work item revision but a random ID that is unique for this event.", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", eventAttributes)
	a.Attribute("relationships", eventRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type", "relationships", "attributes", "id")
})

var eventAttributes = a.Type("EventAttributes", func() {
	a.Attribute("revisionId", d.UUID, "ID of the revision", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Description(`JSONAPI store for all the "attributes" of a event. +See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("timestamp", d.DateTime, "When the event occurred", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("name", d.String, "[DEPRECATED] The name of the event occured", func() {
		a.Example("system.title")
	})
	a.Attribute("onField", d.String, "The field on which the event occurred", func() {
		a.Example("system_title")
	})

	a.Attribute("oldValue", d.Any, "The user who was assigned to (or unassigned from). Only for 'assigned' and 'unassigned' events.", func() {
		a.Example("813a456e-1c8a-48df-ac15-84065ee039f7")
	})
	a.Attribute("newValue", d.Any, "The user who performed the assignment (or unassignment). Only for 'assigned' and 'unassigned' events..", func() {
		a.Example("813a456e-1c8a-48df-ac15-84065ee039f7")
	})
	a.Required("timestamp", "name", "onField", "revisionId")
})

var eventRelationships = a.Type("EventRelations", func() {
	a.Attribute("modifier", relationGeneric, "This defines the modifier of the event")
	a.Attribute("oldValue", relationGenericList)
	a.Attribute("newValue", relationGenericList)
	a.Attribute("workItemType", relationGeneric, "The type of the work item at the event's point in time")

	a.Required("workItemType", "modifier")
})

var eventList = JSONList(
	"Event", "Holds the response of events",
	event,
	nil,
	nil,
)

var eventSingle = JSONSingle(
	"Event", "Holds a single Event",
	event,
	nil)

var _ = a.Resource("work_item_events", func() {
	a.Parent("workitem")

	a.Action("list", func() {
		a.Routing(
			a.GET("events"),
		)
		a.Params(func() {
			a.Param("revisionID", d.UUID, "an optional revision ID to filter events by")
		})
		a.Description("List events associated with the given work item")
		a.UseTrait("conditional") // Refer: goasupport/conditional_request/generator.go
		a.Response(d.OK, eventList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})
