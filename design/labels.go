package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var label = a.Type("Label", func() {
	a.Description(`JSONAPI store for the data of a Label. See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("labels")
	})
	a.Attribute("id", d.UUID, "ID of label", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", labelAttributes)
	a.Attribute("relationships", labelRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

var labelAttributes = a.Type("LabelAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a Label. See also http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("name", d.String, "The Label name", nameValidationFunction)
	a.Attribute("created-at", d.DateTime, "When the label was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("updated-at", d.DateTime, "When the label was updated", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control (optional during creating)", func() {
		a.Example(23)
	})
	a.Attribute("text-color", d.String, "Text color in hex code format. See also http://www.color-hex.com", func() {
		a.Example("#ffa7cb")
	})
	a.Attribute("background-color", d.String, "Background color in hex code format. See also http://www.color-hex.com", func() {
		a.Example("#ffa7cb")
	})
	a.Attribute("border-color", d.String, "Border color in hex code format. See also http://www.color-hex.com", func() {
		a.Example("#ffa7cb")
	})
})

var labelRelationships = a.Type("LabelRelations", func() {
	a.Attribute("space", relationGeneric, "This defines the owning space")
})

var labelList = JSONList(
	"Label", "Holds the list of Labels",
	label,
	pagingLinks,
	meta)

var labelSingle = JSONSingle(
	"Label", "Holds a single Label",
	label,
	nil)

var _ = a.Resource("label", func() {
	a.Parent("space")
	a.BasePath("/labels")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:labelID"),
		)
		a.Description("Retrieve label for the given id.")
		a.Params(func() {
			a.Param("labelID", d.UUID, "ID of the label")
		})
		a.UseTrait("conditional")
		a.Response(d.OK, labelSingle)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List Labels.")
		a.UseTrait("conditional")
		a.Response(d.OK, labelList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("create label with id, name and color.")
		a.Payload(labelSingle)
		a.Response(d.Created, "/labels/.*", func() {
			a.Media(labelSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Conflict, JSONAPIErrors)
	})

	a.Action("update", func() {
		a.Security("jwt")
		a.Routing(
			a.PATCH("/:labelID"),
		)
		a.Description("update the label for the given id.")
		a.Params(func() {
			a.Param("labelID", d.UUID, "ID of the label to update")
		})
		a.Payload(labelSingle)
		a.Response(d.OK, func() {
			a.Media(labelSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.Conflict, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
	})
})

var _ = a.Resource("work_item_labels", func() {
	a.Parent("workitem")

	a.Action("list", func() {
		a.Routing(
			a.GET("labels"),
		)
		a.Description("List labels associated with the given work item")
		a.UseTrait("conditional")
		a.Response(d.OK, labelList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
})
