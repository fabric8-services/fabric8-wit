package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// SimpleSpace describe a space
var simpleSpace = a.Type("SimpleSpace", func() {
	a.Description(`a space consisting of multiple applications`)
	a.Attribute("id", d.UUID)
	//a.Attribute("name", d.String)
	a.Attribute("applications", a.ArrayOf(simpleApp))
})

// SimpleApp describe an application within a space
var simpleApp = a.Type("SimpleApp", func() {
	a.Description(`a description of an application`)
	a.Attribute("id", d.UUID)
	a.Attribute("name", d.String)
	a.Attribute("pipeline", a.ArrayOf(simpleDeployment))
})

// simpleDeployment describe an element of an application pipeline
var simpleDeployment = a.Type("simpleDeployment", func() {
	a.Description(`a deployment (a step in a pipeline, e.g. 'build')`)
	a.Attribute("id", d.UUID)
	a.Attribute("name", d.String)
	a.Attribute("stats", envStats)
	a.Attribute("quota", envStats)
})

// simpleDeployment describe an element of an application pipeline
var simpleEnvironment = a.Type("simpleEnvironment", func() {
	a.Description(`a shared environment`)
	a.Attribute("id", d.UUID)
	a.Attribute("name", d.String)
	a.Attribute("quota", envStats)
})

var envStats = a.Type("EnvStats", func() {
	a.Description(`statistics and quotas for an enviromnent or deployment`)
	a.Attribute("cpucores", envStatCores)
	a.Attribute("memory", envStatMemory)
	a.Attribute("pods", envStatPods)
})

var envStatCores = a.Type("EnvStatCores", func() {
	a.Description(`CPU core stats`)
	a.Attribute("used", d.Integer)
	a.Attribute("quota", d.Integer)
})

var envStatMemory = a.Type("EnvStatMemory", func() {
	a.Description(`memory stats`)
	a.Attribute("used", d.Integer)
	a.Attribute("quota", d.Integer)
	a.Attribute("units", d.String)
})

var envStatPods = a.Type("EnvStatPods", func() {
	a.Description(`pod stats`)
	a.Attribute("starting", d.Integer)
	a.Attribute("running", d.Integer)
	a.Attribute("stopping", d.Integer)
	a.Attribute("quota", d.Integer)
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

var simpleDeploymentSingle = JSONSingle(
	"SimpleDeployment", "Holds a single response to a space/application/deployment request",
	simpleDeployment,
	nil)

var simpleEnvironmentStatSingle = JSONSingle(
	"EnvStats", "Holds a single response to a pipeline/stats request",
	envStats,
	nil)

var _ = a.Resource("apps", func() {
	a.BasePath("/apps")
	// to add auth security:
	// a.Security("jwt")

	a.Action("showSpace", func() {
		a.Routing(
			a.GET("/spaces/:spaceID"),
		)
		a.Description("list applications in a space")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
		})
		a.Response(d.OK, simpleSpaceSingle)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("showSpaceApp", func() {
		a.Routing(
			a.GET("/spaces/:spaceID/applications/:appID"),
		)
		a.Description("list application")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
			a.Param("appID", d.UUID, "ID of the application")
		})
		a.Response(d.OK, simpleAppSingle)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("showApp", func() {
		a.Routing(
			a.GET("/applications/:appID"),
		)
		a.Description("list application")
		a.Params(func() {
			a.Param("appID", d.UUID, "ID of the application")
		})
		a.Response(d.OK, simpleAppSingle)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("showSpaceAppDeployment", func() {
		a.Routing(
			a.GET("/spaces/:spaceID/applications/:appID/deployments/:deployID"),
		)
		a.Description("list pipe element")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
			a.Param("appID", d.UUID, "ID of the application")
			a.Param("deployID", d.UUID, "ID of the pipe deployment")
		})
		a.Response(d.OK, simpleDeploymentSingle)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("showEnvironment", func() {
		a.Routing(
			a.GET("/environments/:envID"),
		)
		a.Description("list environment")
		a.Params(func() {
			a.Param("envID", d.UUID, "ID of the environment")
		})
		a.Response(d.OK, simpleEnvironmentSingle)
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
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("showDeploymentStats", func() {
		a.Routing(
			a.GET("/deployments/:deployID/stats"),
		)
		a.Description("list pipe element statistics")
		a.Params(func() {
			a.Param("deployID", d.UUID, "ID of the pipe element")
		})
		a.Response(d.OK, simpleDeploymentSingle)
		a.Response(d.NotFound, JSONAPIErrors)
	})

})
