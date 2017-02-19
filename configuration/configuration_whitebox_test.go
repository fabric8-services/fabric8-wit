package configuration

import (
	"net/http"
	"testing"

	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
)

func TestOpenIDConnectPathOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	path := openIDConnectPath("somesufix")
	assert.Equal(t, "auth/realms/demo/protocol/openid-connect/somesufix", path)
}

func TestGetKeycloakURLOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	url, err := getKeycloakURL(req, "somepath")
	assert.Nil(t, err)
	assert.Equal(t, "http://sso.service.domain.org/somepath", url)
}

func TestGetKeycloakURLForTooShortHostFails(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	r := &goa.RequestData{
		Request: &http.Request{Host: "org"},
	}
	_, err := getKeycloakURL(r, "somepath")
	assert.NotNil(t, err)
}

func TestDemoApiAlmightyIoExceptionOK(t *testing.T) {
	// demo.api.almighty.io doesn't follow the service name convention <serviceName>.<domain>
	// The correct name would be something like API.demo.almighty.io which is to be converted to SSO.demo.almighty.io
	// So, it should be treated as an exception

	resource.Require(t, resource.UnitTest)
	t.Parallel()

	r := &goa.RequestData{
		Request: &http.Request{Host: "demo.api.almighty.io"},
	}

	url, err := getKeycloakURL(r, "somepath")
	assert.Nil(t, err)
	assert.Equal(t, "http://sso.demo.almighty.io/somepath", url)
}
