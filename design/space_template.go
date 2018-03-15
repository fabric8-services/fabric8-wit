package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var spaceTemplate = a.Type("SpaceTemplate", func() {
	a.Description(`JSONAPI store for the data of a space template. See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("spacetemplates")
	})
	a.Attribute("id", d.UUID, "ID of space template", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", spaceTemplateAttributes)
	a.Attribute("links", genericLinks)
	a.Attribute("relationships", spaceTemplateRelationships)
	a.Required("type", "attributes")
})

var spaceTemplateAttributes = a.Type("SpaceTemplateAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a space template. See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("name", d.String, mandatoryOnCreate("name of the space template"), func() {
		a.Example("Issue Tracking")
		a.MaxLength(62) // maximum space name length is 62 characters
		a.MinLength(1)  // minimum space name length is 1 characters
		a.Pattern("^[^_|-].*")
	})
	a.Attribute("created-at", d.DateTime, "When the space template was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("updated-at", d.DateTime, "When the space template was updated", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("description", d.String, "optional description of the space template", func() {
		a.Example("A very simple development methodology focused on the tracking of Issues and the Tasks needed to be completed to resolve a particular Issue.")
	})
	// a.Attribute("template", d.String, "base64 encoded YAML template (no newlines allowed in Base64)", func() {
	// 	a.Example("d29ya19pdGVtX3R5cGVzOiAhIW1hcAo=")
	// 	// Minimum length is 4 because an empty string is disallowed and the
	// 	// minimum base64 string is 4.
	// 	a.MinLength(4)
	// 	// found here: http://stackoverflow.com/questions/475074/regex-to-parse-or-validate-base64-data
	// 	a.Pattern("^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$")
	// 	// We don't accept templates that are bigger than 1MB of characters
	// 	a.MaxLength(1048576)
	// })
	a.Attribute("version", d.Integer, "version for optimistic concurrency control (optional during creating)", func() {
		a.Example(23)
	})
})

var spaceTemplateRelationships = a.Type("SpaceTemplateRelationships", func() {
	a.Attribute("workitemlinktypes", relationGeneric, "Space template can have one or many work item link types")
	a.Attribute("workitemtypes", relationGeneric, "Space template can have one or many work item types")
	a.Attribute("workitemtypegroups", relationGeneric, "Space template can have one or many work item type groups")
})

var spaceTemplateRelation = a.Type("SpaceTemplateRelation", func() {
	a.Attribute("data", spaceTemplateRelationData)
	a.Attribute("links", genericLinks)
	a.Required("data")
})

var spaceTemplateRelationData = a.Type("SpaceTemplateRelationData", func() {
	a.Attribute("id", d.UUID, "unique id of the space template")
	a.Attribute("type", d.String, "type of the user identity", func() {
		a.Enum("spacetemplates")
	})
	a.Required("type", "id")
})

var spaceTemplateList = JSONList(
	"SpaceTemplate", "Holds the list of space templates",
	spaceTemplate,
	pagingLinks,
	meta)

var spaceTemplateSingle = JSONSingle(
	"SpaceTemplate", "Holds a single space template",
	spaceTemplate,
	nil)

var _ = a.Resource("space_template", func() {
	a.BasePath("/spacetemplates")
	a.Action("show", func() {
		a.Routing(
			a.GET("/:spaceTemplateID"),
		)
		a.Description("Retrieve space template with given ID")
		a.Params(func() {
			a.Param("spaceTemplateID", d.UUID, "id of the space template to fetch")
		})
		a.UseTrait("conditional")
		a.Response(d.OK, spaceTemplateSingle)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List space templates")
		a.UseTrait("conditional")
		a.Response(d.OK, spaceTemplateList)
		a.Response(d.NotModified)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})
