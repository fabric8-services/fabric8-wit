package controller_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/resource"
	testjwt "github.com/fabric8-services/fabric8-wit/test/jwt"
	testrecorder "github.com/fabric8-services/fabric8-wit/test/recorder"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/suite"
)

type UserControllerTestSuite struct {
	gormtestsupport.DBTestSuite
	config controller.UserControllerConfiguration
}

type UserControllerConfig struct {
	url string
}

func (t UserControllerConfig) GetAuthServiceURL() string {
	return t.url
}

func (t UserControllerConfig) GetAuthShortServiceHostName() string {
	return ""
}

func (t UserControllerConfig) GetCacheControlUser() string {
	return ""
}

func (t UserControllerConfig) IsAuthorizationEnabled() bool {
	return false
}

func TestUserController(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &UserControllerTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *UserControllerTestSuite) NewSecuredController(options ...configuration.HTTPClientOption) (*goa.Service, *controller.UserController) {
	svc := goa.New("user-controller")
	userCtrl := controller.NewUserController(svc, s.GormDB, s.config, options...)
	return svc, userCtrl
}

func (s *UserControllerTestSuite) NewUnsecuredController(options ...configuration.HTTPClientOption) (*goa.Service, *controller.UserController) {
	svc := goa.New("user-controller")
	userCtrl := controller.NewUserController(svc, s.GormDB, s.config, options...)
	return svc, userCtrl
}

func (s *UserControllerTestSuite) TestListSpaces() {

	s.config = UserControllerConfig{
		url: "https://auth",
	}

	s.T().Run("ok", func(t *testing.T) {

		t.Run("user has no role in any space", func(t *testing.T) {
			// given
			ctx, err := testjwt.NewJWTContext("aa8bffab-c505-40b6-8e87-cd8b0fc1a0c4", "")
			require.NoError(t, err)
			r, err := testrecorder.New("../test/data/auth/list_spaces",
				testrecorder.WithJWTMatcher("../test/jwt/public_key.pem"))
			require.NoError(t, err)
			defer r.Stop()
			svc, userCtrl := s.NewSecuredController(configuration.WithRoundTripper(r.Transport))
			// when
			_, result := test.ListSpacesUserOK(t, ctx, svc, userCtrl)
			// then
			require.Empty(t, result.Data)
		})

		t.Run("user has a role in 1 space", func(t *testing.T) {
			// given
			tf.NewTestFixture(t, s.DB, tf.Spaces(1, func(fxt *tf.TestFixture, idx int) error {
				if idx == 0 {
					id, err := uuid.FromString("6c378ed7-67cf-4e09-b099-c25bf8202617")
					if err != nil {
						return errs.Wrapf(err, "failed to set ID for space in test fixture")
					}
					fxt.Spaces[idx].ID = id
					fxt.Spaces[idx].Name = "space1"
				}
				return nil
			}))

			ctx, err := testjwt.NewJWTContext("bcdd0b29-123d-11e8-a8bc-b69930b94f5c", "")
			require.NoError(t, err)
			r, err := testrecorder.New("../test/data/auth/list_spaces",
				testrecorder.WithJWTMatcher("../test/jwt/public_key.pem"))
			require.NoError(t, err)
			defer r.Stop()
			svc, userCtrl := s.NewSecuredController(configuration.WithRoundTripper(r.Transport))
			// when
			_, result := test.ListSpacesUserOK(t, ctx, svc, userCtrl)
			// then
			require.Len(t, result.Data, 1)
			assert.Equal(t, "space1", result.Data[0].Attributes.Name)
			assert.NotNil(t, result.Data[0].Links.Self)
		})

		t.Run("user has a role in 2 spaces", func(t *testing.T) {
			// given
			tf.NewTestFixture(t, s.DB, tf.Spaces(2, func(fxt *tf.TestFixture, idx int) error {
				if idx == 0 {
					id, err := uuid.FromString("6bfa9182-dc81-4bc1-a694-c2e96ec23d3e")
					if err != nil {
						return errs.Wrapf(err, "failed to set ID for space in test fixture")
					}
					fxt.Spaces[idx].ID = id
					fxt.Spaces[idx].Name = "space1"
				} else if idx == 1 {
					id, err := uuid.FromString("2423d75d-ae5d-4bc5-818b-8e3fa4e2167c")
					if err != nil {
						return errs.Wrapf(err, "failed to set ID for space in test fixture")
					}
					fxt.Spaces[idx].ID = id
					fxt.Spaces[idx].Name = "space2"
				}
				return nil
			}))

			ctx, err := testjwt.NewJWTContext("83fdcae2-634f-4a52-958a-f723cb621700", "")
			require.NoError(t, err)
			r, err := testrecorder.New("../test/data/auth/list_spaces",
				testrecorder.WithJWTMatcher("../test/jwt/public_key.pem"))
			require.NoError(t, err)
			defer r.Stop()
			svc, userCtrl := s.NewSecuredController(configuration.WithRoundTripper(r.Transport))
			// when
			_, result := test.ListSpacesUserOK(t, ctx, svc, userCtrl)
			// then
			compareWithGoldenAgnostic(t, filepath.Join("test-files", "endpoints", "listspaces", "ok.res.payload.golden.json"), result)
		})

	})

	s.T().Run("unauthorized", func(t *testing.T) {

		t.Run("missing token", func(t *testing.T) {
			// given
			ctx := context.Background()
			r, err := testrecorder.New("../test/data/auth/list_spaces",
				testrecorder.WithJWTMatcher("../test/jwt/public_key.pem"))
			require.NoError(t, err)
			defer r.Stop()
			svc, userCtrl := s.NewUnsecuredController(configuration.WithRoundTripper(r.Transport))
			// when/then
			test.ListSpacesUserUnauthorized(t, ctx, svc, userCtrl)
		})
	})

}
