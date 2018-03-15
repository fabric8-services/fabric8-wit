package controller_test

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"time"

	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//-----------------------------------------------------------------------------
// Test Suite setup
//-----------------------------------------------------------------------------

// The workItemTypesSuite has state the is relevant to all tests.
type workItemTypesSuite struct {
	gormtestsupport.DBTestSuite
	typesCtrl    *WorkitemtypesController
	linkTypeCtrl *WorkItemLinkTypeController
	linkCatCtrl  *WorkItemLinkCategoryController
	spaceCtrl    *SpaceController
	svc          *goa.Service
	testDir      string
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemTypes(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemTypesSuite{
		DBTestSuite: gormtestsupport.NewDBTestSuite(""),
	})
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemTypesSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.testDir = filepath.Join("test-files", "work_item_type")
}

// The SetupTest method will be run before every test in the suite.
func (s *workItemTypesSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	idn := &account.Identity{
		ID:           uuid.Nil,
		Username:     "TestDeveloper",
		ProviderType: "test provider",
	}
	s.svc = testsupport.ServiceAsUser("workItemLinkSpace-Service", *idn)
	s.spaceCtrl = NewSpaceController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration, &DummyResourceManager{})
	require.NotNil(s.T(), s.spaceCtrl)
	s.typesCtrl = NewWorkitemtypesController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.linkTypeCtrl = NewWorkItemLinkTypeController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(s.svc, gormapplication.NewGormDB(s.DB))

}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

// createWorkItemTypeAnimal defines a work item type "animal" that consists of
// two fields ("animal-type" and "color"). The type is mandatory but the color is not.
func (s *workItemTypesSuite) createWorkItemTypeAnimal() *app.WorkItemTypeSingle {
	witRepo := workitem.NewWorkItemTypeRepository(s.DB)
	wit, err := witRepo.Create(context.Background(), spacetemplate.SystemLegacyTemplateID, &animalID, nil, "animal", ptr.String("Description for 'animal'"), "fa-hand-lizard-o", workitem.FieldDefinitions{
		"animal_type": {
			Required:    true,
			Description: "Description for animal_type field",
			Label:       "Animal Type",
			Type: &workitem.EnumType{
				SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
				BaseType:   workitem.SimpleType{Kind: workitem.KindString},
				Values:     []interface{}{"elephant", "blue whale", "Tyrannosaurus rex"},
			},
		},
		"color": {
			Required:    true,
			Description: "Description for color field",
			Label:       "Color",
			Type:        &workitem.SimpleType{Kind: workitem.KindString},
		},
	})
	require.Nil(s.T(), err)
	reqLong := &http.Request{Host: "api.service.domain.org"}
	witData := ConvertWorkItemTypeFromModel(reqLong, wit)
	return &app.WorkItemTypeSingle{
		Data: &witData,
	}
}

// createWorkItemTypePerson defines a work item type "person" that consists of
// a required "name" field.
func (s *workItemTypesSuite) createWorkItemTypePerson() *app.WorkItemTypeSingle {
	witRepo := workitem.NewWorkItemTypeRepository(s.DB)
	wit, err := witRepo.Create(context.Background(), spacetemplate.SystemLegacyTemplateID, &personID, nil, "person", ptr.String("Description for 'person'"), "fa-user", workitem.FieldDefinitions{
		"name": {
			Required:    true,
			Description: "Description for Name field",
			Label:       "Name",
			Type:        &workitem.SimpleType{Kind: workitem.KindString},
		},
	})
	require.Nil(s.T(), err)
	reqLong := &http.Request{Host: "api.service.domain.org"}
	witData := ConvertWorkItemTypeFromModel(reqLong, wit)
	return &app.WorkItemTypeSingle{
		Data: &witData,
	}
}

//-----------------------------------------------------------------------------
// Test on work item types retrieval (single and list)
//-----------------------------------------------------------------------------

func (s *workItemTypesSuite) TestValidate() {
	// given
	desc := "Description for 'person'"
	id := personID
	reqLong := &http.Request{Host: "api.service.domain.org"}
	//spaceSelfURL := rest.AbsoluteURL(reqLong, app.SpaceHref(space.SystemSpace.String()))
	spaceTemplateID := spacetemplate.SystemLegacyTemplateID
	spaceTemplateSelfURL := rest.AbsoluteURL(reqLong, app.SpaceTemplateHref(spaceTemplateID.String()))
	payload := app.WorkItemTypeSingle{
		Data: &app.WorkItemTypeData{
			ID:   &id,
			Type: "workitemtypes",
			Attributes: &app.WorkItemTypeAttributes{
				Name:        "",
				Description: &desc,
				Icon:        "fa-user",
				Fields: map[string]*app.FieldDefinition{
					"name": {
						Required:    true,
						Description: "Description for name field",
						Label:       "Name",
						Type: &app.FieldType{
							Kind: "string",
						},
					},
				},
			},
			Relationships: &app.WorkItemTypeRelationships{
				SpaceTemplate: app.NewSpaceTemplateRelation(spaceTemplateID, spaceTemplateSelfURL),
			},
		},
	}

	s.T().Run("valid", func(t *testing.T) {
		// given
		p := payload
		p.Data.Attributes.Name = "Valid Name 0baa42b5-fa52-4ee2-847d-ef26b23fbb6e"
		// when
		err := p.Validate()
		// then
		require.NoError(t, err)
	})

	s.T().Run("invalid - oversized name", func(t *testing.T) {
		// given
		p := payload
		p.Data.Attributes.Name = testsupport.TestOversizedNameObj
		// when
		err := p.Validate()
		// then
		require.Error(t, err)
		gerr, ok := err.(*goa.ErrorResponse)
		require.True(t, ok)
		gerr.ID = "IGNORE_ME"
		compareWithGolden(t, filepath.Join(s.testDir, "validate", "invalid_oversized_name.golden.json"), gerr)
	})

	s.T().Run("invalid - name starts with underscore", func(t *testing.T) {
		// given
		p := payload
		p.Data.Attributes.Name = "_person"
		// when
		err := p.Validate()
		// then
		require.Error(t, err)
		gerr, ok := err.(*goa.ErrorResponse)
		require.True(t, ok)
		gerr.ID = "IGNORE_ME"
		compareWithGolden(t, filepath.Join(s.testDir, "validate", "invalid_name_starts_with_underscore.golden.json"), gerr)
	})
}

func (s *workItemTypesSuite) TestList() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypes(3, tf.SetWorkItemTypeNames("task", "bug", "feature")))
		// when
		// Paging in the format <start>,<limit>"
		page := "0,-1"
		res, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typesCtrl, fxt.SpaceTemplates[0].ID, &page, nil, nil)
		// then
		require.NotNil(t, witCollection)
		toBeFound := id.Slice{fxt.WorkItemTypes[0].ID, fxt.WorkItemTypes[1].ID, fxt.WorkItemTypes[2].ID}.ToMap()
		for _, wit := range witCollection.Data {
			delete(toBeFound, *wit.ID)
		}
		require.Empty(t, toBeFound, "failed to find these work item types: %+v", toBeFound)

		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok.res.payload.golden.json"), witCollection)

		require.NotNil(t, res.Header()[app.LastModified])
		assert.Equal(t, app.ToHTTPTime(fxt.WorkItemTypes[0].UpdatedAt), res.Header()[app.LastModified][0])
		require.NotNil(t, res.Header()[app.CacheControl])
		assert.NotNil(t, res.Header()[app.CacheControl][0])
		require.NotNil(t, res.Header()[app.ETag])
		assert.Equal(t, generateWorkItemTypesTag(*witCollection), res.Header()[app.ETag][0])
	})

	s.T().Run("ok - using expired IfModifiedSince header", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypes(1, tf.SetWorkItemTypeNames("bug")))
		// when
		// Paging in the format <start>,<limit>"
		lastModified := app.ToHTTPTime(time.Now().Add(-1 * time.Hour))
		page := "0,-1"
		res, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typesCtrl, fxt.SpaceTemplates[0].ID, &page, &lastModified, nil)
		// then
		require.NotNil(t, witCollection)
		require.NotNil(t, witCollection.Data[0].ID)
		require.Equal(t, fxt.WorkItemTypes[0].ID, *witCollection.Data[0].ID)
		require.NotNil(t, res.Header()[app.LastModified])
		assert.Equal(t, app.ToHTTPTime(fxt.WorkItemTypes[0].UpdatedAt), res.Header()[app.LastModified][0])
		require.NotNil(t, res.Header()[app.CacheControl])
		assert.NotNil(t, res.Header()[app.CacheControl][0])
		require.NotNil(t, res.Header()[app.ETag])
		assert.Equal(t, generateWorkItemTypesTag(*witCollection), res.Header()[app.ETag][0])
	})

	s.T().Run("ok - using IfNoneMatch header", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypes(3))
		// when
		// Paging in the format <start>,<limit>"
		etag := "foo"
		page := "0,-1"
		res, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typesCtrl, fxt.SpaceTemplates[0].ID, &page, nil, &etag)
		// then
		require.NotNil(t, witCollection)
		toBeFound := id.Slice{fxt.WorkItemTypes[0].ID, fxt.WorkItemTypes[1].ID, fxt.WorkItemTypes[2].ID}.ToMap()
		for _, wit := range witCollection.Data {
			delete(toBeFound, *wit.ID)
		}
		require.Empty(t, toBeFound, "failed to find these work item types: %+v", toBeFound)

		require.NotNil(t, res.Header()[app.LastModified])
		assert.Equal(t, app.ToHTTPTime(fxt.WorkItemTypes[0].UpdatedAt), res.Header()[app.LastModified][0])
		require.NotNil(t, res.Header()[app.CacheControl])
		assert.NotNil(t, res.Header()[app.CacheControl][0])
		require.NotNil(t, res.Header()[app.ETag])
		assert.Equal(t, generateWorkItemTypesTag(*witCollection), res.Header()[app.ETag][0])
	})

	s.T().Run("not modified - using IfModifiedSince header", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypes(1))
		// when/then
		// Paging in the format <start>,<limit>"
		lastModified := app.ToHTTPTime(fxt.WorkItemTypes[0].UpdatedAt)
		page := "0,-1"
		test.ListWorkitemtypesNotModified(t, nil, nil, s.typesCtrl, fxt.SpaceTemplates[0].ID, &page, &lastModified, nil)
	})

	s.T().Run("not modified - using IfNoneMatch header", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypes(1))
		// Paging in the format <start>,<limit>"
		page := "0,-1"
		_, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typesCtrl, fxt.SpaceTemplates[0].ID, &page, nil, nil)
		require.NotNil(t, witCollection)
		// when/then
		ifNoneMatch := generateWorkItemTypesTag(*witCollection)
		test.ListWorkitemtypesNotModified(t, nil, nil, s.typesCtrl, fxt.SpaceTemplates[0].ID, &page, nil, &ifNoneMatch)
	})
}

//-----------------------------------------------------------------------------
// Test on work item type authorization
//-----------------------------------------------------------------------------

// This test case will check authorized access to Create/Update/Delete APIs
func (s *workItemTypesSuite) TestUnauthorizeWorkItemTypeCreate() {
	UnauthorizeCreateUpdateDeleteTest(s.T(), s.getWorkItemTypeTestDataFunc(), func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWIT-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkitemtypeController(service, gormapplication.NewGormDB(s.DB), s.Configuration)
		app.MountWorkitemtypeController(service, controller)
		return nil
	})
}

func (s *workItemTypesSuite) getWorkItemTypeTestDataFunc() func(*testing.T) []testSecureAPI {
	return func(t *testing.T) []testSecureAPI {
		return []testSecureAPI{
			// Try fetching a random work Item Type
			// We do not have security on GET hence this should return 404 not found
			{
				method:             http.MethodGet,
				url:                fmt.Sprintf(endpointWorkItemTypes, space.SystemSpace.String()) + "/2e889d4e-49a9-463b-8cd4-6a3a95155103",
				expectedStatusCode: http.StatusNotFound,
				expectedErrorCode:  jsonapi.ErrorCodeNotFound,
				payload:            nil,
				jwtToken:           "",
			}, {
				method:             http.MethodGet,
				url:                fmt.Sprintf(endpointWorkItemTypesSourceLinkTypes, space.SystemSpace, "2e889d4e-49a9-463b-8cd4-6a3a95155103"),
				expectedStatusCode: http.StatusNotFound,
				expectedErrorCode:  jsonapi.ErrorCodeNotFound,
				payload:            nil,
				jwtToken:           "",
			}, {
				method:             http.MethodGet,
				url:                fmt.Sprintf(endpointWorkItemTypesTargetLinkTypes, space.SystemSpace, "2e889d4e-49a9-463b-8cd4-6a3a95155103"),
				expectedStatusCode: http.StatusNotFound,
				expectedErrorCode:  jsonapi.ErrorCodeNotFound,
				payload:            nil,
				jwtToken:           "",
			},
		}
	}
}
