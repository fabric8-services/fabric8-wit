package controller_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/kubernetesV1"
)

type testKubeClientV1 struct {
	closed bool
	// Don't implement methods we don't yet need
	kubernetesV1.KubeClientInterface
}

func (kc *testKubeClientV1) Close() {
	kc.closed = true
}

type testKubeClientGetterV1 struct {
	client *testKubeClientV1
}

func (g *testKubeClientGetterV1) GetKubeClientV1(ctx context.Context) (kubernetesV1.KubeClientInterface, error) {
	// Overwrites previous clients created by this getter
	g.client = &testKubeClientV1{}
	// Also return an error to avoid executing remainder of calling method
	return g.client, errors.New("Test")
}

func TestAPIMethodsCloseKubeV1(t *testing.T) {
	testCases := []struct {
		name   string
		method func(*controller.AppsController) error
	}{
		{"SetDeployment", func(ctrl *controller.AppsController) error {
			count := 1
			ctx := &app.SetDeploymentAppsContext{
				PodCount: &count,
			}
			return ctrl.SetDeployment(ctx)
		}},
		{"ShowDeploymentStatSeries", func(ctrl *controller.AppsController) error {
			ctx := &app.ShowDeploymentStatSeriesAppsContext{}
			return ctrl.ShowDeploymentStatSeries(ctx)
		}},
		{"ShowDeploymentStats", func(ctrl *controller.AppsController) error {
			ctx := &app.ShowDeploymentStatsAppsContext{}
			return ctrl.ShowDeploymentStats(ctx)
		}},
		{"ShowEnvironment", func(ctrl *controller.AppsController) error {
			ctx := &app.ShowEnvironmentAppsContext{}
			return ctrl.ShowEnvironment(ctx)
		}},
		{"ShowSpace", func(ctrl *controller.AppsController) error {
			ctx := &app.ShowSpaceAppsContext{}
			return ctrl.ShowSpace(ctx)
		}},
		{"ShowSpaceApp", func(ctrl *controller.AppsController) error {
			ctx := &app.ShowSpaceAppAppsContext{}
			return ctrl.ShowSpaceApp(ctx)
		}},
		{"ShowSpaceAppDeployment", func(ctrl *controller.AppsController) error {
			ctx := &app.ShowSpaceAppDeploymentAppsContext{}
			return ctrl.ShowSpaceAppDeployment(ctx)
		}},
		{"ShowEnvAppPods", func(ctrl *controller.AppsController) error {
			ctx := &app.ShowEnvAppPodsAppsContext{}
			return ctrl.ShowEnvAppPods(ctx)
		}},
		{"ShowSpaceEnvironments", func(ctrl *controller.AppsController) error {
			ctx := &app.ShowSpaceEnvironmentsAppsContext{}
			return ctrl.ShowSpaceEnvironments(ctx)
		}},
	}
	// Check that each API method creating a KubeClientInterface also closes it
	getter := &testKubeClientGetterV1{}
	controller := &controller.AppsController{
		KubeClientGetterV1: getter,
	}
	for _, testCase := range testCases {
		err := testCase.method(controller)
		assert.Error(t, err, "Expected error \"Test\": "+testCase.name)
		// Check Close was called before returning
		assert.NotNil(t, getter.client, "No Kube client created: "+testCase.name)
		assert.True(t, getter.client.closed, "Kube client not closed: "+testCase.name)
	}
}
