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
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem/typegroup"
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
	priv, _ := wittoken.RSAPrivateKey()
	s.svc = testsupport.ServiceAsUser("WITG-Service", wittoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	s.typeGroupCtrl = NewWorkItemTypeGroupController(s.svc, gormapplication.NewGormDB(s.DB))
}

func (s *workItemTypeGroupSuite) TearDownTest() {
	s.clean()
}

func (s *workItemTypeGroupSuite) TestListTypeGroups() {
	sapcetemplateID := space.SystemSpace // must be valid space ID
	_, groups := test.ListWorkItemTypeGroupOK(s.T(), nil, s.svc, s.typeGroupCtrl, sapcetemplateID)
	assert.NotEmpty(s.T(), groups)
	require.Len(s.T(), groups.Data.Attributes.Hierarchy, 4)
	require.Equal(s.T(), typegroup.GroupPortfolio, groups.Data.Attributes.Hierarchy[0].Group)
	require.Equal(s.T(), typegroup.GroupPortfolio, groups.Data.Attributes.Hierarchy[1].Group)
	require.Equal(s.T(), typegroup.GroupRequirements, groups.Data.Attributes.Hierarchy[2].Group)
	require.Equal(s.T(), typegroup.GroupExecution, groups.Data.Attributes.Hierarchy[3].Group)

	assert.Equal(s.T(), typegroup.Portfolio0.WorkItemTypeCollection, groups.Data.Attributes.Hierarchy[0].WitCollection)
	assert.Equal(s.T(), typegroup.Portfolio1.WorkItemTypeCollection, groups.Data.Attributes.Hierarchy[1].WitCollection)
	assert.Equal(s.T(), typegroup.Requirements0.WorkItemTypeCollection, groups.Data.Attributes.Hierarchy[2].WitCollection)
	assert.Equal(s.T(), typegroup.Execution0.WorkItemTypeCollection, groups.Data.Attributes.Hierarchy[3].WitCollection)
}

func (s *workItemTypeGroupSuite) TestListTypeGroupsNotFound() {
	sapcetemplateID := uuid.NewV4()
	test.ListWorkItemTypeGroupNotFound(s.T(), nil, s.svc, s.typeGroupCtrl, sapcetemplateID)
}
