package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
	"fmt"
	)

// genericLinksForWorkItem defines generic relations links that are specific to a workitem
var genericLinksForRoot = a.Type("GenericLinksForRoot", func() {
	a.Attribute("self", d.String)
})

var root = a.Type("Root", func() {

	a.Description(`JSONAPI store for the data of a Label. See also http://jsonapi.org/format/#document-resource-object`)
	//a.Attribute("relationships", d.Any)
	a.Attribute("relationships", a.HashOf(d.String, d.Any), "User context information of any type as a json", func() {
		a.Example(map[string]interface{}{"last_visited_url": "https://a.openshift.io", "space": "3d6dab8d-f204-42e8-ab29-cdb1c93130ad"})
	})
	a.Attribute("attributes", d.Any)
	a.Attribute("id", d.UUID, "ID of root", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("links", genericLinksForRoot)

})

var rootSingle = JSONSingle(
	"Root", "Holds a single Area",
	root,
	nil)

var _ = a.Resource("root", func() {
	a.BasePath("/root")
	fmt.Println(root)
	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List collaborators for the given space ID. ")
		a.Response(d.OK, rootSingle)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})
