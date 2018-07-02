package controller_test

import (
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"

	"time"

	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type workItemTypesSuite struct {
	gormtestsupport.DBTestSuite
	typeCtrl     *WorkitemtypesController
	linkTypeCtrl *WorkItemLinkTypeController
	linkCatCtrl  *WorkItemLinkCategoryController
	spaceCtrl    *SpaceController
	svc          *goa.Service
	testDir      string
}

func TestSuiteWorkItemTypes(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemTypesSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *workItemTypesSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.testDir = filepath.Join("test-files", "work_item_type")
}

func (s *workItemTypesSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	idn := &account.Identity{
		ID:           uuid.Nil,
		Username:     "TestDeveloper",
		ProviderType: "test provider",
	}
	s.svc = testsupport.ServiceAsUser("workItemLinkSpace-Service", *idn)
	s.spaceCtrl = NewSpaceController(s.svc, s.GormDB, s.Configuration, &DummyResourceManager{})
	require.NotNil(s.T(), s.spaceCtrl)
	s.typeCtrl = NewWorkitemtypesController(s.svc, s.GormDB, s.Configuration)
	s.linkTypeCtrl = NewWorkItemLinkTypeController(s.svc, s.GormDB, s.Configuration)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(s.svc, s.GormDB)
}

func (s *workItemTypesSuite) TestList() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItemTypes(2))

	s.T().Run("not found using non existing space", func(t *testing.T) {
		// given
		spaceID := uuid.NewV4()
		// when
		res, jerrs := test.ListWorkitemtypesNotFound(t, nil, nil, s.typeCtrl, spaceID, nil, nil)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_found_using_non_existing_space.res.payload.golden.json"), jerrs)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_found_using_non_existing_space.res.headers.golden.json"), res.Header())
	})

	s.T().Run("ok", func(t *testing.T) {
		t.Run("generated", func(t *testing.T) {
			// when
			res, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typeCtrl, fxt.SpaceTemplates[0].ID, nil, nil)
			// then
			require.NotNil(t, witCollection)
			require.Nil(t, witCollection.Validate())

			toBeFound := id.Slice{fxt.WorkItemTypes[0].ID, fxt.WorkItemTypes[1].ID}.ToMap()
			for _, wit := range witCollection.Data {
				_, ok := toBeFound[*wit.ID]
				assert.True(t, ok, "failed to find work item type %s in expected list", *wit.ID)
				delete(toBeFound, *wit.ID)
			}
			require.Empty(t, toBeFound, "failed to find these expected work item types: %v", toBeFound)

			require.NotNil(t, res.Header()[app.LastModified])
			assert.Equal(t, app.ToHTTPTime(fxt.WorkItemTypes[1].UpdatedAt), res.Header()[app.LastModified][0])
			require.NotNil(t, res.Header()[app.CacheControl])
			assert.NotNil(t, res.Header()[app.CacheControl][0])
			require.NotNil(t, res.Header()[app.ETag])
			assert.Equal(t, generateWorkItemTypesTag(*witCollection), res.Header()[app.ETag][0])

			compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok.res.payload.golden.json"), witCollection)
			safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
			compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok.res.headers.golden.json"), res.Header())
			assertResponseHeaders(t, res)
		})
		// Test pre-defined work item types
		spaceTemplates := map[string]uuid.UUID{
			"base":   spacetemplate.SystemBaseTemplateID,
			"scrum":  spacetemplate.SystemScrumTemplateID,
			"legacy": spacetemplate.SystemLegacyTemplateID,
			"agile":  spacetemplate.SystemAgileTemplateID,
		}
		for name, ID := range spaceTemplates {
			t.Run(name, func(t *testing.T) {
				// when
				res, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typeCtrl, ID, nil, nil)
				// then
				require.NotNil(t, witCollection)
				compareWithGoldenAgnosticTime(t, filepath.Join(s.testDir, "list", fmt.Sprintf("ok_%s.res.payload.golden.json", name)), witCollection)
				safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
				compareWithGoldenAgnosticTime(t, filepath.Join(s.testDir, "list", fmt.Sprintf("ok_%s.res.headers.golden.json", name)), res.Header())
				assertResponseHeaders(t, res)
			})
		}
	})

	s.T().Run("ok - using expired IfModifiedSince header", func(t *testing.T) {
		// when
		lastModified := app.ToHTTPTime(time.Now().Add(-1 * time.Hour))
		res, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typeCtrl, fxt.SpaceTemplates[0].ID, &lastModified, nil)
		// then
		require.NotNil(t, witCollection)
		require.Nil(t, witCollection.Validate())

		toBeFound := id.Slice{fxt.WorkItemTypes[0].ID, fxt.WorkItemTypes[1].ID}.ToMap()
		for _, wit := range witCollection.Data {
			_, ok := toBeFound[*wit.ID]
			assert.True(t, ok, "failed to find work item type %s in expected list", *wit.ID)
			delete(toBeFound, *wit.ID)
		}
		require.Empty(t, toBeFound, "failed to find these expected work item types: %v", toBeFound)

		require.NotNil(t, res.Header()[app.LastModified])
		assert.Equal(t, app.ToHTTPTime(fxt.WorkItemTypes[1].UpdatedAt), res.Header()[app.LastModified][0])
		require.NotNil(t, res.Header()[app.CacheControl])
		assert.NotNil(t, res.Header()[app.CacheControl][0])
		require.NotNil(t, res.Header()[app.ETag])
		assert.Equal(t, generateWorkItemTypesTag(*witCollection), res.Header()[app.ETag][0])

		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok_using_expired_ifmodifiedsince_header.res.payload.golden.json"), witCollection)
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok_using_expired_ifmodifiedsince_header.res.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("ok - using IfNoneMatch header", func(t *testing.T) {
		// when
		etag := "foo"
		res, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typeCtrl, fxt.SpaceTemplates[0].ID, nil, &etag)
		// then
		require.NotNil(t, witCollection)
		require.Nil(t, witCollection.Validate())

		toBeFound := id.Slice{fxt.WorkItemTypes[0].ID, fxt.WorkItemTypes[1].ID}.ToMap()
		for _, wit := range witCollection.Data {
			_, ok := toBeFound[*wit.ID]
			assert.True(t, ok, "failed to find work item type %s in expected list", *wit.ID)
			delete(toBeFound, *wit.ID)
		}
		require.Empty(t, toBeFound, "failed to find these expected work item types: %v", toBeFound)

		require.NotNil(t, res.Header()[app.LastModified])
		assert.Equal(t, app.ToHTTPTime(fxt.WorkItemTypes[1].UpdatedAt), res.Header()[app.LastModified][0])
		require.NotNil(t, res.Header()[app.CacheControl])
		assert.NotNil(t, res.Header()[app.CacheControl][0])
		require.NotNil(t, res.Header()[app.ETag])
		assert.Equal(t, generateWorkItemTypesTag(*witCollection), res.Header()[app.ETag][0])
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok_using_expired_ifnonematch_header.res.payload.golden.json"), witCollection)
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok_using_expired_ifnonematch_header.res.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified - using IfModifiedSince header", func(t *testing.T) {
		// given
		lastModified := app.ToHTTPTime(fxt.WorkItemTypes[1].UpdatedAt)
		// when
		res := test.ListWorkitemtypesNotModified(t, nil, nil, s.typeCtrl, fxt.SpaceTemplates[0].ID, &lastModified, nil)
		// then
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_modified_using_ifmodifiedsince_header.res.headers.golden.json"), res.Header())
	})

	s.T().Run("not modified - using IfNoneMatch header", func(t *testing.T) {
		// given
		_, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typeCtrl, fxt.SpaceTemplates[0].ID, nil, nil)
		require.NotNil(t, witCollection)
		// when
		ifNoneMatch := generateWorkItemTypesTag(*witCollection)
		res := test.ListWorkitemtypesNotModified(t, nil, nil, s.typeCtrl, fxt.SpaceTemplates[0].ID, nil, &ifNoneMatch)
		// then
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_modified_using_ifnonematch_header.res.headers.golden.json"), res.Header())
	})
}

func (s *workItemTypesSuite) TestValidate() {
	// given
	desc := "Description for 'person'"
	id := uuid.NewV4()
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
