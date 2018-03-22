package controller_test

import (
	"os"
	"path/filepath"
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
	"github.com/fabric8-services/fabric8-wit/ptr"
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
	db      *gormapplication.GormDB
	testDir string
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
	rest.testDir = filepath.Join("test-files", "area")
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
			fxt := tf.NewTestFixture(t, rest.DB, tf.Identities(2), tf.Areas(1))
			parentArea := fxt.Areas[0]
			parentID := parentArea.ID
			ca := newCreateChildAreaPayload("TestSuccessCreateChildArea")
			owner := fxt.Identities[0]
			svc, ctrl := rest.SecuredControllerWithIdentity(owner)
			// when
			resp, created := test.CreateChildAreaCreated(t, svc.Context, svc, ctrl, parentID.String(), ca)
			// then
			assert.Equal(t, *ca.Data.Attributes.Name, *created.Data.Attributes.Name)
			assert.Equal(t, parentID.String(), *created.Data.Relationships.Parent.Data.ID)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "ok.golden.json"), created)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "ok.headers.golden.json"), resp.Header())
			// try creating child area with different identity: should fail
			otherIdentity := fxt.Identities[1]
			svc, ctrl = rest.SecuredControllerWithIdentity(otherIdentity)
			resp, err := test.CreateChildAreaForbidden(t, svc.Context, svc, ctrl, parentID.String(), ca)
			compareWithGolden(t, filepath.Join(rest.testDir, "create", "forbidden.golden.json"), err)
			compareWithGolden(t, filepath.Join(rest.testDir, "create", "failed.headers.golden.json"), resp.Header())
		})

		t.Run("Multiple Children", func(t *testing.T) {
			/*
				TestAreaREST ---> TestSuccessCreateMultiChildArea-0 ----> TestSuccessCreateMultiChildArea-0-0
			*/
			// given
			fxt := tf.NewTestFixture(t, rest.DB, tf.Areas(1))
			parentArea := fxt.Areas[0]
			parentID := parentArea.ID
			ca := newCreateChildAreaPayload("TestSuccessCreateMultiChildArea-0")
			owner := fxt.Identities[0]
			svc, ctrl := rest.SecuredControllerWithIdentity(owner)
			// when
			resp, created := test.CreateChildAreaCreated(t, svc.Context, svc, ctrl, parentID.String(), ca)
			// then
			assert.Equal(t, *ca.Data.Attributes.Name, *created.Data.Attributes.Name)
			assert.Equal(t, parentID.String(), *created.Data.Relationships.Parent.Data.ID)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "multiple_child", "child1_ok.golden.json"), created)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "multiple_child", "ok.headers.golden.json"), resp.Header())
			// Create a child of the child created above.
			ca = newCreateChildAreaPayload("TestSuccessCreateMultiChildArea-0-0")
			newParentID := *created.Data.Relationships.Parent.Data.ID
			// when
			resp, created = test.CreateChildAreaCreated(t, svc.Context, svc, ctrl, newParentID, ca)
			// then
			assert.Equal(t, *ca.Data.Attributes.Name, *created.Data.Attributes.Name)
			assert.Equal(t, newParentID, *created.Data.Relationships.Parent.Data.ID)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "multiple_child", "child2_ok.golden.json"), created)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "multiple_child", "ok.headers.golden.json"), resp.Header())
		})
	})

	rest.T().Run("Failure", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, rest.DB, tf.Areas(1))
		parentArea := fxt.Areas[0]
		parentID := parentArea.ID
		childAreaPayload := newCreateChildAreaPayload(uuid.NewV4().String())
		owner := fxt.Identities[0]
		svc, ctrl := rest.SecuredControllerWithIdentity(owner)
		t.Run("Duplicate Child Area", func(t *testing.T) {
			// when
			_, created := test.CreateChildAreaCreated(t, svc.Context, svc, ctrl, parentID.String(), childAreaPayload)
			// then
			assert.Equal(t, *childAreaPayload.Data.Attributes.Name, *created.Data.Attributes.Name)
			assert.Equal(t, parentID.String(), *created.Data.Relationships.Parent.Data.ID)

			// try creating the same area again
			resp, errs := test.CreateChildAreaConflict(t, svc.Context, svc, ctrl, parentID.String(), childAreaPayload)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "conflict.golden.json"), errs)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "failed.headers.golden.json"), resp.Header())
		})

		t.Run("Missing Name", func(t *testing.T) {
			// when
			childAreaPayload.Data.Attributes.Name = nil
			// then
			resp, errs := test.CreateChildAreaBadRequest(t, svc.Context, svc, ctrl, parentID.String(), childAreaPayload)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "missing_name.golden.json"), errs)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "failed.headers.golden.json"), resp.Header())
		})

		t.Run("Invalid Parent", func(t *testing.T) {
			// when
			createChildAreaPayload := newCreateChildAreaPayload("TestFailCreateChildAreaWithInvalidsParent")
			// then
			resp, errs := test.CreateChildAreaNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), createChildAreaPayload)
			// Ignore error ID
			errs.Errors[0].ID = ptr.String("IGNORE_ME")
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "invalid_parent.golden.json"), errs)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "failed.headers.golden.json"), resp.Header())
		})

		t.Run("Unauthorized", func(t *testing.T) {
			// when
			svc, ctrl := rest.UnSecuredController()
			// then
			resp, errs := test.CreateChildAreaUnauthorized(t, svc.Context, svc, ctrl, parentID.String(), childAreaPayload)
			// Ignore error ID
			errs.Errors[0].ID = ptr.String("IGNORE_ME")
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "unauthorized.golden.json"), errs)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "failed.headers.golden.json"), resp.Header())
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
			res, area := test.ShowAreaOK(t, svc.Context, svc, ctrl, a.ID.String(), nil, nil)
			res.Header().Set("Etag", "IGNORE_ME")
			//then
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "ok.golden.json"), area)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "ok.headers.golden.json"), res.Header())
		})

		t.Run("Using ExpiredIfModifedSince Header", func(t *testing.T) {
			// when
			ifModifiedSince := app.ToHTTPTime(a.UpdatedAt.Add(-1 * time.Hour))
			res, area := test.ShowAreaOK(t, svc.Context, svc, ctrl, a.ID.String(), &ifModifiedSince, nil)
			res.Header().Set("Etag", "IGNORE_ME")
			//then
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "expired_if_modified_since.golden.json"), area)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "ok.headers.golden.json"), res.Header())
		})

		t.Run("Using ExpiredIfNoneMatch Header", func(t *testing.T) {
			// when
			ifNoneMatch := "foo"
			res, area := test.ShowAreaOK(t, svc.Context, svc, ctrl, a.ID.String(), nil, &ifNoneMatch)
			res.Header().Set("Etag", "IGNORE_ME")
			//then
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "expired_if_none_match.golden.json"), area)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "ok.headers.golden.json"), res.Header())
		})

		t.Run("Not Modified Using IfModifedSince Header", func(t *testing.T) {
			// when
			ifModifiedSince := app.ToHTTPTime(a.UpdatedAt)
			area := test.ShowAreaNotModified(t, svc.Context, svc, ctrl, a.ID.String(), &ifModifiedSince, nil)
			area.Header().Set("Etag", "IGNORE_ME")
			//then
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "if_modified_since_headers.golden.json"), area.Header())
		})

		t.Run("Not Modified IfNoneMatch Header", func(t *testing.T) {
			// when
			ifNoneMatch := app.GenerateEntityTag(a)
			area := test.ShowAreaNotModified(t, svc.Context, svc, ctrl, a.ID.String(), nil, &ifNoneMatch)
			area.Header().Set("Etag", "IGNORE_ME")
			//then
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "if_none_match_headers.golden.json"), area.Header())
		})
	})

	rest.T().Run("Failure", func(t *testing.T) {
		// Setup
		svc, ctrl := rest.SecuredController()
		t.Run("Not Found", func(t *testing.T) {
			// when
			resp, area := test.ShowAreaNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), nil, nil)
			//then
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "not_found.golden.json"), area)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "not_found_headers.golden.json"), resp.Header())
		})
	})
}

func (rest *TestAreaREST) TestAreaPayload() {
	rest.T().Run("Failure", func(t *testing.T) {
		t.Run("Validate Area name Length", func(t *testing.T) {
			// given
			ca := newCreateChildAreaPayload(testsupport.TestOversizedNameObj)
			// then
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "payload", "invalid_name_length.golden.json"), ca)
		})

		t.Run("Validate Area name Start With", func(t *testing.T) {
			// given
			ca := newCreateChildAreaPayload("_TestSuccessCreateChildArea")
			// then
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "payload", "invalid_name_start.golden.json"), ca)
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
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.Areas(1))
	parentArea := fxt.Areas[0]
	owner := fxt.Identities[0]
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	rest.T().Run("Success", func(t *testing.T) {
		childArea := rest.createChildArea("TestShowChildrenArea", *parentArea, svc, ctrl)
		t.Run("OK", func(t *testing.T) {
			res, result := test.ShowChildrenAreaOK(t, svc.Context, svc, ctrl, parentArea.ID.String(), nil, nil)
			res.Header().Set("Etag", "IGNORE_ME")
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "ok.golden.json"), result)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "ok.headers.golden.json"), res.Header())
		})
		t.Run("Using ExpiredIfModifedSince Header", func(t *testing.T) {
			ifModifiedSince := app.ToHTTPTime(parentArea.UpdatedAt.Add(-1 * time.Hour))
			res, result := test.ShowChildrenAreaOK(t, svc.Context, svc, ctrl, parentArea.ID.String(), &ifModifiedSince, nil)
			res.Header().Set("Etag", "IGNORE_ME")
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "ok.golden.json"), result)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "ok.headers.golden.json"), res.Header())
		})

		t.Run("Using ExpiredIfNoneMatch Header", func(t *testing.T) {
			ifNoneMatch := "foo"
			res, result := test.ShowChildrenAreaOK(t, svc.Context, svc, ctrl, parentArea.ID.String(), nil, &ifNoneMatch)
			res.Header().Set("Etag", "IGNORE_ME")
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "ok.golden.json"), result)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "ok.headers.golden.json"), res.Header())
		})

		t.Run("Not Modified Using IfModifedSince Header", func(t *testing.T) {
			ifModifiedSince := app.ToHTTPTime(*childArea.Data.Attributes.UpdatedAt)
			res := test.ShowChildrenAreaNotModified(t, svc.Context, svc, ctrl, parentArea.ID.String(), &ifModifiedSince, nil)
			res.Header().Set("Etag", "IGNORE_ME")
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "not_modified_header.golden.json"), res.Header())
		})

		t.Run("Not Modified IfNoneMatch Header", func(t *testing.T) {
			modelChildArea := convertAreaToModel(*childArea)
			ifNoneMatch := app.GenerateEntityTag(modelChildArea)
			res := test.ShowChildrenAreaNotModified(t, svc.Context, svc, ctrl, parentArea.ID.String(), nil, &ifNoneMatch)
			res.Header().Set("Etag", "IGNORE_ME")
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "not_modified_header.golden.json"), res.Header())
		})
	})

	rest.T().Run("Failure", func(t *testing.T) {
		t.Run("Not Found", func(t *testing.T) {
			res, result := test.ShowChildrenAreaNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), nil, nil)
			res.Header().Set("Etag", "IGNORE_ME")
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "not_found.golden.json"), result)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "not_found.header.golden.json"), res.Header())
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
