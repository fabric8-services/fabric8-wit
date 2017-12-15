package testfixture_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/workitem/link"

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
}

func (s *testFixtureSuite) TestNewFixture_Advanced() {
	s.T().Run("implicitly created entities", func(t *testing.T) {
		c, err := tf.NewFixture(s.DB, tf.WorkItems(2))
		require.NoError(t, err)
		require.Nil(t, c.Check())
	})
	s.T().Run("explicitly create entities", func(t *testing.T) {
		// given
		c, err := tf.NewFixture(s.DB, tf.WorkItems(2))
		require.NoError(t, err)
		require.Nil(t, c.Check())

		// manually use values from previous fixture over fields from first fixture
		c1, err := tf.NewFixtureIsolated(s.DB, tf.WorkItems(3, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].SpaceID = c.Spaces[0].ID
			fxt.WorkItems[idx].Type = c.WorkItemTypes[0].ID
			fxt.WorkItems[idx].Fields[workitem.SystemCreator] = c.Identities[0].ID.String()
			return nil
		}))
		require.NoError(t, err)
		require.Nil(t, c1.Check())
	})
	s.T().Run("create 100 comments by 100 authors on 1 workitem", func(t *testing.T) {
		c, err := tf.NewFixture(s.DB, tf.Identities(100), tf.Comments(100, func(fxt *tf.TestFixture, idx int) error {
			fxt.Comments[idx].Creator = fxt.Identities[idx].ID
			return nil
		}))
		require.NoError(t, err)
		require.Nil(t, c.Check())
	})
	s.T().Run("create 10 links between 20 work items with a network topology link type", func(t *testing.T) {
		c, err := tf.NewFixture(s.DB, tf.WorkItemLinks(10), tf.WorkItemLinkTypes(1, tf.SetTopologies(link.TopologyNetwork)))
		require.NoError(t, err)
		require.Nil(t, c.Check())
	})
	s.T().Run("test CreateWorkItemEnvironment error", func(t *testing.T) {
		c, err := tf.NewFixture(s.DB, tf.CreateWorkItemEnvironment(), tf.Spaces(2))
		require.Error(t, err)
		require.Nil(t, c)
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
		require.NoError(t, err)
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
			require.Error(t, err)
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
		require.NoError(t, err)
		require.Nil(t, c.Check())
		// manual checking
		require.Len(t, c.Identities, n)
	})
	t.Run("work item link categories", func(t *testing.T) {
		// given
		c, err := fxtCtor(db, tf.WorkItemLinkCategories(n))
		// then
		require.NoError(t, err)
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

func (s *testFixtureSuite) TestWorkItemLinks() {
	s.T().Run("standard", func(t *testing.T) {
		// when
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinks(3))
		// then
		require.Len(t, fxt.WorkItemLinks, 3)
		require.Len(t, fxt.WorkItems, 6)
	})
	s.T().Run("custom", func(t *testing.T) {
		t.Run("missing work items and link setup", func(t *testing.T) {
			// when
			fxt, err := tf.NewFixture(s.DB, tf.WorkItemLinksCustom(3))
			// then we expect an error because you're supposed to create work items
			// yourself and link them on your own when using the custom method
			require.Error(t, err)
			require.Nil(t, fxt)
		})
		t.Run("missing link setup", func(t *testing.T) {
			// when
			fxt, err := tf.NewFixture(s.DB, tf.WorkItemLinksCustom(3), tf.WorkItems(3))
			// then we expect an error because you're supposed to setup links
			// yourself when using the custom method
			require.Error(t, err)
			require.Nil(t, fxt)
		})
		t.Run("ok", func(t *testing.T) {
			// when
			fxt, err := tf.NewFixture(s.DB,
				tf.WorkItems(3),
				tf.WorkItemLinksCustom(2, func(fxt *tf.TestFixture, idx int) error {
					l := fxt.WorkItemLinks[idx]
					switch idx {
					case 0:
						l.SourceID = fxt.WorkItems[0].ID
						l.TargetID = fxt.WorkItems[1].ID
					case 1:
						l.SourceID = fxt.WorkItems[1].ID
						l.TargetID = fxt.WorkItems[2].ID
					}
					return nil
				}),
			)
			// then we expect an error because you're supposed to setup links
			// yourself when using the custom method
			require.NoError(t, err)
			require.NotNil(t, fxt)
			require.Len(t, fxt.WorkItemLinks, 2)
			require.Len(t, fxt.WorkItems, 3)
		})
		t.Run("mixture not allowed (normal first)", func(t *testing.T) {
			// when
			fxt, err := tf.NewFixture(s.DB,
				tf.WorkItems(3),
				tf.WorkItemLinks(1),
				tf.WorkItemLinksCustom(2, func(fxt *tf.TestFixture, idx int) error {
					l := fxt.WorkItemLinks[idx]
					switch idx {
					case 0:
						l.SourceID = fxt.WorkItems[0].ID
						l.TargetID = fxt.WorkItems[1].ID
					case 1:
						l.SourceID = fxt.WorkItems[1].ID
						l.TargetID = fxt.WorkItems[2].ID
					}
					return nil
				}),
			)
			// then we expect an error because you're supposed to mix
			// WorkItemLinks and WorkItemLinksCustom
			require.Error(t, err)
			require.Nil(t, fxt)
		})
		t.Run("mixture not allowed (normal second)", func(t *testing.T) {
			// when
			fxt, err := tf.NewFixture(s.DB,
				tf.WorkItems(3),
				tf.WorkItemLinksCustom(2, func(fxt *tf.TestFixture, idx int) error {
					l := fxt.WorkItemLinks[idx]
					switch idx {
					case 0:
						l.SourceID = fxt.WorkItems[0].ID
						l.TargetID = fxt.WorkItems[1].ID
					case 1:
						l.SourceID = fxt.WorkItems[1].ID
						l.TargetID = fxt.WorkItems[2].ID
					}
					return nil
				}),
				tf.WorkItemLinks(1),
			)
			// then we expect an error because you're supposed to mix
			// WorkItemLinks and WorkItemLinksCustom
			require.Error(t, err)
			require.Nil(t, fxt)
		})
	})
}
