package controller_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/controller"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/goadesign/goa"
)

type FeatureControllerTestConfig struct {
	url string
}

func (c *FeatureControllerTestConfig) GetTogglesServiceURL() string {
	return c.url
}

func SecuredFeaturesController(identity account.Identity, url string) (*goa.Service, *controller.FeaturesController) {
	svc := testsupport.ServiceAsUser("Features-Service", identity)
	return svc, controller.NewFeaturesController(svc, &FeatureControllerTestConfig{
		url: url,
	})

}

func UnsecuredFeaturesController(url string) (*goa.Service, *controller.FeaturesController) {
	svc := goa.New("Features-Service")
	return svc, controller.NewFeaturesController(svc, &FeatureControllerTestConfig{
		url: url,
	})
}

// newTestServer returns a new HTTP server to mock the toggles-service endpoint
// and return very basic responses based on the request URI.
func newTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "feature_ok") {
			fmt.Fprintln(w, "Hello, feature!")
		} else if strings.Contains(r.RequestURI, "feature_not_found") {
			http.Error(w, "feature not found", http.StatusNotFound)
		}
	}))
}

func TestShowFeatures(t *testing.T) {
	// given
	ts := newTestServer()
	defer ts.Close()

	t.Run("anonymous user", func(t *testing.T) {
		// given
		svc, featuresCtrl := UnsecuredFeaturesController(ts.URL)

		t.Run("feature ok", func(t *testing.T) {
			test.ShowFeaturesOK(t, context.Background(), svc, featuresCtrl, "feature_ok")
		})

		t.Run("feature not found", func(t *testing.T) {
			test.ShowFeaturesNotFound(t, context.Background(), svc, featuresCtrl, "feature_not_found")
		})
	})

	t.Run("logged-in user", func(t *testing.T) {
		// given
		svc, featuresCtrl := SecuredFeaturesController(testsupport.TestIdentity, ts.URL)

		t.Run("feature ok", func(t *testing.T) {
			test.ShowFeaturesOK(t, context.Background(), svc, featuresCtrl, "feature_ok")
		})

		t.Run("feature not found", func(t *testing.T) {
			test.ShowFeaturesNotFound(t, context.Background(), svc, featuresCtrl, "feature_not_found")
		})
	})
}
func TestListFeatures(t *testing.T) {
	// given
	ts := newTestServer()
	defer ts.Close()

	t.Run("anonymous user", func(t *testing.T) {
		// given
		svc, featuresCtrl := UnsecuredFeaturesController(ts.URL)

		t.Run("feature ok", func(t *testing.T) {
			test.ListFeaturesOK(t, context.Background(), svc, featuresCtrl)
		})

	})

	t.Run("logged-in user", func(t *testing.T) {
		// given
		svc, featuresCtrl := SecuredFeaturesController(testsupport.TestIdentity, ts.URL)

		t.Run("feature ok", func(t *testing.T) {
			test.ListFeaturesOK(t, context.Background(), svc, featuresCtrl)
		})

	})
}
