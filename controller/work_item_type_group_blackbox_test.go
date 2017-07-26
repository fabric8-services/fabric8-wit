package controller_test

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	almtoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type workItemTypeGroupSuite struct {
	gormtestsupport.DBTestSuite
	clean         func()
	svc           *goa.Service
	typeGroupCtrl *WorkItemTypeGroupController
}

func TestRunWorkItemTypeGroupSuite(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemTypeGroupSuite{
		DBTestSuite: gormtestsupport.NewDBTestSuite(""),
	})
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemTypeGroupSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	ctx := migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(ctx)
}

// The SetupTest method will be run before every test in the suite.
func (s *workItemTypeGroupSuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	s.svc = testsupport.ServiceAsUser("WITG-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	s.typeGroupCtrl = NewWorkItemTypeGroupController(s.svc, gormapplication.NewGormDB(s.DB))
}

func (s *workItemTypeGroupSuite) TearDownTest() {
	s.clean()
}

func (s *workItemTypeGroupSuite) TestListTypeGroups() {
	sapcetemplateID := uuid.NewV4()
	_, groups := test.ListWorkItemTypeGroupOK(s.T(), nil, s.svc, s.typeGroupCtrl, sapcetemplateID)
	assert.NotEmpty(s.T(), groups)
	require.Len(s.T(), groups.Data.Attributes.Hierarchy, 3)
}
