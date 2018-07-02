package controller_test

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type testSpaceTemplateSuite struct {
	gormtestsupport.DBTestSuite
	ctx     context.Context
	testDir string
}

func TestSpaceTemplateSuite(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &testSpaceTemplateSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *testSpaceTemplateSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.testDir = filepath.Join("test-files", "space_templates")
}

func (s *testSpaceTemplateSuite) SecuredController() (*goa.Service, *SpaceTemplateController) {
	svc := testsupport.ServiceAsUser("SpaceTemplate-Service", testsupport.TestIdentity)
	return svc, NewSpaceTemplateController(svc, s.GormDB, s.Configuration)
}

func (s *testSpaceTemplateSuite) TestSpaceTemplate_Show() {
	s.T().Run("non-existing template", func(t *testing.T) {
		// given
		svc, ctrl := s.SecuredController()
		// when
		_, jerr := test.ShowSpaceTemplateNotFound(s.T(), svc.Context, svc, ctrl, uuid.NewV4(), nil, nil)
		// then
		require.NotNil(t, jerr)
		require.Equal(t, 1, len(jerr.Errors))
		require.NotNil(t, jerr.Errors[0].Status)
		require.Equal(t, strconv.Itoa(http.StatusNotFound), *jerr.Errors[0].Status)
	})

	s.T().Run("ok", func(t *testing.T) {
		// given
		svc, ctrl := s.SecuredController()
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1))

		testData := map[string]uuid.UUID{
			"generated_template": fxt.SpaceTemplates[0].ID,
			"scrum_template":     spacetemplate.SystemScrumTemplateID,
			"base_template":      spacetemplate.SystemBaseTemplateID,
			"legacy_template":    spacetemplate.SystemLegacyTemplateID,
			"agile_template":     spacetemplate.SystemAgileTemplateID,
		}
		// when
		for name, spaceTemplateID := range testData {
			t.Run(name, func(t *testing.T) {
				res, actual := test.ShowSpaceTemplateOK(t, svc.Context, svc, ctrl, spaceTemplateID, nil, nil)
				// then
				safeOverriteHeader(t, res, app.ETag, "m2MLfQTqVSfIsr8Dt9pjMQ==")
				compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", fmt.Sprintf("ok_%s.res.payload.golden.json", name)), actual)
				compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", fmt.Sprintf("ok_%s.res.headers.golden.json", name)), res.Header())
				require.NotNil(t, actual)
				assertResponseHeaders(t, res)
				actualModel := convertSpaceTemplateSingleToModel(t, *actual)
				loadedTempl, err := spacetemplate.NewRepository(s.DB).Load(s.Ctx, spaceTemplateID)
				require.NoError(t, err)
				require.True(t, loadedTempl.Equal(actualModel))
			})
		}
	})

	s.T().Run("existing template (using expired If-Modified-Since header)", func(t *testing.T) {
		// given
		svc, ctrl := s.SecuredController()
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1))
		ifModifiedSince := app.ToHTTPTime(fxt.SpaceTemplates[0].UpdatedAt.Add(-1 * time.Hour))
		// when
		res, actual := test.ShowSpaceTemplateOK(t, svc.Context, svc, ctrl, fxt.SpaceTemplates[0].ID, &ifModifiedSince, nil)
		// then
		require.NotNil(t, actual)
		safeOverriteHeader(t, res, app.ETag, "m2MLfQTqVSfIsr8Dt9pjMQ==")
		assertResponseHeaders(t, res)
		actualModel := convertSpaceTemplateSingleToModel(t, *actual)
		require.True(t, fxt.SpaceTemplates[0].Equal(actualModel))
	})

	s.T().Run("not modified (using If-Modified-Since header)", func(t *testing.T) {
		// given
		svc, ctrl := s.SecuredController()
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1))
		ifModifiedSince := app.ToHTTPTime(fxt.SpaceTemplates[0].UpdatedAt)
		// when
		res := test.ShowSpaceTemplateNotModified(t, svc.Context, svc, ctrl, fxt.SpaceTemplates[0].ID, &ifModifiedSince, nil)
		// then
		safeOverriteHeader(t, res, app.ETag, "m2MLfQTqVSfIsr8Dt9pjMQ==")
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified (using If-None-Match header)", func(t *testing.T) {
		// given
		svc, ctrl := s.SecuredController()
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1))
		ifNoneMatch := app.GenerateEntityTag(fxt.SpaceTemplates[0])
		// when
		res := test.ShowSpaceTemplateNotModified(t, svc.Context, svc, ctrl, fxt.SpaceTemplates[0].ID, nil, &ifNoneMatch)
		// then
		safeOverriteHeader(t, res, app.ETag, "m2MLfQTqVSfIsr8Dt9pjMQ==")
		assertResponseHeaders(t, res)
	})
}

func (s *testSpaceTemplateSuite) TestSpaceTemplate_List() {
	// given
	svc, ctrl := s.SecuredController()
	checkToBeFound := func(t *testing.T, fxt *tf.TestFixture, spaceTemplateList *app.SpaceTemplateList) {
		require.NotNil(t, spaceTemplateList)
		toBeFound := id.MapFromSlice(id.Slice{
			fxt.SpaceTemplates[0].ID,
			fxt.SpaceTemplates[1].ID,
			fxt.SpaceTemplates[2].ID,
			fxt.SpaceTemplates[3].ID,
		})
		for _, st := range spaceTemplateList.Data {
			delete(toBeFound, *st.ID)
		}
		require.Empty(t, toBeFound, "not all space templates created in this test were found: %s", toBeFound)
	}

	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.SpaceTemplates(4))
		// when
		res, spaceTemplateList := test.ListSpaceTemplateOK(t, svc.Context, svc, ctrl, nil, nil)
		// then
		checkToBeFound(t, fxt, spaceTemplateList)
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified (using expired If-Modified-Since header)", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.SpaceTemplates(4))
		ifModifiedSince := app.ToHTTPTime(fxt.SpaceTemplates[0].UpdatedAt.Add(-1 * time.Hour))
		// when
		res, spaceTemplateList := test.ListSpaceTemplateOK(t, svc.Context, svc, ctrl, &ifModifiedSince, nil)
		// then
		checkToBeFound(t, fxt, spaceTemplateList)
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified (using If-Modified-Since header)", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.SpaceTemplates(4))
		ifModifiedSince := app.ToHTTPTime(fxt.SpaceTemplates[0].UpdatedAt)
		// when
		res := test.ListSpaceTemplateNotModified(t, svc.Context, svc, ctrl, &ifModifiedSince, nil)
		// then
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified (using If-None-Match header)", func(t *testing.T) {
		// given
		_ = tf.NewTestFixture(s.T(), s.DB, tf.SpaceTemplates(4))
		_, spaceTemplateList := test.ListSpaceTemplateOK(t, svc.Context, svc, ctrl, nil, nil)
		arr := make([]app.ConditionalRequestEntity, len(spaceTemplateList.Data))
		for i, v := range spaceTemplateList.Data {
			arr[i] = convertSpaceTemplateToModel(t, *v)
		}
		ifNoneMatch := app.GenerateEntitiesTag(arr)
		// when
		res := test.ListSpaceTemplateNotModified(t, svc.Context, svc, ctrl, nil, &ifNoneMatch)
		// then
		assertResponseHeaders(t, res)
	})
}

func convertSpaceTemplateSingleToModel(t *testing.T, appSpaceTemplate app.SpaceTemplateSingle) spacetemplate.SpaceTemplate {
	return convertSpaceTemplateToModel(t, *appSpaceTemplate.Data)
}

func convertSpaceTemplateToModel(t *testing.T, appSpaceTemplate app.SpaceTemplate) spacetemplate.SpaceTemplate {
	var desc string
	if appSpaceTemplate.Attributes.Description != nil {
		desc = *appSpaceTemplate.Attributes.Description
	}

	// bs, err := base64.StdEncoding.DecodeString(*appSpaceTemplate.Attributes.Template)
	// require.Nil(t, err, "failed to decode template from base64")

	return spacetemplate.SpaceTemplate{
		ID:           *appSpaceTemplate.ID,
		Name:         *appSpaceTemplate.Attributes.Name,
		Description:  &desc,
		CanConstruct: *appSpaceTemplate.Attributes.CanConstruct,
		// Template:    string(bs),
		Version: *appSpaceTemplate.Attributes.Version,
		Lifecycle: gormsupport.Lifecycle{
			UpdatedAt: *appSpaceTemplate.Attributes.UpdatedAt,
			CreatedAt: *appSpaceTemplate.Attributes.CreatedAt,
		},
	}
}
