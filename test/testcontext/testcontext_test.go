package testcontext_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	p "github.com/fabric8-services/fabric8-wit/test/testcontext"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestRunTestContextSuite(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &testContextSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../../config.yaml")})
}

type testContextSuite struct {
	gormtestsupport.DBTestSuite
	clean func()
}

func (s *testContextSuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}
func (s *testContextSuite) TearDownTest() {
	s.clean()
}

func (s *testContextSuite) TestNewContext_Advanced() {
	s.T().Run("implicitly created entities", func(t *testing.T) {
		c := p.NewContext(t, s.DB, p.WorkItems(2))
		c.CheckWorkItems(2)
	})
	s.T().Run("explicitly create entities", func(t *testing.T) {
		// given
		c := p.NewContext(t, s.DB, p.WorkItems(2))
		c.CheckWorkItems(2)

		// manually use values from previous context over fields from first context
		c1 := p.NewContextIsolated(t, s.DB, p.WorkItems(3, func(ctx *p.TestContext, idx int) {
			ctx.WorkItems[idx].SpaceID = c.Spaces[0].ID
			ctx.WorkItems[idx].Type = c.WorkItemTypes[0].ID
			ctx.WorkItems[idx].Fields[workitem.SystemCreator] = c.Identities[0].ID.String()
		}))
		c1.CheckWorkItems(3)
	})
	s.T().Run("create 100 comments by 100 authors on 1 workitem", func(t *testing.T) {
		c := p.NewContext(t, s.DB, p.Identities(100), p.Comments(100, func(ctx *p.TestContext, idx int) {
			ctx.Comments[idx].Creator = ctx.Identities[idx].ID
		}))
		c.CheckComments(100)
		c.CheckIdentities(100)
	})
	s.T().Run("create 10 links between 20 work items with a network topology link type", func(t *testing.T) {
		c := p.NewContext(t, s.DB, p.WorkItemLinks(10), p.WorkItemLinkTypes(1, p.TopologyNetwork()))
		c.CheckWorkItemLinks(10)
		c.CheckWorkItemLinkTypes(1)
		c.CheckWorkItems(20)
	})
}

func (s *testContextSuite) TestNewContext() {
	// Number of objects to create of each type
	n := 3

	s.T().Run("identities", func(t *testing.T) {
		// given
		c := p.NewContext(t, s.DB, p.Identities(n))
		// then
		c.CheckIdentities(n)
		// manual checking
		require.Len(t, c.Identities, n)
	})
	s.T().Run("work item link categories", func(t *testing.T) {
		// given
		c := p.NewContext(t, s.DB, p.WorkItemLinkCategories(n))
		// then
		c.CheckWorkItemLinkCategories(n)
		// manual checking
		require.Len(t, c.WorkItemLinkCategories, n)
	})
	s.T().Run("spaces", func(t *testing.T) {
		// given
		c := p.NewContext(t, s.DB, p.Spaces(n))
		// then
		c.CheckSpaces(n)
		// manual checking
		require.Len(t, c.Spaces, n)
		require.Len(t, c.Identities, 1)
	})
	s.T().Run("work item link types", func(t *testing.T) {
		// given
		c := p.NewContext(t, s.DB, p.WorkItemLinkTypes(n))
		// then
		c.CheckWorkItemLinkTypes(n)
		// manual checking
		require.Len(t, c.WorkItemLinkTypes, n)
		require.Len(t, c.WorkItemLinkCategories, 1)
		require.Len(t, c.Identities, 1)
	})
	s.T().Run("codebases", func(t *testing.T) {
		// given
		c := p.NewContext(t, s.DB, p.Codebases(n))
		// then
		c.CheckCodebases(n)
		// manual checking
		require.Len(t, c.Codebases, n)
		require.Len(t, c.Spaces, 1)
		require.Len(t, c.Identities, 1)
	})
	s.T().Run("work item types", func(t *testing.T) {
		// given
		c := p.NewContext(t, s.DB, p.WorkItemTypes(n))
		// then
		c.CheckWorkItemTypes(n)
		// manual checking
		require.Len(t, c.WorkItemTypes, n)
		require.Len(t, c.Spaces, 1)
		require.Len(t, c.Identities, 1)
	})
	s.T().Run("iterations", func(t *testing.T) {
		// given
		c := p.NewContext(t, s.DB, p.Iterations(n))
		// then
		c.CheckIterations(n)
		// manual checking
		require.Len(t, c.Iterations, n)
		require.Len(t, c.Spaces, 1)
		require.Len(t, c.Identities, 1)
	})
	s.T().Run("areas", func(t *testing.T) {
		// given
		c := p.NewContext(t, s.DB, p.Areas(n))
		// then
		c.CheckAreas(n)
		// manual checking
		require.Len(t, c.Areas, n)
		require.Len(t, c.Spaces, 1)
		require.Len(t, c.Identities, 1)
	})
	s.T().Run("work items", func(t *testing.T) {
		// given
		c := p.NewContext(t, s.DB, p.WorkItems(n))
		// then
		c.CheckWorkItems(n)
		// manual checking
		require.Len(t, c.WorkItems, n)
		require.Len(t, c.Identities, 1)
		require.Len(t, c.WorkItemTypes, 1)
		require.Len(t, c.Spaces, 1)
	})
	s.T().Run("comments", func(t *testing.T) {
		// given
		c := p.NewContext(t, s.DB, p.Comments(n))
		// then
		c.CheckComments(n)
		// manual checking
		require.Len(t, c.Comments, n)
		require.Len(t, c.WorkItems, 1)
		require.Len(t, c.Identities, 1)
		require.Len(t, c.WorkItemTypes, 1)
		require.Len(t, c.Spaces, 1)
	})
	s.T().Run("work item links", func(t *testing.T) {
		// given
		c := p.NewContext(t, s.DB, p.WorkItemLinks(n))
		// then
		c.CheckWorkItemLinks(n)
		// manual checking
		require.Len(t, c.WorkItemLinks, n)
		require.Len(t, c.WorkItems, 2*n)
		require.Len(t, c.WorkItemTypes, 1)
		require.Len(t, c.WorkItemLinkTypes, 1)
		require.Len(t, c.Spaces, 1)
		require.Len(t, c.Identities, 1)
	})
}
