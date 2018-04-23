package controller_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/kubernetes"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/goatest"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/pkg/api/v1"
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
		{"ShowSpace", func(ctrl *controller.DeploymentsController) error {
			ctx := &app.ShowSpaceDeploymentsContext{}
			return ctrl.ShowSpace(ctx)
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

type ContextMock struct {
	valueResult interface{}
	context.Context
}

func (c ContextMock) Value(key interface{}) interface{} {
	return c.valueResult
}

type ResponseWriterMock struct {
	header     http.Header
	writeValue int
	writeError error
	http.ResponseWriter
}

func (r ResponseWriterMock) Header() http.Header {
	return r.header
}

func (r ResponseWriterMock) WriteHeader(int) {
}

func (r ResponseWriterMock) Write(b []byte) (int, error) {
	return r.writeValue, r.writeError
}

type ClientGetterMock struct {
	kubeClientInterface kubernetes.KubeClientInterface
	getKubeClientError  error
	configRegistry      *configuration.Registry
	osioClient          controller.OpenshiftIOClient
	err                 error
}

func (c ClientGetterMock) GetKubeClient(ctx context.Context) (kubernetes.KubeClientInterface, error) {
	return c.kubeClientInterface, c.getKubeClientError
}

func (c ClientGetterMock) GetConfig() *configuration.Registry {
	return c.configRegistry
}

func (c ClientGetterMock) GetAndCheckOSIOClient(ctx context.Context) (controller.OpenshiftIOClient, error) {
	return c.osioClient, c.err
}

type KubeClientMock struct {
	scaleDeploymentResult  *int
	scaleDeploymentError   error
	simpleStatSeries       *app.SimpleDeploymentStatSeries
	simpleStatSeriesError  error
	deploymentStats        *app.SimpleDeploymentStats
	deploymentStatsError   error
	simpleEnvironment      *app.SimpleEnvironment
	simpleEnvironmentError error
	simpleSpace            *app.SimpleSpace
	simpleSpaceError       error
	simpleApp              *app.SimpleApp
	simpleAppError         error
	simpleDeployment       *app.SimpleDeployment
	simpleDeploymentError  error
	envAppPods             []v1.Pod
	envAppPodsError        error
	environments           []*app.SimpleEnvironment
	environmentsError      error
	kubernetes.KubeClientInterface
}

func (k KubeClientMock) ScaleDeployment(spaceName string, appName string, envName string, deployNumber int) (*int, error) {
	return k.scaleDeploymentResult, k.scaleDeploymentError
}

func (k KubeClientMock) GetDeploymentStatSeries(spaceName string, appName string, envName string, startTime time.Time, endTime time.Time, limit int) (*app.SimpleDeploymentStatSeries, error) {
	return k.simpleStatSeries, k.simpleStatSeriesError
}

func (k KubeClientMock) GetDeploymentStats(spaceName string, appName string, envName string, startTime time.Time) (*app.SimpleDeploymentStats, error) {
	return k.deploymentStats, k.deploymentStatsError
}

func (k KubeClientMock) GetEnvironment(envName string) (*app.SimpleEnvironment, error) {
	return k.simpleEnvironment, k.simpleEnvironmentError
}

func (k KubeClientMock) GetSpace(spaceName string) (*app.SimpleSpace, error) {
	return k.simpleSpace, k.simpleSpaceError
}

func (k KubeClientMock) GetApplication(spaceName string, appName string) (*app.SimpleApp, error) {
	return k.simpleApp, k.simpleAppError
}

func (k KubeClientMock) GetDeployment(spaceName string, appName string, envName string) (*app.SimpleDeployment, error) {
	return k.simpleDeployment, k.simpleDeploymentError
}

func (k KubeClientMock) GetPodsInNamespace(nameSpace string, appName string) ([]v1.Pod, error) {
	return k.envAppPods, k.envAppPodsError
}

func (k KubeClientMock) GetEnvironments() ([]*app.SimpleEnvironment, error) {
	return k.environments, k.environmentsError
}

func (k KubeClientMock) Close() {
}

type OSIOClientMock struct {
	namespaceAttributes *app.NamespaceAttributes
	namespaceTypeError  error
	userService         *app.UserService
	userServiceError    error
	space               *app.Space
	spaceError          error
}

func (o OSIOClientMock) GetNamespaceByType(ctx context.Context, userService *app.UserService, namespaceType string) (*app.NamespaceAttributes, error) {
	return o.namespaceAttributes, o.namespaceTypeError
}

func (o OSIOClientMock) GetUserServices(ctx context.Context) (*app.UserService, error) {
	return o.userService, o.userServiceError
}

func (o OSIOClientMock) GetSpaceByID(ctx context.Context, spaceID uuid.UUID) (*app.Space, error) {
	return o.space, o.spaceError
}

type ContextResponderMock struct {
	deploymentStatSeriesAppsError       error
	deploymentStatSeriesAppsVerifier    func(res *app.SimpleDeploymentStatSeriesSingle) error
	deploymentStatsError                error
	deploymentStatsVerifier             func(res *app.SimpleDeploymentStatsSingle) error
	deploymentSpaceError                error
	deploymentSpaceVerifier             func(res *app.SimpleSpaceSingle) error
	deploymentsEnvironmentsError        error
	deploymentsEnvironmentsVerifier     func(res *app.SimpleEnvironmentList) error
	deploymentsSetDeploymentError       error
	deploymentsSetDeploymentVerifier    func(res []byte) error
	deploymentsDeleteDeploymentError    error
	deploymentsDeleteDeploymentVerifier func(res []byte) error
}

func (c ContextResponderMock) SendShowDeploymentStatSeriesAppsOK(res *app.SimpleDeploymentStatSeriesSingle, ctx *app.ShowDeploymentStatSeriesDeploymentsContext) error {
	if c.deploymentStatSeriesAppsVerifier != nil {
		return c.deploymentStatSeriesAppsVerifier(res)
	}
	return c.deploymentStatSeriesAppsError
}

func (c ContextResponderMock) SendShowDeploymentStatsOK(res *app.SimpleDeploymentStatsSingle, ctx *app.ShowDeploymentStatsDeploymentsContext) error {
	if c.deploymentStatsVerifier != nil {
		return c.deploymentStatsVerifier(res)
	}
	return c.deploymentStatsError
}

func (c ContextResponderMock) SendSpaceOK(res *app.SimpleSpaceSingle, ctx *app.ShowSpaceDeploymentsContext) error {
	if c.deploymentSpaceVerifier != nil {
		return c.deploymentSpaceVerifier(res)
	}
	return c.deploymentSpaceError
}

func (c ContextResponderMock) SendEnvironmentsOK(res *app.SimpleEnvironmentList, ctx *app.ShowSpaceEnvironmentsDeploymentsContext) error {
	if c.deploymentsEnvironmentsVerifier != nil {
		return c.deploymentsEnvironmentsVerifier(res)
	}
	return c.deploymentsEnvironmentsError
}

func (c ContextResponderMock) SendSetDeploymentOK(res []byte, ctx *app.SetDeploymentDeploymentsContext) error {
	if c.deploymentsSetDeploymentVerifier != nil {
		return c.deploymentsSetDeploymentVerifier(res)
	}
	return c.deploymentsSetDeploymentError
}

func (c ContextResponderMock) SendDeleteDeploymentOK(res []byte, ctx *app.DeleteDeploymentDeploymentsContext) error {
	if c.deploymentsDeleteDeploymentVerifier != nil {
		return c.deploymentsDeleteDeploymentVerifier(res)
	}
	return c.deploymentsDeleteDeploymentError
}

func TestSetDeployment(t *testing.T) {
	testCases := []struct {
		testName         string
		deployCtx        *app.SetDeploymentDeploymentsContext
		clientGetter     controller.ClientGetter
		contextResponder controller.ContextResponder
		shouldError      bool
	}{
		{
			testName:    "Nil pod count should fail early with an error",
			deployCtx:   &app.SetDeploymentDeploymentsContext{},
			shouldError: true,
		},
		{
			testName: "Failure to get the kube client fails with an error",
			deployCtx: &app.SetDeploymentDeploymentsContext{
				PodCount: new(int),
			},
			clientGetter: ClientGetterMock{
				getKubeClientError: errors.New("some-error"),
			},
			shouldError: true,
		},
		{
			testName: "Unable to get OSIO space",
			deployCtx: &app.SetDeploymentDeploymentsContext{
				PodCount: new(int),
			},
			clientGetter: ClientGetterMock{
				osioClient: OSIOClientMock{
					spaceError: errors.New("no-space"),
				},
			},
			shouldError: true,
		},
		{
			testName: "Cannot write response",
			deployCtx: &app.SetDeploymentDeploymentsContext{
				Context: ContextMock{
					valueResult: "someValue",
				},
				PodCount:   new(int),
				DeployName: "deployName",
				ResponseData: &goa.ResponseData{
					ResponseWriter: ResponseWriterMock{
						header:     map[string][]string{},
						writeError: errors.New("some-error"),
					},
				},
			},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					scaleDeploymentResult: new(int),
				},
				osioClient: OSIOClientMock{
					space: &app.Space{
						Attributes: &app.SpaceAttributes{
							Name: ptr.String("spaceName"),
						},
					},
				},
			},
			contextResponder: ContextResponderMock{
				deploymentsSetDeploymentError: errors.New("some-error"),
			},
			shouldError: true,
		},
		{
			testName: "Can set deployments correctly",
			deployCtx: &app.SetDeploymentDeploymentsContext{
				Context: ContextMock{
					valueResult: "someValue",
				},
				PodCount:   new(int),
				DeployName: "deployName",
				ResponseData: &goa.ResponseData{
					ResponseWriter: ResponseWriterMock{
						header:     map[string][]string{},
						writeValue: 0,
					},
				},
			},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					scaleDeploymentResult: new(int),
				},
				osioClient: OSIOClientMock{
					space: &app.Space{
						Attributes: &app.SpaceAttributes{
							Name: ptr.String("spaceName"),
						},
					},
				},
			},
			contextResponder: ContextResponderMock{
				deploymentsSetDeploymentVerifier: func(res []byte) error {
					if len(res) == 0 {
						return nil
					} else {
						return errors.New("expected data to be empty")
					}
				},
			},
			shouldError: false,
		},
	}

	for _, testCase := range testCases {
		deploymentsController := &controller.DeploymentsController{
			ClientGetter:     testCase.clientGetter,
			ContextResponder: testCase.contextResponder,
		}

		err := deploymentsController.SetDeployment(testCase.deployCtx)
		if testCase.shouldError {
			assert.NotNil(t, err, testCase.testName)
		} else {
			assert.Nil(t, err, testCase.testName)
		}
	}
}

func TestShowDeploymentStatSeries(t *testing.T) {
	startTime := ptr.Float64(1.0)
	endTime := ptr.Float64(0.0)

	testCases := []struct {
		testName         string
		deployCtx        *app.ShowDeploymentStatSeriesDeploymentsContext
		clientGetter     controller.ClientGetter
		contextResponder controller.ContextResponder
		shouldError      bool
	}{
		{
			testName: "Does not allow bad timestamps",
			deployCtx: &app.ShowDeploymentStatSeriesDeploymentsContext{
				Start: startTime,
				End:   endTime,
			},
			shouldError: true,
		},
		{
			testName:  "Errors if getting the KubeClient fails",
			deployCtx: &app.ShowDeploymentStatSeriesDeploymentsContext{},
			clientGetter: ClientGetterMock{
				getKubeClientError: errors.New("some-error"),
			},
			shouldError: true,
		},
		{
			testName:  "Errors if the OSIO client cannot get the space",
			deployCtx: &app.ShowDeploymentStatSeriesDeploymentsContext{},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					scaleDeploymentResult: new(int),
				},
				osioClient: OSIOClientMock{
					spaceError: errors.New("space-error"),
				},
			},
			shouldError: true,
		},
		{
			testName:  "Getting the stat series fails with an error",
			deployCtx: &app.ShowDeploymentStatSeriesDeploymentsContext{},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					simpleStatSeriesError: errors.New("some-error"),
				},
				osioClient: OSIOClientMock{
					space: &app.Space{
						Attributes: &app.SpaceAttributes{
							Name: ptr.String("spaceName"),
						},
					},
				},
			},
			shouldError: true,
		},
		{
			testName:  "Getting the stat series fails if the returned struct is nil",
			deployCtx: &app.ShowDeploymentStatSeriesDeploymentsContext{},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					simpleStatSeries: nil,
				},
				osioClient: OSIOClientMock{
					space: &app.Space{
						Attributes: &app.SpaceAttributes{
							Name: ptr.String("spaceName"),
						},
					},
				},
			},
			shouldError: true,
		},
		{
			testName: "Cannot write response",
			deployCtx: &app.ShowDeploymentStatSeriesDeploymentsContext{
				Context: ContextMock{},
				ResponseData: &goa.ResponseData{
					Service: &goa.Service{},
					ResponseWriter: ResponseWriterMock{
						header: map[string][]string{},
					},
				},
			},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					simpleStatSeries: &app.SimpleDeploymentStatSeries{},
				},
				osioClient: OSIOClientMock{
					space: &app.Space{
						Attributes: &app.SpaceAttributes{
							Name: ptr.String("spaceName"),
						},
					},
				},
			},
			contextResponder: ContextResponderMock{
				deploymentStatSeriesAppsError: errors.New("some-error"),
			},
			shouldError: true,
		},
		{
			testName: "Successful response writing for deployment stat series",
			deployCtx: &app.ShowDeploymentStatSeriesDeploymentsContext{
				ResponseData: &goa.ResponseData{
					Service: &goa.Service{
						Encoder: &goa.HTTPEncoder{},
					},
					ResponseWriter: ResponseWriterMock{
						header: map[string][]string{},
					},
				},
			},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					simpleStatSeries: &app.SimpleDeploymentStatSeries{
						Start: startTime,
						End:   endTime,
					},
				},
				osioClient: OSIOClientMock{
					space: &app.Space{
						Attributes: &app.SpaceAttributes{
							Name: ptr.String("spaceName"),
						},
					},
				},
			},
			contextResponder: ContextResponderMock{
				deploymentStatSeriesAppsVerifier: func(res *app.SimpleDeploymentStatSeriesSingle) error {
					if res.Data.Start == startTime && res.Data.End == endTime {
						return nil
					} else {
						return errors.New("expected mock object/data was not returned")
					}
				},
			},
			shouldError: false,
		},
	}

	for _, testCase := range testCases {
		deploymentsController := &controller.DeploymentsController{
			ClientGetter:     testCase.clientGetter,
			ContextResponder: testCase.contextResponder,
		}

		err := deploymentsController.ShowDeploymentStatSeries(testCase.deployCtx)
		if testCase.shouldError {
			assert.NotNil(t, err, testCase.testName)
		} else {
			assert.Nil(t, err, testCase.testName)
		}
	}
}

func TestShowDeploymentStats(t *testing.T) {
	mockTime := ptr.Float64(1.5)
	mockValue := ptr.Float64(4.2)

	testCases := []struct {
		testName         string
		deployCtx        *app.ShowDeploymentStatsDeploymentsContext
		clientGetter     controller.ClientGetter
		contextResponder controller.ContextResponder
		shouldError      bool
	}{
		{
			testName:  "Errors if getting the KubeClient fails",
			deployCtx: &app.ShowDeploymentStatsDeploymentsContext{},
			clientGetter: ClientGetterMock{
				getKubeClientError: errors.New("some-error"),
			},
			shouldError: true,
		},
		{
			testName: "Getting the stat series fails with an error",
			deployCtx: &app.ShowDeploymentStatsDeploymentsContext{
				AppName:    "appName",
				DeployName: "deployName",
			},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					deploymentStatsError: errors.New("some-error"),
				},
				osioClient: OSIOClientMock{
					space: &app.Space{
						Attributes: &app.SpaceAttributes{
							Name: ptr.String("spaceName"),
						},
					},
				},
			},
			shouldError: true,
		},
		{
			testName: "Getting the stat series fails with an error when sending http OK",
			deployCtx: &app.ShowDeploymentStatsDeploymentsContext{
				AppName:    "appName",
				DeployName: "deployName",
			},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					deploymentStats: &app.SimpleDeploymentStats{},
				},
				osioClient: OSIOClientMock{
					space: &app.Space{
						Attributes: &app.SpaceAttributes{
							Name: ptr.String("spaceName"),
						},
					},
				},
			},
			contextResponder: ContextResponderMock{
				deploymentStatsError: errors.New("some-error"),
			},
			shouldError: true,
		},
		{
			testName: "Successful response writing for deployment stats",
			deployCtx: &app.ShowDeploymentStatsDeploymentsContext{
				AppName:    "appName",
				DeployName: "deployName",
			},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					deploymentStats: &app.SimpleDeploymentStats{
						Attributes: &app.SimpleDeploymentStatsAttributes{
							Cores: &app.TimedNumberTuple{
								Time:  mockTime,
								Value: mockValue,
							},
						},
					},
				},
				osioClient: OSIOClientMock{
					space: &app.Space{
						Attributes: &app.SpaceAttributes{
							Name: ptr.String("spaceName"),
						},
					},
				},
			},
			contextResponder: ContextResponderMock{
				deploymentStatsVerifier: func(res *app.SimpleDeploymentStatsSingle) error {
					if res.Data.Attributes.Cores.Value == mockValue && res.Data.Attributes.Cores.Time == mockTime {
						return nil
					} else {
						return errors.New("expected mock object/data was not returned")
					}
				},
			},
			shouldError: false,
		},
	}

	for _, testCase := range testCases {
		deploymentsController := &controller.DeploymentsController{
			ClientGetter:     testCase.clientGetter,
			ContextResponder: testCase.contextResponder,
		}

		err := deploymentsController.ShowDeploymentStats(testCase.deployCtx)
		if testCase.shouldError {
			assert.NotNil(t, err, testCase.testName)
		} else {
			assert.Nil(t, err, testCase.testName)
		}
	}
}

func TestShowSpace(t *testing.T) {
	mockSpaceName := "mockSpaceName"

	testCases := []struct {
		testName         string
		deployCtx        *app.ShowSpaceDeploymentsContext
		clientGetter     controller.ClientGetter
		contextResponder controller.ContextResponder
		shouldError      bool
	}{
		{
			testName:  "Errors if getting the KubeClient fails",
			deployCtx: &app.ShowSpaceDeploymentsContext{},
			clientGetter: ClientGetterMock{
				getKubeClientError: errors.New("some-error"),
			},
			shouldError: true,
		},
		{
			testName: "Getting the name of the space from kube fails with an error",
			deployCtx: &app.ShowSpaceDeploymentsContext{
				SpaceID: uuid.Nil,
			},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					deploymentStatsError: errors.New("some-error"),
				},
				osioClient: OSIOClientMock{
					spaceError: errors.New("some-error"),
				},
			},
			shouldError: true,
		},
		{
			testName: "Getting the space fails with an error",
			deployCtx: &app.ShowSpaceDeploymentsContext{
				SpaceID: uuid.Nil,
			},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					simpleSpaceError: errors.New("some-error"),
				},
				osioClient: OSIOClientMock{
					space: &app.Space{
						Attributes: &app.SpaceAttributes{
							Name: ptr.String("spaceName"),
						},
					},
				},
			},
			shouldError: true,
		},
		{
			testName: "Sending space by context fails",
			deployCtx: &app.ShowSpaceDeploymentsContext{
				SpaceID: uuid.Nil,
			},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					simpleSpace: &app.SimpleSpace{},
				},
				osioClient: OSIOClientMock{
					space: &app.Space{
						Attributes: &app.SpaceAttributes{
							Name: ptr.String("spaceName"),
						},
					},
				},
			},
			contextResponder: ContextResponderMock{
				deploymentSpaceError: errors.New("some-error"),
			},
			shouldError: true,
		},
		{
			testName: "Sending space by context succeeds",
			deployCtx: &app.ShowSpaceDeploymentsContext{
				SpaceID: uuid.Nil,
			},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					simpleSpace: &app.SimpleSpace{
						Attributes: &app.SimpleSpaceAttributes{
							Name: mockSpaceName,
						},
					},
				},
				osioClient: OSIOClientMock{
					space: &app.Space{
						Attributes: &app.SpaceAttributes{
							Name: ptr.String("spaceName"),
						},
					},
				},
			},
			contextResponder: ContextResponderMock{
				deploymentSpaceVerifier: func(res *app.SimpleSpaceSingle) error {
					if res.Data.Attributes.Name == mockSpaceName {
						return nil
					} else {
						return errors.New("expected mock object/data was not returned")
					}
				},
			},
			shouldError: false,
		},
	}

	for _, testCase := range testCases {
		deploymentsController := &controller.DeploymentsController{
			ClientGetter:     testCase.clientGetter,
			ContextResponder: testCase.contextResponder,
		}

		err := deploymentsController.ShowSpace(testCase.deployCtx)
		if testCase.shouldError {
			assert.NotNil(t, err, testCase.testName)
		} else {
			assert.Nil(t, err, testCase.testName)
		}
	}
}

func TestShowSpaceEnvironments(t *testing.T) {
	mockEnv := new(app.SimpleEnvironment)

	testCases := []struct {
		testName         string
		deployCtx        *app.ShowSpaceEnvironmentsDeploymentsContext
		clientGetter     controller.ClientGetter
		contextResponder controller.ContextResponder
		shouldError      bool
	}{
		{
			testName:  "Errors if getting the KubeClient fails",
			deployCtx: &app.ShowSpaceEnvironmentsDeploymentsContext{},
			clientGetter: ClientGetterMock{
				getKubeClientError: errors.New("some-error"),
			},
			shouldError: true,
		},
		{
			testName: "Getting environments fails",
			deployCtx: &app.ShowSpaceEnvironmentsDeploymentsContext{
				SpaceID: uuid.Nil,
			},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					environmentsError: errors.New("some-error"),
				},
			},
			shouldError: true,
		},
		{
			testName: "Getting environments as nil returns an error",
			deployCtx: &app.ShowSpaceEnvironmentsDeploymentsContext{
				SpaceID: uuid.Nil,
			},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					environments: nil,
				},
			},
			shouldError: true,
		},
		{
			testName: "Sending by context with failure is an error",
			deployCtx: &app.ShowSpaceEnvironmentsDeploymentsContext{
				SpaceID: uuid.Nil,
			},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					environments: []*app.SimpleEnvironment{},
				},
			},
			contextResponder: ContextResponderMock{
				deploymentsEnvironmentsError: errors.New("some-error"),
			},
			shouldError: true,
		},
		{
			testName: "Sending by context sucessfully causes no errors",
			deployCtx: &app.ShowSpaceEnvironmentsDeploymentsContext{
				SpaceID: uuid.Nil,
			},
			clientGetter: ClientGetterMock{
				kubeClientInterface: KubeClientMock{
					environments: []*app.SimpleEnvironment{
						mockEnv,
					},
				},
			},
			contextResponder: ContextResponderMock{
				deploymentsEnvironmentsVerifier: func(res *app.SimpleEnvironmentList) error {
					if len(res.Data) == 1 && res.Data[0] == mockEnv {
						return nil
					} else {
						return errors.New("expected mock object/data was not returned")
					}
				},
			},
			shouldError: false,
		},
	}

	for _, testCase := range testCases {
		deploymentsController := &controller.DeploymentsController{
			ClientGetter:     testCase.clientGetter,
			ContextResponder: testCase.contextResponder,
		}

		err := deploymentsController.ShowSpaceEnvironments(testCase.deployCtx)
		if testCase.shouldError {
			assert.NotNil(t, err, testCase.testName)
		} else {
			assert.Nil(t, err, testCase.testName)
		}
	}
}
