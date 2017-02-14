package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("workitemtype", func() {

	a.BasePath("/workitemtypes")

	a.Action("show", func() {

		a.Routing(
			a.GET("/:name"),
		)
		a.Description("Retrieve work item type with given name.")
		a.Params(func() {
			a.Param("name", d.String, "name")
		})
		a.Response(d.OK, func() {
			a.Media(workItemType)
		})
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("Create work item type.")
		a.Payload(CreateWorkItemTypePayload)
		a.Response(d.Created, "/workitemtypes/.*", func() {
			a.Media(workItemType)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List work item types.")
		a.Params(func() {
			a.Param("page", d.String, "Paging in the format <start>,<limit>")
		})
		a.Response(d.OK, func() {
			a.Media(a.CollectionOf(workItemType))
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("list-source-link-types", func() {
		a.Routing(
			a.GET("/:name/source-link-types"),
		)
		a.Params(func() {
			a.Param("name", d.String, "name")
		})
		a.Description(`Retrieve work item link types where the
given work item type can be used in the source of the link.`)
		a.Response(d.OK, func() {
			a.Media(workItemLinkTypeList)
		})
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("list-target-link-types", func() {
		a.Routing(
			a.GET("/:name/target-link-types"),
		)
		a.Params(func() {
			a.Param("name", d.String, "name")
		})
		a.Description(`Retrieve work item link types where the
given work item type can be used in the target of the link.`)
		a.Response(d.OK, func() {
			a.Media(workItemLinkTypeList)
		})
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})

var _ = a.Resource("status", func() {

	a.DefaultMedia(ALMStatus)
	a.BasePath("/status")

	a.Action("show", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("Show the status of the current running instance")
		a.Response(d.OK)
		a.Response(d.ServiceUnavailable, ALMStatus)
	})
})

var _ = a.Resource("login", func() {

	a.BasePath("/login")

	a.Action("authorize", func() {
		a.Routing(
			a.GET("authorize"),
		)
		a.Description("Authorize with the ALM")
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.TemporaryRedirect)
	})

	a.Action("generate", func() {
		a.Routing(
			a.GET("generate"),
		)
		a.Description("Generates a set of Tokens for different Auth levels. NOT FOR PRODUCTION. Only available if server is running in dev mode")
		a.Response(d.OK, func() {
			a.Media(a.CollectionOf(AuthToken))
		})
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})

var _ = a.Resource("tracker", func() {
	a.BasePath("/trackers")

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List all tracker configurations.")
		a.Params(func() {
			a.Param("filter", d.String, "a query language expression restricting the set of found items")
			a.Param("page", d.String, "Paging in the format <start>,<limit>")
		})
		a.Response(d.OK, func() {
			a.Media(a.CollectionOf(Tracker))
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("show", func() {
		a.Routing(
			a.GET("/:id"),
		)
		a.Description("Retrieve tracker configuration for the given id.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Response(d.OK, func() {
			a.Media(Tracker)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("Add new tracker configuration.")
		a.Payload(CreateTrackerAlternatePayload)
		a.Response(d.Created, "/trackers/.*", func() {
			a.Media(Tracker)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
	a.Action("delete", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/:id"),
		)
		a.Description("Delete tracker configuration.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Response(d.OK)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
	a.Action("update", func() {
		a.Security("jwt")
		a.Routing(
			a.PUT("/:id"),
		)
		a.Description("Update tracker configuration.")
		a.Payload(UpdateTrackerAlternatePayload)
		a.Response(d.OK, func() {
			a.Media(Tracker)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})

})

var _ = a.Resource("trackerquery", func() {
	a.BasePath("/trackerqueries")
	a.Action("show", func() {
		a.Routing(
			a.GET("/:id"),
		)
		a.Description("Retrieve tracker configuration for the given id.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Response(d.OK, func() {
			a.Media(TrackerQuery)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("Add new tracker query.")
		a.Payload(CreateTrackerQueryAlternatePayload)
		a.Response(d.Created, "/trackerqueries/.*", func() {
			a.Media(TrackerQuery)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
	a.Action("update", func() {
		a.Security("jwt")
		a.Routing(
			a.PUT("/:id"),
		)
		a.Description("Update tracker query.")
		a.Payload(UpdateTrackerQueryAlternatePayload)
		a.Response(d.OK, func() {
			a.Media(TrackerQuery)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
	a.Action("delete", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/:id"),
		)
		a.Description("Delete tracker query")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Response(d.OK)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List all tracker queries.")
		a.Response(d.OK, func() {
			a.Media(a.CollectionOf(TrackerQuery))
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
})
