package controller_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/goadesign/goa"
	"github.com/goadesign/goa/goatest"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/kubernetes"
)

type testKubeClient struct {
	fixture       *deploymentsTestFixture
	closed        bool
	deleteResults *deleteTestResults
	// Don't implement methods we don't yet need
	kubernetes.KubeClientInterface
}

type testOSIOClient struct {
	fixture *deploymentsTestFixture
	// Don't implement methods we don't yet need
	controller.OpenshiftIOClient
}

func (kc *testKubeClient) Close() {
	kc.closed = true
}

type deploymentsTestFixture struct {
	kube         *testKubeClient
	spaceMapping map[string]string
	deploymentsTestErrors
}

type deploymentsTestErrors struct {
	getKubeClientError    error
	deleteDeploymentError error
}

func (fixture *deploymentsTestFixture) GetKubeClient(ctx context.Context) (kubernetes.KubeClientInterface, error) {
	// Overwrites previous clients created by this getter
	fixture.kube = &testKubeClient{
		fixture: fixture,
	}
	return fixture.kube, fixture.getKubeClientError
}

type deleteTestResults struct {
	spaceName string
	appName   string
	envName   string
}

func (c *testKubeClient) DeleteDeployment(spaceName string, appName string, envName string) error {
	c.deleteResults = &deleteTestResults{
		spaceName: spaceName,
		appName:   appName,
		envName:   envName,
	}
	return c.fixture.deleteDeploymentError
}

func (fixture *deploymentsTestFixture) GetAndCheckOSIOClient(ctx context.Context) (controller.OpenshiftIOClient, error) {
	return &testOSIOClient{
		fixture: fixture,
	}, nil
}

func (c *testOSIOClient) GetSpaceByID(ctx context.Context, spaceID uuid.UUID) (*app.Space, error) {
	var spaceName *string
	uuidString := spaceID.String()
	name, pres := c.fixture.spaceMapping[uuidString]
	if pres {
		spaceName = &name
	}
	space := &app.Space{
		Attributes: &app.SpaceAttributes{
			Name: spaceName,
		},
	}
	return space, nil
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
	fixture := &deploymentsTestFixture{
		// Also return an error to avoid executing remainder of calling method
		deploymentsTestErrors: deploymentsTestErrors{
			getKubeClientError: errors.New("Test"),
		},
	}
	controller := &controller.DeploymentsController{
		ClientGetter: fixture,
	}
	for _, testCase := range testCases {
		err := testCase.method(controller)
		require.Error(t, err, "Expected error \"Test\": "+testCase.name)
		// Check Close was called before returning
		require.NotNil(t, fixture.kube, "No Kube client created: "+testCase.name)
		require.True(t, fixture.kube.closed, "Kube client not closed: "+testCase.name)
	}
}

func TestDeleteDeployment(t *testing.T) {
	const uuidStr = "ed3b4c4d-5a47-44ec-8b73-9a0fbc902184"
	const spaceName = "mySpace"
	const appName = "myApp"
	const envName = "myEnv"

	expectedResults := &deleteTestResults{
		spaceName: spaceName,
		appName:   appName,
		envName:   envName,
	}
	testCases := []struct {
		testName   string
		deleteFunc func(t goatest.TInterface, ctx context.Context, service *goa.Service, ctrl app.DeploymentsController,
			spaceID uuid.UUID, appName string, deployName string) (http.ResponseWriter, *app.JSONAPIErrors)
		spaceUUID       string
		expectedResults *deleteTestResults
		deploymentsTestErrors
	}{
		{
			testName: "Basic",
			deleteFunc: func(t goatest.TInterface, ctx context.Context, service *goa.Service, ctrl app.DeploymentsController,
				spaceID uuid.UUID, appName string, deployName string) (http.ResponseWriter, *app.JSONAPIErrors) {
				// Wrap test method to return additional *app.JSONAPIErrors value
				return test.DeleteDeploymentDeploymentsOK(t, ctx, service, ctrl, spaceID, appName, deployName), nil
			},
			spaceUUID:       uuidStr,
			expectedResults: expectedResults,
		},
		{
			testName:   "Delete Failure",
			deleteFunc: test.DeleteDeploymentDeploymentsInternalServerError,
			spaceUUID:  uuidStr,
			deploymentsTestErrors: deploymentsTestErrors{
				deleteDeploymentError: errors.New("TEST"), // Return expected error from DeleteDeployment
			},
			expectedResults: expectedResults,
		},
		{
			testName:   "Space Not Found",
			deleteFunc: test.DeleteDeploymentDeploymentsNotFound,
			spaceUUID:  "9de7a4bc-d098-4867-809c-759e2cd824f4", // Different UUID
		},
		{
			testName:   "Auth Failure",
			deleteFunc: test.DeleteDeploymentDeploymentsUnauthorized,
			spaceUUID:  uuidStr,
			deploymentsTestErrors: deploymentsTestErrors{
				getKubeClientError: errors.New("TEST"), // Return expected error from GetKubeClient
			},
		},
	}
	fixture := &deploymentsTestFixture{
		spaceMapping: map[string]string{uuidStr: spaceName},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			fixture.deploymentsTestErrors = testCase.deploymentsTestErrors

			// Create controller and install our test fixture
			svc, controller, err := createDeploymentsController()
			require.NoError(t, err, "Failed to create controller")
			controller.ClientGetter = fixture

			spUUID, err := uuid.FromString(testCase.spaceUUID)
			require.NoError(t, err, "Bad UUID")
			// Invoke Goa-generated test method used by this test case
			testCase.deleteFunc(t, svc.Context, svc, controller, spUUID, appName, envName)

			// Check arguments passed to DeleteDeployment
			if testCase.expectedResults != nil {
				results := fixture.kube.deleteResults
				require.NotNil(t, results, "DeleteDeployment not called")
				require.Equal(t, testCase.expectedResults.spaceName, results.spaceName, "Incorrect space name")
				require.Equal(t, testCase.expectedResults.appName, results.appName, "Incorrect application name")
				require.Equal(t, testCase.expectedResults.envName, results.envName, "Incorrect environment name")
			}

			// Check KubeClient is closed
			require.True(t, fixture.kube.closed, "KubeClient is still open")
		})
	}
}

func createDeploymentsController() (*goa.Service, *controller.DeploymentsController, error) {
	svc := goa.New("deployment-service-test")
	config, err := configuration.New("../config.yaml")
	if err != nil {
		return nil, nil, err
	}
	return svc, controller.NewDeploymentsController(svc, config), nil
}
