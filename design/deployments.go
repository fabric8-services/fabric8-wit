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
	a.Required("type", "id", "attributes")
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
	// in the future perhaps: a.Attribute("self", d.String)
})

var simpleDeploymentAttributes = a.Type("SimpleDeploymentAttributes", func() {
	a.Description(`a deployment (a step in a pipeline, e.g. 'build')`)
	a.Attribute("id", d.UUID)
	a.Attribute("name", d.String)
	a.Attribute("version", d.String)
	a.Attribute("pods", a.ArrayOf(a.ArrayOf(d.String)))
	a.Attribute("pod_total", d.Integer)
	a.Required("name")
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
	a.Attribute("cpucores", envStatCores)
	a.Attribute("memory", envStatMemory)
})

var envStatCores = a.Type("EnvStatCores", func() {
	a.Description(`CPU core stats`)
	a.Attribute("used", d.Number)
	a.Attribute("quota", d.Number)
})

var envStatMemory = a.Type("EnvStatMemory", func() {
	a.Description(`memory stats`)
	a.Attribute("used", d.Number)
	a.Attribute("quota", d.Number)
	a.Attribute("units", d.String)
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

var simpleSpaceSingle = JSONSingle(
	"SimpleSpace", "Holds a single response to a space request",
	simpleSpace,
	nil)

var simpleAppSingle = JSONSingle(
	"SimpleApplication", "Holds a single response to a space/application request",
	simpleApp,
	nil)

var simpleEnvironmentSingle = JSONSingle(
	"SimpleEnvironment", "Holds a single response to a space/environment request",
	simpleEnvironment,
	nil)

var simpleEnvironmentMultiple = JSONList(
	"SimpleEnvironment", "Holds a response to a space/environment request",
	simpleEnvironment,
	nil,
	nil)

var simplePod = a.Type("SimplePod", func() {
	a.Description("wrapper for a kubernetes Pod")
	a.Attribute("pod", d.Any)
})

var simplePodMultiple = JSONList(
	"SimplePod", "Holds a list of pods",
	simplePod,
	nil,
	nil)

var simpleDeploymentSingle = JSONSingle(
	"SimpleDeployment", "Holds a single response to a space/application/deployment request",
	simpleDeployment,
	nil)

var simpleDeploymentStatsSingle = JSONSingle(
	"SimpleDeploymentStats", "Holds a single response to a space/application/deployment/stats request",
	simpleDeploymentStats,
	nil)

var simpleDeploymentStatSeriesSingle = JSONSingle(
	"SimpleDeploymentStatSeries", "HOlds a response to a stat series query",
	simpleDeploymentStatSeries,
	nil)

var simpleEnvironmentStatSingle = JSONSingle(
	"EnvStats", "Holds a single response to a pipeline/stats request",
	envStats,
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
		})
		a.Response(d.OK, simpleSpaceSingle)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("showSpaceApp", func() {
		a.Routing(
			a.GET("/spaces/:spaceID/applications/:appName"),
		)
		a.Description("list application")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
			a.Param("appName", d.String, "Name of the application")
		})
		a.Response(d.OK, simpleAppSingle)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("showSpaceAppDeployment", func() {
		a.Routing(
			a.GET("/spaces/:spaceID/applications/:appName/deployments/:deployName"),
		)
		a.Description("list deployment")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
			a.Param("appName", d.String, "Name of the application")
			a.Param("deployName", d.String, "Name of the deployment")
		})
		a.Response(d.OK, simpleDeploymentSingle)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
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
	})

	a.Action("showSpaceEnvironments", func() {
		a.Routing(
			a.GET("/spaces/:spaceID/environments"),
		)
		a.Description("list all environments for a space")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
		})
		a.Response(d.OK, simpleEnvironmentMultiple)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("showEnvironment", func() {
		a.Routing(
			a.GET("/environments/:envName"),
		)
		a.Description("list environment")
		a.Params(func() {
			a.Param("envName", d.String, "Name of the environment")
		})
		a.Response(d.OK, simpleEnvironmentSingle)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("showEnvAppPods", func() {
		a.Routing(
			a.GET("/environments/:envName/applications/:appName/pods"),
		)
		a.Description("list application pods")
		a.Params(func() {
			a.Param("envName", d.String, "Name of the environment")
			a.Param("appName", d.String, "Name of the application")
		})
		// TODO - find a way to use predefined structs in goa DSL
		// until then, hand code JSON response here instead of []v1.Pod
		a.Response(d.OK, "application/json")
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

})
