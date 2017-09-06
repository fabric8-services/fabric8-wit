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

func TestRunSetup(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &setupSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../../config.yaml")})
}

type setupSuite struct {
	gormtestsupport.DBTestSuite
	clean func()
}

func (s *setupSuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}
func (s *setupSuite) TearDownTest() {
	s.clean()
}

func requireMinLen(t *testing.T, minLen, actualLen int) {
	require.Condition(t, func() (success bool) { return actualLen >= minLen }, "minimum length must of %d is not reached with %d", minLen, actualLen)
}

func (s *setupSuite) TestNewContext() {
	// // given
	c := p.NewContext(s.T(), s.DB, p.WorkItems(2))

	s.T().Run("implicitly create entities", func(t *testing.T) {
		require.Len(t, c.Identities, 1)
		require.Len(t, c.Spaces, 1)
		require.Len(t, c.WorkItemTypes, 1)
		require.Len(t, c.WorkItems, 2)
	})
	s.T().Run("explicitly create entities", func(t *testing.T) {
		// manually use values from previous context over fields from first context
		ctx := p.NewContextIsolated(t, s.DB, p.WorkItems(3, func(ctx *p.TestContext, idx int) {
			ctx.WorkItems[idx].SpaceID = c.Spaces[0].ID
			ctx.WorkItems[idx].Type = c.WorkItemTypes[0].ID
			ctx.WorkItems[idx].Fields[workitem.SystemCreator] = c.Identities[0].ID.String()
		}))
		require.Len(t, ctx.Identities, 0)
		require.Len(t, ctx.Spaces, 0)
		require.Len(t, ctx.WorkItemTypes, 0)
		require.Len(t, ctx.WorkItems, 3)
	})
	s.T().Run("create 100 comments by 100 authors on 1 workitem", func(t *testing.T) {
		ctx := p.NewContext(t, s.DB, p.Identities(100), p.Comments(100, func(ctx *p.TestContext, idx int) {
			ctx.Comments[idx].Creator = ctx.Identities[idx].ID
		}))
		require.Len(t, ctx.Identities, 100)
		require.Len(t, ctx.Spaces, 1)
		require.Len(t, ctx.WorkItemTypes, 1)
		require.Len(t, ctx.WorkItems, 1)
		require.Len(t, ctx.Comments, 100)
	})
	s.T().Run("create 10 links between 20 work items with a network topology link type", func(t *testing.T) {
		ctx := p.NewContext(t, s.DB, p.WorkItemLinks(10), p.WorkItemLinkTypes(1, p.TopologyNetwork()))
		require.Len(t, ctx.Identities, 1)
		require.Len(t, ctx.Spaces, 1)
		require.Len(t, ctx.WorkItemTypes, 1)
		require.Len(t, ctx.WorkItems, 20)
		require.Len(t, ctx.WorkItemLinks, 10)
	})
}
