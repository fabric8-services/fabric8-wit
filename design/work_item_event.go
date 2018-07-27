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
	a.Attribute("id", d.UUID, "ID of event", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", a.HashOf(d.String, d.Any), func() {
		a.Example(map[string]interface{}{"version": "1", "system.state": "new", "system.title": "Example story"})
	})
	a.Attribute("relationships", eventRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type")
})

var eventAttributes = a.Type("EventAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a event. +See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("timestamp", d.DateTime, "When the event occurred", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("name", d.String, "The name of the event occured", func() {
		a.Example("closed")
	})
	a.Attribute("oldValue", d.String, "The user who was assigned to (or unassigned from). Only for 'assigned' and 'unassigned' events.", func() {
		a.Example("813a456e-1c8a-48df-ac15-84065ee039f7")
	})
	a.Attribute("newValue", d.String, "The user who performed the assignment (or unassignment). Only for 'assigned' and 'unassigned' events..", func() {
		a.Example("813a456e-1c8a-48df-ac15-84065ee039f7")
	})
	a.Required("timestamp", "name")
})

var eventRelationships = a.Type("EventRelations", func() {
	a.Attribute("modifier", relationGeneric, "This defines the modifier of the event")
	a.Attribute("oldValue", relationGenericList)
	a.Attribute("newValue", relationGenericList)
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
		a.Description("List events associated with the given work item")
		a.UseTrait("conditional") // Refer: goasupport/conditional_request/generator.go
		a.Response(d.OK, eventList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})

// user can list an event from the wit service
// if the event id is given
var _ = a.Resource("event", func() {
	a.BasePath("/events")
	a.Action("show", func() {
		a.Routing(
			a.GET("/:eventID"),
		)
		a.Description("Retrieve event with given id")
		a.Params(func() {
			a.Param("eventID", d.UUID, "event identifier")
		})
		a.Response(d.OK, func() {
			a.Media(eventList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIError)
		a.Response(d.NotFound, JSONAPIError)
	})
})
