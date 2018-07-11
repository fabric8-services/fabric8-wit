package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
	"fmt"
	)

var root = a.Type("Root", func() {

	a.Description(`JSONAPI store for the data of a Root. See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("relationships", a.HashOf(d.String, d.Any), "User context information of any type as a json", func() {
		a.Example(map[string]interface{}{"codebases_che_start": "{}"})
	})
	a.Attribute("basePath", d.String)
	a.Attribute("attributes", d.Any)
	a.Attribute("id", d.UUID, "ID of root", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("links", genericLinksForRoot)

})

// genericLinksForRoot defines generic relations links that are specific to a root
var genericLinksForRoot = a.Type("GenericLinksForRoot", func() {
	a.Attribute("self", d.String)
})

var rootSingle = JSONSingle(
	"Root", "Holds a single Root",
	root,
	nil)

var _ = a.Resource("root", func() {
	a.BasePath("/root")
	fmt.Println(root)
	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List all endpoints. ")
		a.Response(d.OK, rootSingle)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})
