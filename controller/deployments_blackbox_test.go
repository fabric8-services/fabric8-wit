package controller_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/kubernetes"
)

type testKubeClient struct {
	closed bool
	// Don't implement methods we don't yet need
	kubernetes.KubeClientInterface
}

func (kc *testKubeClient) Close() {
	kc.closed = true
}

type testKubeClientGetter struct {
	client *testKubeClient
}

func (g *testKubeClientGetter) GetKubeClient(ctx context.Context) (kubernetes.KubeClientInterface, error) {
	// Overwrites previous clients created by this getter
	g.client = &testKubeClient{}
	// Also return an error to avoid executing remainder of calling method
	return g.client, errors.New("Test")
}

func TestAPIMethodsCloseKube(t *testing.T) {
	testCases := []struct {
		name   string
		method func(*controller.DeploymentsController) error
	}{
		{"SetDeployment", func(ctrl *controller.DeploymentsController) error {
			count := 1
			ctx := &app.SetDeploymentDeploymentsContext{
				PodCount: &count,
			}
			return ctrl.SetDeployment(ctx)
		}},
		{"ShowDeploymentStatSeries", func(ctrl *controller.DeploymentsController) error {
			ctx := &app.ShowDeploymentStatSeriesDeploymentsContext{}
			return ctrl.ShowDeploymentStatSeries(ctx)
		}},
		{"ShowDeploymentStats", func(ctrl *controller.DeploymentsController) error {
			ctx := &app.ShowDeploymentStatsDeploymentsContext{}
			return ctrl.ShowDeploymentStats(ctx)
		}},
		{"ShowEnvironment", func(ctrl *controller.DeploymentsController) error {
			ctx := &app.ShowEnvironmentDeploymentsContext{}
			return ctrl.ShowEnvironment(ctx)
		}},
		{"ShowSpace", func(ctrl *controller.DeploymentsController) error {
			ctx := &app.ShowSpaceDeploymentsContext{}
			return ctrl.ShowSpace(ctx)
		}},
		{"ShowSpaceApp", func(ctrl *controller.DeploymentsController) error {
			ctx := &app.ShowSpaceAppDeploymentsContext{}
			return ctrl.ShowSpaceApp(ctx)
		}},
		{"ShowSpaceAppDeployment", func(ctrl *controller.DeploymentsController) error {
			ctx := &app.ShowSpaceAppDeploymentDeploymentsContext{}
			return ctrl.ShowSpaceAppDeployment(ctx)
		}},
		{"ShowEnvAppPods", func(ctrl *controller.DeploymentsController) error {
			ctx := &app.ShowEnvAppPodsDeploymentsContext{}
			return ctrl.ShowEnvAppPods(ctx)
		}},
		{"ShowSpaceEnvironments", func(ctrl *controller.DeploymentsController) error {
			ctx := &app.ShowSpaceEnvironmentsDeploymentsContext{}
			return ctrl.ShowSpaceEnvironments(ctx)
		}},
	}
	// Check that each API method creating a KubeClientInterface also closes it
	getter := &testKubeClientGetter{}
	controller := &controller.DeploymentsController{
		KubeClientGetter: getter,
	}
	for _, testCase := range testCases {
		err := testCase.method(controller)
		assert.Error(t, err, "Expected error \"Test\": "+testCase.name)
		// Check Close was called before returning
		assert.NotNil(t, getter.client, "No Kube client created: "+testCase.name)
		assert.True(t, getter.client.closed, "Kube client not closed: "+testCase.name)
	}
}
