package controller_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/label"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestLabelREST struct {
	gormtestsupport.DBTestSuite
	db    *gormapplication.GormDB
	clean func()
}

func TestRunLabelREST(t *testing.T) {
	resource.Require(t, resource.Database)
	pwd, err := os.Getwd()
	require.Nil(t, err)
	suite.Run(t, &TestLabelREST{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

func (rest *TestLabelREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestLabelREST) TearDownTest() {
	rest.clean()
}

func (rest *TestLabelREST) TestCreateLabel() {
	c, err := tf.NewFixture(rest.DB, tf.Spaces(1))
	require.Nil(rest.T(), err)
	require.Nil(rest.T(), c.Check())
	i, err := tf.NewFixture(rest.DB, tf.Identities(1))
	require.Nil(rest.T(), err)
	require.Nil(rest.T(), c.Check())
	priv, _ := wittoken.RSAPrivateKey()
	svc := testsupport.ServiceAsUser("Label-Service", wittoken.NewManagerWithPrivateKey(priv), *i.Identities[0])

	ctrl := NewLabelController(svc, rest.db, rest.Configuration)
	pl := app.CreateLabelPayload{
		Data: &app.Label{
			Attributes: &app.LabelAttributes{Name: "some color"},
			Type:       label.APIStringTypeLabels,
		},
	}
	_, created := test.CreateLabelCreated(rest.T(), svc.Context, svc, ctrl, c.Spaces[0].ID, &pl)
	assert.Equal(rest.T(), pl.Data.Attributes.Name, created.Data.Attributes.Name)
	assert.Equal(rest.T(), "#000000", *created.Data.Attributes.TextColor)
	assert.Equal(rest.T(), "#FFFFFF", *created.Data.Attributes.BackgroundColor)
	assert.False(rest.T(), created.Data.Attributes.CreatedAt.After(time.Now()), "Label was not created, CreatedAt after Now()")
}

func (rest *TestLabelREST) TestCreateLabelWithWhiteSpace() {
	c, err := tf.NewFixture(rest.DB, tf.Spaces(1))
	require.Nil(rest.T(), err)
	require.Nil(rest.T(), c.Check())
	i, err := tf.NewFixture(rest.DB, tf.Identities(1))
	require.Nil(rest.T(), err)
	require.Nil(rest.T(), c.Check())
	priv, _ := wittoken.RSAPrivateKey()
	svc := testsupport.ServiceAsUser("Label-Service", wittoken.NewManagerWithPrivateKey(priv), *i.Identities[0])

	ctrl := NewLabelController(svc, rest.db, rest.Configuration)
	pl := app.CreateLabelPayload{
		Data: &app.Label{
			Attributes: &app.LabelAttributes{Name: "	  some color  "},
			Type: label.APIStringTypeLabels,
		},
	}
	_, created := test.CreateLabelCreated(rest.T(), svc.Context, svc, ctrl, c.Spaces[0].ID, &pl)
	assertLabelLinking(rest.T(), created.Data)
	assert.Equal(rest.T(), strings.TrimSpace(pl.Data.Attributes.Name), created.Data.Attributes.Name)
	assert.Equal(rest.T(), "#000000", *created.Data.Attributes.TextColor)
	assert.Equal(rest.T(), "#FFFFFF", *created.Data.Attributes.BackgroundColor)
	assert.False(rest.T(), created.Data.Attributes.CreatedAt.After(time.Now()), "Label was not created, CreatedAt after Now()")
}

func (rest *TestLabelREST) TestListLabel() {
	c, err := tf.NewFixture(rest.DB, tf.Spaces(1))
	require.Nil(rest.T(), err)
	require.Nil(rest.T(), c.Check())
	i, err := tf.NewFixture(rest.DB, tf.Identities(1))
	require.Nil(rest.T(), err)
	require.Nil(rest.T(), c.Check())
	priv, _ := wittoken.RSAPrivateKey()
	svc := testsupport.ServiceAsUser("Label-Service", wittoken.NewManagerWithPrivateKey(priv), *i.Identities[0])

	ctrl := NewLabelController(svc, rest.db, rest.Configuration)

	_, labels := test.ListLabelOK(rest.T(), svc.Context, svc, ctrl, c.Spaces[0].ID, nil, nil)
	require.Empty(rest.T(), labels.Data, "labels found")
	pl := app.CreateLabelPayload{
		Data: &app.Label{
			Attributes: &app.LabelAttributes{Name: "some color"},
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
	require.Nil(rest.T(), err)
	priv, _ := wittoken.RSAPrivateKey()
	svc := testsupport.ServiceAsUser("Label-Service", wittoken.NewManagerWithPrivateKey(priv), *i.Identities[0])

	ctrl := NewLabelController(svc, rest.db, rest.Configuration)

	_, labels2 := test.ShowLabelOK(rest.T(), svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Labels[0].ID.String(), nil, nil)
	assertLabelLinking(rest.T(), labels2.Data)
	require.NotEmpty(rest.T(), labels2.Data, "labels found")
	assert.Equal(rest.T(), testFxt.Labels[0].Name, labels2.Data.Attributes.Name)
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
