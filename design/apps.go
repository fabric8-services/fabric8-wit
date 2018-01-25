package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// SimpleSpace describe a space
var simpleSpaceV1 = a.Type("SimpleSpaceV1", func() {
	a.Description(`a space consisting of multiple applications`)
	a.Attribute("id", d.UUID)
	a.Attribute("name", d.String)
	a.Attribute("applications", a.ArrayOf(simpleAppV1))
	a.Required("applications")
})

// SimpleApp describe an application within a space
var simpleAppV1 = a.Type("SimpleAppV1", func() {
	a.Description(`a description of an application`)
	a.Attribute("id", d.UUID)
	a.Attribute("name", d.String)
	a.Attribute("pipeline", a.ArrayOf(simpleDeploymentV1))
	a.Required("pipeline")
})

// simpleDeployment describe an element of an application pipeline
var simpleDeploymentV1 = a.Type("SimpleDeploymentV1", func() {
	a.Description(`a deployment (a step in a pipeline, e.g. 'build')`)
	a.Attribute("id", d.UUID)
	a.Attribute("name", d.String)
	a.Attribute("version", d.String)
	a.Attribute("pods", podStatsV1)
})

// simpleDeployment describe an element of an application pipeline
var simpleEnvironmentV1 = a.Type("SimpleEnvironmentV1", func() {
	a.Description(`a shared environment`)
	a.Attribute("id", d.UUID)
	a.Attribute("name", d.String)
	a.Attribute("quota", envStatsV1)
})

var envStatsV1 = a.Type("EnvStatsV1", func() {
	a.Description("resource usage and quotas for an environment")
	a.Attribute("cpucores", envStatCoresV1)
	a.Attribute("memory", envStatMemoryV1)
})

var envStatCoresV1 = a.Type("EnvStatCoresV1", func() {
	a.Description(`CPU core stats`)
	a.Attribute("used", d.Number)
	a.Attribute("quota", d.Number)
})

var envStatMemoryV1 = a.Type("EnvStatMemoryV1", func() {
	a.Description(`memory stats`)
	a.Attribute("used", d.Number)
	a.Attribute("quota", d.Number)
	a.Attribute("units", d.String)
})

var podStatsV1 = a.Type("PodStatsV1", func() {
	a.Description(`pod stats`)
	a.Attribute("starting", d.Integer)
	a.Attribute("running", d.Integer)
	a.Attribute("stopping", d.Integer)
	a.Attribute("total", d.Integer)
})

var timedNumberTupleV1 = a.Type("TimedNumberTupleV1", func() {
	a.Description("a set of time and number values")
	a.Attribute("time", d.Number)
	a.Attribute("value", d.Number)
})

var simpleDeploymentStatsV1 = a.Type("SimpleDeploymentStatsV1", func() {
	a.Description("current deployment stats")
	a.Attribute("cores", timedNumberTupleV1)
	a.Attribute("memory", timedNumberTupleV1)
	a.Attribute("net_tx", timedNumberTupleV1)
	a.Attribute("net_rx", timedNumberTupleV1)
})

var simpleDeploymentStatSeriesV1 = a.Type("SimpleDeploymentStatSeriesV1", func() {
	a.Description("pod stat series")
	a.Attribute("start", d.Number)
	a.Attribute("end", d.Number)
	a.Attribute("memory", a.ArrayOf(timedNumberTupleV1))
	a.Attribute("cores", a.ArrayOf(timedNumberTupleV1))
	a.Attribute("net_tx", a.ArrayOf(timedNumberTupleV1))
	a.Attribute("net_rx", a.ArrayOf(timedNumberTupleV1))
})

var simpleSpaceSingleV1 = JSONSingle(
	"SimpleSpace", "Holds a single response to a space request",
	simpleSpaceV1,
	nil)

var simpleAppSingleV1 = JSONSingle(
	"SimpleApplication", "Holds a single response to a space/application request",
	simpleAppV1,
	nil)

var simpleEnvironmentSingleV1 = JSONSingle(
	"SimpleEnvironment", "Holds a single response to a space/environment request",
	simpleEnvironmentV1,
	nil)

var simpleEnvironmentMultipleV1 = JSONList(
	"SimpleEnvironment", "Holds a response to a space/environment request",
	simpleEnvironmentV1,
	nil,
	nil)

var simplePodV1 = a.Type("SimplePodV1", func() {
	a.Description("wrapper for a kubernetes Pod")
	a.Attribute("pod", d.Any)
})

var simplePodMultipleV1 = JSONList(
	"SimplePod", "Holds a list of pods",
	simplePodV1,
	nil,
	nil)

var simpleDeploymentSingleV1 = JSONSingle(
	"SimpleDeployment", "Holds a single response to a space/application/deployment request",
	simpleDeploymentV1,
	nil)

var simpleDeploymentStatsSingleV1 = JSONSingle(
	"SimpleDeploymentStats", "Holds a single response to a space/application/deployment/stats request",
	simpleDeploymentStatsV1,
	nil)

var simpleDeploymentStatSeriesSingleV1 = JSONSingle(
	"SimpleDeploymentStatSeries", "HOlds a response to a stat series query",
	simpleDeploymentStatSeriesV1,
	nil)

var simpleEnvironmentStatSingleV1 = JSONSingle(
	"EnvStats", "Holds a single response to a pipeline/stats request",
	envStatsV1,
	nil)

var _ = a.Resource("apps", func() {
	a.BasePath("/apps")

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
		a.Response(d.OK, simpleSpaceSingleV1)
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
		a.Response(d.OK, simpleAppSingleV1)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("showSpaceAppDeployment", func() {
		a.Routing(
			a.GET("/spaces/:spaceID/applications/:appName/deployments/:deployName"),
		)
		a.Description("list pipe element")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
			a.Param("appName", d.String, "Name of the application")
			a.Param("deployName", d.String, "Name of the pipe deployment")
		})
		a.Response(d.OK, simpleDeploymentSingleV1)
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
		a.Response(d.OK, simpleDeploymentStatsSingleV1)
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
		a.Response(d.OK, simpleDeploymentStatSeriesSingleV1)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("setDeployment", func() {
		a.Routing(
			a.PUT("/spaces/:spaceID/applications/:appName/deployments/:deployName/control"),
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
		a.Response(d.OK, simpleEnvironmentMultipleV1)
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
		a.Response(d.OK, simpleEnvironmentSingleV1)
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
