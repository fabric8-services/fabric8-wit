package testfixture_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestRunTestFixtureSuite(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &testFixtureSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../../config.yaml")})
}

type testFixtureSuite struct {
	gormtestsupport.DBTestSuite
	clean func()
}

func (s *testFixtureSuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}
func (s *testFixtureSuite) TearDownTest() {
	s.clean()
}

func (s *testFixtureSuite) TestNewFixture_Advanced() {
	s.T().Run("implicitly created entities", func(t *testing.T) {
		c, err := tf.NewFixture(s.DB, tf.WorkItems(2))
		require.Nil(t, err)
		require.Nil(t, c.Check())
	})
	s.T().Run("explicitly create entities", func(t *testing.T) {
		// given
		c, err := tf.NewFixture(s.DB, tf.WorkItems(2))
		require.Nil(t, err)
		require.Nil(t, c.Check())

		// manually use values from previous fixture over fields from first fixture
		c1, err := tf.NewFixtureIsolated(s.DB, tf.WorkItems(3, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].SpaceID = c.Spaces[0].ID
			fxt.WorkItems[idx].Type = c.WorkItemTypes[0].ID
			fxt.WorkItems[idx].Fields[workitem.SystemCreator] = c.Identities[0].ID.String()
			return nil
		}))
		require.Nil(t, err)
		require.Nil(t, c1.Check())
	})
	s.T().Run("create 100 comments by 100 authors on 1 workitem", func(t *testing.T) {
		c, err := tf.NewFixture(s.DB, tf.Identities(100), tf.Comments(100, func(fxt *tf.TestFixture, idx int) error {
			fxt.Comments[idx].Creator = fxt.Identities[idx].ID
			return nil
		}))
		require.Nil(t, err)
		require.Nil(t, c.Check())
	})
	s.T().Run("create 10 links between 20 work items with a network topology link type", func(t *testing.T) {
		c, err := tf.NewFixture(s.DB, tf.WorkItemLinks(10), tf.WorkItemLinkTypes(1, tf.TopologyNetwork()))
		require.Nil(t, err)
		require.Nil(t, c.Check())
	})
}

func (s *testFixtureSuite) TestNewFixture() {
	checkNewFixture(s.T(), s.DB, 3, false)
}

func (s *testFixtureSuite) TestNewFixtureIsolated() {
	checkNewFixture(s.T(), s.DB, 3, true)
}

func checkNewFixture(t *testing.T, db *gorm.DB, n int, isolated bool) {
	// when not creating in isolation we want tests to check for created items
	// and a valid fixture
	fxtCtor := tf.NewFixture
	checkCtorErrFunc := func(t *testing.T, err error) {
		require.Nil(t, err)
	}
	checkFunc := func(t *testing.T, fxt *tf.TestFixture) {
		require.NotNil(t, fxt)
		require.Nil(t, fxt.Check())
	}

	// when creating in isolation we want tests to check for not existing items
	// and an invalid fixture
	if isolated {
		fxtCtor = tf.NewFixtureIsolated
		checkCtorErrFunc = func(t *testing.T, err error) {
			require.NotNil(t, err)
		}
		checkFunc = func(t *testing.T, fxt *tf.TestFixture) {
			require.Nil(t, fxt)
		}
	}

	// identity and work item link categories will always work

	t.Run("identities", func(t *testing.T) {
		// given
		c, err := fxtCtor(db, tf.Identities(n))
		// then
		require.Nil(t, err)
		require.Nil(t, c.Check())
		// manual checking
		require.Len(t, c.Identities, n)
	})
	t.Run("work item link categories", func(t *testing.T) {
		// given
		c, err := fxtCtor(db, tf.WorkItemLinkCategories(n))
		// then
		require.Nil(t, err)
		require.Nil(t, c.Check())
		// manual checking
		require.Len(t, c.WorkItemLinkCategories, n)
	})

	t.Run("spaces", func(t *testing.T) {
		// given
		c, err := fxtCtor(db, tf.Spaces(n))
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
		c, err := fxtCtor(db, tf.WorkItemLinkTypes(n))
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
		c, err := fxtCtor(db, tf.Codebases(n))
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
		c, err := fxtCtor(db, tf.WorkItemTypes(n))
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
		c, err := fxtCtor(db, tf.Iterations(n))
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
		c, err := fxtCtor(db, tf.Areas(n))
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
		c, err := fxtCtor(db, tf.WorkItems(n))
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
		c, err := fxtCtor(db, tf.Comments(n))
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
		c, err := fxtCtor(db, tf.WorkItemLinks(n))
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
	t.Run("labels", func(t *testing.T) {
		// given
		c, err := fxtCtor(db, tf.Labels(n))
		// then
		checkCtorErrFunc(t, err)
		checkFunc(t, c)
		// manual checking
		if !isolated {
			require.Len(t, c.Labels, n)
			require.Len(t, c.Spaces, 1)
		}
	})
}
