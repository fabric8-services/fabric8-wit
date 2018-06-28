package link_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/ptr"
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
	workitemRepo     *workitem.GormWorkItemRepository
}

func TestRunLinkRepoBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &linkRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *linkRepoBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.workitemLinkRepo = link.NewWorkItemLinkRepository(s.DB)
	s.workitemRepo = workitem.NewWorkItemRepository(s.DB)
}

func (s *linkRepoBlackBoxTest) TestList() {
	// tests total number of workitem children returned by list is equal to the
	// total number of workitem children created and total number of workitem
	// children in a page are equal to the "limit" specified
	s.T().Run("ok - count child work items", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItems(4, tf.SetWorkItemTitles("parent", "child1", "child2", "child3")),
			tf.WorkItemLinksCustom(3, func(fxt *tf.TestFixture, idx int) error {
				l := fxt.WorkItemLinks[idx]
				l.LinkTypeID = link.SystemWorkItemLinkTypeParentChildID
				l.SourceID = fxt.WorkItems[0].ID
				l.TargetID = fxt.WorkItems[idx+1].ID
				return nil
			}),
		)
		res, count, err := s.workitemLinkRepo.ListWorkItemChildren(s.Ctx, fxt.WorkItemByTitle("parent").ID, ptr.Int(0), ptr.Int(1))
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, 3, int(count))
		require.Equal(t, fxt.WorkItemByTitle("child3").ID, res[0].ID)
	})
}

func (s *linkRepoBlackBoxTest) TestReorder() {
	// setup creates 1 parent with 3 children
	setup := func(t *testing.T) *tf.TestFixture {
		var fxt *tf.TestFixture
		t.Run("setup", func(t *testing.T) {
			fxt = tf.NewTestFixture(t, s.DB,
				tf.WorkItems(4, tf.SetWorkItemTitles("parent", "child1", "child2", "child3")),
				tf.WorkItemLinks(3, func(fxt *tf.TestFixture, idx int) error {
					l := fxt.WorkItemLinks[idx]
					l.LinkTypeID = link.SystemWorkItemLinkTypeParentChildID
					l.SourceID = fxt.WorkItems[0].ID
					l.TargetID = fxt.WorkItems[idx+1].ID
					return nil
				}),
			)
			// Expect children in descending order (sorted by their execution order)
			beforeReorder, _, err := s.workitemLinkRepo.ListWorkItemChildren(s.Ctx, fxt.WorkItems[0].ID, nil, nil)
			require.NoError(t, err)
			require.Len(t, beforeReorder, 3)
			require.Equal(t, fxt.WorkItemByTitle("child3").ID, beforeReorder[0].ID)
			require.Equal(t, fxt.WorkItemByTitle("child2").ID, beforeReorder[1].ID)
			require.Equal(t, fxt.WorkItemByTitle("child1").ID, beforeReorder[2].ID)
		})
		require.NotNil(t, fxt)
		return fxt
	}
	s.T().Run(string(workitem.DirectionAbove), func(t *testing.T) {
		fxt := setup(t)
		// when moving child1 above child2
		_, err := s.workitemRepo.Reorder(s.Ctx, fxt.Spaces[0].ID, workitem.DirectionAbove, &fxt.WorkItemByTitle("child2").ID, *fxt.WorkItemByTitle("child1"), fxt.Identities[0].ID)
		require.NoError(t, err)
		// then
		afterReorder, _, err := s.workitemLinkRepo.ListWorkItemChildren(s.Ctx, fxt.WorkItemByTitle("parent").ID, nil, nil)
		require.NoError(t, err)
		require.Len(t, afterReorder, 3)
		require.Equal(t, fxt.WorkItemByTitle("child3").ID, afterReorder[0].ID)
		require.Equal(t, fxt.WorkItemByTitle("child1").ID, afterReorder[1].ID)
		require.Equal(t, fxt.WorkItemByTitle("child2").ID, afterReorder[2].ID)
	})
	s.T().Run(string(workitem.DirectionBelow), func(t *testing.T) {
		fxt := setup(t)
		// when moving child3 below child2
		_, err := s.workitemRepo.Reorder(s.Ctx, fxt.Spaces[0].ID, workitem.DirectionBelow, &fxt.WorkItemByTitle("child2").ID, *fxt.WorkItemByTitle("child3"), fxt.Identities[0].ID)
		require.NoError(t, err)
		// then
		afterReorder, _, err := s.workitemLinkRepo.ListWorkItemChildren(s.Ctx, fxt.WorkItemByTitle("parent").ID, nil, nil)
		require.NoError(t, err)
		require.Len(t, afterReorder, 3)
		require.Equal(t, fxt.WorkItemByTitle("child2").ID, afterReorder[0].ID)
		require.Equal(t, fxt.WorkItemByTitle("child3").ID, afterReorder[1].ID)
		require.Equal(t, fxt.WorkItemByTitle("child1").ID, afterReorder[2].ID)
	})
	s.T().Run(string(workitem.DirectionTop), func(t *testing.T) {
		fxt := setup(t)
		// when moving child1 to top
		_, err := s.workitemRepo.Reorder(s.Ctx, fxt.Spaces[0].ID, workitem.DirectionTop, nil, *fxt.WorkItemByTitle("child1"), fxt.Identities[0].ID)
		require.NoError(t, err)
		// then
		afterReorder, _, err := s.workitemLinkRepo.ListWorkItemChildren(s.Ctx, fxt.WorkItemByTitle("parent").ID, nil, nil)
		require.NoError(t, err)
		require.Len(t, afterReorder, 3)
		require.Equal(t, fxt.WorkItemByTitle("child1").ID, afterReorder[0].ID)
		require.Equal(t, fxt.WorkItemByTitle("child3").ID, afterReorder[1].ID)
		require.Equal(t, fxt.WorkItemByTitle("child2").ID, afterReorder[2].ID)
	})
	s.T().Run(string(workitem.DirectionBottom), func(t *testing.T) {
		fxt := setup(t)
		// when moving child3 to bottom
		_, err := s.workitemRepo.Reorder(s.Ctx, fxt.Spaces[0].ID, workitem.DirectionBottom, nil, *fxt.WorkItemByTitle("child3"), fxt.Identities[0].ID)
		require.NoError(t, err)
		// then
		afterReorder, _, err := s.workitemLinkRepo.ListWorkItemChildren(s.Ctx, fxt.WorkItemByTitle("parent").ID, nil, nil)
		require.NoError(t, err)
		require.Len(t, afterReorder, 3)
		require.Equal(t, fxt.WorkItemByTitle("child2").ID, afterReorder[0].ID)
		require.Equal(t, fxt.WorkItemByTitle("child1").ID, afterReorder[1].ID)
		require.Equal(t, fxt.WorkItemByTitle("child3").ID, afterReorder[2].ID)
	})
	s.T().Run("invalid", func(t *testing.T) {
		fxt := setup(t)
		directions := []workitem.DirectionType{workitem.DirectionAbove, workitem.DirectionBelow, workitem.DirectionBottom, workitem.DirectionTop}
		for _, direction := range directions {
			t.Run(string(direction), func(t *testing.T) {
				t.Run("empty workitem", func(t *testing.T) {
					_, err := s.workitemRepo.Reorder(s.Ctx, fxt.Spaces[0].ID, direction, &fxt.WorkItemByTitle("child2").ID, workitem.WorkItem{}, fxt.Identities[0].ID)
					require.Error(t, err)
				})
				t.Run("targetID", func(t *testing.T) {
					switch direction {
					case workitem.DirectionAbove, workitem.DirectionBelow:
						t.Run("unknown", func(t *testing.T) {
							_, err := s.workitemRepo.Reorder(s.Ctx, fxt.Spaces[0].ID, direction, ptr.UUID(uuid.NewV4()), *fxt.WorkItemByTitle("child2"), fxt.Identities[0].ID)
							require.Error(t, err)
						})
						t.Run("nil", func(t *testing.T) {
							_, err := s.workitemRepo.Reorder(s.Ctx, fxt.Spaces[0].ID, direction, nil, *fxt.WorkItemByTitle("child2"), fxt.Identities[0].ID)
							require.Error(t, err)
						})
					case workitem.DirectionTop, workitem.DirectionBottom:
						_, err := s.workitemRepo.Reorder(s.Ctx, fxt.Spaces[0].ID, direction, ptr.UUID(uuid.NewV4()), *fxt.WorkItemByTitle("child2"), fxt.Identities[0].ID)
						require.Error(t, err)
					}
				})
				t.Run("invalid space ID", func(t *testing.T) {
					_, err := s.workitemRepo.Reorder(s.Ctx, uuid.NewV4(), direction, &fxt.WorkItemByTitle("child1").ID, *fxt.WorkItemByTitle("child2"), fxt.Identities[0].ID)
					require.Error(t, err)
				})
				t.Run("unknown modifier", func(t *testing.T) {
					_, err := s.workitemRepo.Reorder(s.Ctx, fxt.Spaces[0].ID, direction, &fxt.WorkItemByTitle("child1").ID, *fxt.WorkItemByTitle("child2"), uuid.NewV4())
					require.Error(t, err)
				})
			})
		}
	})
}

func (s *linkRepoBlackBoxTest) TestWorkItemHasChildren() {
	s.T().Run("work item has no child after deletion", func(t *testing.T) {
		// given a work item link
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItems(2), // parent + child 1
			tf.WorkItemLinksCustom(1, func(fxt *tf.TestFixture, idx int) error {
				l := fxt.WorkItemLinks[idx]
				l.LinkTypeID = link.SystemWorkItemLinkTypeParentChildID
				l.SourceID = fxt.WorkItems[0].ID
				l.TargetID = fxt.WorkItems[idx+1].ID
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

// createLinksConcurrently accepts a list of function of which only 1 is
// supposed to succeed; the rest are supposed to fail for (concurrency reasons).
func (s *linkRepoBlackBoxTest) createLinksConcurrently(t *testing.T, fxt *tf.TestFixture, numLinksExpected int, fns ...func(fxt *tf.TestFixture, workItemLinkRepo link.WorkItemLinkRepository) error) {
	N := len(fns)
	wgBegin := sync.WaitGroup{} // synced begin
	wgBegin.Add(N)
	wgFinish := sync.WaitGroup{} // synced end
	wgFinish.Add(N)
	errs := make([]error, N)
	errCnt := 0

	for n := 0; n < N; n++ {
		go func(i int) {
			defer func() {
				wgFinish.Done()
			}()

			// Make sure that each go routine operates on its own
			// transaction
			db := s.DB.Begin()
			require.NoError(t, db.Error)
			defer func() {
				if errs[i] != nil {
					db.Rollback()
				} else {
					db.Commit()
				}
				require.NoError(t, db.Error)
			}()

			workitemLinkRepo := link.NewWorkItemLinkRepository(db)

			// barrier to synchronize creation of links
			wgBegin.Done()
			wgBegin.Wait()

			// Execute the i-th function
			errs[i] = fns[i](fxt, workitemLinkRepo)

			if errs[i] != nil {
				errCnt++
			}
		}(n)
	}
	wgFinish.Wait()
	// require.Equal(t, numLinksExpected, errCnt, "expected %d out of %d concurrent routines to fail but here %d failed: %+v", numLinksExpected, N, errCnt, errs)
	t.Run(fmt.Sprintf("total #links is %d", numLinksExpected), func(t *testing.T) {
		links := []link.WorkItemLink{}
		db := s.DB.Table(link.WorkItemLink{}.TableName()).Where("link_type_id = ?", fxt.WorkItemLinkTypes[0].ID).Find(&links)
		require.NoError(t, db.Error)
		require.Len(t, links, numLinksExpected)
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
	})

	s.T().Run("fail", func(t *testing.T) {
		t.Run("single-parent violation in tree topology", func(t *testing.T) {
			// given 2 work items linked with one tree-topology link type
			fxt := tf.NewTestFixture(t, s.DB,
				tf.WorkItems(3, tf.SetWorkItemTitles("A", "B", "C")),
				tf.WorkItemLinkTypes(1, tf.SetTopologies(link.TopologyTree)),
				tf.WorkItemLinks(1, tf.BuildLinks(tf.LinkChain("A", "B")...)),
			)
			// when
			_, err := s.workitemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("C").ID, fxt.WorkItemByTitle("B").ID, fxt.WorkItemLinkTypes[0].ID, fxt.Identities[0].ID)
			// then expect an error because a parent/link relation already exists with the child item
			require.Error(t, err)
		})
		t.Run("cross-space linking", func(t *testing.T) {
			// given 2 work items in two different spaces isn't allowed in any topology
			topos := []link.Topology{link.TopologyDependency, link.TopologyDirectedNetwork, link.TopologyNetwork, link.TopologyTree}
			for _, topo := range topos {
				fxt1 := tf.NewTestFixture(t, s.DB,
					tf.WorkItems(1, tf.SetWorkItemTitles("A")),
					tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
				)
				fxt2 := tf.NewTestFixture(t, s.DB,
					tf.WorkItems(1, tf.SetWorkItemTitles("B")),
				)
				// when
				_, err := s.workitemLinkRepo.Create(s.Ctx, fxt1.WorkItemByTitle("A").ID, fxt2.WorkItemByTitle("B").ID, fxt1.WorkItemLinkTypes[0].ID, fxt1.Identities[0].ID)
				// then
				require.Error(t, err)
			}
		})
	})

	s.T().Run("cycle detection", func(t *testing.T) {
		t.Run("serial", func(t *testing.T) {
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
		t.Run("concurrent", func(t *testing.T) {
			// Scenarios
			//
			//  I:
			//  	Topology: -
			//  	Alice   : A*B
			//  	Bob     : B*A
			//
			//  II:
			// 		A->B->C * D->E->F
			// 		   ^         |
			// 		    \__ * ___/
			//
			//  III:
			//		A->B->C * D->E->F
			//		 ^              |
			//		  \____ * ______/
			// Map topologies to expected error during cycle
			topos := map[link.Topology][]bool{
				link.TopologyNetwork:         {false, false, false},
				link.TopologyDirectedNetwork: {false, false, false},
				link.TopologyTree:            {true, true, true},
				link.TopologyDependency:      {true, true, true},
			}
			for topo, errorExpected := range topos {
				t.Run("topology: "+topo.String(), func(t *testing.T) {
					t.Run("create A->B and B->A", func(t *testing.T) {
						fxt := tf.NewTestFixture(t, s.DB,
							tf.WorkItems(2, tf.SetWorkItemTitles("A", "B")),
							tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
						)
						addon := 1
						if !errorExpected[0] {
							addon++
						}
						s.createLinksConcurrently(t, fxt, addon,
							func(fxt *tf.TestFixture, workItemLinkRepo link.WorkItemLinkRepository) error {
								_, err := workItemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("A").ID, fxt.WorkItemByTitle("B").ID, fxt.WorkItemLinkTypes[0].ID, fxt.Identities[0].ID)
								return err
							}, func(fxt *tf.TestFixture, workItemLinkRepo link.WorkItemLinkRepository) error {
								_, err := workItemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("B").ID, fxt.WorkItemByTitle("A").ID, fxt.WorkItemLinkTypes[0].ID, fxt.Identities[0].ID)
								return err
							},
						)
					})
					// Concurrently create these links
					// A->B->C * D->E->F
					//    ^         |
					//     \___*____|
					t.Run("given A->B->C and D->E->F, now create E->B and C->D", func(t *testing.T) {
						chain := append(tf.LinkChain("A", "B", "C"), tf.LinkChain("D", "E", "F")...)
						fxt := tf.NewTestFixture(t, s.DB,
							tf.WorkItems(6, tf.SetWorkItemTitles("A", "B", "C", "D", "E", "F")),
							tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
							tf.WorkItemLinksCustom(len(chain), tf.BuildLinks(chain...)),
						)
						addon := 1
						if !errorExpected[1] {
							addon++
						}
						s.createLinksConcurrently(t, fxt, len(chain)+addon,
							func(fxt *tf.TestFixture, workItemLinkRepo link.WorkItemLinkRepository) error {
								_, err := workItemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("C").ID, fxt.WorkItemByTitle("D").ID, fxt.WorkItemLinkTypes[0].ID, fxt.Identities[0].ID)
								return err
							}, func(fxt *tf.TestFixture, workItemLinkRepo link.WorkItemLinkRepository) error {
								_, err := workItemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("E").ID, fxt.WorkItemByTitle("B").ID, fxt.WorkItemLinkTypes[0].ID, fxt.Identities[0].ID)
								return err
							},
						)
					})
					// Concurrently create these links
					// A->B->C * D->E->F
					// ^               |
					//  \______*______/
					t.Run("given A->B->C and D->E->F, now create C->D and F->A", func(t *testing.T) {
						chain := append(tf.LinkChain("A", "B", "C"), tf.LinkChain("D", "E", "F")...)
						fxt := tf.NewTestFixture(t, s.DB,
							tf.WorkItems(6, tf.SetWorkItemTitles("A", "B", "C", "D", "E", "F")),
							tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
							tf.WorkItemLinksCustom(len(chain), tf.BuildLinks(chain...)),
						)
						addon := 1
						if !errorExpected[2] {
							addon++
						}
						s.createLinksConcurrently(t, fxt, len(chain)+addon,
							func(fxt *tf.TestFixture, workItemLinkRepo link.WorkItemLinkRepository) error {
								_, err := workItemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("C").ID, fxt.WorkItemByTitle("D").ID, fxt.WorkItemLinkTypes[0].ID, fxt.Identities[0].ID)
								return err
							}, func(fxt *tf.TestFixture, workItemLinkRepo link.WorkItemLinkRepository) error {
								_, err := workItemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("F").ID, fxt.WorkItemByTitle("A").ID, fxt.WorkItemLinkTypes[0].ID, fxt.Identities[0].ID)
								return err
							},
						)
					})
				})
			}
		})
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

func (s *linkRepoBlackBoxTest) TestGetAncestors() {
	validateAncestry := func(t *testing.T, fxt *tf.TestFixture, toBeFound map[link.Ancestor]struct{}, ancestors []link.Ancestor) {
		// uncomment for more information:
		// for _, ancestor := range ancestors {
		// 	t.Logf("Ancestor: %s For: %s IsRoot: %t\n",
		// 		fxt.WorkItemByID(ancestor.ID).Fields[workitem.SystemTitle].(string),
		// 		fxt.WorkItemByID(ancestor.OriginalChildID).Fields[workitem.SystemTitle].(string),
		// 		ancestor.IsRoot,
		// 	)
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
		require.Empty(t, toBeFound, "failed to find these expected ancestors: %s", func() string {
			titles := []string{}
			for ancestor := range toBeFound {
				titles = append(titles, fmt.Sprintf("\"%s\" (%s)", fxt.WorkItemByID(ancestor.ID).Fields[workitem.SystemTitle].(string), ancestor.ID))
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
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, link.AncestorLevelAll, E)
					require.NoError(t, err)
					validateAncestry(t, fxt, nil, ancestors)
				})

				t.Run("ancestors for A (none expected)", func(t *testing.T) {
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, link.AncestorLevelAll, A)
					require.NoError(t, err)
					validateAncestry(t, fxt, nil, ancestors)
				})

				t.Run("ancestors for D (expecting A,B,C)", func(t *testing.T) {
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, link.AncestorLevelAll, D)
					require.NoError(t, err)
					validateAncestry(t, fxt, map[link.Ancestor]struct{}{
						{ID: A, DirectChildID: B, Level: 3, OriginalChildID: D, IsRoot: true}:  {},
						{ID: B, DirectChildID: C, Level: 2, OriginalChildID: D, IsRoot: false}: {},
						{ID: C, DirectChildID: D, Level: 1, OriginalChildID: D, IsRoot: false}: {},
					}, ancestors)
				})

				t.Run("ancestors for C (expecting A,B)", func(t *testing.T) {
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, link.AncestorLevelAll, C)
					require.NoError(t, err)
					validateAncestry(t, fxt, map[link.Ancestor]struct{}{
						{ID: A, DirectChildID: B, Level: 2, OriginalChildID: C, IsRoot: true}:  {},
						{ID: B, DirectChildID: C, Level: 1, OriginalChildID: C, IsRoot: false}: {},
					}, ancestors)
				})

				t.Run("ancestors for D, and C (expecting, A,B,C and A,B)", func(t *testing.T) {
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, link.AncestorLevelAll, D, C)
					require.NoError(t, err)
					validateAncestry(t, fxt, map[link.Ancestor]struct{}{
						// for D
						{ID: A, DirectChildID: B, Level: 3, OriginalChildID: D, IsRoot: true}:  {},
						{ID: B, DirectChildID: C, Level: 2, OriginalChildID: D, IsRoot: false}: {},
						{ID: C, DirectChildID: D, Level: 1, OriginalChildID: D, IsRoot: false}: {},
						// for C
						{ID: A, DirectChildID: B, Level: 2, OriginalChildID: C, IsRoot: true}:  {},
						{ID: B, DirectChildID: C, Level: 1, OriginalChildID: C, IsRoot: false}: {},
					}, ancestors)
				})

				t.Run("E up to parent (none expected)", func(t *testing.T) {
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, link.AncestorLevelParent, E)
					require.NoError(t, err)
					validateAncestry(t, fxt, nil, ancestors)
				})
				t.Run("C up to parent (expecting B)", func(t *testing.T) {
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, link.AncestorLevelParent, C)
					require.NoError(t, err)
					validateAncestry(t, fxt, map[link.Ancestor]struct{}{
						// for C
						{ID: B, DirectChildID: C, Level: 1, OriginalChildID: C, IsRoot: false}: {},
					}, ancestors)
				})
				t.Run("C up to grandparent (expecting A,B)", func(t *testing.T) {
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, link.AncestorLevelGrandParent, C)
					require.NoError(t, err)
					validateAncestry(t, fxt, map[link.Ancestor]struct{}{
						// for C
						{ID: A, DirectChildID: B, Level: 2, OriginalChildID: C, IsRoot: true}:  {},
						{ID: B, DirectChildID: C, Level: 1, OriginalChildID: C, IsRoot: false}: {},
					}, ancestors)
				})
				t.Run("D up to great-grandparent (expecting A,B,C)", func(t *testing.T) {
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, link.AncestorLevelGreatGrandParent, D)
					require.NoError(t, err)
					validateAncestry(t, fxt, map[link.Ancestor]struct{}{
						// for D
						{ID: A, DirectChildID: B, Level: 3, OriginalChildID: D, IsRoot: true}:  {},
						{ID: B, DirectChildID: C, Level: 2, OriginalChildID: D, IsRoot: false}: {},
						{ID: C, DirectChildID: D, Level: 1, OriginalChildID: D, IsRoot: false}: {},
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
					ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, link.AncestorLevelAll, Y, E)
					require.NoError(t, err)
					validateAncestry(t, fxt, map[link.Ancestor]struct{}{
						// for Y
						{ID: X, DirectChildID: Y, Level: 1, OriginalChildID: Y, IsRoot: true}: {},
						// for E
						{ID: A, DirectChildID: D, Level: 2, OriginalChildID: E, IsRoot: true}:  {},
						{ID: D, DirectChildID: E, Level: 1, OriginalChildID: E, IsRoot: false}: {},
					}, ancestors)
				})
			})

		})
	}
}

func (s *linkRepoBlackBoxTest) TestAncestorList() {
	s.T().Run("A,B,C,D,E", func(t *testing.T) {
		chain := tf.LinkChain("A", "B", "C", "D", "E")
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItemLinkTypes(1, tf.SetTopologies(link.TopologyTree)),
			tf.WorkItems(5, tf.SetWorkItemTitles("A", "B", "C", "D", "E")),
			tf.WorkItemLinksCustom(4, tf.BuildLinks(chain...)),
		)
		// to shorten the test code below
		A := fxt.WorkItemByTitle("A").ID
		B := fxt.WorkItemByTitle("B").ID
		C := fxt.WorkItemByTitle("C").ID
		D := fxt.WorkItemByTitle("D").ID
		E := fxt.WorkItemByTitle("E").ID

		// given
		ancestors, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypes[0].ID, link.AncestorLevelAll, E)
		require.NoError(t, err)

		t.Run("GetParentOf", func(t *testing.T) {
			t.Run("A", func(t *testing.T) {
				a := ancestors.GetParentOf(A)
				require.Nil(t, a)
			})
			t.Run("E", func(t *testing.T) {
				a := ancestors.GetParentOf(E)
				require.NotNil(t, a)
				require.Equal(t, D, a.ID)
			})
			t.Run("C", func(t *testing.T) {
				a := ancestors.GetParentOf(C)
				require.NotNil(t, a)
				require.Equal(t, B, a.ID)
			})
		})
	})
}

func (s *linkRepoBlackBoxTest) TestListChildLinks() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItems(4, tf.SetWorkItemTitles("A", "B", "C", "D")),
			tf.WorkItemLinksCustom(3, tf.BuildLinks(tf.L("A", "B"), tf.L("A", "C"), tf.L("C", "D"))),
		)
		A := fxt.WorkItemByTitle("A").ID
		B := fxt.WorkItemByTitle("B").ID
		C := fxt.WorkItemByTitle("C").ID
		linkType := fxt.WorkItemLinkTypes[0].ID
		// when
		childLinks, err := s.workitemLinkRepo.ListChildLinks(s.Ctx, linkType, A)
		// then
		require.NoError(t, err)
		var foundAB, foundAC bool
		cnt := 0
		for _, l := range childLinks {
			if l.SourceID == A && l.TargetID == B {
				foundAB = true
			}
			if l.SourceID == A && l.TargetID == C {
				foundAC = true
			}
			cnt++
		}
		require.Equal(t, 2, cnt)
		require.True(t, foundAB, "failed to find link A-B")
		require.True(t, foundAC, "failed to find link A-C")
	})
}

func (s *linkRepoBlackBoxTest) TestDeleteLinkAndListChildren() {
	s.T().Run("delete link and list children", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItems(2),
			tf.WorkItemLinksCustom(1, func(fxt *tf.TestFixture, idx int) error {
				l := fxt.WorkItemLinks[idx]
				l.LinkTypeID = link.SystemWorkItemLinkTypeParentChildID
				l.SourceID = fxt.WorkItems[0].ID
				l.TargetID = fxt.WorkItems[idx+1].ID
				return nil
			}),
		)
		hasChildren, err := s.workitemLinkRepo.WorkItemHasChildren(s.Ctx, fxt.WorkItems[0].ID)
		require.NoError(t, err)
		require.True(t, hasChildren)

		childrenList, totalCount, err := s.workitemLinkRepo.ListWorkItemChildren(s.Ctx, fxt.WorkItems[0].ID, nil, nil)
		require.NoError(t, err)
		require.Equal(t, 1, totalCount)
		require.Len(t, childrenList, 1)
		require.Equal(t, childrenList[0].ID, fxt.WorkItems[1].ID)

		// delete work item link
		err = s.workitemLinkRepo.Delete(s.Ctx, fxt.WorkItemLinks[0].ID, fxt.Identities[0].ID)
		require.NoError(t, err)

		hasChildren, err = s.workitemLinkRepo.WorkItemHasChildren(s.Ctx, fxt.WorkItems[0].ID)
		require.NoError(t, err)
		require.False(t, hasChildren)

		childrenList, totalCount, err = s.workitemLinkRepo.ListWorkItemChildren(s.Ctx, fxt.WorkItems[0].ID, nil, nil)
		require.NoError(t, err)
		require.Equal(t, 0, totalCount)
		require.Len(t, childrenList, 0)
	})
}
