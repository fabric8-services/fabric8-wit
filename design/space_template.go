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
	a.Required("type")
})

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
		a.Response(d.OK, spaceTemplateSingle)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

})
