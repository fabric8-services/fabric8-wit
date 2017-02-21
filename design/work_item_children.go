package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("work-item-children", func() {
	a.Parent("workitem")

	a.Action("list", func() {
		a.Routing(
			a.GET("children"),
		)
		a.Description("List children associated with the given work item")
		a.Response(d.OK, func() {
			a.Media(workItemList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
})
