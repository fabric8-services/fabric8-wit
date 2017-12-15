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

type TestLabelREST struct {
	gormtestsupport.DBTestSuite
	db      *gormapplication.GormDB
	testDir string
}

func TestRunLabelREST(t *testing.T) {
	resource.Require(t, resource.Database)
	pwd, err := os.Getwd()
	require.NoError(t, err)
	suite.Run(t, &TestLabelREST{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

func (rest *TestLabelREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.testDir = filepath.Join("test-files", "label")
}

func (rest *TestLabelREST) TestCreateLabel() {
	c, err := tf.NewFixture(rest.DB, tf.Spaces(1))
	require.NoError(rest.T(), err)
	require.Nil(rest.T(), c.Check())
	i, err := tf.NewFixture(rest.DB, tf.Identities(1))
	require.NoError(rest.T(), err)
	require.Nil(rest.T(), c.Check())
	svc := testsupport.ServiceAsUser("Label-Service", *i.Identities[0])

	ctrl := NewLabelController(svc, rest.db, rest.Configuration)
	color := "some color"
	pl := app.CreateLabelPayload{
		Data: &app.Label{
			Attributes: &app.LabelAttributes{Name: &color},
			Type:       label.APIStringTypeLabels,
		},
	}
	_, created := test.CreateLabelCreated(rest.T(), svc.Context, svc, ctrl, c.Spaces[0].ID, &pl)
	assert.Equal(rest.T(), pl.Data.Attributes.Name, created.Data.Attributes.Name)
	assert.Equal(rest.T(), "#000000", *created.Data.Attributes.TextColor)
	assert.Equal(rest.T(), "#FFFFFF", *created.Data.Attributes.BackgroundColor)
	assert.Equal(rest.T(), "#000000", *created.Data.Attributes.BorderColor)
	assert.False(rest.T(), created.Data.Attributes.CreatedAt.After(time.Now()), "Label was not created, CreatedAt after Now()")
}

func (rest *TestLabelREST) TestCreateLabelWithWhiteSpace() {
	c, err := tf.NewFixture(rest.DB, tf.Spaces(1))
	require.NoError(rest.T(), err)
	require.Nil(rest.T(), c.Check())
	i, err := tf.NewFixture(rest.DB, tf.Identities(1))
	require.NoError(rest.T(), err)
	require.Nil(rest.T(), c.Check())
	svc := testsupport.ServiceAsUser("Label-Service", *i.Identities[0])

	ctrl := NewLabelController(svc, rest.db, rest.Configuration)
	color := "	  some color  "
	pl := app.CreateLabelPayload{
		Data: &app.Label{
			Attributes: &app.LabelAttributes{Name: &color},
			Type:       label.APIStringTypeLabels,
		},
	}
	_, created := test.CreateLabelCreated(rest.T(), svc.Context, svc, ctrl, c.Spaces[0].ID, &pl)
	assertLabelLinking(rest.T(), created.Data)
	assert.Equal(rest.T(), strings.TrimSpace(*pl.Data.Attributes.Name), *created.Data.Attributes.Name)
	assert.Equal(rest.T(), "#000000", *created.Data.Attributes.TextColor)
	assert.Equal(rest.T(), "#FFFFFF", *created.Data.Attributes.BackgroundColor)
	assert.Equal(rest.T(), "#000000", *created.Data.Attributes.BorderColor)
	assert.False(rest.T(), created.Data.Attributes.CreatedAt.After(time.Now()), "Label was not created, CreatedAt after Now()")
}

func (rest *TestLabelREST) TestUpdate() {
	resetFn := rest.DisableGormCallbacks()
	defer resetFn()

	testFxt := tf.NewTestFixture(rest.T(), rest.DB, tf.Labels(1))
	svc := testsupport.ServiceAsUser("Label-Service", *testFxt.Identities[0])
	ctrl := NewLabelController(svc, rest.db, rest.Configuration)

	rest.T().Run("update label", func(t *testing.T) {
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
		compareWithGoldenUUIDAgnostic(t, filepath.Join(rest.testDir, "update", "ok.label.golden.json"), updated)
		compareWithGoldenUUIDAgnostic(t, filepath.Join(rest.testDir, "update", "ok.headers.golden.json"), resp)

		_, labels2 := test.ShowLabelOK(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Labels[0].ID, nil, nil)
		assertLabelLinking(t, labels2.Data)
		require.NotEmpty(t, labels2.Data, "labels found")
		assert.Equal(t, newName, *labels2.Data.Attributes.Name)
	})

	rest.T().Run("update label with version conflict", func(t *testing.T) {
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
		compareWithGoldenUUIDAgnostic(t, filepath.Join(rest.testDir, "update", "conflict.errors.golden.json"), jerrs)
	})

	rest.T().Run("update label with bad parameter", func(t *testing.T) {
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
		compareWithGoldenUUIDAgnostic(t, filepath.Join(rest.testDir, "update", "badparam_version.errors.golden.json"), jerrs)
	})

	rest.T().Run("update label with bad parameter - name", func(t *testing.T) {
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
		compareWithGoldenUUIDAgnostic(t, filepath.Join(rest.testDir, "update", "badparam_name.errors.golden.json"), jerrs)
	})

	rest.T().Run("update label with unauthorized", func(t *testing.T) {
		svc := goa.New("Label-Service")
		ctrl := NewLabelController(svc, rest.db, rest.Configuration)

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
		compareWithGoldenUUIDAgnostic(t, filepath.Join(rest.testDir, "update", "unauthorized.errors.golden.json"), jerrs)
	})
	rest.T().Run("update label not found", func(t *testing.T) {
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

func (rest *TestLabelREST) TestListLabel() {
	c, err := tf.NewFixture(rest.DB, tf.Spaces(1))
	require.NoError(rest.T(), err)
	require.Nil(rest.T(), c.Check())
	i, err := tf.NewFixture(rest.DB, tf.Identities(1))
	require.NoError(rest.T(), err)
	require.Nil(rest.T(), c.Check())
	svc := testsupport.ServiceAsUser("Label-Service", *i.Identities[0])

	ctrl := NewLabelController(svc, rest.db, rest.Configuration)

	_, labels := test.ListLabelOK(rest.T(), svc.Context, svc, ctrl, c.Spaces[0].ID, nil, nil)
	require.Empty(rest.T(), labels.Data, "labels found")
	color := "some color"
	pl := app.CreateLabelPayload{
		Data: &app.Label{
			Attributes: &app.LabelAttributes{Name: &color},
			Type:       label.APIStringTypeLabels,
		},
	}
	test.CreateLabelCreated(rest.T(), svc.Context, svc, ctrl, c.Spaces[0].ID, &pl)
	_, labels2 := test.ListLabelOK(rest.T(), svc.Context, svc, ctrl, c.Spaces[0].ID, nil, nil)
	assertLabelLinking(rest.T(), labels2.Data[0])
	require.NotEmpty(rest.T(), labels2.Data, "labels found")
	require.Len(rest.T(), labels2.Data, 1)
}

func (rest *TestLabelREST) TestShowLabel() {
	testFxt := tf.NewTestFixture(rest.T(), rest.DB, tf.Labels(1))
	i, err := tf.NewFixture(rest.DB, tf.Identities(1))
	require.NoError(rest.T(), err)
	svc := testsupport.ServiceAsUser("Label-Service", *i.Identities[0])

	ctrl := NewLabelController(svc, rest.db, rest.Configuration)

	_, labels2 := test.ShowLabelOK(rest.T(), svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Labels[0].ID, nil, nil)
	assertLabelLinking(rest.T(), labels2.Data)
	require.NotEmpty(rest.T(), labels2.Data, "labels found")
	assert.Equal(rest.T(), testFxt.Labels[0].Name, *labels2.Data.Attributes.Name)
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
