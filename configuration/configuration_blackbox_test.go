package configuration

import (
	"fmt"
	"os"
	"testing"

	"net/http"

	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
)

var req *goa.RequestData

func TestMain(m *testing.M) {
	if err := Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	req = &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	os.Exit(m.Run())
}

func TestGetKeycloakEndpointAuthOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	url, err := GetKeycloakEndpointAuth(req)
	assert.Nil(t, err)
	// In dev mode it's always the defualt value regardless of the request
	assert.Equal(t, "http://sso.demo.almighty.io/auth/realms/demo/protocol/openid-connect/auth", url)
}

func TestGetKeycloakEndpointTokenOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	url, err := GetKeycloakEndpointToken(req)
	assert.Nil(t, err)
	// In dev mode it's always the defualt value regardless of the request
	assert.Equal(t, "http://sso.demo.almighty.io/auth/realms/demo/protocol/openid-connect/token", url)
}

func TestGetKeycloakEndpointUserInfoOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	url, err := GetKeycloakEndpointUserInfo(req)
	assert.Nil(t, err)
	// In dev mode it's always the defualt value regardless of the request
	assert.Equal(t, "http://sso.demo.almighty.io/auth/realms/demo/protocol/openid-connect/userinfo", url)
}
