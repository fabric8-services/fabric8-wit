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
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"time"

	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//-----------------------------------------------------------------------------
// Test Suite setup
//-----------------------------------------------------------------------------

// The WorkItemTypeTestSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type workItemTypeSuite struct {
	gormtestsupport.DBTestSuite
	typeCtrl     *WorkitemtypeController
	linkTypeCtrl *WorkItemLinkTypeController
	linkCatCtrl  *WorkItemLinkCategoryController
	spaceCtrl    *SpaceController
	svc          *goa.Service
	testDir      string
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemType(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemTypeSuite{
		DBTestSuite: gormtestsupport.NewDBTestSuite(""),
	})
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemTypeSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.testDir = filepath.Join("test-files", "work_item_type")
}

var (
	animalID = uuid.FromStringOrNil("729431f2-bca4-4062-9087-c751807b569f")
	personID = uuid.FromStringOrNil("22a1e4f1-7e9d-4ce8-ac87-fe7c79356b16")
)

// The SetupTest method will be run before every test in the suite.
func (s *workItemTypeSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	idn := &account.Identity{
		ID:           uuid.Nil,
		Username:     "TestDeveloper",
		ProviderType: "test provider",
	}
	s.svc = testsupport.ServiceAsUser("workItemLinkSpace-Service", *idn)
	s.spaceCtrl = NewSpaceController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration, &DummyResourceManager{})
	require.NotNil(s.T(), s.spaceCtrl)
	s.typeCtrl = NewWorkitemtypeController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.linkTypeCtrl = NewWorkItemLinkTypeController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(s.svc, gormapplication.NewGormDB(s.DB))

}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

// createWorkItemTypeAnimal defines a work item type "animal" that consists of
// two fields ("animal-type" and "color"). The type is mandatory but the color is not.
func (s *workItemTypeSuite) createWorkItemTypeAnimal() *app.WorkItemTypeSingle {
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
	}, true)
	require.Nil(s.T(), err)
	reqLong := &http.Request{Host: "api.service.domain.org"}
	witData := ConvertWorkItemTypeFromModel(reqLong, wit)
	return &app.WorkItemTypeSingle{
		Data: &witData,
	}
}

// createWorkItemTypePerson defines a work item type "person" that consists of
// a required "name" field.
func (s *workItemTypeSuite) createWorkItemTypePerson() *app.WorkItemTypeSingle {
	witRepo := workitem.NewWorkItemTypeRepository(s.DB)
	wit, err := witRepo.Create(context.Background(), spacetemplate.SystemLegacyTemplateID, &personID, nil, "person", ptr.String("Description for 'person'"), "fa-user", workitem.FieldDefinitions{
		"name": {
			Required:    true,
			Description: "Description for Name field",
			Label:       "Name",
			Type:        &workitem.SimpleType{Kind: workitem.KindString},
		},
	}, true)
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

func (s *workItemTypeSuite) TestValidate() {
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
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "validate", "invalid_oversized_name.golden.json"), gerr)
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
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "validate", "invalid_name_starts_with_underscore.golden.json"), gerr)
	})
}

func (s *workItemTypeSuite) TestShow() {

	// given
	wit := s.createWorkItemTypeAnimal()
	require.NotNil(s.T(), wit)
	require.NotNil(s.T(), wit.Data)
	require.NotNil(s.T(), wit.Data.ID)

	s.T().Run("ok", func(t *testing.T) {
		// when
		res, actual := test.ShowWorkitemtypeOK(t, nil, nil, s.typeCtrl, *wit.Data.ID, nil, nil)
		// then
		require.NotNil(t, actual)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.wit.golden.json"), actual)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.headers.golden.json"), res.Header())
	})

	s.T().Run("ok - using expired IfModifiedSince header", func(t *testing.T) {
		// when
		lastModified := app.ToHTTPTime(wit.Data.Attributes.CreatedAt.Add(-1 * time.Hour))
		res, actual := test.ShowWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, *wit.Data.ID, &lastModified, nil)
		// then
		require.NotNil(t, actual)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_lastmodified_header.wit.golden.json"), actual)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_lastmodified_header.headers.golden.json"), res.Header())
	})

	s.T().Run("ok - using IfNoneMatch header", func(t *testing.T) {
		// when
		ifNoneMatch := "foo"
		res, actual := test.ShowWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, *wit.Data.ID, nil, &ifNoneMatch)
		// then
		require.NotNil(t, actual)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_etag_header.wit.golden.json"), actual)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_etag_header.headers.golden.json"), res.Header())
	})

	s.T().Run("not modified - using IfModifiedSince header", func(t *testing.T) {
		// when
		lastModified := app.ToHTTPTime(time.Now().Add(119 * time.Second))
		res := test.ShowWorkitemtypeNotModified(s.T(), nil, nil, s.typeCtrl, *wit.Data.ID, &lastModified, nil)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_modified_using_if_modified_since_header.headers.golden.json"), res.Header())
	})

	s.T().Run("not modified - using IfNoneMatch header", func(t *testing.T) {
		// when
		etag := generateWorkItemTypeTag(*wit)
		res := test.ShowWorkitemtypeNotModified(s.T(), nil, nil, s.typeCtrl, *wit.Data.ID, nil, &etag)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_modified_using_ifnonematch_header.headers.golden.json"), res.Header())
	})
}

// used for testing purpose only
func convertWorkItemTypeToModel(data app.WorkItemTypeData) workitem.WorkItemType {
	return workitem.WorkItemType{
		ID:      *data.ID,
		Version: *data.Attributes.Version,
	}
}

func generateWorkItemTypesTag(entities app.WorkItemTypeList) string {
	modelEntities := make([]app.ConditionalRequestEntity, len(entities.Data))
	for i, entityData := range entities.Data {
		modelEntities[i] = convertWorkItemTypeToModel(*entityData)
	}
	return app.GenerateEntitiesTag(modelEntities)
}

func generateWorkItemTypeTag(entity app.WorkItemTypeSingle) string {
	return app.GenerateEntityTag(convertWorkItemTypeToModel(*entity.Data))
}

func generateWorkItemLinkTypesTag(entities app.WorkItemLinkTypeList) string {
	modelEntities := make([]app.ConditionalRequestEntity, len(entities.Data))
	for i, entityData := range entities.Data {
		e, _ := ConvertWorkItemLinkTypeToModel(app.WorkItemLinkTypeSingle{Data: entityData})
		modelEntities[i] = e
	}
	return app.GenerateEntitiesTag(modelEntities)
}

func generateWorkItemLinkTypeTag(entity app.WorkItemLinkTypeSingle) string {
	e, _ := ConvertWorkItemLinkTypeToModel(entity)
	return app.GenerateEntityTag(e)
}

func convertWorkItemTypesToConditionalEntities(workItemTypeList app.WorkItemTypeList) []app.ConditionalRequestEntity {
	conditionalWorkItemTypes := make([]app.ConditionalRequestEntity, len(workItemTypeList.Data))
	for i, data := range workItemTypeList.Data {
		conditionalWorkItemTypes[i] = convertWorkItemTypeToModel(*data)
	}
	return conditionalWorkItemTypes
}

func getWorkItemLinkTypeUpdatedAt(appWorkItemLinkType app.WorkItemLinkTypeSingle) time.Time {
	return *appWorkItemLinkType.Data.Attributes.UpdatedAt
}

//-----------------------------------------------------------------------------
// Test on work item type authorization
//-----------------------------------------------------------------------------

// This test case will check authorized access to Create/Update/Delete APIs
func (s *workItemTypeSuite) TestUnauthorizeWorkItemTypeCreate() {
	UnauthorizeCreateUpdateDeleteTest(s.T(), s.getWorkItemTypeTestDataFunc(), func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWIT-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkitemtypeController(service, gormapplication.NewGormDB(s.DB), s.Configuration)
		app.MountWorkitemtypeController(service, controller)
		return nil
	})
}

func (s *workItemTypeSuite) getWorkItemTypeTestDataFunc() func(*testing.T) []testSecureAPI {
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
