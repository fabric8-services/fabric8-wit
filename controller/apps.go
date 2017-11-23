package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

var (
	spacename = "spacename"
	appname   = "appname"
	appname2  = "secondapp"
	pipename  = "build"
	pipename2 = "deploy"
	dummy     uuid.UUID

	cpus   = 4
	cpumax = 10

	mem      = 13674823
	memmax   = 100000000
	memunits = "bytes"

	podstart    = 1
	podrunning  = 6
	podstopping = 2
	podquota    = 10
)

// AppsController implements the apps resource.
type AppsController struct {
	*goa.Controller
}

// NewAppsController creates a apps controller.
func NewAppsController(service *goa.Service) *AppsController {
	return &AppsController{Controller: service.NewController("AppsController")}
}

// ShowApp runs the showApp action.
func (c *AppsController) ShowApp(ctx *app.ShowAppAppsContext) error {
	// AppsController_ShowApp: start_implement

	res := &app.SimpleApplicationSingle{
		Data: &app.SimpleApp{
			Name: &appname,
			ID:   &ctx.AppID,
			Pipeline: []*app.SimpleDeployment{
				&app.SimpleDeployment{
					Name: &pipename,
					ID:   &ctx.AppID,
					Stats: &app.EnvStats{
						Cpucores: &app.EnvStatCores{
							Used: &cpus,
						},
						Memory: &app.EnvStatMemory{
							Used:  &mem,
							Units: &memunits,
						},
						Pods: &app.EnvStatPods{
							Starting: &podstart,
							Running:  &podrunning,
							Stopping: &podstopping,
						},
					},
				},
				&app.SimpleDeployment{
					Name: &pipename2,
					ID:   &ctx.AppID,
					Stats: &app.EnvStats{
						Cpucores: &app.EnvStatCores{
							Used: &cpus,
						},
						Memory: &app.EnvStatMemory{
							Used:  &mem,
							Units: &memunits,
						},
						Pods: &app.EnvStatPods{
							Starting: &podstart,
							Running:  &podrunning,
							Stopping: &podstopping,
						},
					},
				},
			},
		},
	}

	// AppsController_ShowApp: end_implement
	return ctx.OK(res)
}

// ShowDeploymentStats runs the showDeploymentStats action.
func (c *AppsController) ShowDeploymentStats(ctx *app.ShowDeploymentStatsAppsContext) error {
	// AppsController_ShowDeploymentStats: start_implement

	res := &app.SimpleDeploymentSingle{
		Data: &app.SimpleDeployment{
			Name: &pipename,
			ID:   &ctx.DeployID,
			Stats: &app.EnvStats{
				Cpucores: &app.EnvStatCores{
					Used: &cpus,
				},
				Memory: &app.EnvStatMemory{
					Used:  &mem,
					Units: &memunits,
				},
				Pods: &app.EnvStatPods{
					Starting: &podstart,
					Running:  &podrunning,
					Stopping: &podstopping,
				},
			},
		},
	}

	// AppsController_ShowDeploymentStats: end_implement
	return ctx.OK(res)
}

// ShowEnvironment runs the showEnvironment action.
func (c *AppsController) ShowEnvironment(ctx *app.ShowEnvironmentAppsContext) error {
	// AppsController_ShowEnvironment: start_implement

	res := &app.SimpleEnvironmentSingle{
		Data: &app.SimpleEnvironment{
			Name: &pipename,
			ID:   &ctx.EnvID,
			Quota: &app.EnvStats{
				Cpucores: &app.EnvStatCores{
					Quota: &cpumax,
					Used:  &cpus,
				},
				Memory: &app.EnvStatMemory{
					Quota: &memmax,
					Used:  &mem,
					Units: &memunits,
				},
			},
		},
	}

	// AppsController_ShowEnvironment: end_implement
	return ctx.OK(res)
}

// ShowSpace runs the showSpace action.
func (c *AppsController) ShowSpace(ctx *app.ShowSpaceAppsContext) error {
	// AppsController_ShowSpace: start_implement

	res := &app.SimpleSpaceSingle{
		Data: &app.SimpleSpace{
			//Name: &spacename,
			ID: &ctx.SpaceID,
			Applications: []*app.SimpleApp{
				&app.SimpleApp{
					Name: &appname,
					ID:   &ctx.SpaceID,
					Pipeline: []*app.SimpleDeployment{
						&app.SimpleDeployment{
							Name: &pipename,
							ID:   &ctx.SpaceID,
							Stats: &app.EnvStats{
								Cpucores: &app.EnvStatCores{
									Used: &cpus,
								},
								Memory: &app.EnvStatMemory{
									Used:  &mem,
									Units: &memunits,
								},
								Pods: &app.EnvStatPods{
									Starting: &podstart,
									Running:  &podrunning,
									Stopping: &podstopping,
								},
							},
						},
						&app.SimpleDeployment{
							Name: &pipename2,
							ID:   &ctx.SpaceID,
							Stats: &app.EnvStats{
								Cpucores: &app.EnvStatCores{
									Used: &cpus,
								},
								Memory: &app.EnvStatMemory{
									Used:  &mem,
									Units: &memunits,
								},
								Pods: &app.EnvStatPods{
									Starting: &podstart,
									Running:  &podrunning,
									Stopping: &podstopping,
								},
							},
						},
					},
				},
				&app.SimpleApp{
					Name: &appname2,
					ID:   &ctx.SpaceID,
					Pipeline: []*app.SimpleDeployment{
						&app.SimpleDeployment{
							Name: &pipename,
							ID:   &ctx.SpaceID,
							Stats: &app.EnvStats{
								Cpucores: &app.EnvStatCores{
									Used: &cpus,
								},
								Memory: &app.EnvStatMemory{
									Used:  &mem,
									Units: &memunits,
								},
								Pods: &app.EnvStatPods{
									Starting: &podstart,
									Running:  &podrunning,
									Stopping: &podstopping,
								},
							},
						},
						&app.SimpleDeployment{
							Name: &pipename2,
							ID:   &ctx.SpaceID,
							Stats: &app.EnvStats{
								Cpucores: &app.EnvStatCores{
									Used: &cpus,
								},
								Memory: &app.EnvStatMemory{
									Used:  &mem,
									Units: &memunits,
								},
								Pods: &app.EnvStatPods{
									Starting: &podstart,
									Running:  &podrunning,
									Stopping: &podstopping,
								},
							},
						},
					},
				},
			},
		},
	}

	// AppsController_ShowSpace: end_implement
	return ctx.OK(res)
}

// ShowSpaceApp runs the showSpaceApp action.
func (c *AppsController) ShowSpaceApp(ctx *app.ShowSpaceAppAppsContext) error {
	// AppsController_ShowSpaceApp: start_implement

	res := &app.SimpleApplicationSingle{
		Data: &app.SimpleApp{
			Name: &appname,
			ID:   &ctx.SpaceID,
			Pipeline: []*app.SimpleDeployment{
				&app.SimpleDeployment{
					Name: &pipename,
					ID:   &ctx.SpaceID,
					Stats: &app.EnvStats{
						Cpucores: &app.EnvStatCores{
							Used: &cpus,
						},
						Memory: &app.EnvStatMemory{
							Used:  &mem,
							Units: &memunits,
						},
						Pods: &app.EnvStatPods{
							Starting: &podstart,
							Running:  &podrunning,
							Stopping: &podstopping,
						},
					},
				},
				&app.SimpleDeployment{
					Name: &pipename2,
					ID:   &ctx.SpaceID,
					Stats: &app.EnvStats{
						Cpucores: &app.EnvStatCores{
							Used: &cpus,
						},
						Memory: &app.EnvStatMemory{
							Used:  &mem,
							Units: &memunits,
						},
						Pods: &app.EnvStatPods{
							Starting: &podstart,
							Running:  &podrunning,
							Stopping: &podstopping,
						},
					},
				},
			},
		},
	}

	// AppsController_ShowSpaceApp: end_implement
	return ctx.OK(res)
}

// ShowSpaceAppDeployment runs the showSpaceAppDeployment action.
func (c *AppsController) ShowSpaceAppDeployment(ctx *app.ShowSpaceAppDeploymentAppsContext) error {
	// AppsController_ShowSpaceAppDeployment: start_implement

	res := &app.SimpleDeploymentSingle{
		Data: &app.SimpleDeployment{
			Name: &pipename,
			ID:   &ctx.SpaceID,
			Stats: &app.EnvStats{
				Cpucores: &app.EnvStatCores{
					Used: &cpus,
				},
				Memory: &app.EnvStatMemory{
					Used:  &mem,
					Units: &memunits,
				},
				Pods: &app.EnvStatPods{
					Starting: &podstart,
					Running:  &podrunning,
					Stopping: &podstopping,
				},
			},
		},
	}

	// AppsController_ShowSpaceAppDeployment: end_implement
	return ctx.OK(res)
}

// ShowSpaceEnvironments runs the showSpaceEnvironments action.
func (c *AppsController) ShowSpaceEnvironments(ctx *app.ShowSpaceEnvironmentsAppsContext) error {
	// AppsController_ShowSpaceEnvironments: start_implement

	res := &app.SimpleEnvironmentList{
		Data: []*app.SimpleEnvironment{
			{
				Name: &pipename,
				ID:   &ctx.SpaceID,
				Quota: &app.EnvStats{
					Cpucores: &app.EnvStatCores{
						Quota: &cpumax,
						Used:  &cpus,
					},
					Memory: &app.EnvStatMemory{
						Quota: &memmax,
						Used:  &mem,
						Units: &memunits,
					},
				},
			},
			{
				Name: &pipename2,
				ID:   &ctx.SpaceID,
				Quota: &app.EnvStats{
					Cpucores: &app.EnvStatCores{
						Used:  &cpus,
						Quota: &cpumax,
					},
					Memory: &app.EnvStatMemory{
						Quota: &memmax,
						Used:  &mem,
						Units: &memunits,
					},
				},
			},
		},
	}

	// AppsController_ShowSpaceEnvironments: end_implement
	return ctx.OK(res)
}
