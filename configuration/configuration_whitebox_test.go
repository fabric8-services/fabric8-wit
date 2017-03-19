package configuration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
)

var reqLong *goa.RequestData
var reqShort *goa.RequestData
var config *ConfigurationData

func init() {

	// ensure that the content here is executed only once.
	reqLong = &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	reqShort = &goa.RequestData{
		Request: &http.Request{Host: "api.domain.org"},
	}
	resetConfiguration()
}

func resetConfiguration() {
	var err error
	config, err = GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

func TestOpenIDConnectPathOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	path := config.openIDConnectPath("somesufix")
	assert.Equal(t, "auth/realms/fabric8/protocol/openid-connect/somesufix", path)
}

func TestGetKeycloakURLOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	url, err := config.getKeycloakURL(reqLong, "somepath")
	assert.Nil(t, err)
	assert.Equal(t, "http://sso.service.domain.org/somepath", url)

	url, err = config.getKeycloakURL(reqShort, "somepath2")
	assert.Nil(t, err)
	assert.Equal(t, "http://sso.domain.org/somepath2", url)
}

func TestGetKeycloakURLForTooShortHostFails(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	r := &goa.RequestData{
		Request: &http.Request{Host: "org"},
	}
	_, err := config.getKeycloakURL(r, "somepath")
	assert.NotNil(t, err)
}

func TestDemoApiAlmightyIoExceptionOK(t *testing.T) {
	// demo.api.almighty.io doesn't follow the service name convention <serviceName>.<domain>
	// The correct name would be something like API.demo.almighty.io which is to be converted to sso.prod-preview.openshift.io
	// So, it should be treated as an exception

	resource.Require(t, resource.UnitTest)
	t.Parallel()

	r := &goa.RequestData{
		Request: &http.Request{Host: "sso.prod-preview.openshift.io"},
	}

	url, err := config.getKeycloakURL(r, "somepath3")
	assert.Nil(t, err)
	assert.Equal(t, "http://sso.prod-preview.openshift.io/somepath3", url)
}
