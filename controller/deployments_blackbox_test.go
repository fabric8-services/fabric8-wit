package controller_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gojuno/minimock"
	"golang.org/x/net/websocket"
	"k8s.io/client-go/tools/cache"

	"github.com/goadesign/goa"
	"github.com/goadesign/goa/goatest"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/controller"
	witerrors "github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/kubernetes"
	"github.com/fabric8-services/fabric8-wit/space"
	testcontroller "github.com/fabric8-services/fabric8-wit/test/controller"
	testk8s "github.com/fabric8-services/fabric8-wit/test/kubernetes"
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
			testName:   "Delete Bad Request",
			deleteFunc: test.DeleteDeploymentDeploymentsBadRequest,
			spaceUUID:  uuidStr,
			deploymentsTestErrors: deploymentsTestErrors{
				deleteDeploymentError: witerrors.NewBadParameterErrorFromString("TEST"), // Return expected error from DeleteDeployment
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
			deleteFunc: test.DeleteDeploymentDeploymentsInternalServerError,
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

func TestShowSpace(t *testing.T) {
	// given
	spaceName := "mySpace"
	clientGetterMock := testcontroller.NewClientGetterMock(t)
	svc, ctrl, err := createDeploymentsController()
	require.NoError(t, err)
	ctrl.ClientGetter = clientGetterMock

	t.Run("ok", func(t *testing.T) {
		// given
		kubeClientMock := testk8s.NewKubeClientMock(t)
		kubeClientMock.GetSpaceFunc = func(spaceName string) (*app.SimpleSpace, error) {
			return &app.SimpleSpace{
				Type: "space",
				Attributes: &app.SimpleSpaceAttributes{
					Name:         spaceName,
					Applications: []*app.SimpleApp{},
				},
			}, nil
		}
		kubeClientMock.CloseFunc = func() {}
		clientGetterMock.GetKubeClientFunc = func(ctx context.Context) (kubernetes.KubeClientInterface, error) {
			return kubeClientMock, nil
		}
		clientGetterMock.GetAndCheckOSIOClientFunc = func(ctx context.Context) (controller.OpenshiftIOClient, error) {
			return createOSIOClientMock(t, spaceName), nil
		}
		// when
		_, result := test.ShowSpaceDeploymentsOK(t, context.Background(), svc, ctrl, space.SystemSpace)
		// then
		assert.Equal(t, space.SystemSpace, result.Data.ID, "space ID should be %s", space.SystemSpace.String())
		assert.NotNil(t, result.Data.Attributes, "space attributes must be non-nil")
		assert.Equal(t, spaceName, result.Data.Attributes.Name, "space ID should be %s", space.SystemSpace.String())
		// verify that the Close method was called
		assert.Equal(t, uint64(1), kubeClientMock.CloseCounter)
	})

	t.Run("failure", func(t *testing.T) {

		t.Run("kube client init failure", func(t *testing.T) {
			// given
			clientGetterMock.GetKubeClientFunc = func(p context.Context) (r kubernetes.KubeClientInterface, r1 error) {
				return nil, fmt.Errorf("failure")
			}
			// when/then
			test.ShowSpaceDeploymentsInternalServerError(t, context.Background(), svc, ctrl, space.SystemSpace)
		})

		t.Run("get space bad request", func(t *testing.T) {
			// given
			kubeClientMock := testk8s.NewKubeClientMock(t)
			kubeClientMock.GetSpaceFunc = func(spaceName string) (*app.SimpleSpace, error) {
				return nil, witerrors.NewBadParameterErrorFromString("TEST")
			}
			kubeClientMock.CloseFunc = func() {}
			clientGetterMock.GetKubeClientFunc = func(ctx context.Context) (kubernetes.KubeClientInterface, error) {
				return kubeClientMock, nil
			}
			clientGetterMock.GetAndCheckOSIOClientFunc = func(ctx context.Context) (controller.OpenshiftIOClient, error) {
				return createOSIOClientMock(t, spaceName), nil
			}
			// when
			test.ShowSpaceDeploymentsBadRequest(t, context.Background(), svc, ctrl, space.SystemSpace)
			// then verify that the Close method was called
			assert.Equal(t, uint64(1), kubeClientMock.CloseCounter)
		})
	})
}

func TestSetDeployment(t *testing.T) {
	// given
	spaceName := "mySpace"
	appName := "myApp"
	envName := "run"
	newPodNum := 5
	oldPodNum := 3

	clientGetterMock := testcontroller.NewClientGetterMock(t)
	svc, ctrl, err := createDeploymentsController()
	require.NoError(t, err)
	ctrl.ClientGetter = clientGetterMock

	t.Run("ok", func(t *testing.T) {
		// given
		kubeClientMock := testk8s.NewKubeClientMock(t)
		defer kubeClientMock.Finish()
		kubeClientMock.ScaleDeploymentMock.Expect(spaceName, appName, envName, newPodNum).Return(&oldPodNum, nil)
		kubeClientMock.CloseFunc = func() {}
		clientGetterMock.GetKubeClientFunc = func(ctx context.Context) (kubernetes.KubeClientInterface, error) {
			return kubeClientMock, nil
		}
		clientGetterMock.GetAndCheckOSIOClientFunc = func(ctx context.Context) (controller.OpenshiftIOClient, error) {
			return createOSIOClientMock(t, spaceName), nil
		}
		// when
		test.SetDeploymentDeploymentsOK(t, context.Background(), svc, ctrl, space.SystemSpace,
			appName, envName, &newPodNum)

		// then
		// verify that the Close method was called
		assert.Equal(t, uint64(1), kubeClientMock.CloseCounter)
	})

	t.Run("failure", func(t *testing.T) {

		t.Run("kube client init failure", func(t *testing.T) {
			// given
			kubeClientMock := testk8s.NewKubeClientMock(t)
			defer kubeClientMock.Finish()
			clientGetterMock.GetKubeClientFunc = func(p context.Context) (r kubernetes.KubeClientInterface, r1 error) {
				return nil, fmt.Errorf("failure")
			}
			// when/then
			test.SetDeploymentDeploymentsInternalServerError(t, context.Background(), svc, ctrl, space.SystemSpace,
				appName, envName, &newPodNum)
		})

		t.Run("scale deployment bad request", func(t *testing.T) {
			// given
			kubeClientMock := testk8s.NewKubeClientMock(t)
			defer kubeClientMock.Finish()
			kubeClientMock.ScaleDeploymentMock.Expect(spaceName, appName, envName, newPodNum).Return(nil,
				witerrors.NewBadParameterErrorFromString("TEST"))
			kubeClientMock.CloseFunc = func() {}
			clientGetterMock.GetKubeClientFunc = func(ctx context.Context) (kubernetes.KubeClientInterface, error) {
				return kubeClientMock, nil
			}

			clientGetterMock.GetAndCheckOSIOClientFunc = func(ctx context.Context) (controller.OpenshiftIOClient, error) {
				return createOSIOClientMock(t, spaceName), nil
			}
			// when
			test.SetDeploymentDeploymentsBadRequest(t, context.Background(), svc, ctrl, space.SystemSpace,
				appName, envName, &newPodNum)
			// then verify that the Close method was called
			assert.Equal(t, uint64(1), kubeClientMock.CloseCounter)
		})
	})
}

func TestShowDeploymentStats(t *testing.T) {
	// given
	spaceName := "mySpace"
	appName := "myApp"
	envName := "run"
	startTimeMilli := float64(1527796723000)
	startTime := convertToTime(int64(startTimeMilli))

	stats := &app.SimpleDeploymentStats{
		Type:       "deploymentstats",
		Attributes: &app.SimpleDeploymentStatsAttributes{},
	}

	clientGetterMock := testcontroller.NewClientGetterMock(t)
	svc, ctrl, err := createDeploymentsController()
	require.NoError(t, err)
	ctrl.ClientGetter = clientGetterMock

	t.Run("ok", func(t *testing.T) {
		// given
		kubeClientMock := testk8s.NewKubeClientMock(t)
		defer kubeClientMock.Finish()
		kubeClientMock.GetDeploymentStatsMock.Expect(spaceName, appName, envName, startTime).Return(stats, nil)
		kubeClientMock.CloseFunc = func() {}
		clientGetterMock.GetKubeClientFunc = func(ctx context.Context) (kubernetes.KubeClientInterface, error) {
			return kubeClientMock, nil
		}
		clientGetterMock.GetAndCheckOSIOClientFunc = func(ctx context.Context) (controller.OpenshiftIOClient, error) {
			return createOSIOClientMock(t, spaceName), nil
		}
		// when
		_, result := test.ShowDeploymentStatsDeploymentsOK(t, context.Background(), svc, ctrl, space.SystemSpace,
			appName, envName, &startTimeMilli)

		// then
		assert.Equal(t, stats, result.Data, "deployment stats do not match")
		// verify that the Close method was called
		assert.Equal(t, uint64(1), kubeClientMock.CloseCounter)
	})

	t.Run("failure", func(t *testing.T) {

		t.Run("kube client init failure", func(t *testing.T) {
			// given
			kubeClientMock := testk8s.NewKubeClientMock(t)
			defer kubeClientMock.Finish()
			clientGetterMock.GetKubeClientFunc = func(p context.Context) (r kubernetes.KubeClientInterface, r1 error) {
				return nil, fmt.Errorf("failure")
			}
			// when/then
			test.ShowDeploymentStatsDeploymentsInternalServerError(t, context.Background(), svc, ctrl, space.SystemSpace,
				appName, envName, &startTimeMilli)
		})

		t.Run("get deployment stats bad request", func(t *testing.T) {
			// given
			kubeClientMock := testk8s.NewKubeClientMock(t)
			defer kubeClientMock.Finish()
			kubeClientMock.GetDeploymentStatsMock.Expect(spaceName, appName, envName, startTime).Return(stats,
				witerrors.NewBadParameterErrorFromString("TEST"))
			kubeClientMock.CloseFunc = func() {}
			clientGetterMock.GetKubeClientFunc = func(ctx context.Context) (kubernetes.KubeClientInterface, error) {
				return kubeClientMock, nil
			}

			clientGetterMock.GetAndCheckOSIOClientFunc = func(ctx context.Context) (controller.OpenshiftIOClient, error) {
				return createOSIOClientMock(t, spaceName), nil
			}
			// when
			test.ShowDeploymentStatsDeploymentsBadRequest(t, context.Background(), svc, ctrl, space.SystemSpace,
				appName, envName, &startTimeMilli)
			// then verify that the Close method was called
			assert.Equal(t, uint64(1), kubeClientMock.CloseCounter)
		})
	})
}

func TestShowDeploymentStatSeries(t *testing.T) {
	// given
	spaceName := "mySpace"
	appName := "myApp"
	envName := "run"
	startTimeMilli := float64(1527796723000)
	startTime := convertToTime(int64(startTimeMilli))
	endTimeMilli := float64(1527796753000)
	endTime := convertToTime(int64(endTimeMilli))
	limit := 5

	stats := &app.SimpleDeploymentStatSeries{
		Start: &startTimeMilli,
		End:   &endTimeMilli,
	}

	clientGetterMock := testcontroller.NewClientGetterMock(t)
	svc, ctrl, err := createDeploymentsController()
	require.NoError(t, err)
	ctrl.ClientGetter = clientGetterMock

	t.Run("ok", func(t *testing.T) {
		// given
		kubeClientMock := testk8s.NewKubeClientMock(t)
		defer kubeClientMock.Finish()
		kubeClientMock.GetDeploymentStatSeriesMock.Expect(spaceName, appName, envName, startTime, endTime, limit).Return(stats, nil)
		kubeClientMock.CloseFunc = func() {}
		clientGetterMock.GetKubeClientFunc = func(ctx context.Context) (kubernetes.KubeClientInterface, error) {
			return kubeClientMock, nil
		}
		clientGetterMock.GetAndCheckOSIOClientFunc = func(ctx context.Context) (controller.OpenshiftIOClient, error) {
			return createOSIOClientMock(t, spaceName), nil
		}
		// when
		_, result := test.ShowDeploymentStatSeriesDeploymentsOK(t, context.Background(), svc, ctrl, space.SystemSpace,
			appName, envName, &endTimeMilli, &limit, &startTimeMilli)

		// then
		assert.Equal(t, stats, result.Data, "deployment stats do not match")
		// verify that the Close method was called
		assert.Equal(t, uint64(1), kubeClientMock.CloseCounter)
	})

	t.Run("failure", func(t *testing.T) {

		t.Run("kube client init failure", func(t *testing.T) {
			// given
			kubeClientMock := testk8s.NewKubeClientMock(t)
			defer kubeClientMock.Finish()
			clientGetterMock.GetKubeClientFunc = func(p context.Context) (r kubernetes.KubeClientInterface, r1 error) {
				return nil, fmt.Errorf("failure")
			}
			// when/then
			test.ShowDeploymentStatSeriesDeploymentsInternalServerError(t, context.Background(), svc, ctrl, space.SystemSpace,
				appName, envName, &endTimeMilli, &limit, &startTimeMilli)

		})

		t.Run("get deployment stats bad request", func(t *testing.T) {
			// given
			kubeClientMock := testk8s.NewKubeClientMock(t)
			defer kubeClientMock.Finish()
			kubeClientMock.GetDeploymentStatSeriesMock.Expect(spaceName, appName, envName, startTime, endTime,
				limit).Return(stats, witerrors.NewBadParameterErrorFromString("TEST"))
			kubeClientMock.CloseFunc = func() {}
			clientGetterMock.GetKubeClientFunc = func(ctx context.Context) (kubernetes.KubeClientInterface, error) {
				return kubeClientMock, nil
			}

			clientGetterMock.GetAndCheckOSIOClientFunc = func(ctx context.Context) (controller.OpenshiftIOClient, error) {
				return createOSIOClientMock(t, spaceName), nil
			}
			// when
			test.ShowDeploymentStatSeriesDeploymentsBadRequest(t, context.Background(), svc, ctrl, space.SystemSpace,
				appName, envName, &endTimeMilli, &limit, &startTimeMilli)
			// then verify that the Close method was called
			assert.Equal(t, uint64(1), kubeClientMock.CloseCounter)
		})
	})
}

func convertToTime(unixMillis int64) time.Time {
	return time.Unix(0, unixMillis*int64(time.Millisecond))
}
func TestShowSpaceEnvironments(t *testing.T) {
	// given
	clientGetterMock := testcontroller.NewClientGetterMock(t)
	svc, ctrl, err := createDeploymentsController()
	require.NoError(t, err)
	ctrl.ClientGetter = clientGetterMock

	t.Run("ok", func(t *testing.T) {
		// given
		envName := "foo"
		kubeClientMock := testk8s.NewKubeClientMock(t)
		kubeClientMock.GetEnvironmentsFunc = func() ([]*app.SimpleEnvironment, error) {
			return []*app.SimpleEnvironment{
				{
					ID:   "foo",
					Type: "environment",
					Attributes: &app.SimpleEnvironmentAttributes{
						Name: &envName,
					},
				},
			}, nil
		}
		kubeClientMock.CloseFunc = func() {}
		clientGetterMock.GetKubeClientFunc = func(p context.Context) (kubernetes.KubeClientInterface, error) {
			return kubeClientMock, nil
		}
		osioClientMock := testcontroller.NewOSIOClientMock(t)
		clientGetterMock.GetAndCheckOSIOClientFunc = func(p context.Context) (controller.OpenshiftIOClient, error) {
			return osioClientMock, nil
		}
		// when
		test.ShowSpaceEnvironmentsDeploymentsOK(t, context.Background(), svc, ctrl, space.SystemSpace)
		// then verify that the Close method was called
		assert.Equal(t, uint64(1), kubeClientMock.CloseCounter)
	})

	t.Run("failure", func(t *testing.T) {

		t.Run("kube client init failure", func(t *testing.T) {
			// given
			clientGetterMock.GetKubeClientFunc = func(p context.Context) (r kubernetes.KubeClientInterface, r1 error) {
				return nil, fmt.Errorf("failure")
			}
			// when/then
			test.ShowSpaceEnvironmentsDeploymentsInternalServerError(t, context.Background(), svc, ctrl, space.SystemSpace)
		})

		t.Run("get environments bad request", func(t *testing.T) {
			// given
			kubeClientMock := testk8s.NewKubeClientMock(t)
			kubeClientMock.GetEnvironmentsFunc = func() ([]*app.SimpleEnvironment, error) {
				return nil, witerrors.NewBadParameterErrorFromString("TEST")
			}
			kubeClientMock.CloseFunc = func() {}
			clientGetterMock.GetKubeClientFunc = func(p context.Context) (kubernetes.KubeClientInterface, error) {
				return kubeClientMock, nil
			}
			osioClientMock := testcontroller.NewOSIOClientMock(t)
			clientGetterMock.GetAndCheckOSIOClientFunc = func(p context.Context) (controller.OpenshiftIOClient, error) {
				return osioClientMock, nil
			}
			// when
			test.ShowSpaceEnvironmentsDeploymentsBadRequest(t, context.Background(), svc, ctrl, space.SystemSpace)
			// then verify that the Close method was called
			assert.Equal(t, uint64(1), kubeClientMock.CloseCounter)
		})
	})
}

func TestShowAllEnvironments(t *testing.T) {
	// given
	clientGetterMock := testcontroller.NewClientGetterMock(t)
	svc, ctrl, err := createDeploymentsController()
	require.NoError(t, err)
	ctrl.ClientGetter = clientGetterMock

	t.Run("ok", func(t *testing.T) {
		// given
		envName1 := "foo1"
		envName2 := "foo2"
		kubeClientMock := testk8s.NewKubeClientMock(t)
		kubeClientMock.GetEnvironmentsFunc = func() ([]*app.SimpleEnvironment, error) {
			return []*app.SimpleEnvironment{
				{
					ID:   "foo1",
					Type: "environment",
					Attributes: &app.SimpleEnvironmentAttributes{
						Name: &envName1,
					},
				},
				{
					ID:   "foo2",
					Type: "environment",
					Attributes: &app.SimpleEnvironmentAttributes{
						Name: &envName2,
					},
				},
			}, nil
		}
		kubeClientMock.CloseFunc = func() {}
		clientGetterMock.GetKubeClientFunc = func(p context.Context) (kubernetes.KubeClientInterface, error) {
			return kubeClientMock, nil
		}
		osioClientMock := testcontroller.NewOSIOClientMock(t)
		clientGetterMock.GetAndCheckOSIOClientFunc = func(p context.Context) (controller.OpenshiftIOClient, error) {
			return osioClientMock, nil
		}
		// when
		test.ShowAllEnvironmentsDeploymentsOK(t, context.Background(), svc, ctrl)
		// then verify that the Close method was called
		assert.Equal(t, uint64(1), kubeClientMock.CloseCounter)

	})

	t.Run("failure", func(t *testing.T) {

		t.Run("kube client init failure", func(t *testing.T) {
			// given
			clientGetterMock.GetKubeClientFunc = func(p context.Context) (r kubernetes.KubeClientInterface, r1 error) {
				return nil, fmt.Errorf("failure")
			}
			// when/then
			test.ShowAllEnvironmentsDeploymentsInternalServerError(t, context.Background(), svc, ctrl)
		})

		t.Run("get all environments bad request", func(t *testing.T) {
			// given
			kubeClientMock := testk8s.NewKubeClientMock(t)
			kubeClientMock.GetEnvironmentsFunc = func() ([]*app.SimpleEnvironment, error) {
				return nil, witerrors.NewBadParameterErrorFromString("TEST")
			}
			kubeClientMock.CloseFunc = func() {}
			clientGetterMock.GetKubeClientFunc = func(p context.Context) (kubernetes.KubeClientInterface, error) {
				return kubeClientMock, nil
			}
			osioClientMock := testcontroller.NewOSIOClientMock(t)
			clientGetterMock.GetAndCheckOSIOClientFunc = func(p context.Context) (controller.OpenshiftIOClient, error) {
				return osioClientMock, nil
			}
			// when
			test.ShowAllEnvironmentsDeploymentsBadRequest(t, context.Background(), svc, ctrl)
			// then verify that the Close method was called
			assert.Equal(t, uint64(1), kubeClientMock.CloseCounter)
		})
	})

}

func createOSIOClientMock(t minimock.Tester, spaceName string) *testcontroller.OSIOClientMock {
	osioClientMock := testcontroller.NewOSIOClientMock(t)
	osioClientMock.GetSpaceByIDFunc = func(ctx context.Context, spaceID uuid.UUID) (*app.Space, error) {
		return &app.Space{
			ID: &spaceID,
			Attributes: &app.SpaceAttributes{
				Name: &spaceName,
			},
		}, nil
	}
	return osioClientMock
}

type testWatchItem struct {
	Name   string
	Object testInnerItem
}

type testInnerItem struct {
	Number int
}

func TestWatchEnvironmentEvents(t *testing.T) {
	clientGetterMock := testcontroller.NewClientGetterMock(t)
	svc, ctrl, err := createDeploymentsController()
	require.NoError(t, err)
	ctrl.ClientGetter = clientGetterMock

	t.Run("ok", func(t *testing.T) {
		testItems := []testWatchItem{
			testWatchItem{
				Name: "one",
				Object: testInnerItem{
					Number: 2,
				},
			},
			testWatchItem{
				Name: "two",
				Object: testInnerItem{
					Number: 3,
				},
			},
		}

		mockKeyFunc := func(obj interface{}) (string, error) {
			if v, ok := obj.(testWatchItem); ok {
				return v.Name, nil
			}
			return "default", nil
		}

		kubeClientMock := testk8s.NewKubeClientMock(t)
		kubeClientMock.WatchEventsInNamespaceFunc = func(p string) (r *cache.FIFO, r1 chan struct{}) {
			fmt.Println("Watch events function called")
			store := cache.NewFIFO(mockKeyFunc)
			for _, item := range testItems {
				store.Add(item)
			}

			return store, make(chan struct{})
		}
		kubeClientMock.CloseFunc = func() {}
		clientGetterMock.GetKubeClientFunc = func(p context.Context) (kubernetes.KubeClientInterface, error) {
			return kubeClientMock, nil
		}
		osioClientMock := testcontroller.NewOSIOClientMock(t)
		clientGetterMock.GetAndCheckOSIOClientFunc = func(p context.Context) (controller.OpenshiftIOClient, error) {
			return osioClientMock, nil
		}

		conn := WatchEnvironmentEventsDeploymentsOK(t, context.Background(), svc, ctrl, space.SystemSpace)

		var buf []byte
		for _, item := range testItems {
			// buffer 256 is an arbitrary choice that fits the test items
			// Manually unmarshal ws frame. Second element contains length
			// Object is marshaled as JSON and placed in index 2:length+2
			buf = make([]byte, 256)
			conn.Read(buf)
			frameLength := int(buf[1])
			startPos := 2
			endPos := startPos + frameLength
			var m testWatchItem
			err = websocket.JSON.Unmarshal(buf[startPos:endPos], 1, &m)
			assert.Equal(t, m, item)
		}
	})
}

type wsRecorder struct {
	*httptest.ResponseRecorder
	server net.Conn
}

func (r *wsRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	rw := bufio.NewReadWriter(bufio.NewReader(r.server), bufio.NewWriter(r.server))
	return r.server, rw, nil
}

func WatchEnvironmentEventsDeploymentsOK(t goatest.TInterface, ctx context.Context, service *goa.Service, ctrl app.DeploymentsController, spaceID uuid.UUID) net.Conn {
	var (
		logBuf     bytes.Buffer
		respSetter goatest.ResponseSetterFunc = func(r interface{}) {}
	)
	if service == nil {
		service = goatest.Service(&logBuf, respSetter)
	} else {
		logger := log.New(&logBuf, "", log.Ltime)
		service.WithLogger(goa.NewLogger(logger))
		newEncoder := func(io.Writer) goa.Encoder { return respSetter }
		service.Encoder = goa.NewHTTPEncoder()
		service.Encoder.Register(newEncoder, "*/*")
	}

	conn, server := net.Pipe()
	rw := &wsRecorder{
		httptest.NewRecorder(),
		server,
	}
	u := &url.URL{
		Scheme: "ws",
		Path:   fmt.Sprintf("/api/deployments/spaces/%v/environments", spaceID),
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		panic("invalid test " + err.Error())
	}
	req.Header.Add("Sec-Websocket-Version", "13")
	req.Header.Add("Sec-Websocket-Key", "G7YfpwECvn2g+GPiIT9K6A==")
	req.Header.Add("Upgrade", "websocket")
	req.Header.Add("Connection", "Upgrade")
	req.Header.Add("Origin", "https://localhost:8080")

	prms := url.Values{}
	prms["spaceID"] = []string{fmt.Sprintf("%v", spaceID)}
	if ctx == nil {
		ctx = context.Background()
	}
	goaCtx := goa.NewContext(goa.WithAction(ctx, "DeploymentsTest"), rw, req, prms)
	watchEnvirnomentEventsCtx, _err := app.NewWatchEnvironmentEventsDeploymentsContext(goaCtx, req, service)
	if _err != nil {
		panic("invalid test data " + _err.Error())
	}

	go func() {
		_err = ctrl.WatchEnvironmentEvents(watchEnvirnomentEventsCtx)
	}()

	if _err != nil {
		t.Fatalf("controller returned %+v, logs:\n%s", _err, logBuf.String())
	}

	var expected = "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: 0v75TdGGa4rJ+EXs1fpIBirdeG8=\r\n\r\n"
	buf := make([]byte, 256)
	conn.Read(buf)
	actual := strings.Trim(string(buf), "\x00")

	assert.Equal(t, expected, actual)

	return conn
}
