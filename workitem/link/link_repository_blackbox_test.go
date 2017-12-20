package link_test

import (
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem/link"

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
			tf.WorkItems(1, tf.SetWorkItemTitles("someWorkItem")),
			tf.WorkItemLinkTypes(1, tf.SetTopologies(link.TopologyTree), tf.SetWorkItemLinkTypeNames("tree-type")),
		)
		// when
		err := s.workitemLinkRepo.ValidateTopology(s.Ctx, nil, fxt.WorkItemByTitle("someWorkItem").ID, *fxt.WorkItemLinkTypeByName("tree-type"))
		// then: there must be no error because no link exists
		require.NoError(t, err)
	})

	s.T().Run("ok - link exists but ignored", func(t *testing.T) {
		err := s.workitemLinkRepo.ValidateTopology(s.Ctx, &fxt.WorkItemByTitle("parent").ID, fxt.WorkItemByTitle("child").ID, *fxt.WorkItemLinkTypeByName("tree-type"))
		// then: there must be no error because the existing link was ignored
		require.NoError(t, err)
	})

	s.T().Run("ok - no link with same type", func(t *testing.T) {
		// when using another link type to validate
		err := s.workitemLinkRepo.ValidateTopology(s.Ctx, nil, fxt.WorkItemByTitle("child").ID, *fxt.WorkItemLinkTypeByName("another-type"))
		// then: there must be no error because no link of the same type exists
		require.NoError(t, err)
	})

	s.T().Run("fail - link exists", func(t *testing.T) {
		err := s.workitemLinkRepo.ValidateTopology(s.Ctx, nil, fxt.WorkItemByTitle("child").ID, *fxt.WorkItemLinkTypeByName("tree-type"))
		// then: there must be an error because a link of the same type already exists
		require.Error(t, err)
	})

	s.T().Run("fail - another link exists", func(t *testing.T) {
		err := s.workitemLinkRepo.ValidateTopology(s.Ctx, &fxt.WorkItemByTitle("another-item").ID, fxt.WorkItemByTitle("child").ID, *fxt.WorkItemLinkTypeByName("tree-type"))
		// then: there must be an error because a link of the same type already exists with another parent
		require.Error(t, err)
	})
}

func (s *linkRepoBlackBoxTest) TestCreate() {
	s.T().Run("ok", func(t *testing.T) {
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
		//   C = the element that is causing the cycle
		//
		// Scenarios
		// ---------
		//
		//   I:        II:       III:      IV:       V:
		//
		//    C         C         C         C         A
		//     *         \         *         *         \
		//      A         A         A         A         B
		//       \         \         \         \         *
		//        B         B         C         B         C
		//         \         *         \
		//          C         C         B
		//
		// I, II, III are cycles
		// IV and V are no cycles.

		// Map topologies to expected error during cycle
		topos := map[link.Topology][5]bool{
			link.TopologyNetwork:         {false, false, false, false, false},
			link.TopologyDirectedNetwork: {false, false, false, false, false},
			link.TopologyTree:            {true, true, true, false, false},
			link.TopologyDependency:      {true, true, true, false, false},
		}
		for topo, errorExpected := range topos {
			t.Run("topology: "+topo.String(), func(t *testing.T) {
				//   I:
				//
				//    C
				//     *
				//      A
				//       \
				//        B
				//         \
				//          C
				t.Run("Scenario I", func(t *testing.T) {
					// given
					fxt := tf.NewTestFixture(t, s.DB,
						tf.WorkItems(3, tf.SetWorkItemTitles("A", "B", "C")),
						tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
						tf.WorkItemLinks(2, func(fxt *tf.TestFixture, idx int) error {
							l := fxt.WorkItemLinks[idx]
							switch idx {
							case 0:
								l.SourceID = fxt.WorkItemByTitle("A").ID
								l.TargetID = fxt.WorkItemByTitle("B").ID
							case 1:
								l.SourceID = fxt.WorkItemByTitle("B").ID
								l.TargetID = fxt.WorkItemByTitle("C").ID
							}
							return nil
						}),
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
				//   II:
				//
				//    C
				//     \
				//      A
				//       \
				//        B
				//         *
				//          C
				t.Run("Scenario II", func(t *testing.T) {
					// given
					fxt := tf.NewTestFixture(t, s.DB,
						tf.WorkItems(3, tf.SetWorkItemTitles("A", "C", "B")),
						tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
						tf.WorkItemLinks(2, func(fxt *tf.TestFixture, idx int) error {
							l := fxt.WorkItemLinks[idx]
							switch idx {
							case 0:
								l.SourceID = fxt.WorkItemByTitle("A").ID
								l.TargetID = fxt.WorkItemByTitle("C").ID
							case 1:
								l.SourceID = fxt.WorkItemByTitle("C").ID
								l.TargetID = fxt.WorkItemByTitle("B").ID
							}
							return nil
						}),
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
				//  III:
				//
				//   C
				//    *
				//     A
				//      \
				//       C
				//        \
				//         B
				t.Run("Scenario III", func(t *testing.T) {
					// given
					fxt := tf.NewTestFixture(t, s.DB,
						tf.WorkItems(3, tf.SetWorkItemTitles("A", "C", "B")),
						tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
						tf.WorkItemLinks(2, func(fxt *tf.TestFixture, idx int) error {
							l := fxt.WorkItemLinks[idx]
							switch idx {
							case 0:
								l.SourceID = fxt.WorkItemByTitle("A").ID
								l.TargetID = fxt.WorkItemByTitle("C").ID
							case 1:
								l.SourceID = fxt.WorkItemByTitle("C").ID
								l.TargetID = fxt.WorkItemByTitle("B").ID
							}
							return nil
						}),
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
				//  IV:
				//
				//   C
				//    *
				//     A
				//      \
				//       B
				t.Run("Scenario IV", func(t *testing.T) {
					// given
					fxt := tf.NewTestFixture(t, s.DB,
						tf.WorkItems(3, tf.SetWorkItemTitles("A", "B", "C")),
						tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
						tf.WorkItemLinks(1, func(fxt *tf.TestFixture, idx int) error {
							l := fxt.WorkItemLinks[idx]
							switch idx {
							case 0:
								l.SourceID = fxt.WorkItemByTitle("A").ID
								l.TargetID = fxt.WorkItemByTitle("B").ID
							}
							return nil
						}),
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
				//  V:
				//
				//   A
				//    \
				//     B
				//      *
				//       C
				t.Run("Scenario V", func(t *testing.T) {
					// given
					fxt := tf.NewTestFixture(t, s.DB,
						tf.WorkItems(3, tf.SetWorkItemTitles("A", "B", "C")),
						tf.WorkItemLinkTypes(1, tf.SetTopologies(topo)),
						tf.WorkItemLinks(1, func(fxt *tf.TestFixture, idx int) error {
							l := fxt.WorkItemLinks[idx]
							switch idx {
							case 0:
								l.SourceID = fxt.WorkItemByTitle("A").ID
								l.TargetID = fxt.WorkItemByTitle("B").ID
							}
							return nil
						}),
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
			})
		}
	})
}

func (s *linkRepoBlackBoxTest) TestSave() {
	s.T().Run("ok", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinks(1))
		_, err := s.workitemLinkRepo.Save(s.Ctx, *fxt.WorkItemLinks[0], fxt.Identities[0].ID)
		require.NoError(t, err)
	})
}

func (s *linkRepoBlackBoxTest) TestExistsLink() {
	s.T().Run("link exists", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinks(1))
		err := s.workitemLinkRepo.CheckExists(s.Ctx, fxt.WorkItemLinks[0].ID.String())
		require.NoError(t, err)
	})

	s.T().Run("link doesn't exist", func(t *testing.T) {
		err := s.workitemLinkRepo.CheckExists(s.Ctx, uuid.NewV4().String())
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
	s.T().Run("ok", func(t *testing.T) {

		setup := func(t *testing.T, withCycle bool) *tf.TestFixture {
			// Test setup
			//     scenario     1
			//       experience 1.1
			//       experience 1.2
			//     scenario     2
			//      experience  2.1
			//       feature    2.1.1
			//        task      2.1.1.1
			//        task      2.1.1.2
			//       feature    2.1.2
			//        task      2.1.2.1

			type testData struct {
				title    string
				typeName string
			}
			td := []testData{
				{"s 1", "scenario"},
				{"e 1.1", "experience"},
				{"e 1.2", "experience"},
				{"s 2", "scenario"},
				{"e 2.1", "experience"},
				{"f 2.1.1", "feature"},
				{"t 2.1.1.1", "task"},
				{"t 2.1.1.2", "task"},
				{"f 2.1.2", "feature"},
				{"t 2.1.2.1", "task"},
			}

			numLinks := 8
			if withCycle {
				numLinks += 1
			}
			return tf.NewTestFixture(t, s.DB,
				tf.WorkItemTypes(4, tf.SetWorkItemTypeNames("scenario", "experience", "feature", "task")),
				tf.WorkItemLinkTypes(1, tf.SetWorkItemLinkTypeNames("parenting"), tf.SetTopologies(link.TopologyTree)),
				tf.WorkItems(len(td), func(fxt *tf.TestFixture, idx int) error {
					fxt.WorkItems[idx].Fields[workitem.SystemTitle] = td[idx].title
					fxt.WorkItems[idx].Type = fxt.WorkItemTypeByName(td[idx].typeName).ID
					return nil
				}),
				tf.WorkItemLinksCustom(numLinks, func(fxt *tf.TestFixture, idx int) error {
					l := fxt.WorkItemLinks[idx]
					switch idx {
					case 0:
						l.SourceID = fxt.WorkItemByTitle("s 1").ID
						l.TargetID = fxt.WorkItemByTitle("e 1.1").ID
					case 1:
						l.SourceID = fxt.WorkItemByTitle("s 1").ID
						l.TargetID = fxt.WorkItemByTitle("e 1.2").ID
					case 2:
						l.SourceID = fxt.WorkItemByTitle("s 2").ID
						l.TargetID = fxt.WorkItemByTitle("e 2.1").ID
					case 3:
						l.SourceID = fxt.WorkItemByTitle("e 2.1").ID
						l.TargetID = fxt.WorkItemByTitle("f 2.1.1").ID
					case 4:
						l.SourceID = fxt.WorkItemByTitle("e 2.1").ID
						l.TargetID = fxt.WorkItemByTitle("f 2.1.2").ID
					case 5:
						l.SourceID = fxt.WorkItemByTitle("f 2.1.1").ID
						l.TargetID = fxt.WorkItemByTitle("t 2.1.1.1").ID
					case 6:
						l.SourceID = fxt.WorkItemByTitle("f 2.1.1").ID
						l.TargetID = fxt.WorkItemByTitle("t 2.1.1.2").ID
					case 7:
						l.SourceID = fxt.WorkItemByTitle("f 2.1.2").ID
						l.TargetID = fxt.WorkItemByTitle("t 2.1.2.1").ID
					case 8:
						// This link is only created when a cycle was requested
						l.SourceID = fxt.WorkItemByTitle("t 2.1.2.1").ID
						l.TargetID = fxt.WorkItemByTitle("s 2").ID
					}
					return nil
				}),
			)
		}

		validateAncestry := func(t *testing.T, fxt *tf.TestFixture, toBeFound map[uuid.UUID]struct{}, ancestorIDs []uuid.UUID) {
			for _, id := range ancestorIDs {
				wi := fxt.WorkItemByID(id)
				assert.NotNil(t, wi, "failed to find work item with ID: %s", id)
				if wi != nil {
					t.Logf("found work item: %s", wi.Fields[workitem.SystemTitle].(string))
				}
				_, ok := toBeFound[id]
				require.True(t, ok, "found unexpected work item: %s", fxt.WorkItemByID(id).Fields[workitem.SystemTitle].(string))
				delete(toBeFound, id)
			}
			require.Empty(t, toBeFound, "failed to find these work items in ancestor list: %s", func() string {
				titles := []string{}
				for id := range toBeFound {
					titles = append(titles, "\""+fxt.WorkItemByID(id).Fields[workitem.SystemTitle].(string)+"\"")
				}
				return strings.Join(titles, ", ")
			}())
		}

		t.Run("complex scenario", func(t *testing.T) {
			t.Run("search for tasks", func(t *testing.T) {
				// given
				fxt := setup(t, false)

				// when fetching the ancestors for all tasks
				ancestorIDs, _, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypeByName("parenting", fxt.Spaces[0].ID).ID,
					fxt.WorkItemByTitle("t 2.1.1.1").ID,
					fxt.WorkItemByTitle("t 2.1.1.2").ID,
					fxt.WorkItemByTitle("t 2.1.2.1").ID)

				// then
				require.NoError(t, err)
				toBeFound := map[uuid.UUID]struct{}{
					fxt.WorkItemByTitle("s 2").ID:     {},
					fxt.WorkItemByTitle("e 2.1").ID:   {},
					fxt.WorkItemByTitle("f 2.1.1").ID: {},
					fxt.WorkItemByTitle("f 2.1.2").ID: {},
				}
				validateAncestry(t, fxt, toBeFound, ancestorIDs)
			})
			t.Run("search for experience", func(t *testing.T) {
				// given
				fxt := setup(t, false)

				// when fetching the ancestors for all tasks
				ancestorIDs, _, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypeByName("parenting", fxt.Spaces[0].ID).ID,
					fxt.WorkItemByTitle("e 2.1").ID)

				// then
				require.NoError(t, err)
				toBeFound := map[uuid.UUID]struct{}{
					fxt.WorkItemByTitle("s 2").ID: {},
				}
				validateAncestry(t, fxt, toBeFound, ancestorIDs)
			})
		})

		t.Run("non-existent child workitem", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinkTypes(1, tf.SetWorkItemLinkTypeNames("parenting"), tf.SetTopologies(link.TopologyTree)))
			// when
			ancestorIDs, _, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypeByName("parenting").ID, uuid.NewV4())
			// then
			require.NoError(t, err)
			require.Empty(t, ancestorIDs)
		})

		t.Run("no given child work item", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinkTypes(1, tf.SetWorkItemLinkTypeNames("parenting"), tf.SetTopologies(link.TopologyTree)))
			// when
			ancestorIDs, _, err := s.workitemLinkRepo.GetAncestors(s.Ctx, fxt.WorkItemLinkTypeByName("parenting").ID)
			// then
			require.NoError(t, err)
			require.Empty(t, ancestorIDs)
		})
	})
}
