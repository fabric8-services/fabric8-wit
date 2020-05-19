package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// SimpleSpace describe a space
var simpleSpace = a.Type("SimpleSpace", func() {
	a.Description(`a space consisting of multiple applications`)
	a.Attribute("type", d.String, "The type of the related resource", func() {
		a.Enum("space")
		a.Default("space")
	})
	a.Attribute("id", d.UUID, "ID of the space", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", simpleSpaceAttributes)
	a.Attribute("links", simpleSpaceLinks)
	a.Required("type", "id", "attributes")
})

var simpleSpaceLinks = a.Type("SimpleSpaceLinks", func() {
	a.Description(`relevant links for a space object`)
	a.Attribute("space", linkWithAccess)
	a.Attribute("deployments", linkWithAccess)
	a.Required("space")
})

var simpleSpaceAttributes = a.Type("SimpleSpaceAttributes", func() {
	a.Description(`a space consisting of multiple applications`)
	a.Attribute("name", d.String)
	a.Attribute("applications", a.ArrayOf(simpleApp))
	a.Required("name", "applications")
})

// SimpleApp describe an application within a space
var simpleApp = a.Type("SimpleApp", func() {
	a.Description(`a description of an application`)
	a.Attribute("type", d.String, "The type of the related resource", func() {
		a.Enum("application")
	})
	a.Attribute("id", d.String, "ID of the application (same as 'name')")
	a.Attribute("attributes", simpleAppAttributes)
	a.Required("type", "id", "attributes")
})

var simpleAppAttributes = a.Type("SimpleAppAttributes", func() {
	a.Description(`a description of an application`)
	a.Attribute("name", d.String)
	a.Attribute("deployments", a.ArrayOf(simpleDeployment))
	a.Required("name", "deployments")
})

// simpleDeployment describe an element of an application pipeline
var simpleDeployment = a.Type("SimpleDeployment", func() {
	a.Description(`a deployment (a step in a pipeline, e.g. 'build')`)
	a.Attribute("type", d.String, "The type of the related resource", func() {
		a.Enum("deployment")
	})
	a.Attribute("id", d.String, "ID of the deployment (same as 'name')")
	a.Attribute("attributes", simpleDeploymentAttributes)
	a.Attribute("links", genericLinksForDeployment)
	a.Required("type", "id", "attributes")
})

var genericLinksForDeployment = a.Type("GenericLinksForDeployment", func() {
	a.Attribute("console", d.String)
	a.Attribute("logs", d.String)
	a.Attribute("application", d.String)
	a.Attribute("self", linkWithAccess)
	a.Attribute("stats", linkWithAccess)
	a.Attribute("stat_series", linkWithAccess)
})

var simpleDeploymentAttributes = a.Type("SimpleDeploymentAttributes", func() {
	a.Description(`a deployment (a step in a pipeline, e.g. 'build')`)
	a.Attribute("id", d.UUID)
	a.Attribute("name", d.String)
	a.Attribute("version", d.String)
	a.Attribute("pods", a.ArrayOf(a.ArrayOf(d.String)))
	a.Attribute("pod_total", d.Integer)
	a.Attribute("pods_quota", podsQuota)
	a.Required("name", "pods")
})

var podsQuota = a.Type("PodsQuota", func() {
	a.Description(`resource quotas for pods of a deployment`)
	a.Attribute("cpucores", d.Number)
	a.Attribute("memory", d.Number)
})

var simpleEnvironment = a.Type("SimpleEnvironment", func() {
	a.Description(`a shared environment`)
	a.Attribute("type", d.String, "The type of the related resource", func() {
		a.Enum("environment")
	})
	a.Attribute("id", d.String, "ID of the environment (same as 'name')")
	a.Attribute("attributes", simpleEnvironmentAttributes)
	a.Required("type", "id", "attributes")
})

var simpleEnvironmentAttributes = a.Type("SimpleEnvironmentAttributes", func() {
	a.Description(`a shared environment`)
	a.Attribute("name", d.String)
	a.Attribute("quota", envStats)
})

var envStats = a.Type("EnvStats", func() {
	a.Description("resource usage and quotas for an environment")
	a.Attribute("cpucores", envStatQuota)
	a.Attribute("memory", envStatQuota)
	a.Attribute("pods", envStatQuota)
	a.Attribute("replication_controllers", envStatQuota)
	a.Attribute("resource_quotas", envStatQuota)
	a.Attribute("services", envStatQuota)
	a.Attribute("secrets", envStatQuota)
	a.Attribute("config_maps", envStatQuota)
	a.Attribute("persistent_volume_claims", envStatQuota)
	a.Attribute("image_streams", envStatQuota)
})

var envStatQuota = a.Type("EnvStatQuota", func() {
	a.Description(`environment object counts`)
	a.Attribute("used", d.Number)
	a.Attribute("quota", d.Number)
})

var timedNumberTuple = a.Type("TimedNumberTuple", func() {
	a.Description("a set of time and number values")
	a.Attribute("time", d.Number)
	a.Attribute("value", d.Number)
})

var simpleDeploymentStats = a.Type("SimpleDeploymentStats", func() {
	a.Description("current deployment stats")
	a.Attribute("type", d.String, "The type of the related resource", func() {
		a.Enum("deploymentstats")
	})
	a.Attribute("id", d.String, "ID of the deployment (same as 'name')")
	a.Attribute("attributes", simpleDeploymentStatsAttributes)
	a.Required("type", "id", "attributes")
})

var simpleDeploymentStatsAttributes = a.Type("SimpleDeploymentStatsAttributes", func() {
	a.Description("current deployment stats")
	a.Attribute("cores", timedNumberTuple)
	a.Attribute("memory", timedNumberTuple)
	a.Attribute("net_tx", timedNumberTuple)
	a.Attribute("net_rx", timedNumberTuple)
})

var simpleDeploymentStatSeries = a.Type("SimpleDeploymentStatSeries", func() {
	a.Description("pod stat series")
	a.Attribute("start", d.Number)
	a.Attribute("end", d.Number)
	a.Attribute("memory", a.ArrayOf(timedNumberTuple))
	a.Attribute("cores", a.ArrayOf(timedNumberTuple))
	a.Attribute("net_tx", a.ArrayOf(timedNumberTuple))
	a.Attribute("net_rx", a.ArrayOf(timedNumberTuple))
})

var simpleDeploymentPodLimitRange = a.Type("SimpleDeploymentPodLimitRange", func() {
	a.Description("pod limit range")
	a.Attribute("limits", podsQuota)
})

var spaceAndOtherEnvironmentUsage = a.Type("SpaceAndOtherEnvironmentUsage", func() {
	a.Description("Environment usage by specific space and all others")
	a.Attribute("type", d.String, "The type of the related resource", func() {
		a.Enum("environment")
	})
	a.Attribute("id", d.String, "ID of the environment (same as 'name')")
	a.Attribute("attributes", spaceAndOtherEnvironmentAttributes)
	a.Required("type", "id", "attributes")
})

var spaceAndOtherEnvironmentAttributes = a.Type("SpaceAndOtherEnvironmentUsageAttributes", func() {
	a.Description("Attributes for environment usage info for a single space")
	a.Attribute("name", d.String)
	a.Attribute("space_usage", spaceEnvironmentUsageQuota)
	a.Attribute("other_usage", envStats)
})

var spaceEnvironmentUsageQuota = a.Type("SpaceEnvironmentUsageQuota", func() {
	a.Description("Quota info for space-aware environment usage")
	a.Attribute("cpucores", d.Number)
	a.Attribute("memory", d.Number)
})

var spaceAndOtherEnvironmentUsageMultiple = JSONList(
	"spaceAndOtherEnvironmentUsage",
	"Holds a response to environment usage for a space compared to other spaces",
	spaceAndOtherEnvironmentUsage,
	nil,
	nil)

var simpleSpaceSingle = JSONSingle(
	"SimpleSpace", "Holds a single response to a space request",
	simpleSpace,
	nil)

var simpleEnvironmentMultiple = JSONList(
	"SimpleEnvironment", "Holds a response to a environment request",
	simpleEnvironment,
	nil,
	nil)

var simpleDeploymentStatsSingle = JSONSingle(
	"SimpleDeploymentStats", "Holds a single response to a space/application/deployment/stats request",
	simpleDeploymentStats,
	nil)

var simpleDeploymentStatSeriesSingle = JSONSingle(
	"SimpleDeploymentStatSeries", "Holds a response to a stat series query",
	simpleDeploymentStatSeries,
	nil)

var simpleDeploymentPodLimitRangeSingle = JSONSingle(
	"simpleDeploymentPodLimitRange", "Holds a response to a pod limit range query",
	simpleDeploymentPodLimitRange,
	nil)

var _ = a.Resource("deployments", func() {
	a.BasePath("/deployments")

	// An auth token is required to call the auth API to get an OpenShift auth token.
	a.Security("jwt")

	a.Action("showSpace", func() {
		a.Routing(
			a.GET("/spaces/:spaceID"),
		)
		a.Description("list applications in a space")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
			a.Param("qp", d.Boolean, "if true, query and return permissions for this space")
		})
		a.Response(d.OK, simpleSpaceSingle)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
	})

	a.Action("showDeploymentStats", func() {
		a.Routing(
			a.GET("/spaces/:spaceID/applications/:appName/deployments/:deployName/stats"),
		)
		a.Description("get deployment statistics")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
			a.Param("appName", d.String, "Name of the application")
			a.Param("deployName", d.String, "Name of the deployment")
			a.Param("start", d.Number, "start time in millis")
		})
		a.Response(d.OK, simpleDeploymentStatsSingle)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
	})

	a.Action("showDeploymentStatSeries", func() {
		a.Routing(
			a.GET("/spaces/:spaceID/applications/:appName/deployments/:deployName/statseries"),
		)
		a.Description("list deployment statistics")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
			a.Param("appName", d.String, "Name of the application")
			a.Param("deployName", d.String, "Name of the deployment")
			a.Param("start", d.Number, "start time in millis")
			a.Param("end", d.Number, "end time in millis")
			a.Param("limit", d.Integer, "maximum number of data points to return")
		})
		a.Response(d.OK, simpleDeploymentStatSeriesSingle)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
	})

	a.Action("showDeploymentPodLimitRange", func() {
		a.Routing(
			a.GET("/spaces/:spaceID/applications/:appName/deployments/:deployName/podlimits"),
		)
		a.Description("get pod resource limit range")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
			a.Param("appName", d.String, "Name of the application")
			a.Param("deployName", d.String, "Name of the deployment")
		})
		a.Response(d.OK, simpleDeploymentPodLimitRangeSingle)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
	})

	a.Action("setDeployment", func() {
		a.Routing(
			a.PUT("/spaces/:spaceID/applications/:appName/deployments/:deployName"),
		)
		a.Description("set deployment pod count")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
			a.Param("appName", d.String, "Name of the application")
			a.Param("deployName", d.String, "Name of the deployment")
			a.Param("podCount", d.Integer, "desired running pod count")
		})
		a.Response(d.OK)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
	})

	a.Action("deleteDeployment", func() {
		a.Routing(
			a.DELETE("/spaces/:spaceID/applications/:appName/deployments/:deployName"),
		)
		a.Description("Delete a deployment of an application")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
			a.Param("appName", d.String, "Name of the application")
			a.Param("deployName", d.String, "Name of the deployment")
		})
		a.Response(d.OK)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
	})

	// FIXME Keep original API around until frontend is completely moved over to
	// showEnvironmentsBySpace, since this is a breaking change.
	a.Action("showSpaceEnvironments", func() {
		a.Routing(
			a.GET("/spaces/:spaceID/environments"),
		)
		a.Description("DEPRECATED: please use /environments/spaces/:spaceID instead")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
		})
		a.Response(d.OK, simpleEnvironmentMultiple)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
	})

	a.Action("showEnvironmentsBySpace", func() {
		a.Routing(
			a.GET("/environments/spaces/:spaceID"),
		)
		a.Description("list all environments for a space and information for all others")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
		})
		a.Response(d.OK, spaceAndOtherEnvironmentUsageMultiple)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
	})

	a.Action("showAllEnvironments", func() {
		a.Routing(
			a.GET("/environments"),
		)
		a.Description("list all environments")
		a.Response(d.OK, simpleEnvironmentMultiple)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
	})

	a.Action("watchEnvironmentEvents", func() {
		a.Security("jwt-query-param")
		a.Routing(
			a.GET("/environments/:envName/events/watch"),
		)
		a.Params(func() {
			a.Param("envName", d.String, "Name of the environment")
		})
		a.Description("watch events for an environment")
		a.Scheme("wss")
		a.Response(d.SwitchingProtocols)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})
