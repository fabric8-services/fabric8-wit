package controller_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/area"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestAreaREST struct {
	gormtestsupport.DBTestSuite
	testDir string
}

func TestRunAreaREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestAreaREST{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (rest *TestAreaREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
	rest.testDir = filepath.Join("test-files", "area")
}

func (rest *TestAreaREST) SecuredController() (*goa.Service, *AreaController) {
	svc := testsupport.ServiceAsUser("Area-Service", testsupport.TestIdentity)
	return svc, NewAreaController(svc, rest.GormDB, rest.Configuration)
}

func (rest *TestAreaREST) SecuredControllerWithIdentity(idn *account.Identity) (*goa.Service, *AreaController) {
	svc := testsupport.ServiceAsUser("Area-Service", *idn)
	return svc, NewAreaController(svc, rest.GormDB, rest.Configuration)
}

func (rest *TestAreaREST) UnSecuredController() (*goa.Service, *AreaController) {
	svc := goa.New("Area-Service")
	return svc, NewAreaController(svc, rest.GormDB, rest.Configuration)
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
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "ok.res.payload.golden.json"), created)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "ok.res.headers.golden.json"), resp.Header())

			t.Run("Multiple Children", func(t *testing.T) {
				// Create a child of the child created above.
				ca = newCreateChildAreaPayload("TestSuccessCreateMultiChildArea-0")
				newParentID := *created.Data.Relationships.Parent.Data.ID
				// when
				resp, created = test.CreateChildAreaCreated(t, svc.Context, svc, ctrl, newParentID, ca)
				// then
				assert.Equal(t, *ca.Data.Attributes.Name, *created.Data.Attributes.Name)
				assert.Equal(t, newParentID, *created.Data.Relationships.Parent.Data.ID)
				compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "ok.child1_ok.res.payload.golden.json"), created)
				compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "ok.res.headers.golden.json"), resp.Header())
			})
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
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "conflict.res.payload.golden.json"), errs)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "conflict.res.headers.golden.json"), resp.Header())
		})

		t.Run("Missing Name", func(t *testing.T) {
			// when
			childAreaPayload.Data.Attributes.Name = nil
			// then
			resp, errs := test.CreateChildAreaBadRequest(t, svc.Context, svc, ctrl, parentID.String(), childAreaPayload)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "missing_name.res.payload.golden.json"), errs)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "missing_name.res.headers.golden.json"), resp.Header())
		})

		t.Run("Invalid Parent", func(t *testing.T) {
			// when
			createChildAreaPayload := newCreateChildAreaPayload("TestFailCreateChildAreaWithInvalidParent")
			// then
			resp, errs := test.CreateChildAreaNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), createChildAreaPayload)
			// Ignore error ID
			errs.Errors[0].ID = ptr.String("IGNORE_ME")
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "invalid_parent.res.payload.golden.json"), errs)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "invalid_parent.res.headers.golden.json"), resp.Header())
		})

		t.Run("Unauthorized", func(t *testing.T) {
			// when
			svc, ctrl := rest.UnSecuredController()
			// then
			resp, errs := test.CreateChildAreaUnauthorized(t, svc.Context, svc, ctrl, parentID.String(), childAreaPayload)
			// Ignore error ID
			errs.Errors[0].ID = ptr.String("IGNORE_ME")
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "unauthorized.res.payload.golden.json"), errs)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "unauthorized.res.headers.golden.json"), resp.Header())
		})

		t.Run("Different Identity", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB, tf.Identities(1))
			// try creating child area with different identity: should fail
			otherIdentity := fxt.Identities[0]
			parentID := parentArea.ID
			svc, ctrl = rest.SecuredControllerWithIdentity(otherIdentity)
			resp, err := test.CreateChildAreaForbidden(t, svc.Context, svc, ctrl, parentID.String(), childAreaPayload)
			compareWithGolden(t, filepath.Join(rest.testDir, "create", "forbidden.res.payload.golden.json"), err)
			compareWithGolden(t, filepath.Join(rest.testDir, "create", "forbidden.res.headers.golden.json"), resp.Header())
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
			safeOverriteHeader(t, res, app.ETag, "0icd7ov5CqwDXN6Fx9z18g==")
			//then
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "ok.res.payload.golden.json"), area)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "ok.res.headers.golden.json"), res.Header())
		})

		t.Run("Using ExpiredIfModifedSince Header", func(t *testing.T) {
			// when
			ifModifiedSince := app.ToHTTPTime(a.UpdatedAt.Add(-1 * time.Hour))
			res, area := test.ShowAreaOK(t, svc.Context, svc, ctrl, a.ID.String(), &ifModifiedSince, nil)
			safeOverriteHeader(t, res, app.ETag, "0icd7ov5CqwDXN6Fx9z18g==")
			//then
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "expired_if_modified_since.res.payload.golden.json"), area)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "ok.res.headers.golden.json"), res.Header())
		})

		t.Run("Using ExpiredIfNoneMatch Header", func(t *testing.T) {
			// when
			ifNoneMatch := "foo"
			res, area := test.ShowAreaOK(t, svc.Context, svc, ctrl, a.ID.String(), nil, &ifNoneMatch)
			safeOverriteHeader(t, res, app.ETag, "0icd7ov5CqwDXN6Fx9z18g==")
			//then
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "expired_if_none_match.res.payload.golden.json"), area)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "ok.res.headers.golden.json"), res.Header())
		})

		t.Run("Not Modified Using IfModifedSince Header", func(t *testing.T) {
			// when
			ifModifiedSince := app.ToHTTPTime(a.UpdatedAt)
			area := test.ShowAreaNotModified(t, svc.Context, svc, ctrl, a.ID.String(), &ifModifiedSince, nil)
			safeOverriteHeader(t, area, app.ETag, "0icd7ov5CqwDXN6Fx9z18g==")
			//then
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "if_modified_since_headers.res.headers.golden.json"), area.Header())
		})

		t.Run("Not Modified IfNoneMatch Header", func(t *testing.T) {
			// when
			ifNoneMatch := app.GenerateEntityTag(a)
			area := test.ShowAreaNotModified(t, svc.Context, svc, ctrl, a.ID.String(), nil, &ifNoneMatch)
			safeOverriteHeader(t, area, app.ETag, "0icd7ov5CqwDXN6Fx9z18g==")
			//then
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "if_none_match_headers.res.headers.golden.json"), area.Header())
		})
	})

	rest.T().Run("Failure", func(t *testing.T) {
		// Setup
		svc, ctrl := rest.SecuredController()
		t.Run("Not Found", func(t *testing.T) {
			// when
			resp, area := test.ShowAreaNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), nil, nil)
			//then
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "not_found.res.payload.golden.json"), area)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "not_found.res.headers.golden.json"), resp.Header())
		})
	})
}

func (rest *TestAreaREST) TestAreaPayload() {
	rest.T().Run("Failure", func(t *testing.T) {
		t.Run("Validate Area name Length", func(t *testing.T) {
			// given
			ca := newCreateChildAreaPayload(testsupport.TestOversizedNameObj)
			// then
			err := ca.Validate()
			// Validate payload function returns an error
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), "length of type.name must be less than or equal to 63")
		})

		t.Run("Validate Area name Start With", func(t *testing.T) {
			// given
			ca := newCreateChildAreaPayload("_TestSuccessCreateChildArea")
			// then
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
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.Areas(1))
	parentArea := fxt.Areas[0]
	owner := fxt.Identities[0]
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	rest.T().Run("Success", func(t *testing.T) {
		childArea := rest.createChildArea("TestShowChildrenArea", *parentArea, svc, ctrl)
		t.Run("OK", func(t *testing.T) {
			res, result := test.ShowChildrenAreaOK(t, svc.Context, svc, ctrl, parentArea.ID.String(), nil, nil)
			safeOverriteHeader(t, res, app.ETag, "0icd7ov5CqwDXN6Fx9z18g==")
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "ok.res.payload.golden.json"), result)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "ok.res.headers.golden.json"), res.Header())
		})
		t.Run("Using ExpiredIfModifedSince Header", func(t *testing.T) {
			ifModifiedSince := app.ToHTTPTime(parentArea.UpdatedAt.Add(-1 * time.Hour))
			res, result := test.ShowChildrenAreaOK(t, svc.Context, svc, ctrl, parentArea.ID.String(), &ifModifiedSince, nil)
			safeOverriteHeader(t, res, app.ETag, "0icd7ov5CqwDXN6Fx9z18g==")
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "ok.res.payload.golden.json"), result)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "ok.res.headers.golden.json"), res.Header())
		})

		t.Run("Using ExpiredIfNoneMatch Header", func(t *testing.T) {
			ifNoneMatch := "foo"
			res, result := test.ShowChildrenAreaOK(t, svc.Context, svc, ctrl, parentArea.ID.String(), nil, &ifNoneMatch)
			safeOverriteHeader(t, res, app.ETag, "0icd7ov5CqwDXN6Fx9z18g==")
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "ok.res.payload.golden.json"), result)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "ok.res.headers.golden.json"), res.Header())
		})

		t.Run("Not Modified Using IfModifedSince Header", func(t *testing.T) {
			ifModifiedSince := app.ToHTTPTime(*childArea.Data.Attributes.UpdatedAt)
			res := test.ShowChildrenAreaNotModified(t, svc.Context, svc, ctrl, parentArea.ID.String(), &ifModifiedSince, nil)
			safeOverriteHeader(t, res, app.ETag, "0icd7ov5CqwDXN6Fx9z18g==")
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "not_modified_header.res.headers.golden.json"), res.Header())
		})

		t.Run("Not Modified IfNoneMatch Header", func(t *testing.T) {
			modelChildArea := ConvertAreaToModel(*childArea)
			ifNoneMatch := app.GenerateEntityTag(modelChildArea)
			res := test.ShowChildrenAreaNotModified(t, svc.Context, svc, ctrl, parentArea.ID.String(), nil, &ifNoneMatch)
			safeOverriteHeader(t, res, app.ETag, "0icd7ov5CqwDXN6Fx9z18g==")
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "not_modified_header.res.headers.golden.json"), res.Header())
		})
	})

	rest.T().Run("Failure", func(t *testing.T) {
		t.Run("Not Found", func(t *testing.T) {
			res, result := test.ShowChildrenAreaNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), nil, nil)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "not_found.res.payload.golden.json"), result)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "showChildren", "not_found.res.header.golden.json"), res.Header())
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
