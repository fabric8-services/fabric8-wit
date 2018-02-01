package controller_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

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
		SpaceHttpResponse:            nil,
		SpaceHttpResponseError:       nil,
		UserServiceHttpResponse:      nil,
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
			SpaceHttpResponse:            nil,
			SpaceHttpResponseError:       nil,
			UserServiceHttpResponse:      mockResponse,
			UserServiceHttpResponseError: nil,
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
		SpaceHttpResponse:            nil,
		SpaceHttpResponseError:       nil,
		UserServiceHttpResponse:      mockResponse,
		UserServiceHttpResponseError: nil,
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, jsonReader)

	_, err := mockOSIOClient.GetUserServices(&MockContext{})
	require.Error(t, err)
}

func TestUserServiceWithProperJSON(t *testing.T) {
	jsonReader := &JsonResponseReader{
		jsonBytes: bytes.NewBuffer([]byte(`
			{
				"data": {
					"attributes": {
						"bio": "",
						"cluster": "https://some-random-cluster-dot.com",
						"company": "",
						"contextInformation": {
							"experimentalFeatures": {
								"enabled": true
							},
							"recentContexts": [
								{
									"space": "00000000-0000-0000-0000-000000000000",
									"user": "11111111-1111-1111-1111-111111111111"
								},
								{
									"space": null,
									"user": "22222222-2222-2222-2222-222222222222"
								}
							],
							"recentSpaces": ["33333333-3333-3333-3333-333333333333"]
						},
						"created-at": "2017-11-03T16:39:45.566361Z",
						"email": "email@somerandomemailhere.email",
						"emailPrivate": true,
						"emailVerified": true,
						"fullName": "Dr. Legit Email",
						"identityID": "44444444-4444-4444-4444-444444444444",
						"imageURL": "https://www.gravatar.com/avatar/00000000000000000000000000000000.jpg",
						"providerType": "kc",
						"registrationCompleted": true,
						"updated-at": "2018-01-16T19:43:41.859203Z",
						"url": "",
						"userID": "55555555-5555-5555-5555-555555555555",
						"username": "username"
					},
					"id": "66666666-6666-6666-6666-666666666666",
					"links": {
						"related": "https://auth.openshift.io/api/users/77777777-7777-7777-7777-777777777777",
						"self": "https://auth.openshift.io/api/users/88888888-8888-8888-8888-888888888888"
					},
					"type": "identities"
				}
			}`)),
	}
	mockResponse := &http.Response{
		Body:       &MockResponseBodyReader{},
		StatusCode: http.StatusOK,
	}
	mockWitClient := &MockWitClient{
		SpaceHttpResponse:            nil,
		SpaceHttpResponseError:       nil,
		UserServiceHttpResponse:      mockResponse,
		UserServiceHttpResponseError: nil,
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, jsonReader)

	userService, err := mockOSIOClient.GetUserServices(&MockContext{})
	require.NoError(t, err)
	require.NotNil(t, userService)
	require.Equal(t, "https://auth.openshift.io/api/users/77777777-7777-7777-7777-777777777777", *userService.Links.Related)
	require.Equal(t, "https://auth.openshift.io/api/users/88888888-8888-8888-8888-888888888888", *userService.Links.Self)
	require.Equal(t, "66666666-6666-6666-6666-666666666666", userService.ID.String())
	require.Equal(t, "identities", userService.Type)
}

func TestGetSpaceByIDWithShowSpaceError(t *testing.T) {
	mockContext := &MockContext{}
	mockWitClient := &MockWitClient{
		SpaceHttpResponse:            nil,
		SpaceHttpResponseError:       errors.New("error"),
		UserServiceHttpResponse:      nil,
		UserServiceHttpResponseError: nil,
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
			SpaceHttpResponse:            mockResponse,
			SpaceHttpResponseError:       nil,
			UserServiceHttpResponse:      nil,
			UserServiceHttpResponseError: nil,
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
		SpaceHttpResponse:            mockResponse,
		SpaceHttpResponseError:       nil,
		UserServiceHttpResponse:      nil,
		UserServiceHttpResponseError: nil,
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, jsonReader)

	_, err := mockOSIOClient.GetSpaceByID(&MockContext{}, uuid.Nil)
	require.Error(t, err)
}

func TestGetSpaceByIDWithProperJSON(t *testing.T) {
	jsonReader := &JsonResponseReader{
		jsonBytes: bytes.NewBuffer([]byte(`{
			"data": {
				"attributes": {
					"created-at": "2017-12-01T18:34:06.393371Z",
					"description": "",
					"name": "yet_another",
					"updated-at": "2017-12-01T18:34:06.393371Z",
					"version": 0
				},
				"id": "00000000-0000-0000-0000-000000000000",
				"links": {
					"backlog": {
						"meta": {
							"totalCount": 0
						},
						"self": "https://api.openshift.io/api/spaces/00000000-0000-0000-0000-000000000000/backlog"
					},
					"filters": "https://api.openshift.io/api/filters",
					"related": "https://api.openshift.io/api/spaces/00000000-0000-0000-0000-000000000000",
					"self": "https://api.openshift.io/api/spaces/00000000-0000-0000-0000-000000000000",
					"workitemlinktypes": "https://api.openshift.io/api/spaces/00000000-0000-0000-0000-000000000000/workitemlinktypes",
					"workitemtypes": "https://api.openshift.io/api/spaces/00000000-0000-0000-0000-000000000000/workitemtypes"
				},
				"relationships": {
					"areas": {
						"links": {
							"related": "https://api.openshift.io/api/spaces/00000000-0000-0000-0000-000000000000/areas"
						}
					}
				},
				"type": "spaces"
			}
		}`)),
	}
	mockContext := &MockContext{}
	mockResponse := &http.Response{
		Body:       &MockResponseBodyReader{},
		StatusCode: http.StatusOK,
	}
	mockWitClient := &MockWitClient{
		SpaceHttpResponse:            mockResponse,
		SpaceHttpResponseError:       nil,
		UserServiceHttpResponse:      nil,
		UserServiceHttpResponseError: nil,
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, jsonReader)

	space, err := mockOSIOClient.GetSpaceByID(mockContext, uuid.Nil)
	require.NoError(t, err)
	require.NotNil(t, space)
	require.Equal(t, "https://api.openshift.io/api/spaces/00000000-0000-0000-0000-000000000000", *space.Links.Self)
	require.Equal(t, "https://api.openshift.io/api/spaces/00000000-0000-0000-0000-000000000000", *space.Links.Related)
	require.Equal(t, "https://api.openshift.io/api/spaces/00000000-0000-0000-0000-000000000000/backlog", *space.Links.Backlog.Self)
	require.Equal(t, 0, space.Links.Backlog.Meta.TotalCount)
	require.Equal(t, "https://api.openshift.io/api/filters", *space.Links.Filters)
	require.Equal(t, "https://api.openshift.io/api/spaces/00000000-0000-0000-0000-000000000000/workitemlinktypes", *space.Links.Workitemlinktypes)
	require.Equal(t, "https://api.openshift.io/api/spaces/00000000-0000-0000-0000-000000000000/workitemtypes", *space.Links.Workitemtypes)
	require.Equal(t, "00000000-0000-0000-0000-000000000000", space.ID.String())
	require.Equal(t, "spaces", space.Type)
	require.Equal(t, "", *space.Attributes.Description)
	require.Equal(t, "yet_another", *space.Attributes.Name)
}

func TestGetNamespaceByTypeErrorFromUserServices(t *testing.T) {
	mockWitClient := &MockWitClient{
		SpaceHttpResponse:            nil,
		SpaceHttpResponseError:       nil,
		UserServiceHttpResponse:      nil,
		UserServiceHttpResponseError: errors.New("error"),
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, &controller.IOResponseReader{})

	namespaceAttributes, err := mockOSIOClient.GetNamespaceByType(&MockContext{}, nil, "namespace")
	require.Error(t, err)
	require.Nil(t, namespaceAttributes)
}

func TestGetNamespaceByTypeNoMatch(t *testing.T) {
	mockWitClient := &MockWitClient{
		SpaceHttpResponse:            nil,
		SpaceHttpResponseError:       nil,
		UserServiceHttpResponse:      nil,
		UserServiceHttpResponseError: nil,
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, &controller.IOResponseReader{})
	mockUserService := &app.UserService{
		Attributes: &app.UserServiceAttributes{
			Namespaces: make([]*app.NamespaceAttributes, 0),
			CreatedAt:  nil,
		},
	}

	namespaceAttributes, err := mockOSIOClient.GetNamespaceByType(&MockContext{}, mockUserService, "namespace")
	require.NoError(t, err)
	require.Nil(t, namespaceAttributes)
}

func TestGetNamespaceByTypeMatchNamespace(t *testing.T) {
	NAMESPACE_TYPE := "desiredType"
	jsonProvider := &JsonResponseReader{
		jsonBytes: bytes.NewBuffer([]byte(`{
			"data": {
				"attributes": {
					"bio": "",
					"cluster": "https://some-random-cluster-dot.com",
					"company": "",
					"contextInformation": {
						"experimentalFeatures": {
							"enabled": true
						},
						"recentContexts": [
							{
								"space": "00000000-0000-0000-0000-000000000000",
								"user": "11111111-1111-1111-1111-111111111111"
							},
							{
								"space": null,
								"user": "22222222-2222-2222-2222-222222222222"
							}
						],
						"recentSpaces": ["33333333-3333-3333-3333-333333333333"]
					},
					"created-at": "2017-11-03T16:39:45.566361Z",
					"email": "email@somerandomemailhere.email",
					"emailPrivate": true,
					"emailVerified": true,
					"fullName": "Dr. Legit Email",
					"identityID": "44444444-4444-4444-4444-444444444444",
					"imageURL": "https://www.gravatar.com/avatar/00000000000000000000000000000000.jpg",
					"providerType": "kc",
					"registrationCompleted": true,
					"updated-at": "2018-01-16T19:43:41.859203Z",
					"url": "",
					"userID": "55555555-5555-5555-5555-555555555555",
					"username": "username"
				},
				"id": "66666666-6666-6666-6666-666666666666",
				"links": {
					"related": "https://auth.openshift.io/api/users/77777777-7777-7777-7777-777777777777",
					"self": "https://auth.openshift.io/api/users/88888888-8888-8888-8888-888888888888"
				},
				"type": "desiredType"
			}
		}`)),
	}
	mockNamespace := &app.NamespaceAttributes{
		Type: &NAMESPACE_TYPE,
	}
	mockResponse := &http.Response{
		Body:       &MockResponseBodyReader{},
		StatusCode: http.StatusOK,
	}
	mockWitClient := &MockWitClient{
		SpaceHttpResponse:            nil,
		SpaceHttpResponseError:       nil,
		UserServiceHttpResponse:      mockResponse,
		UserServiceHttpResponseError: nil,
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, jsonProvider)
	mockUserService := &app.UserService{
		Attributes: &app.UserServiceAttributes{
			Namespaces: []*app.NamespaceAttributes{mockNamespace},
			CreatedAt:  nil,
		},
	}
	namespaceAttributes, err := mockOSIOClient.GetNamespaceByType(&MockContext{}, mockUserService, NAMESPACE_TYPE)
	require.NoError(t, err)
	require.Equal(t, mockNamespace, namespaceAttributes)
}

func TestGetNamespaceByTypeMatchNamespaceWithDiscoveredUserService(t *testing.T) {
	NAMESPACE_TYPE := "desiredType"
	jsonProvider := &JsonResponseReader{
		jsonBytes: bytes.NewBuffer([]byte(`{
			"data": {
				"attributes": {
					"created-at": "2017-11-03T16:39:45.566361Z",
					"namespaces": [
						{
							"name": "some-name",
							"state": "some-state",
							"type": "desiredType",
							"version": "123"
						}
					]
				},
				"id": "66666666-6666-6666-6666-666666666666",
				"links": {
					"related": "https://auth.openshift.io/api/users/77777777-7777-7777-7777-777777777777",
					"self": "https://auth.openshift.io/api/users/88888888-8888-8888-8888-888888888888"
				},
				"type": "someType"
			}
		}`)),
	}
	mockResponse := &http.Response{
		Body:       &MockResponseBodyReader{},
		StatusCode: http.StatusOK,
	}
	mockWitClient := &MockWitClient{
		SpaceHttpResponse:            nil,
		SpaceHttpResponseError:       nil,
		UserServiceHttpResponse:      mockResponse,
		UserServiceHttpResponseError: nil,
	}
	mockOSIOClient := controller.CreateOSIOClient(mockWitClient, jsonProvider)
	namespaceAttributes, err := mockOSIOClient.GetNamespaceByType(&MockContext{}, nil, NAMESPACE_TYPE)
	require.NoError(t, err)
	require.Equal(t, NAMESPACE_TYPE, *namespaceAttributes.Type)
	require.Equal(t, "some-name", *namespaceAttributes.Name)
	require.Equal(t, "some-state", *namespaceAttributes.State)
	require.Equal(t, "123", *namespaceAttributes.Version)
}
