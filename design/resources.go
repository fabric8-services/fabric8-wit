package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("workitem", func() {
	a.BasePath("/workitems")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:id"),
		)
		a.Description("Retrieve work item with given id.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Response(d.OK, func() {
			a.Media(workItem)
		})
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
		a.Response(d.NotFound)
	})

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List work items.")
		a.Params(func() {
			a.Param("filter", d.String, "a query language expression restricting the set of found work items")
			a.Param("page", d.String, "Paging in the format <start>,<limit>")
		})
		a.Response(d.OK, func() {
			a.Media(a.CollectionOf(workItem))
		})
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("create work item with type and id.")
		a.Payload(CreateWorkItemPayload)
		a.Response(d.Created, "/workitems/.*", func() {
			a.Media(workItem)
		})
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
		a.Response(d.Unauthorized)
	})
	a.Action("delete", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/:id"),
		)
		a.Description("Delete work item with given id.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Response(d.OK)
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
		a.Response(d.NotFound)
		a.Response(d.Unauthorized)
	})
	a.Action("update", func() {
		a.Security("jwt")
		a.Routing(
			a.PUT("/:id"),
		)
		a.Description("update the given work item with given id.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Payload(UpdateWorkItemPayload)
		a.Response(d.OK, func() {
			a.Media(workItem)
		})
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
		a.Response(d.NotFound)
		a.Response(d.Unauthorized)
	})

})

// new version of "list" for migration
var _ = a.Resource("workitem.2", func() {
	a.BasePath("/workitems.2")
	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List work items.")
		a.Params(func() {
			a.Param("filter", d.String, "a query language expression restricting the set of found work items")
			a.Param("page[offset]", d.String, "Paging start position")
			a.Param("page[limit]", d.Integer, "Paging size")
		})
		a.Response(d.OK, func() {
			a.Media(workItemListResponse)
		})
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
	})

})

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
		a.Response(d.NotFound)
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
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
		a.Response(d.Unauthorized)
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
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
	})
})

var _ = a.Resource("user", func() {
	a.BasePath("/user")

	a.Action("show", func() {
		a.Security("jwt")
		a.Routing(
			a.GET(""),
		)
		a.Description("Get the authenticated user")
		a.Response(d.OK, func() {
			a.Media(User)
		})
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
		a.Response(d.Unauthorized)
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
		a.Response(d.Unauthorized)
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
		a.Response(d.Unauthorized)
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
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
		a.Response(d.NotFound)
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
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
		a.Response(d.NotFound)
	})

	a.Action("create", func() {
		a.Routing(
			a.POST(""),
		)
		a.Description("Add new tracker configuration.")
		a.Payload(CreateTrackerAlternatePayload)
		a.Response(d.Created, "/trackers/.*", func() {
			a.Media(Tracker)
		})
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
		a.Response(d.NotFound)
	})
	a.Action("delete", func() {
		a.Routing(
			a.DELETE("/:id"),
		)
		a.Description("Delete tracker configuration.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Response(d.OK)
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
		a.Response(d.NotFound)
	})
	a.Action("update", func() {
		a.Routing(
			a.PUT("/:id"),
		)
		a.Description("Update tracker configuration.")
		a.Payload(UpdateTrackerAlternatePayload)
		a.Response(d.OK, func() {
			a.Media(Tracker)
		})
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
		a.Response(d.NotFound)
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
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
		a.Response(d.NotFound)
	})

	a.Action("create", func() {
		a.Routing(
			a.POST(""),
		)
		a.Description("Add new tracker query.")
		a.Payload(CreateTrackerQueryAlternatePayload)
		a.Response(d.Created, "/trackerqueries/.*", func() {
			a.Media(TrackerQuery)
		})
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
		a.Response(d.NotFound)
	})
	a.Action("update", func() {
		a.Routing(
			a.PUT("/:id"),
		)
		a.Description("Update tracker query.")
		a.Payload(UpdateTrackerQueryAlternatePayload)
		a.Response(d.OK, func() {
			a.Media(TrackerQuery)
		})
		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})
		a.Response(d.InternalServerError)
		a.Response(d.NotFound)
	})

})

var _ = a.Resource("search", func() {
	a.BasePath("/search")

	a.Action("show", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("Search by ID, URL, full text capability")
		a.Params(func() {
			a.Param("q", d.String, "Search Query")
			a.Param("page[offset]", d.Number, "Paging in the format <start>,<limit>")
			a.Param("page[limit]", d.Number, "Paging in the format <start>,<limit>")
		})
		a.Response(d.OK, func() {
			a.Media(searchResponse)
		})

		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})

		a.Response(d.InternalServerError)
	})
})
