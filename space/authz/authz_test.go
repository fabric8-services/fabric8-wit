package authz_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/fabric8-services/fabric8-wit/auth"
	witerrors "github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	. "github.com/fabric8-services/fabric8-wit/space/authz"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	testsuite "github.com/fabric8-services/fabric8-wit/test/suite"
	"github.com/fabric8-services/fabric8-wit/test/token"

	errs "github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestAuthz(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	suite.Run(t, new(TestAuthzSuite))
}

type TestAuthzSuite struct {
	testsuite.UnitTestSuite
	authzService *AuthzRoleService
	doer         *testsupport.DummyHttpDoer
	authzConfig  *auth.ServiceConfiguration
}

func (s *TestAuthzSuite) SetupSuite() {
	s.UnitTestSuite.SetupSuite()
	s.authzService = NewAuthzService(&authURLConfig{authURL: "https://some.auth.io"})
	doer := testsupport.NewDummyHttpDoer()
	s.authzService.Doer = doer
	s.doer = doer
}

func (s *TestAuthzSuite) TestFailsIfNoTokenInContext() {
	_, err := s.authzService.Authorize(context.Background(), uuid.NewV4().String())
	require.Error(s.T(), err)
	require.IsType(s.T(), witerrors.UnauthorizedError{}, err)
}

func (s *TestAuthzSuite) TestDefaultDoer() {
	c := &authURLConfig{}
	as := NewAuthzService(c)
	assert.Equal(s.T(), c, as.Config)
	assert.Equal(s.T(), rest.DefaultHttpDoer(), as.Doer)
}

func (s *TestAuthzSuite) TestDisabledAuthorization() {
	as := NewAuthzService(&authURLConfig{disabled: true})
	assert.False(s.T(), as.Configuration().IsAuthorizationEnabled())
	ctx, _, _, _ := token.ContextWithTokenAndRequestID(s.T())
	ok, err := as.Authorize(ctx, "")
	require.NoError(s.T(), err)
	assert.True(s.T(), ok)
}

func (s *TestAuthzSuite) TestInvalidAuthURLFails() {
	as := NewAuthzService(&authURLConfig{authURL: "% %"})
	ctx, _, _, _ := token.ContextWithTokenAndRequestID(s.T())
	_, err := as.Authorize(ctx, "")
	require.Error(s.T(), err)
	assert.Equal(s.T(), "parse % %/api/resources//roles: invalid URL escape \"% %\"", err.Error())
}

func (s *TestAuthzSuite) TestAuthorizeOK() {
	ctx, identityID, tokenString, reqID := token.ContextWithTokenAndRequestID(s.T())

	// One admin
	responsePayload := fmt.Sprintf("{\"data\":[{\"role_name\":\"admin\",\"assignee_id\":\"%s\"}]}", identityID.String())
	s.checkAuthorize(ctx, tokenString, reqID, responsePayload, true)

	// One contributor
	responsePayload = fmt.Sprintf("{\"data\":[{\"role_name\":\"contributor\",\"assignee_id\":\"%s\"}]}", identityID.String())
	s.checkAuthorize(ctx, tokenString, reqID, responsePayload, true)

	// Multiple users and the expected one is among them
	responsePayload = fmt.Sprintf("{\"data\":[{\"role_name\":\"admin\",\"assignee_id\":\"%s\"},{\"role_name\":\"contributor\",\"assignee_id\":\"%s\"}]}", uuid.NewV4().String(), identityID.String())
	s.checkAuthorize(ctx, tokenString, reqID, responsePayload, true)

	// Viewer is forbidden
	responsePayload = fmt.Sprintf("{\"data\":[{\"role_name\":\"viewer\",\"assignee_id\":\"%s\"}]}", identityID.String())
	s.checkAuthorize(ctx, tokenString, reqID, responsePayload, false)

	// If the user is not among the roles then it's forbidden too
	responsePayload = fmt.Sprintf("{\"data\":[{\"role_name\":\"admin\",\"assignee_id\":\"%s\"},{\"role_name\":\"contributor\",\"assignee_id\":\"%s\"}]}", uuid.NewV4().String(), uuid.NewV4().String())
	s.checkAuthorize(ctx, tokenString, reqID, responsePayload, false)
}

func (s *TestAuthzSuite) TestAuthorizeFailIfUserCantListRoles() {
	// Forbidden if the user doesn't have permissions to view the roles
	ctx, _, _, _ := token.ContextWithTokenAndRequestID(s.T())

	// Set up expected request
	s.doer.Client.Error = nil
	body := ioutil.NopCloser(bytes.NewReader([]byte{}))
	s.doer.Client.Response = &http.Response{Body: body, StatusCode: http.StatusForbidden, Status: "403"}

	// Forbidden
	ok, err := s.authzService.Authorize(ctx, uuid.NewV4().String())
	require.NoError(s.T(), err)
	assert.False(s.T(), ok)
}

func (s *TestAuthzSuite) TestAuthorizeFailIfInvalidResponse() {
	// Forbidden if the user doesn't have permissions to view the roles
	ctx, _, _, _ := token.ContextWithTokenAndRequestID(s.T())

	// Set up expected request
	s.doer.Client.Error = nil
	body := ioutil.NopCloser(bytes.NewReader([]byte("{[")))
	s.doer.Client.Response = &http.Response{Body: body, StatusCode: http.StatusOK, Status: "200"}

	_, err := s.authzService.Authorize(ctx, uuid.NewV4().String())
	require.Error(s.T(), err)
	assert.IsType(s.T(), witerrors.InternalError{}, errs.Cause(err))
}

func (s *TestAuthzSuite) TestAuthorizeFailWithError() {
	ctx, _, _, _ := token.ContextWithTokenAndRequestID(s.T())
	spaceID := uuid.NewV4().String()

	// Fail if client returned an error
	s.doer.Client.Error = errors.New("oopsie woopsie")
	_, err := s.authzService.Authorize(ctx, spaceID)
	require.Error(s.T(), err)
	assert.Equal(s.T(), "oopsie woopsie", err.Error())

	// Fail if client returned unexpected status
	body := ioutil.NopCloser(bytes.NewReader([]byte{}))
	s.doer.Client.Response = &http.Response{Body: body, StatusCode: http.StatusInternalServerError, Status: "500"}
	s.doer.Client.Error = nil
	_, err = s.authzService.Authorize(ctx, spaceID)
	testsupport.AssertError(s.T(), err, witerrors.InternalError{}, "unable to get space roles. Response status: 500. Response body: ")
}

func (s *TestAuthzSuite) checkAuthorize(ctx context.Context, token, reqID, responsePayload string, expectedAllowed bool) {
	spaceID := uuid.NewV4().String()

	// Set up expected request
	s.doer.Client.Error = nil

	body := ioutil.NopCloser(bytes.NewReader([]byte(responsePayload)))
	s.doer.Client.Response = &http.Response{Body: body, StatusCode: http.StatusOK}

	s.doer.Client.AssertRequest = func(req *http.Request) {
		assert.Equal(s.T(), "GET", req.Method)
		assert.Equal(s.T(), fmt.Sprintf("https://some.auth.io/api/resources/%s/roles", spaceID), req.URL.String())
		assert.Equal(s.T(), "Bearer "+token, req.Header.Get("Authorization"))
		assert.Equal(s.T(), reqID, req.Header.Get("X-Request-Id"))
	}

	// OK
	ok, err := s.authzService.Authorize(ctx, spaceID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), expectedAllowed, ok)
}

type authURLConfig struct {
	authURL  string
	disabled bool
}

func (c *authURLConfig) GetAuthServiceURL() string {
	return c.authURL
}

func (c *authURLConfig) GetAuthShortServiceHostName() string {
	return ""
}

func (c *authURLConfig) IsAuthorizationEnabled() bool {
	return !c.disabled
}
