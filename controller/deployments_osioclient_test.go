package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/ptr"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

const deploymentsOsioTestFilePath = "test-files/deployments_osio/"

// Structs and interfaces for mocking/testing
type MockContext struct {
	context.Context
}

type JsonResponseReader struct {
	jsonBytes *bytes.Buffer
}

func (r *JsonResponseReader) ReadResponse(resp *http.Response) ([]byte, error) {
	return r.jsonBytes.Bytes(), nil
}

type MockResponseBodyReader struct {
	io.ReadCloser
}

func (m MockResponseBodyReader) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (m MockResponseBodyReader) Close() error {
	return nil
}

type MockWitClient struct {
	SpaceHttpResponse            *http.Response
	SpaceHttpResponseError       error
	UserServiceHttpResponse      *http.Response
	UserServiceHttpResponseError error
}

func (m *MockWitClient) ShowSpace(ctx context.Context, path string, ifModifiedSince *string, ifNoneMatch *string) (*http.Response, error) {
	return m.SpaceHttpResponse, m.SpaceHttpResponseError
}

func (m *MockWitClient) ShowUserService(ctx context.Context, path string) (*http.Response, error) {
	return m.UserServiceHttpResponse, m.UserServiceHttpResponseError
}

// Unit tests
func TestGetUserServicesWithShowUserServiceError(t *testing.T) {
	mockWitClient := &MockWitClient{
		UserServiceHttpResponseError: errors.New("error"),
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, &controller.IOResponseReader{})

	_, err := mockOSIOClient.GetUserServices(&MockContext{})
	require.Error(t, err)
}

func TestGetUserServicesBadStatusCodes(t *testing.T) {
	testCases := []struct {
		statusCode  int
		shouldBeNil bool
	}{
		{http.StatusMovedPermanently, false},
		{http.StatusNotFound, true},
		{http.StatusInternalServerError, false},
	}

	for _, testCase := range testCases {
		mockResponse := &http.Response{
			Body:       &MockResponseBodyReader{},
			StatusCode: testCase.statusCode,
		}
		mockWitClient := &MockWitClient{
			UserServiceHttpResponse: mockResponse,
		}
		mockOSIOClient := controller.CreateOSIOClient(mockWitClient, &controller.IOResponseReader{})

		userService, err := mockOSIOClient.GetUserServices(&MockContext{})
		if testCase.shouldBeNil {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
		require.Nil(t, userService)
	}
}

func TestGetUserServiceWithMalformedJSON(t *testing.T) {
	jsonReader := &JsonResponseReader{
		jsonBytes: bytes.NewBuffer([]byte(`{`)),
	}
	mockResponse := &http.Response{
		Body:       &MockResponseBodyReader{},
		StatusCode: http.StatusOK,
	}
	mockWitClient := &MockWitClient{
		UserServiceHttpResponse: mockResponse,
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, jsonReader)

	_, err := mockOSIOClient.GetUserServices(&MockContext{})
	require.Error(t, err)
}

func getTestUserService(t *testing.T) (app.UserServiceSingle, []byte) {
	id := uuid.NewV4()
	expected := app.UserServiceSingle{
		Data: &app.UserService{
			Attributes: &app.UserServiceAttributes{
				CreatedAt: nil,
				Namespaces: []*app.NamespaceAttributes{
					{
						ClusterAppDomain:  ptr.String("http://openshift.io"),
						ClusterConsoleURL: ptr.String("http://openshift.io/url"),
						ClusterMetricsURL: ptr.String("http://openshift.io/url"),
						ClusterURL:        ptr.String("http://openshift.io/url"),
						CreatedAt:         nil,
						Name:              ptr.String("namespaceName"),
						State:             ptr.String("someState"),
						Type:              ptr.String(controller.APIWorkItemTypes),
						UpdatedAt:         nil,
						Version:           ptr.String("1.2.3"),
					},
				},
			},
			ID:    &id,
			Links: nil,
			Type:  controller.APIWorkItemTypes,
		},
	}
	jsonBytes, err := json.MarshalIndent(expected, "", "  ")
	require.NoError(t, err)
	return expected, jsonBytes
}

func TestUserServiceWithProperJSON(t *testing.T) {
	goldenFilePath := deploymentsOsioTestFilePath + "user-service.res.payload.golden.json"
	_, jsonByteData := getTestUserService(t)
	jsonReader := &JsonResponseReader{
		jsonBytes: bytes.NewBuffer(jsonByteData),
	}
	mockResponse := &http.Response{
		Body:       &MockResponseBodyReader{},
		StatusCode: http.StatusOK,
	}
	mockWitClient := &MockWitClient{
		UserServiceHttpResponse: mockResponse,
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, jsonReader)

	userService, err := mockOSIOClient.GetUserServices(&MockContext{})
	require.NoError(t, err)
	require.NotNil(t, userService)
	compareWithGoldenAgnostic(t, goldenFilePath, userService)
}

func TestGetSpaceByIDWithShowSpaceError(t *testing.T) {
	mockContext := &MockContext{}
	mockWitClient := &MockWitClient{
		SpaceHttpResponseError: errors.New("error"),
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, &controller.IOResponseReader{})

	_, err := mockOSIOClient.GetSpaceByID(mockContext, uuid.Nil)
	require.Error(t, err)
}

func TestGetSpaceByIDBadStatusCode(t *testing.T) {
	testCases := []struct {
		statusCode  int
		shouldBeNil bool
	}{
		{http.StatusMovedPermanently, false},
		{http.StatusNotFound, true},
		{http.StatusInternalServerError, false},
	}

	for _, testCase := range testCases {
		mockResponse := &http.Response{
			Body:       &MockResponseBodyReader{},
			StatusCode: testCase.statusCode,
		}
		mockWitClient := &MockWitClient{
			SpaceHttpResponse: mockResponse,
		}
		mockOSIOClient := controller.CreateOSIOClient(mockWitClient, &controller.IOResponseReader{})

		userService, err := mockOSIOClient.GetSpaceByID(&MockContext{}, uuid.Nil)
		if testCase.shouldBeNil {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
		require.Nil(t, userService)
	}
}

func TestGetSpaceByIDWithMalformedJSON(t *testing.T) {
	jsonReader := &JsonResponseReader{
		jsonBytes: bytes.NewBuffer([]byte(`{`)),
	}
	mockResponse := &http.Response{
		Body:       &MockResponseBodyReader{},
		StatusCode: http.StatusOK,
	}
	mockWitClient := &MockWitClient{
		SpaceHttpResponse: mockResponse,
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, jsonReader)

	_, err := mockOSIOClient.GetSpaceByID(&MockContext{}, uuid.Nil)
	require.Error(t, err)
}

func getTestSpace(t *testing.T) (app.SpaceSingle, []byte) {
	id := uuid.NewV4()
	expected := app.SpaceSingle{
		Data: &app.Space{
			ID: &id,
			Attributes: &app.SpaceAttributes{
				Name:        ptr.String("spaceName"),
				Description: ptr.String("space description"),
				Version:     ptr.Int(42),
				CreatedAt:   nil,
				UpdatedAt:   nil,
			},
			Links: &app.GenericLinksForSpace{
				Self:    ptr.String("https://api.openshift.io/api/spaces/" + id.String()),
				Related: ptr.String("https://api.openshift.io/api/spaces/" + id.String()),
				Backlog: &app.BacklogGenericLink{
					Self: ptr.String("https://api.openshift.io/api/spaces/" + id.String() + "/backlog"),
				},
				Filters:           ptr.String("https://api.openshift.io/api/filters"),
				Workitemlinktypes: ptr.String("https://api.openshift.io/api/spaces/" + id.String() + "/workitemlinktypes"),
				Workitemtypes:     ptr.String("https://api.openshift.io/api/spaces/" + id.String() + "/workitemtypes"),
			},
			Type: controller.APIWorkItemTypes,
		},
	}
	jsonBytes, err := json.MarshalIndent(expected, "", "  ")
	require.NoError(t, err)
	return expected, jsonBytes
}

func TestGetSpaceByIDWithProperJSON(t *testing.T) {
	goldenFilePath := deploymentsOsioTestFilePath + "space-id.res.payload.golden.json"
	_, jsonData := getTestSpace(t)
	jsonReader := &JsonResponseReader{
		jsonBytes: bytes.NewBuffer(jsonData),
	}
	mockContext := &MockContext{}
	mockResponse := &http.Response{
		Body:       &MockResponseBodyReader{},
		StatusCode: http.StatusOK,
	}
	mockWitClient := &MockWitClient{
		SpaceHttpResponse: mockResponse,
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, jsonReader)

	space, err := mockOSIOClient.GetSpaceByID(mockContext, uuid.Nil)
	require.NoError(t, err)
	compareWithGoldenAgnostic(t, goldenFilePath, space)
}

func TestGetNamespaceByTypeErrorFromUserServices(t *testing.T) {
	mockWitClient := &MockWitClient{
		UserServiceHttpResponseError: errors.New("error"),
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, &controller.IOResponseReader{})

	namespaceAttributes, err := mockOSIOClient.GetNamespaceByType(&MockContext{}, nil, "namespace")
	require.Error(t, err)
	require.Nil(t, namespaceAttributes)
}

func TestGetNamespaceByTypeNoMatch(t *testing.T) {
	mockOSIOClient := controller.CreateOSIOClient(&MockWitClient{}, &controller.IOResponseReader{})
	mockUserService := &app.UserService{
		Attributes: &app.UserServiceAttributes{
			Namespaces: make([]*app.NamespaceAttributes, 0),
		},
	}

	namespaceAttributes, err := mockOSIOClient.GetNamespaceByType(&MockContext{}, mockUserService, "namespace")
	require.NoError(t, err)
	require.Nil(t, namespaceAttributes)
}

func getTestNamespaceAttributes(t *testing.T) (app.NamespaceAttributes, []byte) {
	expected := app.NamespaceAttributes{
		ClusterAppDomain:  ptr.String("http://openshift.io"),
		ClusterConsoleURL: ptr.String("http://openshift.io/url"),
		ClusterMetricsURL: ptr.String("http://openshift.io/url"),
		ClusterURL:        ptr.String("http://openshift.io/url"),
		CreatedAt:         nil,
		Name:              ptr.String("namespaceName"),
		State:             ptr.String("someState"),
		Type:              ptr.String(controller.APIWorkItemTypes),
		UpdatedAt:         nil,
		Version:           ptr.String("1.2.3"),
	}

	jsonBytes, err := json.MarshalIndent(expected, "", "  ")
	require.NoError(t, err)
	return expected, jsonBytes
}

func TestGetNamespaceByTypeMatchNamespace(t *testing.T) {
	goldenFilePath := deploymentsOsioTestFilePath + "namespace-by-type.res.payload.golden.json"
	mockNamespace, jsonBytes := getTestNamespaceAttributes(t)
	jsonProvider := &JsonResponseReader{
		jsonBytes: bytes.NewBuffer(jsonBytes),
	}
	mockResponse := &http.Response{
		Body:       &MockResponseBodyReader{},
		StatusCode: http.StatusOK,
	}
	mockWitClient := &MockWitClient{
		UserServiceHttpResponse: mockResponse,
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, jsonProvider)
	mockUserService := &app.UserService{
		Attributes: &app.UserServiceAttributes{
			Namespaces: []*app.NamespaceAttributes{&mockNamespace},
		},
	}
	namespaceAttributes, err := mockOSIOClient.GetNamespaceByType(&MockContext{}, mockUserService, *mockNamespace.Type)
	require.NoError(t, err)
	compareWithGoldenAgnostic(t, goldenFilePath, namespaceAttributes)
}

func TestGetNamespaceByTypeMatchNamespaceWithDiscoveredUserService(t *testing.T) {
	goldenFilePath := deploymentsOsioTestFilePath + "namespace-discovered-user-service.res.payload.golden.json"
	mockUserServiceSingle, jsonByteData := getTestUserService(t)
	mockUserService := mockUserServiceSingle.Data
	jsonProvider := &JsonResponseReader{
		jsonBytes: bytes.NewBuffer(jsonByteData),
	}
	mockResponse := &http.Response{
		Body:       &MockResponseBodyReader{},
		StatusCode: http.StatusOK,
	}
	mockWitClient := &MockWitClient{
		UserServiceHttpResponse: mockResponse,
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, jsonProvider)
	namespaceType := *mockUserService.Attributes.Namespaces[0].Type
	namespaceAttributes, err := mockOSIOClient.GetNamespaceByType(&MockContext{}, nil, namespaceType)
	require.NoError(t, err)
	compareWithGoldenAgnostic(t, goldenFilePath, namespaceAttributes)
}
