package controller_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/label"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type LabelControllerTestSuite struct {
	gormtestsupport.DBTestSuite
	db      *gormapplication.GormDB
	testDir string
}

func TestLabelController(t *testing.T) {
	resource.Require(t, resource.Database)
	pwd, err := os.Getwd()
	require.NoError(t, err)
	suite.Run(t, &LabelControllerTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

func (s *LabelControllerTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.db = gormapplication.NewGormDB(s.DB)
	s.testDir = filepath.Join("test-files", "label")
}

func (s *LabelControllerTestSuite) TestCreateLabel() {
	c, err := tf.NewFixture(s.DB, tf.Spaces(1))
	require.NoError(s.T(), err)
	require.Nil(s.T(), c.Check())
	i, err := tf.NewFixture(s.DB, tf.Identities(1))
	require.NoError(s.T(), err)
	require.Nil(s.T(), c.Check())
	svc := testsupport.ServiceAsUser("Label-Service", *i.Identities[0])

	ctrl := NewLabelController(svc, s.db, s.Configuration)
	color := "some color"
	pl := app.CreateLabelPayload{
		Data: &app.Label{
			Attributes: &app.LabelAttributes{Name: &color},
			Type:       label.APIStringTypeLabels,
		},
	}
	_, created := test.CreateLabelCreated(s.T(), svc.Context, svc, ctrl, c.Spaces[0].ID, &pl)
	assert.Equal(s.T(), pl.Data.Attributes.Name, created.Data.Attributes.Name)
	assert.Equal(s.T(), "#000000", *created.Data.Attributes.TextColor)
	assert.Equal(s.T(), "#FFFFFF", *created.Data.Attributes.BackgroundColor)
	assert.Equal(s.T(), "#000000", *created.Data.Attributes.BorderColor)
	assert.False(s.T(), created.Data.Attributes.CreatedAt.After(time.Now()), "Label was not created, CreatedAt after Now()")
}

func (s *LabelControllerTestSuite) TestCreateLabelWithWhiteSpace() {
	c, err := tf.NewFixture(s.DB, tf.Spaces(1))
	require.NoError(s.T(), err)
	require.Nil(s.T(), c.Check())
	i, err := tf.NewFixture(s.DB, tf.Identities(1))
	require.NoError(s.T(), err)
	require.Nil(s.T(), c.Check())
	svc := testsupport.ServiceAsUser("Label-Service", *i.Identities[0])

	ctrl := NewLabelController(svc, s.db, s.Configuration)
	color := "	  some color  "
	pl := app.CreateLabelPayload{
		Data: &app.Label{
			Attributes: &app.LabelAttributes{Name: &color},
			Type:       label.APIStringTypeLabels,
		},
	}
	_, created := test.CreateLabelCreated(s.T(), svc.Context, svc, ctrl, c.Spaces[0].ID, &pl)
	assertLabelLinking(s.T(), created.Data)
	assert.Equal(s.T(), strings.TrimSpace(*pl.Data.Attributes.Name), *created.Data.Attributes.Name)
	assert.Equal(s.T(), "#000000", *created.Data.Attributes.TextColor)
	assert.Equal(s.T(), "#FFFFFF", *created.Data.Attributes.BackgroundColor)
	assert.Equal(s.T(), "#000000", *created.Data.Attributes.BorderColor)
	assert.False(s.T(), created.Data.Attributes.CreatedAt.After(time.Now()), "Label was not created, CreatedAt after Now()")
}

func (s *LabelControllerTestSuite) TestUpdate() {

	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Labels(1))
	svc := testsupport.ServiceAsUser("Label-Service", *testFxt.Identities[0])
	ctrl := NewLabelController(svc, s.db, s.Configuration)

	s.T().Run("update label", func(t *testing.T) {
		newName := "Label New 1001"
		textColor := "#dbe1f6"
		backgroundColor := "#10b2f4"
		borderColor := "#0ccca6"
		payload := app.UpdateLabelPayload{
			Data: &app.Label{
				Attributes: &app.LabelAttributes{
					Name:            &newName,
					Version:         &testFxt.Labels[0].Version,
					TextColor:       &textColor,
					BackgroundColor: &backgroundColor,
					BorderColor:     &borderColor,
				},
				ID:   &testFxt.Labels[0].ID,
				Type: label.APIStringTypeLabels,
			},
		}
		resp, updated := test.UpdateLabelOK(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Labels[0].ID, &payload)
		assert.Equal(t, newName, *updated.Data.Attributes.Name)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "update", "ok.label.golden.json"), updated)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "update", "ok.headers.golden.json"), resp)

		_, labels2 := test.ShowLabelOK(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Labels[0].ID, nil, nil)
		assertLabelLinking(t, labels2.Data)
		require.NotEmpty(t, labels2.Data, "labels found")
		assert.Equal(t, newName, *labels2.Data.Attributes.Name)
	})

	s.T().Run("update label with version conflict", func(t *testing.T) {
		newVersion := testFxt.Labels[0].Version + 2
		payload := app.UpdateLabelPayload{
			Data: &app.Label{
				Attributes: &app.LabelAttributes{
					Name:    &testFxt.Labels[0].Name,
					Version: &newVersion,
				},
				ID:   &testFxt.Labels[0].ID,
				Type: label.APIStringTypeLabels,
			},
		}
		_, jerrs := test.UpdateLabelConflict(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Labels[0].ID, &payload)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.Contains(t, jerrs.Errors[0].Detail, "version conflict")
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "update", "conflict.errors.golden.json"), jerrs)
	})

	s.T().Run("update label with bad parameter", func(t *testing.T) {
		payload := app.UpdateLabelPayload{
			Data: &app.Label{
				Attributes: &app.LabelAttributes{},
				Type:       label.APIStringTypeLabels,
			},
		}

		_, jerrs := test.UpdateLabelBadRequest(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Labels[0].ID, &payload)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.Contains(t, jerrs.Errors[0].Detail, "Bad value for parameter 'data.attributes.version'")
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "update", "badparam_version.errors.golden.json"), jerrs)
	})

	s.T().Run("update label with bad parameter - name", func(t *testing.T) {
		newName := " 	   " // tab & spaces
		newVersion := testFxt.Labels[0].Version + 1
		payload := app.UpdateLabelPayload{
			Data: &app.Label{
				Attributes: &app.LabelAttributes{
					Name:    &newName,
					Version: &newVersion,
				},
				ID:   &testFxt.Labels[0].ID,
				Type: label.APIStringTypeLabels,
			},
		}

		_, jerrs := test.UpdateLabelBadRequest(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Labels[0].ID, &payload)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.Contains(t, jerrs.Errors[0].Detail, "Bad value for parameter 'label name cannot be empty string'")
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "update", "badparam_name.errors.golden.json"), jerrs)
	})

	s.T().Run("update label with unauthorized", func(t *testing.T) {
		svc := goa.New("Label-Service")
		ctrl := NewLabelController(svc, s.db, s.Configuration)

		payload := app.UpdateLabelPayload{
			Data: &app.Label{
				Attributes: &app.LabelAttributes{
					Version: &testFxt.Labels[0].Version,
				},
				Type: label.APIStringTypeLabels,
			},
		}

		_, jerrs := test.UpdateLabelUnauthorized(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Labels[0].ID, &payload)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.Contains(t, jerrs.Errors[0].Detail, "Missing token manager")
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "update", "unauthorized.errors.golden.json"), jerrs)
	})
	s.T().Run("update label not found", func(t *testing.T) {
		newName := "Label New 1002"
		newVersion := testFxt.Labels[0].Version + 1
		id := uuid.NewV4()
		payload := app.UpdateLabelPayload{
			Data: &app.Label{
				Attributes: &app.LabelAttributes{
					Name:    &newName,
					Version: &newVersion,
				},
				ID:   &id,
				Type: label.APIStringTypeLabels,
			},
		}
		test.UpdateLabelNotFound(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, id, &payload)
	})
}

func (s *LabelControllerTestSuite) TestListLabel() {
	c, err := tf.NewFixture(s.DB, tf.Spaces(1))
	require.NoError(s.T(), err)
	require.Nil(s.T(), c.Check())
	i, err := tf.NewFixture(s.DB, tf.Identities(1))
	require.NoError(s.T(), err)
	require.Nil(s.T(), c.Check())
	svc := testsupport.ServiceAsUser("Label-Service", *i.Identities[0])

	ctrl := NewLabelController(svc, s.db, s.Configuration)

	_, labels := test.ListLabelOK(s.T(), svc.Context, svc, ctrl, c.Spaces[0].ID, nil, nil)
	require.Empty(s.T(), labels.Data, "labels found")
	color := "some color"
	pl := app.CreateLabelPayload{
		Data: &app.Label{
			Attributes: &app.LabelAttributes{Name: &color},
			Type:       label.APIStringTypeLabels,
		},
	}
	test.CreateLabelCreated(s.T(), svc.Context, svc, ctrl, c.Spaces[0].ID, &pl)
	_, labels2 := test.ListLabelOK(s.T(), svc.Context, svc, ctrl, c.Spaces[0].ID, nil, nil)
	assertLabelLinking(s.T(), labels2.Data[0])
	require.NotEmpty(s.T(), labels2.Data, "labels found")
	require.Len(s.T(), labels2.Data, 1)
}

func (s *LabelControllerTestSuite) TestShowLabel() {
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Labels(1))
	i, err := tf.NewFixture(s.DB, tf.Identities(1))
	require.NoError(s.T(), err)
	svc := testsupport.ServiceAsUser("Label-Service", *i.Identities[0])

	ctrl := NewLabelController(svc, s.db, s.Configuration)

	_, labels2 := test.ShowLabelOK(s.T(), svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Labels[0].ID, nil, nil)
	assertLabelLinking(s.T(), labels2.Data)
	require.NotEmpty(s.T(), labels2.Data, "labels found")
	assert.Equal(s.T(), testFxt.Labels[0].Name, *labels2.Data.Attributes.Name)
}

func assertLabelLinking(t *testing.T, target *app.Label) {
	assert.NotNil(t, target.ID)
	assert.Equal(t, label.APIStringTypeLabels, target.Type)
	assert.NotNil(t, target.Links.Self)
	require.NotNil(t, target.Relationships)
	require.NotNil(t, target.Relationships.Space)
	require.NotNil(t, target.Relationships.Space.Links)
	require.NotNil(t, target.Relationships.Space.Links.Self)
	assert.True(t, strings.Contains(*target.Relationships.Space.Links.Self, "/api/spaces/"))
}
