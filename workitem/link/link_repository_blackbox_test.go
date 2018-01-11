package link_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	_ "github.com/lib/pq" // need to import postgres driver
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type linkRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	workitemLinkRepo *link.GormWorkItemLinkRepository
}

func TestRunLinkRepoBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &linkRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../../config.yaml")})
}

func (s *linkRepoBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.workitemLinkRepo = link.NewWorkItemLinkRepository(s.DB)
}

func (s *linkRepoBlackBoxTest) TestList() {
	// tests total number of workitem children returned by list is equal to the
	// total number of workitem children created and total number of workitem
	// children in a page are equal to the "limit" specified
	s.T().Run("ok - count child work items", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItems(4), // parent + child 1-3
			tf.WorkItemLinkTypes(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemLinkTypes[idx].ForwardName = "parent of"
				return nil
			}),
			tf.WorkItemLinks(3, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemLinks[idx].SourceID = fxt.WorkItems[0].ID
				fxt.WorkItemLinks[idx].TargetID = fxt.WorkItems[idx+1].ID
				return nil
			}),
		)

		offset := 0
		limit := 1
		res, count, err := s.workitemLinkRepo.ListWorkItemChildren(s.Ctx, fxt.WorkItems[0].ID, &offset, &limit)
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, 3, int(count))
	})
}

func (s *linkRepoBlackBoxTest) TestWorkItemHasChildren() {
	s.T().Run("work item has no child after deletion", func(t *testing.T) {
		// given a work item link
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItems(2), // parent + child 1
			tf.WorkItemLinkTypes(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemLinkTypes[idx].ForwardName = "parent of"
				return nil
			}),
			tf.WorkItemLinks(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemLinks[idx].SourceID = fxt.WorkItems[0].ID
				fxt.WorkItemLinks[idx].TargetID = fxt.WorkItems[idx+1].ID
				return nil
			}),
		)

		// when this work item link is deleted
		err := s.workitemLinkRepo.Delete(s.Ctx, fxt.WorkItemLinks[0].ID, fxt.Identities[0].ID)
		require.NoError(t, err)

		// then it must not have any child
		hasChildren, err := s.workitemLinkRepo.WorkItemHasChildren(s.Ctx, fxt.WorkItems[0].ID)
		// then
		require.NoError(t, err)
		require.False(t, hasChildren)
	})
}

func (s *linkRepoBlackBoxTest) TestValidateTopology() {
	// given 2 work items linked with one tree-topology link type
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.WorkItems(3, tf.SetWorkItemTitles("parent", "child", "another-item")),
		tf.WorkItemLinkTypes(2,
			tf.SetTopologies(link.TopologyTree, link.TopologyTree),
			tf.SetWorkItemLinkTypeNames("tree-type", "another-type"),
		),
		tf.WorkItemLinks(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItemLinks[idx].SourceID = fxt.WorkItemByTitle("parent").ID
			fxt.WorkItemLinks[idx].TargetID = fxt.WorkItemByTitle("child").ID
			fxt.WorkItemLinks[idx].LinkTypeID = fxt.WorkItemLinkTypeByName("tree-type").ID
			return nil
		}),
	)

	s.T().Run("ok - no link", func(t *testing.T) {
		// given link type exists but no link to child item
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItems(2, tf.SetWorkItemTitles("foo", "bar")),
			tf.WorkItemLinkTypes(1, tf.SetTopologies(link.TopologyTree), tf.SetWorkItemLinkTypeNames("tree-type")),
		)
		// when
		err := s.workitemLinkRepo.ValidateTopology(s.Ctx, fxt.WorkItemByTitle("foo").ID, fxt.WorkItemByTitle("bar").ID, *fxt.WorkItemLinkTypeByName("tree-type"))
		// then: there must be no error because no link exists
		require.NoError(t, err)
	})

	s.T().Run("ok - link exists", func(t *testing.T) {
		err := s.workitemLinkRepo.ValidateTopology(s.Ctx, fxt.WorkItemByTitle("parent").ID, fxt.WorkItemByTitle("child").ID, *fxt.WorkItemLinkTypeByName("tree-type"))
		require.Error(t, err)
	})

	s.T().Run("ok - no link with same type", func(t *testing.T) {
		// when using another link type to validate
		err := s.workitemLinkRepo.ValidateTopology(s.Ctx, fxt.WorkItemByTitle("another-item").ID, fxt.WorkItemByTitle("child").ID, *fxt.WorkItemLinkTypeByName("another-type"))
		// then: there must be no error because no link of the same type exists
		require.NoError(t, err)
	})

	s.T().Run("fail - link exists", func(t *testing.T) {
		err := s.workitemLinkRepo.ValidateTopology(s.Ctx, fxt.WorkItemByTitle("another-item").ID, fxt.WorkItemByTitle("child").ID, *fxt.WorkItemLinkTypeByName("tree-type"))
		// then: there must be an error because a link of the same type already exists
		require.Error(t, err)
	})

	s.T().Run("fail - another link exists", func(t *testing.T) {
		err := s.workitemLinkRepo.ValidateTopology(s.Ctx, fxt.WorkItemByTitle("another-item").ID, fxt.WorkItemByTitle("child").ID, *fxt.WorkItemLinkTypeByName("tree-type"))
		// then: there must be an error because a link of the same type already exists with another parent
		require.Error(t, err)
	})
}

func (s *linkRepoBlackBoxTest) TestCreate() {
	s.T().Run("ok", func(t *testing.T) {
		t.Run("serial", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB,
				tf.WorkItems(2, tf.SetWorkItemTitles("parent", "child")),
				tf.WorkItemLinkTypes(1, tf.SetTopologies(link.TopologyTree), tf.SetWorkItemLinkTypeNames("tree-type")),
			)
			// when
			_, err := s.workitemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("parent").ID, fxt.WorkItemByTitle("child").ID, fxt.WorkItemLinkTypeByName("tree-type").ID, fxt.Identities[0].ID)
			// then
			require.NoError(t, err)
		})

		t.Run("2 concurrent requests to create A->B and B->A", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB,
				tf.WorkItems(2, tf.SetWorkItemTitles("A", "B")),
				tf.WorkItemLinkTypes(1, tf.SetTopologies(link.TopologyTree)),
			)

			N := 2
			wgBegin := sync.WaitGroup{}  // synced begin
			wgFinish := sync.WaitGroup{} // synced end
			errs := make([]error, N)
			errCnt := 0

			for n := 0; n < N; n++ {
				wgBegin.Add(1)
				wgFinish.Add(1)

				go func(i int) {
					t.Logf("entering go routine %d\n", i)
					defer func() {
						t.Logf("finishing go routine %d\n", i)
						wgFinish.Done()
					}()

					// Make sure that each go routine operates on its own
					// transaction
					db := s.DB.Begin()
					require.NoError(t, db.Error)
					defer func() {
						if db.Error != nil {
							t.Logf("rolling back transaction %d: %+v\n", i, db.Error)
							db.Rollback()
						} else {
							t.Logf("committing transaction %d\n", i)
							db.Commit()
						}
						require.NoError(t, db.Error)
					}()

					workitemLinkRepo := link.NewWorkItemLinkRepository(db)

					// barrier to synchronize creation of links
					wgBegin.Done()
					wgBegin.Wait()
					t.Logf("Let the games begin: %d", i)

					switch i {
					case 0:
						_, errs[i] = workitemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("A").ID, fxt.WorkItemByTitle("B").ID, fxt.WorkItemLinkTypes[0].ID, fxt.Identities[0].ID)
					case 1:
						_, errs[i] = workitemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("B").ID, fxt.WorkItemByTitle("A").ID, fxt.WorkItemLinkTypes[0].ID, fxt.Identities[0].ID)
					}

					if errs[i] != nil {
						errCnt++
					}
				}(n)
			}
			wgFinish.Wait()
			t.Log("All test go routines have returned.")
			require.Equal(t, N-1, errCnt, "expected %d out of %d concurrent routines to fail but here %d failed: %+v", N-1, N, errCnt, errs)

			t.Run("only one link was created", func(t *testing.T) {
				links := []link.WorkItemLink{}
				db := s.DB.Table(link.WorkItemLink{}.TableName()).Where("link_type_id = ?", fxt.WorkItemLinkTypes[0].ID).Find(&links)
				require.NoError(t, db.Error)
				require.Len(t, links, 1)
			})
		})
	})

	s.T().Run("fail - other parent-child-link exists", func(t *testing.T) {
		// given 2 work items linked with one tree-topology link type
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItems(3, tf.SetWorkItemTitles("parent", "child", "another-item")),
			tf.WorkItemLinkTypes(1,
				tf.SetTopologies(link.TopologyTree),
				tf.SetWorkItemLinkTypeNames("tree-type"),
			),
			tf.WorkItemLinks(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemLinks[idx].SourceID = fxt.WorkItemByTitle("parent").ID
				fxt.WorkItemLinks[idx].TargetID = fxt.WorkItemByTitle("child").ID
				return nil
			}),
		)
		// when try to link parent#2 to child
		_, err := s.workitemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("another-item").ID, fxt.WorkItemByTitle("child").ID, fxt.WorkItemLinkTypeByName("tree-type").ID, fxt.Identities[0].ID)
		// then expect an error because a parent/link relation already exists with the child item
		require.Error(t, err)
	})

	s.T().Run("fail - multiple parents with tree-topology-based link type", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItems(3, tf.SetWorkItemTitles("parent1", "parent2", "child")),
			tf.WorkItemLinkTypes(1, tf.SetTopologies(link.TopologyTree), tf.SetWorkItemLinkTypeNames("tree-type")),
		)
		// when creating link between "parent1" and "child"
		_, err := s.workitemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("parent1").ID, fxt.WorkItemByTitle("child").ID, fxt.WorkItemLinkTypeByName("tree-type").ID, fxt.Identities[0].ID)
		// then it works
		require.NoError(t, err)
		// when creating link between "parent2" and "child"
		_, err = s.workitemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("parent2").ID, fxt.WorkItemByTitle("child").ID, fxt.WorkItemLinkTypeByName("tree-type").ID, fxt.Identities[0].ID)
		// then we expect an error because "child" is already a child of "parent1"
		require.Error(t, err)
	})

	s.T().Run("cycle detection", func(t *testing.T) {

		// These are the scenarios we test here.
		//
		// Legend
		// ------
		//
		//   \ = link
		//   * = new link
		//   C = the element that is potentially causing the cycle
		//
		// Scenarios
		// ---------
		//
		//   I:        II:       III:      IV:       V:       VI:
		//
		//    C         C         C         C         A        A
		//     *         \         *         *         \        \
		//      A         A         A         A         B        C
		//       \         \         \         \         *        \
		//        B         B         C         B         C        B
		//         \         *         \                            *
		//          C         C         B                            C
		//
		// In a "tree" topology:
		//   I, II, III are cycles
		//   IV and V are no cycles.
		//   VI violates the single-parent rule
		//
		// In a "dependency" topology:
		//   I, II, III, and VI are cycles
		//   IV and V are no cycles.

		// Map topologies to expected error during cycle
		topos := map[link.Topology][]bool{
			link.TopologyNetwork:         {false, false, false, false, false, false},
			link.TopologyDirectedNetwork: {false, false, false, false, false, false},
			link.TopologyTree:            {true, true, true, false, false, true},
			link.TopologyDependency:      {true, true, true, false, false, true},
		}
		for topo, errorExpected := range topos {
			t.Run("topology: "+topo.String(), func(t *testing.T) {
				t.Run("Scenario I: C*A-B-C", func(t *testing.T) {
					// given
					fxt := tf.NewTestFixture(t, s.DB,
						tf.WorkItems(3, tf.SetWorkItemTitles("A", "B", "C")),
						tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
						tf.WorkItemLinksCustom(2, tf.BuildLinks(tf.LinkChain("A", "B", "C")...)),
					)
					// when
					_, err := s.workitemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("C").ID, fxt.WorkItemByTitle("A").ID, fxt.WorkItemLinkTypes[0].ID, fxt.Identities[0].ID)
					// then
					if errorExpected[0] {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}
				})
				t.Run("Scenario II: C-A-B*C", func(t *testing.T) {
					// given
					fxt := tf.NewTestFixture(t, s.DB,
						tf.WorkItems(3, tf.SetWorkItemTitles("C", "A", "B")),
						tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
						tf.WorkItemLinksCustom(2, tf.BuildLinks(tf.LinkChain("C", "A", "B")...)),
					)
					// when
					_, err := s.workitemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("B").ID, fxt.WorkItemByTitle("C").ID, fxt.WorkItemLinkTypes[0].ID, fxt.Identities[0].ID)
					// then
					if errorExpected[1] {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}
				})
				t.Run("Scenario III: C*A-C-B", func(t *testing.T) {
					// given
					fxt := tf.NewTestFixture(t, s.DB,
						tf.WorkItems(3, tf.SetWorkItemTitles("A", "C", "B")),
						tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
						tf.WorkItemLinksCustom(2, tf.BuildLinks(tf.LinkChain("A", "C", "B")...)),
					)
					// when
					_, err := s.workitemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("C").ID, fxt.WorkItemByTitle("A").ID, fxt.WorkItemLinkTypes[0].ID, fxt.Identities[0].ID)
					// then
					if errorExpected[2] {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}
				})
				t.Run("Scenario IV: C*A-B", func(t *testing.T) {
					// given
					fxt := tf.NewTestFixture(t, s.DB,
						tf.WorkItems(3, tf.SetWorkItemTitles("A", "B", "C")),
						tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
						tf.WorkItemLinksCustom(1, tf.BuildLinks(tf.LinkChain("A", "B")...)),
					)
					// when
					_, err := s.workitemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("C").ID, fxt.WorkItemByTitle("A").ID, fxt.WorkItemLinkTypes[0].ID, fxt.Identities[0].ID)
					// then
					if errorExpected[3] {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}
				})
				t.Run("Scenario V: A-B*C", func(t *testing.T) {
					// given
					fxt := tf.NewTestFixture(t, s.DB,
						tf.WorkItems(3, tf.SetWorkItemTitles("A", "B", "C")),
						tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
						tf.WorkItemLinksCustom(1, tf.BuildLinks(tf.LinkChain("A", "B")...)),
					)
					// when
					_, err := s.workitemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("B").ID, fxt.WorkItemByTitle("C").ID, fxt.WorkItemLinkTypes[0].ID, fxt.Identities[0].ID)
					// then
					if errorExpected[4] {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}
				})
				t.Run("Scenario VI: A-C-B*C", func(t *testing.T) {
					// given
					fxt := tf.NewTestFixture(t, s.DB,
						tf.WorkItems(3, tf.SetWorkItemTitles("A", "B", "C")),
						tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
						tf.WorkItemLinksCustom(2, tf.BuildLinks(tf.LinkChain("A", "C", "B")...)),
					)
					// when
					_, err := s.workitemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("B").ID, fxt.WorkItemByTitle("C").ID, fxt.WorkItemLinkTypes[0].ID, fxt.Identities[0].ID)
					// then
					if errorExpected[5] {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}
				})
			})
		}
	})
}

func (s *linkRepoBlackBoxTest) TestExistsLink() {
	s.T().Run("link exists", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinks(1))
		err := s.workitemLinkRepo.CheckExists(s.Ctx, fxt.WorkItemLinks[0].ID)
		require.NoError(t, err)
	})

	s.T().Run("link doesn't exist", func(t *testing.T) {
		err := s.workitemLinkRepo.CheckExists(s.Ctx, uuid.NewV4())
		require.IsType(t, errors.NotFoundError{}, err)
	})
}

func (s *linkRepoBlackBoxTest) TestGetParentID() {
	// create 1 links between 2 work items having TopologyNetwork with ForwardName = "parent of"
	fixtures := tf.NewTestFixture(s.T(), s.DB, tf.WorkItemLinks(1), tf.WorkItemLinkTypes(1, tf.SetTopologies(link.TopologyTree), func(fxt *tf.TestFixture, idx int) error {
		fxt.WorkItemLinkTypes[idx].ForwardName = "parent of"
		return nil
	}))
	parentID, err := s.workitemLinkRepo.GetParentID(s.Ctx, fixtures.WorkItems[1].ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), fixtures.WorkItems[0].ID, *parentID)
}

func (s *linkRepoBlackBoxTest) TestGetParentIDNotExist() {
	// create 1 links between 2 work items having TopologyNetwork with ForwardName = "parent of"
	fixtures := tf.NewTestFixture(s.T(), s.DB, tf.WorkItemLinks(1), tf.WorkItemLinkTypes(1, tf.SetTopologies(link.TopologyTree), func(fxt *tf.TestFixture, idx int) error {
		fxt.WorkItemLinkTypes[idx].ForwardName = "parent of"
		return nil
	}))
	parentID, err := s.workitemLinkRepo.GetParentID(s.Ctx, fixtures.WorkItems[0].ID)
	require.Error(s.T(), err)
	assert.Nil(s.T(), parentID)
}

func (s *linkRepoBlackBoxTest) TestGetAncestors() {
	validateAncestry := func(t *testing.T, fxt *tf.TestFixture, toBeFound map[link.Ancestor]struct{}, ancestors []link.Ancestor) {
		// uncomment for more information:
		// for _, ancestor := range ancestors {
		// t.Logf("Ancestor: %s For: %s IsRoot: %t\n",
		// fxt.WorkItemByID(ancestor.ID).Fields[workitem.SystemTitle].(string),
		// fxt.WorkItemByID(ancestor.OriginalChildID).Fields[workitem.SystemTitle].(string),
		// ancestor.IsRoot,
		// )
		// }

		for _, ancestor := range ancestors {
			wi := fxt.WorkItemByID(ancestor.ID)
			assert.NotNil(t, wi, "failed to find work item with ID: %s", ancestor.ID)
			originalChild := fxt.WorkItemByID(ancestor.OriginalChildID)
			assert.NotNil(t, wi, "failed to find work item with ID: %s", ancestor.OriginalChildID)
			if wi != nil {
				t.Logf("found ancestor: %s for: %s (is root: %t)", wi.Fields[workitem.SystemTitle].(string), originalChild.Fields[workitem.SystemTitle].(string), ancestor.IsRoot)
			}
			_, ok := toBeFound[ancestor]
			require.True(t, ok, "found unexpected ancestor: %s", fxt.WorkItemByID(ancestor.ID).Fields[workitem.SystemTitle].(string))
			delete(toBeFound, ancestor)
		}
		require.Empty(t, toBeFound, "failed to find these ancestors in list: %s", func() string {
			titles := []string{}
			for ancestor := range toBeFound {
				titles = append(titles, "\""+fxt.WorkItemByID(ancestor.ID).Fields[workitem.SystemTitle].(string)+"\"")
			}
			return strings.Join(titles, ", ")
		}())
	}

	allTopologies := []link.Topology{link.TopologyDependency, link.TopologyDirectedNetwork, link.TopologyNetwork, link.TopologyTree}

	for _, topo := range allTopologies {
		s.T().Run("topology: "+topo.String(), func(t *testing.T) {

			t.Run("straight chain A-B-C-D", func(t *testing.T) {
				fxt := tf.NewTestFixture(t, s.DB,
					tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
					tf.WorkItems(5, tf.SetWorkItemTitles("A", "B", "C", "D", "E")),
					tf.WorkItemLinksCustom(3, tf.BuildLinks(tf.LinkChain("A", "B", "C", "D")...)),
				)
				// to shorten the test code below
				A := fxt.WorkItemByTitle("A").ID
				B := fxt.WorkItemByTitle("B").ID
				C := fxt.WorkItemByTitle("C").ID
				D := fxt.WorkItemByTitle("D").ID
				E := fxt.WorkItemByTitle("E").ID

				t.Run("ancestors for E (none expected)", func(t *testing.T) {
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, E)
					require.NoError(t, err)
					validateAncestry(t, fxt, nil, ancestors)
				})

				t.Run("ancestors for A (none expected)", func(t *testing.T) {
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, A)
					require.NoError(t, err)
					validateAncestry(t, fxt, nil, ancestors)
				})

				t.Run("ancestors for D (expecting A,B,C)", func(t *testing.T) {
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, D)
					require.NoError(t, err)
					validateAncestry(t, fxt, map[link.Ancestor]struct{}{
						{ID: A, OriginalChildID: D, IsRoot: true}:  {},
						{ID: B, OriginalChildID: D, IsRoot: false}: {},
						{ID: C, OriginalChildID: D, IsRoot: false}: {},
					}, ancestors)
				})

				t.Run("ancestors for C (expecting A,B)", func(t *testing.T) {
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, C)
					require.NoError(t, err)
					validateAncestry(t, fxt, map[link.Ancestor]struct{}{
						{ID: A, OriginalChildID: C, IsRoot: true}:  {},
						{ID: B, OriginalChildID: C, IsRoot: false}: {},
					}, ancestors)
				})

				t.Run("ancestors for D, and C (expecting, A,B,C and A,B)", func(t *testing.T) {
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, D, C)
					require.NoError(t, err)
					validateAncestry(t, fxt, map[link.Ancestor]struct{}{
						// for D
						{ID: A, OriginalChildID: D, IsRoot: true}:  {},
						{ID: B, OriginalChildID: D, IsRoot: false}: {},
						{ID: C, OriginalChildID: D, IsRoot: false}: {},
						// for C
						{ID: A, OriginalChildID: C, IsRoot: true}:  {},
						{ID: B, OriginalChildID: C, IsRoot: false}: {},
					}, ancestors)
				})

			})

			// Two distinct trees:
			//
			//   A
			//   |_ B
			//     |_ C
			//   |_ D
			//     |_ E
			//
			//   X
			//   |_ Y
			t.Run("two distinct trees", func(t *testing.T) {
				chain := tf.LinkChain("A", "B", "C")
				chain = append(chain, tf.LinkChain("A", "D", "E")...)
				chain = append(chain, tf.LinkChain("X", "Y")...)

				fxt := tf.NewTestFixture(t, s.DB,
					tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
					tf.WorkItems(7, tf.SetWorkItemTitles("A", "B", "C", "D", "E", "X", "Y")),
					tf.WorkItemLinksCustom(len(chain), tf.BuildLinks(chain...)),
				)
				// to shorten the test code below
				A := fxt.WorkItemByTitle("A").ID
				// B := fxt.WorkItemByTitle("B").ID
				// C := fxt.WorkItemByTitle("C").ID
				D := fxt.WorkItemByTitle("D").ID
				E := fxt.WorkItemByTitle("E").ID
				X := fxt.WorkItemByTitle("X").ID
				Y := fxt.WorkItemByTitle("Y").ID

				t.Run("ancestors for Y and E (expecting X and A,D)", func(t *testing.T) {
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, Y, E)
					require.NoError(t, err)
					validateAncestry(t, fxt, map[link.Ancestor]struct{}{
						// for Y
						{ID: X, OriginalChildID: Y, IsRoot: true}: {},
						// for E
						{ID: A, OriginalChildID: E, IsRoot: true}:  {},
						{ID: D, OriginalChildID: E, IsRoot: false}: {},
					}, ancestors)
				})
			})

		})
	}
}
