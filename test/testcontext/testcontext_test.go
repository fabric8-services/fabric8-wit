package testcontext_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	p "github.com/fabric8-services/fabric8-wit/test/testcontext"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/jinzhu/gorm"
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
		c, err := p.NewContext(s.DB, p.WorkItems(2))
		require.Nil(t, err)
		require.Nil(t, c.Check())
	})
	s.T().Run("explicitly create entities", func(t *testing.T) {
		// given
		c, err := p.NewContext(s.DB, p.WorkItems(2))
		require.Nil(t, err)
		require.Nil(t, c.Check())

		// manually use values from previous context over fields from first context
		c1, err := p.NewContextIsolated(s.DB, p.WorkItems(3, func(ctx *p.TestContext, idx int) error {
			ctx.WorkItems[idx].SpaceID = c.Spaces[0].ID
			ctx.WorkItems[idx].Type = c.WorkItemTypes[0].ID
			ctx.WorkItems[idx].Fields[workitem.SystemCreator] = c.Identities[0].ID.String()
			return nil
		}))
		require.Nil(t, err)
		require.Nil(t, c1.Check())
	})
	s.T().Run("create 100 comments by 100 authors on 1 workitem", func(t *testing.T) {
		c, err := p.NewContext(s.DB, p.Identities(100), p.Comments(100, func(ctx *p.TestContext, idx int) error {
			ctx.Comments[idx].Creator = ctx.Identities[idx].ID
			return nil
		}))
		require.Nil(t, err)
		require.Nil(t, c.Check())
	})
	s.T().Run("create 10 links between 20 work items with a network topology link type", func(t *testing.T) {
		c, err := p.NewContext(s.DB, p.WorkItemLinks(10), p.WorkItemLinkTypes(1, p.TopologyNetwork()))
		require.Nil(t, err)
		require.Nil(t, c.Check())
	})
}

func (s *testContextSuite) TestNewContext() {
	checkNewContext(s.T(), s.DB, 3, false)
}

func (s *testContextSuite) TestNewContextIsolated() {
	checkNewContext(s.T(), s.DB, 3, true)
}

func checkNewContext(t *testing.T, db *gorm.DB, n int, isolated bool) {
	// when not creating in isolation we want tests to check for created items
	// and a valid context
	ctxCtor := p.NewContext
	checkCtorErrFunc := func(t *testing.T, err error) {
		require.Nil(t, err)
	}
	checkFunc := func(t *testing.T, ctx *p.TestContext) {
		require.NotNil(t, ctx)
		require.Nil(t, ctx.Check())
	}

	// when creating in isolation we want tests to check for not existing items
	// and an invalid context
	if isolated {
		ctxCtor = p.NewContextIsolated
		checkCtorErrFunc = func(t *testing.T, err error) {
			require.NotNil(t, err)
		}
		checkFunc = func(t *testing.T, ctx *p.TestContext) {
			require.Nil(t, ctx)
		}
	}

	// identity and work item link categories will always work

	t.Run("identities", func(t *testing.T) {
		// given
		c, err := ctxCtor(db, p.Identities(n))
		// then
		require.Nil(t, err)
		require.Nil(t, c.Check())
		// manual checking
		require.Len(t, c.Identities, n)
	})
	t.Run("work item link categories", func(t *testing.T) {
		// given
		c, err := ctxCtor(db, p.WorkItemLinkCategories(n))
		// then
		require.Nil(t, err)
		require.Nil(t, c.Check())
		// manual checking
		require.Len(t, c.WorkItemLinkCategories, n)
	})

	t.Run("spaces", func(t *testing.T) {
		// given
		c, err := ctxCtor(db, p.Spaces(n))
		// then
		checkCtorErrFunc(t, err)
		checkFunc(t, c)
		if !isolated {
			// manual checking
			require.Len(t, c.Spaces, n)
			require.Len(t, c.Identities, 1)
		}
	})
	t.Run("work item link types", func(t *testing.T) {
		// given
		c, err := ctxCtor(db, p.WorkItemLinkTypes(n))
		// then
		checkCtorErrFunc(t, err)
		checkFunc(t, c)
		// manual checking
		if !isolated {
			require.Len(t, c.WorkItemLinkTypes, n)
			require.Len(t, c.WorkItemLinkCategories, 1)
			require.Len(t, c.Identities, 1)
		}
	})
	t.Run("codebases", func(t *testing.T) {
		// given
		c, err := ctxCtor(db, p.Codebases(n))
		// then
		checkCtorErrFunc(t, err)
		checkFunc(t, c)
		// manual checking
		if !isolated {
			require.Len(t, c.Codebases, n)
			require.Len(t, c.Spaces, 1)
			require.Len(t, c.Identities, 1)
		}
	})
	t.Run("work item types", func(t *testing.T) {
		// given
		c, err := ctxCtor(db, p.WorkItemTypes(n))
		// then
		checkCtorErrFunc(t, err)
		checkFunc(t, c)
		// manual checking
		if !isolated {
			require.Len(t, c.WorkItemTypes, n)
			require.Len(t, c.Spaces, 1)
			require.Len(t, c.Identities, 1)
		}
	})
	t.Run("iterations", func(t *testing.T) {
		// given
		c, err := ctxCtor(db, p.Iterations(n))
		// then
		checkCtorErrFunc(t, err)
		checkFunc(t, c)
		// manual checking
		if !isolated {
			require.Len(t, c.Iterations, n)
			require.Len(t, c.Spaces, 1)
			require.Len(t, c.Identities, 1)
		}
	})
	t.Run("areas", func(t *testing.T) {
		// given
		c, err := ctxCtor(db, p.Areas(n))
		// then
		checkCtorErrFunc(t, err)
		checkFunc(t, c)
		// manual checking
		if !isolated {
			require.Len(t, c.Areas, n)
			require.Len(t, c.Spaces, 1)
			require.Len(t, c.Identities, 1)
		}
	})
	t.Run("work items", func(t *testing.T) {
		// given
		c, err := ctxCtor(db, p.WorkItems(n))
		// then
		checkCtorErrFunc(t, err)
		checkFunc(t, c)
		// manual checking
		if !isolated {
			require.Len(t, c.WorkItems, n)
			require.Len(t, c.Identities, 1)
			require.Len(t, c.WorkItemTypes, 1)
			require.Len(t, c.Spaces, 1)
		}
	})
	t.Run("comments", func(t *testing.T) {
		// given
		c, err := ctxCtor(db, p.Comments(n))
		// then
		checkCtorErrFunc(t, err)
		checkFunc(t, c)
		// manual checking
		if !isolated {
			require.Len(t, c.Comments, n)
			require.Len(t, c.WorkItems, 1)
			require.Len(t, c.Identities, 1)
			require.Len(t, c.WorkItemTypes, 1)
			require.Len(t, c.Spaces, 1)
		}
	})
	t.Run("work item links", func(t *testing.T) {
		// given
		c, err := ctxCtor(db, p.WorkItemLinks(n))
		// then
		checkCtorErrFunc(t, err)
		checkFunc(t, c)
		// manual checking
		if !isolated {
			require.Len(t, c.WorkItemLinks, n)
			require.Len(t, c.WorkItems, 2*n)
			require.Len(t, c.WorkItemTypes, 1)
			require.Len(t, c.WorkItemLinkTypes, 1)
			require.Len(t, c.Spaces, 1)
			require.Len(t, c.Identities, 1)
		}
	})
}
