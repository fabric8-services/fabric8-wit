package controller_test

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestSpaceTemplateSuite struct {
	gormtestsupport.DBTestSuite
	db  *gormapplication.GormDB
	ctx context.Context
}

func TestRunSpaceTemplateSuite(t *testing.T) {
	resource.Require(t, resource.Database)
	pwd, err := os.Getwd()
	if err != nil {
		require.Nil(t, err)
	}
	suite.Run(t, &TestSpaceTemplateSuite{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

func (s *TestSpaceTemplateSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
}

func (s *TestSpaceTemplateSuite) SetupTest() {
	s.db = gormapplication.NewGormDB(s.DB)
}

func (s *TestSpaceTemplateSuite) SecuredController() (*goa.Service, *SpaceTemplateController) {
	//func securedSpaceTemplateController(db *gormapplication.GormDB, cfg *config.ConfigurationData) (*goa.Service, *SpaceTemplateController) {
	svc := testsupport.ServiceAsUser("SpaceTemplate-Service", testsupport.TestIdentity)
	return svc, NewSpaceTemplateController(svc, s.db, s.Configuration)

	// pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	// //priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	// svc := testsupport.ServiceAsUser("SpaceTemplate-Service", almtoken.NewManager(pub), testsupport.TestIdentity)
	// return svc, NewSpaceTemplateController(svc, db, cfg)
}

// func (s *TestSpaceTemplateSuite) unsecuredSpaceTemplateController() (*goa.Service, *SpaceTemplateController) {
// 	//func unsecuredSpaceTemplateController(db *gormapplication.GormDB, cfg *config.ConfigurationData) (*goa.Service, *SpaceTemplateController) {

// 	svc := testsupport.ServiceAsUser("SpaceTemplate-Service", identity)
// 	return svc, NewSpaceTemplateController(svc, s.db, s.Configuration)
// }

func (s *TestSpaceTemplateSuite) UnSecuredController() (*goa.Service, *SpaceTemplateController) {
	svc := goa.New("SpaceTemplate-Service")
	return svc, NewSpaceTemplateController(svc, s.db, s.Configuration)
}

// func (s *TestSpaceTemplateSuite) createMinimumSpaceTemplate(t *testing.T) *app.SpaceTemplateSingle {
// 	return createMinimumSpaceTemplate(t, s.db, s.Configuration)
// }

// func createMinimumSpaceTemplate(t *testing.T, db *gormapplication.GormDB, cfg *config.ConfigurationData) *app.SpaceTemplateSingle {
// 	// given
// 	var res *app.SpaceTemplateSingle
// 	var appSpaceTemplate *app.SpaceTemplate
// 	svc, ctrl := securedSpaceTemplateController(db, cfg)
// 	req := &goa.RequestData{
// 		Request: &http.Request{Host: "api.service.domain.org"},
// 	}
// 	desc := "a random description"
// 	spaceTemplate := spacetemplate.SpaceTemplate{
// 		ID:          uuid.NewV4(),
// 		Name:        testsupport.CreateRandomValidTestName("MinimumSpaceTemplate-"),
// 		Description: &desc,
// 		Template:    spacetemplate.GetValidEmptyTemplate(),
// 	}
// 	// when
// 	application.Transactional(db, func(appl application.Application) error {
// 		appSpaceTemplate = ConvertSpaceTemplate(appl, req, spaceTemplate)
// 		payload := &app.CreateSpaceTemplatePayload{
// 			Data: appSpaceTemplate,
// 		}
// 		_, res = test.CreateSpaceTemplateCreated(t, svc.Context, svc, ctrl, payload)
// 		return nil
// 	})
// 	// then
// 	require.NotNil(t, res)
// 	require.NotNil(t, res.Data)
// 	require.NotNil(t, res.Data.ID)
// 	require.Equal(t, spaceTemplate.ID, *res.Data.ID)

// 	require.NotNil(t, res.Data.Attributes)
// 	require.NotNil(t, res.Data.Attributes.Name)
// 	require.Equal(t, spaceTemplate.Name, *res.Data.Attributes.Name)

// 	require.NotNil(t, *res.Data.Attributes.Description)
// 	require.Equal(t, *spaceTemplate.Description, *res.Data.Attributes.Description)

// 	require.NotNil(t, *res.Data.Attributes.Template)
// 	require.Equal(t, *appSpaceTemplate.Attributes.Template, *res.Data.Attributes.Template)
// 	return res
// }

// func (s *TestSpaceTemplateSuite) TestSpaceTemplate_Create() {
// 	req := &goa.RequestData{
// 		Request: &http.Request{Host: "api.service.domain.org"},
// 	}

// 	s.T().Run("invalid YAML template", func(t *testing.T) {
// 		t.Parallel()
// 		// given
// 		svc, ctrl := s.SecuredController()
// 		spaceTemplate := spacetemplate.SpaceTemplate{
// 			Name:     testsupport.CreateRandomValidTestName("InvalidYAMLTemplate-"),
// 			Template: "this is not a valid template ;)",
// 		}
// 		// when
// 		payload := &app.CreateSpaceTemplatePayload{
// 			Data: ConvertSpaceTemplate(s.db, req, spaceTemplate),
// 		}
// 		_, jerr := test.CreateSpaceTemplateBadRequest(t, svc.Context, svc, ctrl, payload)
// 		// then
// 		require.NotNil(t, jerr)
// 	})

// 	s.T().Run("empty template (valid YAML)", func(t *testing.T) {
// 		t.Parallel()
// 		// given
// 		svc, ctrl := s.SecuredController()
// 		spaceTemplate := spacetemplate.SpaceTemplate{
// 			Name:     testsupport.CreateRandomValidTestName("EmptyTemplateValidYAML-"),
// 			Template: "test: demo", // this is valid YAML but not a valid template which is fine
// 		}
// 		// when
// 		payload := &app.CreateSpaceTemplatePayload{
// 			Data: ConvertSpaceTemplate(s.db, req, spaceTemplate),
// 		}
// 		_, createdSpaceTemplate := test.CreateSpaceTemplateCreated(t, svc.Context, svc, ctrl, payload)
// 		// then
// 		require.NotNil(t, createdSpaceTemplate)
// 	})

// 	s.T().Run("bare minimum template", func(t *testing.T) {
// 		t.Parallel()
// 		_ = s.createMinimumSpaceTemplate(t)
// 	})

// 	s.T().Run("unauthorized", func(t *testing.T) {
// 		t.Parallel()
// 		// given
// 		svc, ctrl := s.UnSecuredController()
// 		spaceTemplate := spacetemplate.SpaceTemplate{
// 			Name:     testsupport.CreateRandomValidTestName("UnauthorizedTemplateAction-"),
// 			Template: "test: demo", // this is valid YAML but not a valid template which is fine
// 		}
// 		// when
// 		payload := &app.CreateSpaceTemplatePayload{
// 			Data: ConvertSpaceTemplate(s.db, req, spaceTemplate),
// 		}
// 		_, jerrs := test.CreateSpaceTemplateUnauthorized(t, svc.Context, svc, ctrl, payload)
// 		// then
// 		require.NotNil(t, jerrs)
// 	})
// }

// func (s *TestSpaceTemplateSuite) TestSpaceTemplate_ValidateCreatePayload() {
// 	req := &goa.RequestData{
// 		Request: &http.Request{Host: "api.service.domain.org"},
// 	}
// 	spaceTemplate := spacetemplate.SpaceTemplate{
// 		Name:     "foobar",
// 		Template: "work_item_types:",
// 	}

// 	s.T().Run("valid space template create-payload", func(t *testing.T) {
// 		t.Parallel()
// 		// given
// 		payload := &app.CreateSpaceTemplatePayload{
// 			Data: ConvertSpaceTemplate(s.db, req, spaceTemplate),
// 		}
// 		// when
// 		err := payload.Validate()
// 		// then
// 		require.Nil(t, err)
// 	})

// 	s.T().Run("invalid space template create-payload (empty name)", func(t *testing.T) {
// 		t.Parallel()
// 		// given
// 		payload := &app.CreateSpaceTemplatePayload{
// 			Data: ConvertSpaceTemplate(s.db, req, spaceTemplate),
// 		}
// 		emptyName := ""
// 		payload.Data.Attributes.Name = &emptyName
// 		// when
// 		err := payload.Validate()
// 		// then
// 		require.NotNil(t, err)
// 		require.Contains(t, err.Error(), "length of response.name must be greater than or equal to than 1", "failed to detect an empty name string in space template creation payload: %s", *payload.Data.Attributes.Template)
// 	})

// 	s.T().Run("invalid space template create-payload (non-base64 template)", func(t *testing.T) {
// 		t.Parallel()
// 		// given
// 		payload := &app.CreateSpaceTemplatePayload{
// 			Data: ConvertSpaceTemplate(s.db, req, spaceTemplate),
// 		}
// 		invalidBase64Str := "foobar"
// 		payload.Data.Attributes.Template = &invalidBase64Str
// 		// when
// 		err := payload.Validate()
// 		// then
// 		require.NotNil(t, err)
// 		require.Contains(t, err.Error(), "must match the regexp", "failed to detect an invalid base64 template string in space template creation payload: %s", *payload.Data.Attributes.Template)
// 	})

// 	s.T().Run("invalid space template create-payload (empty template)", func(t *testing.T) {
// 		t.Parallel()
// 		// given
// 		payload := &app.CreateSpaceTemplatePayload{
// 			Data: ConvertSpaceTemplate(s.db, req, spaceTemplate),
// 		}
// 		emptyStr := ""
// 		payload.Data.Attributes.Template = &emptyStr
// 		// when
// 		err := payload.Validate()
// 		// then
// 		require.NotNil(t, err)
// 		require.Contains(t, err.Error(), "response.template must be greater than or equal to than 4 but got value \"\"", "failed to detect a too short base64 template string in space template creation payload: %s", *payload.Data.Attributes.Template)
// 	})

// 	s.T().Run("invalid space template create-payload (too long name)", func(t *testing.T) {
// 		t.Parallel()
// 		// given
// 		payload := &app.CreateSpaceTemplatePayload{
// 			Data: ConvertSpaceTemplate(s.db, req, spaceTemplate),
// 		}
// 		tooLongName := strings.Repeat("abcdefg", 10) // 70 chars string
// 		payload.Data.Attributes.Name = &tooLongName
// 		// when
// 		err := payload.Validate()
// 		// then
// 		require.NotNil(t, err)
// 		require.Contains(t, err.Error(), "length of response.name must be less than or equal to than 62")
// 	})

// 	s.T().Run("invalid space template create-payload (name starts with underscore)", func(t *testing.T) {
// 		t.Parallel()
// 		// given
// 		payload := &app.CreateSpaceTemplatePayload{
// 			Data: ConvertSpaceTemplate(s.db, req, spaceTemplate),
// 		}
// 		underscoreName := "_underscoreName"
// 		payload.Data.Attributes.Name = &underscoreName
// 		// when
// 		err := payload.Validate()
// 		// then
// 		require.NotNil(t, err)
// 		require.Contains(t, err.Error(), "response.name must match the regexp")
// 	})

// 	s.T().Run("invalid space template create-payload (name starts with dash)", func(t *testing.T) {
// 		t.Parallel()
// 		// given
// 		payload := &app.CreateSpaceTemplatePayload{
// 			Data: ConvertSpaceTemplate(s.db, req, spaceTemplate),
// 		}
// 		dashName := "-dashName"
// 		payload.Data.Attributes.Name = &dashName
// 		// when
// 		err := payload.Validate()
// 		// then
// 		require.NotNil(t, err)
// 		require.Contains(t, err.Error(), "response.name must match the regexp")
// 	})

// 	s.T().Run("exceeding 1MB template payload", func(t *testing.T) {
// 		t.Parallel()
// 		// given
// 		spaceTemplate := spacetemplate.SpaceTemplate{
// 			Name:     testsupport.CreateRandomValidTestName("EmptyTemplateValidYAML-"),
// 			Template: strings.Repeat("a", 1048576), // this string encoded as base64 will be larger than the max. 1MB
// 		}
// 		payload := &app.CreateSpaceTemplatePayload{
// 			Data: ConvertSpaceTemplate(s.db, req, spaceTemplate),
// 		}
// 		// when
// 		err := payload.Validate()
// 		// then
// 		require.NotNil(t, err)
// 		require.Contains(t, err.Error(), "length of response.template must be less than or equal to than 1048576")
// 	})
// }

func (s *TestSpaceTemplateSuite) TestSpaceTemplate_Show() {
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

	s.T().Run("existing template", func(t *testing.T) {
		// given
		svc, ctrl := s.SecuredController()
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1))
		// when
		res, actual := test.ShowSpaceTemplateOK(t, svc.Context, svc, ctrl, fxt.SpaceTemplates[0].ID, nil, nil)
		// then
		require.NotNil(t, actual)
		assertResponseHeaders(t, res)
		actualModel := convertSpaceTemplateSingleToModel(t, *actual)
		require.True(t, fxt.SpaceTemplates[0].Equal(actualModel))
	})

	s.T().Run("existing template (using expired If-Modified-Since header)", func(t *testing.T) {
		t.Parallel()
		// given
		svc, ctrl := s.SecuredController()
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1))
		ifModifiedSince := app.ToHTTPTime(fxt.SpaceTemplates[0].UpdatedAt.Add(-1 * time.Hour))
		// when
		res, actual := test.ShowSpaceTemplateOK(t, svc.Context, svc, ctrl, fxt.SpaceTemplates[0].ID, &ifModifiedSince, nil)
		// then
		require.NotNil(t, actual)
		assertResponseHeaders(t, res)
		actualModel := convertSpaceTemplateSingleToModel(t, *actual)
		require.True(t, fxt.SpaceTemplates[0].Equal(actualModel))
	})

	s.T().Run("not modified (using If-Modified-Since header)", func(t *testing.T) {
		t.Parallel()
		// given
		svc, ctrl := s.SecuredController()
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1))
		ifModifiedSince := app.ToHTTPTime(fxt.SpaceTemplates[0].UpdatedAt)
		// when
		res := test.ShowSpaceTemplateNotModified(t, svc.Context, svc, ctrl, fxt.SpaceTemplates[0].ID, &ifModifiedSince, nil)
		// then
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified (using If-None-Match header)", func(t *testing.T) {
		t.Parallel()
		// given
		svc, ctrl := s.SecuredController()
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1))
		ifNoneMatch := app.GenerateEntityTag(fxt.SpaceTemplates[0])
		// when
		res := test.ShowSpaceTemplateNotModified(t, svc.Context, svc, ctrl, fxt.SpaceTemplates[0].ID, nil, &ifNoneMatch)
		// then
		assertResponseHeaders(t, res)
	})
}

// func (s *TestSpaceTemplateSuite) TestSpaceTemplate_List() {
// 	s.T().Run("list two existing templates", func(t *testing.T) {
// 		t.Parallel()
// 		// given
// 		svc, ctrl := s.SecuredController()
// 		expected1 := s.createMinimumSpaceTemplate(t)
// 		expected2 := s.createMinimumSpaceTemplate(t)
// 		// when
// 		res, spaceTemplateList := test.ListSpaceTemplateOK(t, svc.Context, svc, ctrl, nil, nil)
// 		// then
// 		require.NotNil(t, spaceTemplateList)
// 		assertResponseHeaders(t, res)
// 		n := len(spaceTemplateList.Data)
// 		require.True(t, n >= 2, "at least the two space templates created in this test must exist (total in list: %d)", n)
// 		toBeFound := 2
// 		for _, st := range spaceTemplateList.Data {
// 			if *st.ID == *expected1.Data.ID || *st.ID == *expected2.Data.ID {
// 				toBeFound--
// 			}
// 		}
// 		require.Equal(t, 0, toBeFound, "not all space templates created in this test were found")
// 	})

// 	s.T().Run("not modified (using expired If-Modified-Since header)", func(t *testing.T) {
// 		t.Parallel()
// 		// given
// 		svc, ctrl := s.SecuredController()
// 		expected1 := s.createMinimumSpaceTemplate(t)
// 		expected2 := s.createMinimumSpaceTemplate(t)
// 		ifModifiedSince := app.ToHTTPTime(expected1.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
// 		// when
// 		res, spaceTemplateList := test.ListSpaceTemplateOK(t, svc.Context, svc, ctrl, &ifModifiedSince, nil)
// 		// then
// 		require.NotNil(t, spaceTemplateList)
// 		assertResponseHeaders(t, res)
// 		n := len(spaceTemplateList.Data)
// 		require.True(t, n >= 2, "at least the two space templates created in this test must exist (total in list: %d)", n)
// 		toBeFound := 2
// 		for _, st := range spaceTemplateList.Data {
// 			if *st.ID == *expected1.Data.ID || *st.ID == *expected2.Data.ID {
// 				toBeFound--
// 			}
// 		}
// 		require.Equal(t, 0, toBeFound, "not all space templates created in this test were found")
// 	})

// 	s.T().Run("not modified (using If-Modified-Since header)", func(t *testing.T) {
// 		t.Parallel()
// 		// given
// 		svc, ctrl := s.SecuredController()
// 		expected := s.createMinimumSpaceTemplate(t)
// 		ifModifiedSince := app.ToHTTPTime(*expected.Data.Attributes.UpdatedAt)
// 		// when
// 		res := test.ListSpaceTemplateNotModified(t, svc.Context, svc, ctrl, &ifModifiedSince, nil)
// 		// then
// 		assertResponseHeaders(t, res)
// 	})

// 	s.T().Run("not modified (using If-None-Match header)", func(t *testing.T) {
// 		// t.Parallel() // this test must not be run in parallel
// 		// given
// 		svc, ctrl := s.SecuredController()
// 		_ = s.createMinimumSpaceTemplate(t)
// 		_, spaceTemplateList := test.ListSpaceTemplateOK(t, svc.Context, svc, ctrl, nil, nil)
// 		arr := make([]app.ConditionalResponseEntity, len(spaceTemplateList.Data))
// 		for i, v := range spaceTemplateList.Data {
// 			arr[i] = convertSpaceTemplateToModel(t, *v)
// 		}
// 		ifNoneMatch := app.GenerateEntitiesTag(arr)
// 		// when
// 		res := test.ListSpaceTemplateNotModified(t, svc.Context, svc, ctrl, nil, &ifNoneMatch)
// 		// then
// 		assertResponseHeaders(t, res)
// 	})
// }

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
		ID:          *appSpaceTemplate.ID,
		Name:        *appSpaceTemplate.Attributes.Name,
		Description: &desc,
		// Template:    string(bs),
		Version: *appSpaceTemplate.Attributes.Version,
		Lifecycle: gormsupport.Lifecycle{
			UpdatedAt: *appSpaceTemplate.Attributes.UpdatedAt,
			CreatedAt: *appSpaceTemplate.Attributes.CreatedAt,
		},
	}
}
