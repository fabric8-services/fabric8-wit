package controller_test

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/fabric8-services/fabric8-wit/configuration"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var spaceConfiguration *configuration.Registry

type DummyResourceManager struct {
}

func (m *DummyResourceManager) CreateSpace(ctx context.Context, request *http.Request, spaceID string) (*authservice.SpaceResource, error) {
	return &authservice.SpaceResource{Data: &authservice.SpaceResourceData{ResourceID: uuid.NewV4().String(), PermissionID: uuid.NewV4().String(), PolicyID: uuid.NewV4().String()}}, nil
}

func (m *DummyResourceManager) DeleteSpace(ctx context.Context, request *http.Request, spaceID string) error {
	return nil
}

func init() {
	var err error
	spaceConfiguration, err = configuration.Get()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

type SpaceControllerTestSuite struct {
	gormtestsupport.DBTestSuite
	db            *gormapplication.GormDB
	iterationRepo iteration.Repository
	testDir       string
}

func TestSpaceController(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &SpaceControllerTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *SpaceControllerTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.db = gormapplication.NewGormDB(s.DB)
	s.iterationRepo = iteration.NewIterationRepository(s.DB)
	s.testDir = filepath.Join("test-files", "space")
}

func (s *SpaceControllerTestSuite) SecuredController(identity account.Identity) (*goa.Service, *SpaceController) {
	svc := testsupport.ServiceAsUser("Space-Service", identity)
	return svc, NewSpaceController(svc, s.db, spaceConfiguration, &DummyResourceManager{})
}

func (s *SpaceControllerTestSuite) UnSecuredController() (*goa.Service, *SpaceController) {
	svc := goa.New("Space-Service")
	return svc, NewSpaceController(svc, s.db, spaceConfiguration, &DummyResourceManager{})
}

func (s *SpaceControllerTestSuite) SecuredSpaceAreaController(identity account.Identity) (*goa.Service, *SpaceAreasController) {
	svc := testsupport.ServiceAsUser("Area-Service", identity)
	return svc, NewSpaceAreasController(svc, s.db, s.Configuration)
}

func (s *SpaceControllerTestSuite) SecuredSpaceIterationController(identity account.Identity) (*goa.Service, *SpaceIterationsController) {
	svc := testsupport.ServiceAsUser("Iteration-Service", identity)
	return svc, NewSpaceIterationsController(svc, s.db, s.Configuration)
}

func (s *SpaceControllerTestSuite) TestValidateSpaceName() {

	s.T().Run("Fail - length", func(t *testing.T) {
		// given
		p := newCreateSpacePayload(&testsupport.TestOversizedNameObj, nil)
		// when
		err := p.Validate()
		// Validate payload function returns an error
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "length of type.name must be less than or equal to 62 but got")
	})

	s.T().Run("Fail - prefix", func(t *testing.T) {
		// given
		invalidSpaceName := "_TestSpace"
		p := newCreateSpacePayload(&invalidSpaceName, nil)
		// when
		err := p.Validate()
		// Validate payload function returns an error
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "type.name must match the regexp")
	})
}

func (s *SpaceControllerTestSuite) TestCreateSpace() {

	s.T().Run("Fail - unsecure", func(t *testing.T) {
		// given
		p := newCreateSpacePayload(nil, nil)
		svc, ctrl := s.UnSecuredController()
		// when/then
		test.CreateSpaceUnauthorized(t, svc.Context, svc, ctrl, p)
	})

	s.T().Run("ok", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("TestSuccessCreateSpace-")
		p := newCreateSpacePayload(&name, nil)
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		// when
		_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
		// then
		require.NotNil(t, created.Data)
		require.NotNil(t, created.Data.Attributes)
		assert.NotNil(t, created.Data.Attributes.CreatedAt)
		assert.NotNil(t, created.Data.Attributes.UpdatedAt)
		require.NotNil(t, created.Data.Attributes.Name)
		assert.Equal(t, name, *created.Data.Attributes.Name)
		require.NotNil(t, created.Data.Links)
		assert.NotNil(t, created.Data.Links.Self)
	})

	s.T().Run("ok with default area", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("TestSuccessCreateSpaceAndDefaultArea-")
		p := newCreateSpacePayload(&name, nil)
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		// when
		_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
		require.NotNil(t, created.Data)
		spaceAreaSvc, spaceAreaCtrl := s.SecuredSpaceAreaController(testsupport.TestIdentity)
		_, areaList := test.ListSpaceAreasOK(t, spaceAreaSvc.Context, spaceAreaSvc, spaceAreaCtrl, *created.Data.ID, nil, nil)
		// then
		// only 1 default gets created.
		assert.Len(t, areaList.Data, 1)
		assert.Equal(t, name, *areaList.Data[0].Attributes.Name)

		// verify if root iteration is created or not
		spaceIterationSvc, spaceIterationCtrl := s.SecuredSpaceIterationController(testsupport.TestIdentity)
		_, iterationList := test.ListSpaceIterationsOK(t, spaceIterationSvc.Context, spaceIterationSvc, spaceIterationCtrl, *created.Data.ID, nil, nil)
		require.Len(t, iterationList.Data, 1)
		assert.Equal(t, name, *iterationList.Data[0].Attributes.Name)
	})

	s.T().Run("ok with description", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("TestSuccessCreateSpaceWithDescription-")
		description := "Space for TestSuccessCreateSpaceWithDescription"
		p := newCreateSpacePayload(&name, &description)
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		// when
		_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
		// then
		assert.NotNil(t, created.Data)
		assert.NotNil(t, created.Data.Attributes)
		assert.NotNil(t, created.Data.Attributes.CreatedAt)
		assert.NotNil(t, created.Data.Attributes.UpdatedAt)
		assert.NotNil(t, created.Data.Attributes.Name)
		assert.Equal(t, name, *created.Data.Attributes.Name)
		assert.NotNil(t, created.Data.Attributes.Description)
		assert.Equal(t, description, *created.Data.Attributes.Description)
		assert.NotNil(t, created.Data.Links)
		assert.NotNil(t, created.Data.Links.Self)
	})

	s.T().Run("ok same name but different owner", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("SameName-")
		description := "Space for TestSuccessCreateSameSpaceNameDifferentOwners"
		newDescription := "Space for TestSuccessCreateSameSpaceNameDifferentOwners2"
		a := newCreateSpacePayload(&name, &description)
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, a)
		// when
		b := newCreateSpacePayload(&name, &newDescription)
		svc2, ctrl2 := s.SecuredController(testsupport.TestIdentity2)
		_, created2 := test.CreateSpaceCreated(t, svc2.Context, svc2, ctrl2, b)
		// then
		assert.NotNil(t, created.Data)
		assert.NotNil(t, created.Data.Attributes)
		assert.NotNil(t, created.Data.Attributes.Name)
		assert.Equal(t, name, *created.Data.Attributes.Name)
		assert.NotNil(t, created2.Data)
		assert.NotNil(t, created2.Data.Attributes)
		assert.NotNil(t, created2.Data.Attributes.Name)
		assert.Equal(t, name, *created2.Data.Attributes.Name)
		assert.NotEqual(t, created.Data.Relationships.OwnedBy.Data.ID, created2.Data.Relationships.OwnedBy.Data.ID)
	})

	s.T().Run("fail same name and same owner", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("SameName-")
		description := "Space for TestSuccessCreateSameSpaceNameDifferentOwners"
		newDescription := "Space for TestSuccessCreateSameSpaceNameDifferentOwners2"
		// when
		a := newCreateSpacePayload(&name, &description)
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, a)
		// then
		assert.NotNil(t, created.Data)
		assert.NotNil(t, created.Data.Attributes)
		assert.NotNil(t, created.Data.Attributes.Name)
		assert.Equal(t, name, *created.Data.Attributes.Name)

		// when
		b := newCreateSpacePayload(&name, &newDescription)
		b.Data.Attributes.Name = &name
		b.Data.Attributes.Description = &newDescription
		test.CreateSpaceConflict(t, svc.Context, svc, ctrl, b)
	})

}

func (s *SpaceControllerTestSuite) TestDeleteSpace() {

	s.T().Run("ok", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("TestFailDeleteSpaceDifferentOwner-")
		description := "Space for TestFailDeleteSpaceDifferentOwner"
		p := newCreateSpacePayload(&name, &description)
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
		// when
		svc2, ctrl2 := s.SecuredController(testsupport.TestIdentity)
		test.DeleteSpaceOK(t, svc2.Context, svc2, ctrl2, *created.Data.ID)
	})

	s.T().Run("fail - different owner", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("TestFailDeleteSpaceDifferentOwner-")
		description := "Space for TestFailDeleteSpaceDifferentOwner"
		p := newCreateSpacePayload(&name, &description)
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
		// when
		svc2, ctrl2 := s.SecuredController(testsupport.TestIdentity2)
		_, errors := test.DeleteSpaceForbidden(t, svc2.Context, svc2, ctrl2, *created.Data.ID)
		// then
		assert.NotEmpty(t, errors.Errors)
		assert.Contains(t, errors.Errors[0].Detail, "user is not the space owner")
	})
}

func (s *SpaceControllerTestSuite) TestUpdateSpace() {

	s.T().Run("ok", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("TestSuccessUpdateSpace-")
		description := "Space for TestSuccessUpdateSpace"
		newName := testsupport.CreateRandomValidTestName("TestSuccessUpdateSpace")
		newDescription := "Space for TestSuccessUpdateSpace2"
		p := newCreateSpacePayload(&name, &description)
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
		u := newUpdateSpacePayload()
		u.Data.ID = created.Data.ID
		u.Data.Attributes.Version = created.Data.Attributes.Version
		u.Data.Attributes.Name = &newName
		u.Data.Attributes.Description = &newDescription
		// when
		_, updated := test.UpdateSpaceOK(t, svc.Context, svc, ctrl, *created.Data.ID, u)
		// then
		assert.Equal(t, newName, *updated.Data.Attributes.Name)
		assert.Equal(t, newDescription, *updated.Data.Attributes.Description)
	})

	s.T().Run("fail - version conflict", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("TestSuccessUpdateSpace-")
		description := "Space for TestSuccessUpdateSpace"
		newName := testsupport.CreateRandomValidTestName("TestSuccessUpdateSpace")
		newDescription := "Space for TestSuccessUpdateSpace2"
		p := newCreateSpacePayload(&name, &description)
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
		u := newUpdateSpacePayload()
		u.Data.ID = created.Data.ID
		u.Data.Attributes.Version = created.Data.Attributes.Version
		u.Data.Attributes.Name = &newName
		u.Data.Attributes.Description = &newDescription
		version := 123456
		u.Data.Attributes.Version = &version
		// when/then
		test.UpdateSpaceConflict(t, svc.Context, svc, ctrl, *created.Data.ID, u)
	})

	s.T().Run("fail - name length", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("TestFailUpdateSpaceNameLength-")
		p := newCreateSpacePayload(&name, nil)
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
		// when / then
		u := newUpdateSpacePayload()
		u.Data.ID = created.Data.ID
		u.Data.Attributes.Version = created.Data.Attributes.Version
		p.Data.Attributes.Name = &testsupport.TestOversizedNameObj
		svc2, ctrl2 := s.SecuredController(testsupport.TestIdentity2)
		test.UpdateSpaceBadRequest(t, svc2.Context, svc2, ctrl2, *created.Data.ID, u)
	})

	s.T().Run("fail - different owner", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("TestFailUpdateSpaceDifferentOwner-")
		description := "Space for TestFailUpdateSpaceDifferentOwner"
		newName := testsupport.CreateRandomValidTestName("TestFailUpdateSpaceDifferentOwner-")
		newDescription := "Space for TestFailUpdateSpaceDifferentOwner2"
		p := newCreateSpacePayload(&name, &description)
		p.Data.Attributes.Name = &name
		p.Data.Attributes.Description = &description
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
		// when
		u := newUpdateSpacePayload()
		u.Data.ID = created.Data.ID
		u.Data.Attributes.Version = created.Data.Attributes.Version
		u.Data.Attributes.Name = &newName
		u.Data.Attributes.Description = &newDescription
		svc2, ctrl2 := s.SecuredController(testsupport.TestIdentity2)
		_, errors := test.UpdateSpaceForbidden(t, svc2.Context, svc2, ctrl2, *created.Data.ID, u)
		// then
		assert.NotEmpty(t, errors.Errors)
		assert.Contains(t, errors.Errors[0].Detail, "User is not the space owner")
	})

	s.T().Run("fail - unsecured", func(t *testing.T) {
		// given
		u := newUpdateSpacePayload()
		svc, ctrl := s.UnSecuredController()
		// when/then
		test.UpdateSpaceUnauthorized(t, svc.Context, svc, ctrl, uuid.NewV4(), u)
	})

	s.T().Run("fail - not found", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("TestFailUpdateSpaceNotFound-")
		version := 0
		id := uuid.NewV4()
		u := newUpdateSpacePayload()
		u.Data.Attributes.Name = &name
		u.Data.Attributes.Version = &version
		u.Data.ID = &id
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		// when/then
		test.UpdateSpaceNotFound(t, svc.Context, svc, ctrl, id, u)
	})

	s.T().Run("fail - missing name", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("TestFailUpdateSpaceMissingName-")
		p := newCreateSpacePayload(&name, nil)
		p.Data.Attributes.Name = &name
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
		u := newUpdateSpacePayload()
		u.Data.ID = created.Data.ID
		u.Data.Attributes.Version = created.Data.Attributes.Version
		// when/then
		test.UpdateSpaceBadRequest(t, svc.Context, svc, ctrl, *created.Data.ID, u)
	})

	s.T().Run("fail - missing version", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("TestFailUpdateSpaceMissingVersion-")
		newName := testsupport.CreateRandomValidTestName("TestFailUpdateSpaceMissingVersion-")
		p := newCreateSpacePayload(&name, nil)
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
		u := newUpdateSpacePayload()
		u.Data.ID = created.Data.ID
		u.Data.Attributes.Name = &newName
		// when/then
		test.UpdateSpaceBadRequest(t, svc.Context, svc, ctrl, *created.Data.ID, u)
	})

}

func (s *SpaceControllerTestSuite) TestShowSpace() {

	// needed to valid comparison with golden files
	resetFn := s.DisableGormCallbacks()
	defer resetFn()

	s.T().Run("ok", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("TestShowSpaceOK-")
		description := "Space for TestShowSpaceOK"
		p := newCreateSpacePayload(&name, &description)
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
		// when
		res, fetched := test.ShowSpaceOK(t, svc.Context, svc, ctrl, *created.Data.ID, nil, nil)
		// then
		eTag, lastModified, _ := assertResponseHeaders(t, res)
		assert.Equal(t, app.ToHTTPTime(getSpaceUpdatedAt(*created)), lastModified)
		assert.Equal(t, generateSpaceTag(*created), eTag)
		compareWithGoldenUUIDAgnostic(t, filepath.Join(s.testDir, "show_space_ok.golden.json"), fetched)
	})

	s.T().Run("conditional request", func(t *testing.T) {
		t.Run("ok with expired modified-since header", func(t *testing.T) {
			// given
			name := testsupport.CreateRandomValidTestName("TestShowSpaceOKUsingExpiredIfModifiedSinceHeader-")
			description := "Space for TestShowSpaceOKUsingExpiredIfModifiedSinceHeader"
			p := newCreateSpacePayload(&name, &description)
			svc, ctrl := s.SecuredController(testsupport.TestIdentity)
			_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
			// when
			ifModifiedSince := app.ToHTTPTime(created.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
			res, _ := test.ShowSpaceOK(t, svc.Context, svc, ctrl, *created.Data.ID, &ifModifiedSince, nil)
			// then
			eTag, lastModified, _ := assertResponseHeaders(t, res)
			assert.Equal(t, app.ToHTTPTime(getSpaceUpdatedAt(*created)), lastModified)
			assert.Equal(t, generateSpaceTag(*created), eTag)
		})

		t.Run("ok with expired if-none-match header", func(t *testing.T) {
			// given
			name := testsupport.CreateRandomValidTestName("TestShowSpaceOKUsingExpiredIfNoneMatchHeader-")
			description := "Space for TestShowSpaceOKUsingExpiredIfNoneMatchHeader"
			p := newCreateSpacePayload(&name, &description)
			svc, ctrl := s.SecuredController(testsupport.TestIdentity)
			_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
			// when
			ifNoneMatch := "foo_etag"
			res, _ := test.ShowSpaceOK(t, svc.Context, svc, ctrl, *created.Data.ID, nil, &ifNoneMatch)
			// then
			eTag, lastModified, _ := assertResponseHeaders(t, res)
			assert.Equal(t, app.ToHTTPTime(getSpaceUpdatedAt(*created)), lastModified)
			assert.Equal(t, generateSpaceTag(*created), eTag)
		})

		t.Run("not modified with modified-since header", func(t *testing.T) {
			// given
			name := testsupport.CreateRandomValidTestName("TestShowSpaceNotModifiedUsingIfModifiedSinceHeader-")
			description := "Space for TestShowSpaceNotModifiedUsingIfModifiedSinceHeader"
			p := newCreateSpacePayload(&name, &description)
			svc, ctrl := s.SecuredController(testsupport.TestIdentity)
			_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
			// when/then
			ifModifiedSince := app.ToHTTPTime(getSpaceUpdatedAt(*created))
			test.ShowSpaceNotModified(t, svc.Context, svc, ctrl, *created.Data.ID, &ifModifiedSince, nil)
		})

		t.Run("not modified with if-none-match header", func(t *testing.T) {
			// given
			name := testsupport.CreateRandomValidTestName("TestShowSpaceNotModifiedUsingIfNoneMatchHeader-")
			description := "Space for TestShowSpaceNotModifiedUsingIfNoneMatchHeader"
			p := newCreateSpacePayload(&name, &description)
			svc, ctrl := s.SecuredController(testsupport.TestIdentity)
			_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
			// when/then
			ifNoneMatch := generateSpaceTag(*created)
			test.ShowSpaceNotModified(t, svc.Context, svc, ctrl, *created.Data.ID, nil, &ifNoneMatch)

		})
	})

	s.T().Run("fail - not found", func(t *testing.T) {
		// given
		svc, ctrl := s.UnSecuredController()
		// when/then
		test.ShowSpaceNotFound(t, svc.Context, svc, ctrl, uuid.NewV4(), nil, nil)
	})

}

func (s *SpaceControllerTestSuite) TestListSpaces() {

	s.T().Run("ok", func(t *testing.T) {
		// given
		name := testsupport.CreateRandomValidTestName("TestListSpacesOK-")
		p := newCreateSpacePayload(&name, nil)
		svc, ctrl := s.SecuredController(testsupport.TestIdentity)
		test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
		// when
		_, list := test.ListSpaceOK(t, svc.Context, svc, ctrl, nil, nil, nil, nil)
		// then
		require.NotNil(t, list)
		require.NotEmpty(t, list.Data)
	})

	s.T().Run("fail - unauthorized", func(t *testing.T) {
		// given
		svc, ctrl := s.UnSecuredController()
		// then
		test.ListSpaceUnauthorized(t, svc.Context, svc, ctrl, nil, nil, nil, nil)
	})

	s.T().Run("conditional request", func(t *testing.T) {

		t.Run("ok with expired modified-since header", func(t *testing.T) {
			// given
			name := testsupport.CreateRandomValidTestName("TestListSpacesOKUsingExpiredIfModifiedSinceHeader-")
			p := newCreateSpacePayload(&name, nil)
			svc, ctrl := s.SecuredController(testsupport.TestIdentity)
			_, createdSpace := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
			// when
			t.Logf("space created at=%s", createdSpace.Data.Attributes.CreatedAt.UTC().String())
			ifModifiedSince := app.ToHTTPTime(createdSpace.Data.Attributes.CreatedAt.Add(-1 * time.Hour))
			t.Logf("requesting with `If-Modified-Since`=%s", ifModifiedSince)
			_, list := test.ListSpaceOK(t, svc.Context, svc, ctrl, nil, nil, &ifModifiedSince, nil)
			// then
			require.NotNil(t, list)
			require.NotEmpty(t, list.Data)
		})

		t.Run("ok with expired if-none-match header", func(t *testing.T) {
			// given
			name := testsupport.CreateRandomValidTestName("TestListSpacesOKUsingExpiredIfNoneMatchHeader-")
			p := newCreateSpacePayload(&name, nil)
			svc, ctrl := s.SecuredController(testsupport.TestIdentity)
			test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
			// when
			ifNoneMatch := "foo-spaces"
			_, list := test.ListSpaceOK(t, svc.Context, svc, ctrl, nil, nil, nil, &ifNoneMatch)
			// then
			require.NotNil(t, list)
			require.NotEmpty(t, list.Data)
		})

		t.Run("not modified with modified-since header", func(t *testing.T) {
			// given
			name := testsupport.CreateRandomValidTestName("TestListSpacesNotModifiedUsingIfModifiedSinceHeader-")
			p := newCreateSpacePayload(&name, nil)
			svc, ctrl := s.SecuredController(testsupport.TestIdentity)
			_, createdSpace := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
			// when/then
			ifModifiedSince := app.ToHTTPTime(*createdSpace.Data.Attributes.UpdatedAt)
			test.ListSpaceNotModified(t, svc.Context, svc, ctrl, nil, nil, &ifModifiedSince, nil)
		})

		t.Run("not modified with if-none-match header", func(t *testing.T) {
			// given
			name := testsupport.CreateRandomValidTestName("TestListSpacesNotModifiedUsingIfNoneMatchHeader-")
			p := newCreateSpacePayload(&name, nil)
			svc, ctrl := s.SecuredController(testsupport.TestIdentity)
			test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
			_, spaceList := test.ListSpaceOK(t, svc.Context, svc, ctrl, nil, nil, nil, nil)
			// when/then
			ifNoneMatch := generateSpacesTag(*spaceList)
			test.ListSpaceNotModified(t, svc.Context, svc, ctrl, nil, nil, nil, &ifNoneMatch)
		})
	})
}

func newCreateSpacePayload(name, description *string) *app.CreateSpacePayload {
	return &app.CreateSpacePayload{
		Data: &app.Space{
			Type: "spaces",
			Attributes: &app.SpaceAttributes{
				Name:        name,
				Description: description,
			},
		},
	}
}

func newUpdateSpacePayload() *app.UpdateSpacePayload {
	return &app.UpdateSpacePayload{
		Data: &app.Space{
			Type:       "spaces",
			Attributes: &app.SpaceAttributes{},
		},
	}
}

func generateSpacesTag(entities app.SpaceList) string {
	modelEntities := make([]app.ConditionalRequestEntity, len(entities.Data))
	for i, entityData := range entities.Data {
		modelEntities[i] = ConvertSpaceToModel(*entityData)
	}
	return app.GenerateEntitiesTag(modelEntities)
}

func generateSpaceTag(entity app.SpaceSingle) string {
	return app.GenerateEntityTag(ConvertSpaceToModel(*entity.Data))
}

func convertSpacesToConditionalEntities(spaceList app.SpaceList) []app.ConditionalRequestEntity {
	conditionalSpaces := make([]app.ConditionalRequestEntity, len(spaceList.Data))
	for i, spaceData := range spaceList.Data {
		conditionalSpaces[i] = ConvertSpaceToModel(*spaceData)
	}
	return conditionalSpaces
}

func getSpaceUpdatedAt(appSpace app.SpaceSingle) time.Time {
	return appSpace.Data.Attributes.UpdatedAt.Truncate(time.Second).UTC()
}
