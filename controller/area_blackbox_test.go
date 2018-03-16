package controller_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/area"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestAreaREST struct {
	gormtestsupport.DBTestSuite
	db *gormapplication.GormDB
}

func TestRunAreaREST(t *testing.T) {
	resource.Require(t, resource.Database)
	pwd, err := os.Getwd()
	require.NoError(t, err)
	suite.Run(t, &TestAreaREST{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

func (rest *TestAreaREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
	rest.db = gormapplication.NewGormDB(rest.DB)
}

func (rest *TestAreaREST) SecuredController() (*goa.Service, *AreaController) {
	svc := testsupport.ServiceAsUser("Area-Service", testsupport.TestIdentity)
	return svc, NewAreaController(svc, rest.db, rest.Configuration)
}

func (rest *TestAreaREST) SecuredControllerWithIdentity(idn *account.Identity) (*goa.Service, *AreaController) {
	svc := testsupport.ServiceAsUser("Area-Service", *idn)
	return svc, NewAreaController(svc, rest.db, rest.Configuration)
}

func (rest *TestAreaREST) UnSecuredController() (*goa.Service, *AreaController) {
	svc := goa.New("Area-Service")
	return svc, NewAreaController(svc, rest.db, rest.Configuration)
}

func (rest *TestAreaREST) TestCreateChildArea() {
	rest.T().Run("Success", func(t *testing.T) {
		t.Run("OK", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, rest.DB, tf.Spaces(1), tf.Areas(1))
			sp := fxt.Spaces[0]
			parentArea := fxt.Areas[0]
			parentID := parentArea.ID
			ca := newCreateChildAreaPayload("TestSuccessCreateChildArea")
			owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerID)
			require.NoError(t, err)
			svc, ctrl := rest.SecuredControllerWithIdentity(owner)
			// when
			_, created := test.CreateChildAreaCreated(t, svc.Context, svc, ctrl, parentID.String(), ca)
			// then
			assert.Equal(t, *ca.Data.Attributes.Name, *created.Data.Attributes.Name)
			fmt.Println(*created.Data.Relationships.Parent.Data.ID)
			assert.Equal(t, parentID.String(), *created.Data.Relationships.Parent.Data.ID)

			// try creating child area with different identity: should fail
			otherIdentity := &account.Identity{
				Username:     "non-space-owner-identity",
				ProviderType: account.KeycloakIDP,
			}
			errInCreateOther := rest.db.Identities().Create(context.Background(), otherIdentity)
			require.NoError(t, errInCreateOther)
			svc, ctrl = rest.SecuredControllerWithIdentity(otherIdentity)
			test.CreateChildAreaForbidden(t, svc.Context, svc, ctrl, parentID.String(), ca)
		})

		t.Run("Multiple Children", func(t *testing.T) {
			/*
				TestAreaREST ---> TestSuccessCreateMultiChildArea-0 ----> TestSuccessCreateMultiChildArea-0-0
			*/
			// given
			fxt := tf.NewTestFixture(t, rest.DB, tf.Spaces(1), tf.Areas(1))
			sp := fxt.Spaces[0]
			parentArea := fxt.Areas[0]
			parentID := parentArea.ID
			ca := newCreateChildAreaPayload("TestSuccessCreateMultiChildArea-0")
			owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerID)
			require.NoError(t, err)
			svc, ctrl := rest.SecuredControllerWithIdentity(owner)
			// when
			_, created := test.CreateChildAreaCreated(t, svc.Context, svc, ctrl, parentID.String(), ca)
			// then
			assert.Equal(t, *ca.Data.Attributes.Name, *created.Data.Attributes.Name)
			assert.Equal(t, parentID.String(), *created.Data.Relationships.Parent.Data.ID)
			// Create a child of the child created above.
			ca = newCreateChildAreaPayload("TestSuccessCreateMultiChildArea-0-0")
			newParentID := *created.Data.Relationships.Parent.Data.ID
			// when
			_, created = test.CreateChildAreaCreated(t, svc.Context, svc, ctrl, newParentID, ca)
			// then
			assert.Equal(t, *ca.Data.Attributes.Name, *created.Data.Attributes.Name)
			assert.NotNil(t, *created.Data.Attributes.CreatedAt)
			assert.NotNil(t, *created.Data.Attributes.Version)
			assert.Equal(t, newParentID, *created.Data.Relationships.Parent.Data.ID)
			assert.Contains(t, *created.Data.Relationships.Children.Links.Self, "children")
		})
	})

	rest.T().Run("Failure", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, rest.DB, tf.Spaces(1), tf.Areas(1))
		sp := fxt.Spaces[0]
		parentArea := fxt.Areas[0]
		parentID := parentArea.ID
		childAreaPayload := newCreateChildAreaPayload(uuid.NewV4().String())
		owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerID)
		require.NoError(t, err)
		svc, ctrl := rest.SecuredControllerWithIdentity(owner)
		t.Run("Duplicate Child Area", func(t *testing.T) {
			// when
			_, created := test.CreateChildAreaCreated(t, svc.Context, svc, ctrl, parentID.String(), childAreaPayload)
			// then
			assert.Equal(t, *childAreaPayload.Data.Attributes.Name, *created.Data.Attributes.Name)
			assert.Equal(t, parentID.String(), *created.Data.Relationships.Parent.Data.ID)

			// try creating the same area again
			test.CreateChildAreaConflict(t, svc.Context, svc, ctrl, parentID.String(), childAreaPayload)

		})

		t.Run("Missing Name", func(t *testing.T) {
			// when
			childAreaPayload.Data.Attributes.Name = nil
			// then
			test.CreateChildAreaBadRequest(t, svc.Context, svc, ctrl, parentID.String(), childAreaPayload)
		})

		t.Run("Invalid Parent", func(t *testing.T) {
			// when
			createChildAreaPayload := newCreateChildAreaPayload("TestFailCreateChildAreaWithInvalidsParent")
			// then
			test.CreateChildAreaNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), createChildAreaPayload)
		})

		t.Run("Unauthorized", func(t *testing.T) {
			// when
			svc, ctrl := rest.UnSecuredController()
			// then
			test.CreateChildAreaUnauthorized(t, svc.Context, svc, ctrl, parentID.String(), childAreaPayload)
		})
	})
}

func (rest *TestAreaREST) TestShowArea() {
	rest.T().Run("Success", func(t *testing.T) {
		// Setup
		a := tf.NewTestFixture(t, rest.DB, tf.Areas(1)).Areas[0]
		svc, ctrl := rest.SecuredController()
		t.Run("OK", func(t *testing.T) {
			// when
			res, _ := test.ShowAreaOK(t, svc.Context, svc, ctrl, a.ID.String(), nil, nil)
			//then
			assertResponseHeaders(t, res)
		})

		t.Run("Using ExpiredIfModifedSince Header", func(t *testing.T) {
			// when
			ifModifiedSince := app.ToHTTPTime(a.UpdatedAt.Add(-1 * time.Hour))
			res, _ := test.ShowAreaOK(t, svc.Context, svc, ctrl, a.ID.String(), &ifModifiedSince, nil)
			//then
			assertResponseHeaders(t, res)
		})

		t.Run("Using ExpiredIfNoneMatch Header", func(t *testing.T) {
			// when
			ifNoneMatch := "foo"
			res, _ := test.ShowAreaOK(t, svc.Context, svc, ctrl, a.ID.String(), nil, &ifNoneMatch)
			//then
			assertResponseHeaders(t, res)
		})

		t.Run("Not Modified Using IfModifedSince Header", func(t *testing.T) {
			// when
			ifModifiedSince := app.ToHTTPTime(a.UpdatedAt)
			res := test.ShowAreaNotModified(t, svc.Context, svc, ctrl, a.ID.String(), &ifModifiedSince, nil)
			//then
			assertResponseHeaders(t, res)
		})

		t.Run("Not Modified IfNoneMatch Header", func(t *testing.T) {
			// when
			ifNoneMatch := app.GenerateEntityTag(a)
			res := test.ShowAreaNotModified(t, svc.Context, svc, ctrl, a.ID.String(), nil, &ifNoneMatch)
			//then
			assertResponseHeaders(t, res)
		})
	})

	rest.T().Run("Failure", func(t *testing.T) {
		// Setup
		svc, ctrl := rest.SecuredController()
		t.Run("Not Found", func(t *testing.T) {
			// when/then
			test.ShowAreaNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), nil, nil)
		})
	})
}

func (rest *TestAreaREST) TestAreaPayload() {
	rest.T().Run("Failure", func(t *testing.T) {
		t.Run("Validate Area name Length", func(t *testing.T) {
			// given
			ca := newCreateChildAreaPayload(testsupport.TestOversizedNameObj)

			err := ca.Validate()
			// Validate payload function returns an error
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), "length of type.name must be less than or equal to 63")
		})

		t.Run("Validate Area name Start With", func(t *testing.T) {
			// given
			ca := newCreateChildAreaPayload("_TestSuccessCreateChildArea")

			err := ca.Validate()
			// Validate payload function returns an error
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), "type.name must match the regexp")
		})
	})
}

func (rest *TestAreaREST) createChildArea(name string, parent area.Area, svc *goa.Service, ctrl *AreaController) *app.AreaSingle {
	ci := newCreateChildAreaPayload(name)
	// when
	_, created := test.CreateChildAreaCreated(rest.T(), svc.Context, svc, ctrl, parent.ID.String(), ci)
	return created
}

func (rest *TestAreaREST) TestShowChildrenArea() {
	// Setup
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.Spaces(1), tf.Areas(1))
	sp := fxt.Spaces[0]
	parentArea := fxt.Areas[0]
	owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerID)
	require.NoError(rest.T(), err)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	childArea := rest.createChildArea("TestShowChildrenArea", *parentArea, svc, ctrl)
	rest.T().Run("Success", func(t *testing.T) {
		t.Run("OK", func(t *testing.T) {
			res, result := test.ShowChildrenAreaOK(rest.T(), svc.Context, svc, ctrl, parentArea.ID.String(), nil, nil)
			assert.Equal(rest.T(), 1, len(result.Data))
			assertResponseHeaders(rest.T(), res)
		})
		t.Run("Using ExpiredIfModifedSince Header", func(t *testing.T) {
			ifModifiedSince := app.ToHTTPTime(parentArea.UpdatedAt.Add(-1 * time.Hour))
			res, result := test.ShowChildrenAreaOK(rest.T(), svc.Context, svc, ctrl, parentArea.ID.String(), &ifModifiedSince, nil)
			assert.Equal(rest.T(), 1, len(result.Data))
			assertResponseHeaders(rest.T(), res)
		})

		t.Run("Using ExpiredIfNoneMatch Header", func(t *testing.T) {
			ifNoneMatch := "foo"
			res, result := test.ShowChildrenAreaOK(rest.T(), svc.Context, svc, ctrl, parentArea.ID.String(), nil, &ifNoneMatch)
			assert.Equal(rest.T(), 1, len(result.Data))
			assertResponseHeaders(rest.T(), res)
		})

		t.Run("Not Modified Using IfModifedSince Header", func(t *testing.T) {
			ifModifiedSince := app.ToHTTPTime(*childArea.Data.Attributes.UpdatedAt)
			res := test.ShowChildrenAreaNotModified(rest.T(), svc.Context, svc, ctrl, parentArea.ID.String(), &ifModifiedSince, nil)
			assertResponseHeaders(rest.T(), res)
		})

		t.Run("Not Modified IfNoneMatch Header", func(t *testing.T) {
			modelChildArea := convertAreaToModel(*childArea)
			ifNoneMatch := app.GenerateEntityTag(modelChildArea)
			res := test.ShowChildrenAreaNotModified(rest.T(), svc.Context, svc, ctrl, parentArea.ID.String(), nil, &ifNoneMatch)
			assertResponseHeaders(rest.T(), res)
		})
	})
}

func ConvertAreaToModel(appArea app.AreaSingle) area.Area {
	return area.Area{
		ID:      *appArea.Data.ID,
		Version: *appArea.Data.Attributes.Version,
		Lifecycle: gormsupport.Lifecycle{
			UpdatedAt: *appArea.Data.Attributes.UpdatedAt,
		},
	}
}

func newCreateChildAreaPayload(name string) *app.CreateChildAreaPayload {
	areaType := area.APIStringTypeAreas
	return &app.CreateChildAreaPayload{
		Data: &app.Area{
			Type: areaType,
			Attributes: &app.AreaAttributes{
				Name: &name,
			},
		},
	}
}
