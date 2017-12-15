package workitem_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type workItemRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo workitem.WorkItemRepository
}

func TestRunWorkItemRepoBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *workItemRepoBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = workitem.NewWorkItemRepository(s.DB)
}

func (s *workItemRepoBlackBoxTest) TestSave() {
	s.T().Run("save work item without assignees & labels", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "some title"
			fxt.WorkItems[idx].Fields[workitem.SystemState] = workitem.SystemStateNew
			return nil
		}))
		wiNew, err := s.repo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemAssignees].([]interface{}), 0)
		require.Len(t, wiNew.Fields[workitem.SystemLabels].([]interface{}), 0)
	})

	s.T().Run("fail - save nil number", func(t *testing.T) {
		// given at least 1 item to avoid RowsEffectedCheck
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))
		// when
		fxt.WorkItems[0].Number = 0
		_, err := s.repo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		// then
		assert.IsType(t, errors.NotFoundError{}, errs.Cause(err))
	})

	s.T().Run("ok - save for unchanged created date", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))
		oldDate, ok := fxt.WorkItems[0].Fields[workitem.SystemCreatedAt].(time.Time)
		require.True(t, ok, "failed to convert interface{} to time.Time")
		wiNew, err := s.repo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		newTime, ok := wiNew.Fields[workitem.SystemCreatedAt].(time.Time)
		require.True(t, ok, "failed to convert interface{} to time.Time")
		// then
		require.NoError(t, err)
		assert.Equal(t, oldDate.UTC(), newTime.UTC())
	})

	s.T().Run("change is not prohibited", func(t *testing.T) {
		// tests that you can change the type of a work item. NOTE: This
		// functionality only works on the DB layer and is not exposed to REST.
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1), tf.WorkItemTypes(2))
		// when
		fxt.WorkItems[0].Type = fxt.WorkItemTypes[1].ID
		newWi, err := s.repo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		// then
		require.NoError(s.T(), err)
		assert.Equal(s.T(), fxt.WorkItemTypes[1].ID, newWi.Type)
	})
}

func (s *workItemRepoBlackBoxTest) TestLoadID() {
	s.T().Run("fail - load nil ID", func(t *testing.T) {
		_, err := s.repo.LoadByID(s.Ctx, uuid.Nil)
		// then
		assert.IsType(t, errors.NotFoundError{}, errs.Cause(err))
	})
}

func (s *workItemRepoBlackBoxTest) TestCreate() {
	s.T().Run("create work item without assignees & labels", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypes(1))
		wi, err := s.repo.Create(
			s.Ctx, fxt.Spaces[0].ID, fxt.WorkItemTypes[0].ID,
			map[string]interface{}{
				workitem.SystemTitle: "some title",
				workitem.SystemState: workitem.SystemStateNew,
			}, fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wi.Fields[workitem.SystemAssignees].([]interface{}), 0)
		require.Len(t, wi.Fields[workitem.SystemLabels].([]interface{}), 0)

	})

	s.T().Run("ok - save assignees", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemAssignees] = []string{"A", "B"}
			return nil
		}))
		// when
		wi, err := s.repo.LoadByID(s.Ctx, fxt.WorkItems[0].ID)
		// then
		require.NoError(t, err)
		require.Len(t, wi.Fields[workitem.SystemAssignees].([]interface{}), 2)
		assert.Equal(t, "A", wi.Fields[workitem.SystemAssignees].([]interface{})[0])
		assert.Equal(t, "B", wi.Fields[workitem.SystemAssignees].([]interface{})[1])
	})

	s.T().Run("ok - create work item with description no markup", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemDescription] = rendering.NewMarkupContentFromLegacy("Description")
			return nil
		}))
		// when
		wi, err := s.repo.LoadByID(s.Ctx, fxt.WorkItems[0].ID)
		// then
		require.NoError(t, err)
		// workitem.WorkItem does not contain the markup associated with the description (yet)
		assert.Equal(t, rendering.NewMarkupContentFromLegacy("Description"), wi.Fields[workitem.SystemDescription])
	})

	s.T().Run("ok - work item with description markup", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemDescription] = rendering.NewMarkupContent("Description", rendering.SystemMarkupMarkdown)
			return nil
		}))
		// when
		wi, err := s.repo.LoadByID(s.Ctx, fxt.WorkItems[0].ID)
		// then
		require.NoError(t, err)
		// workitem.WorkItem does not contain the markup associated with the description (yet)
		assert.Equal(t, rendering.NewMarkupContent("Description", rendering.SystemMarkupMarkdown), wi.Fields[workitem.SystemDescription])
	})

	s.T().Run("ok - code base attributes", func(t *testing.T) {
		// given
		title := "solution on global warming"
		branch := "earth-recycle-101"
		repo := "https://github.com/pranavgore09/go-tutorial.git"
		file := "main.go"
		line := 200
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemTitle] = title
			fxt.WorkItems[idx].Fields[workitem.SystemCodebase] = codebase.Content{
				Branch:     branch,
				Repository: repo,
				FileName:   file,
				LineNumber: line,
			}
			return nil
		}))
		// when
		wi, err := s.repo.LoadByID(s.Ctx, fxt.WorkItems[0].ID)
		// then
		require.NoError(t, err)
		assert.Equal(t, title, wi.Fields[workitem.SystemTitle].(string))
		require.NotNil(t, wi.Fields[workitem.SystemCodebase])
		cb := wi.Fields[workitem.SystemCodebase].(codebase.Content)
		assert.Equal(t, repo, cb.Repository)
		assert.Equal(t, branch, cb.Branch)
		assert.Equal(t, file, cb.FileName)
		assert.Equal(t, line, cb.LineNumber)
	})

	s.T().Run("fail - code base attributes: invalid repo", func(t *testing.T) {
		// given
		title := "solution on global warming"
		branch := "earth-recycle-101"
		repo := "https://non-github.com/pranavgore09/go-tutorial"
		file := "main.go"
		line := 200
		cbase := codebase.Content{
			Branch:     branch,
			Repository: repo,
			FileName:   file,
			LineNumber: line,
		}
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypes(1))
		_, err := s.repo.Create(
			s.Ctx, fxt.Spaces[0].ID, fxt.WorkItemTypes[0].ID,
			map[string]interface{}{
				workitem.SystemTitle:    title,
				workitem.SystemState:    workitem.SystemStateNew,
				workitem.SystemCodebase: cbase,
			}, fxt.Identities[0].ID)
		require.Error(t, err)
	})

	s.T().Run("field types", func(t *testing.T) {
		vals := workitem.GetFieldTypeTestData(t)
		// Get keys from the map above
		kinds := []workitem.Kind{}
		for k := range vals {
			kinds = append(kinds, k)
		}
		fieldName := "fieldundertest"
		// Create a work item type for each kind
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItemTypes(len(kinds), func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemTypes[idx].Name = kinds[idx].String()
				fxt.WorkItemTypes[idx].Fields = map[string]workitem.FieldDefinition{
					fieldName: {
						Required:    true,
						Label:       kinds[idx].String(),
						Description: fmt.Sprintf("This field is used for testing values for the field kind '%s'", kinds[idx]),
						Type: workitem.SimpleType{
							Kind: kinds[idx],
						},
					},
				}
				return nil
			}),
		)
		// when
		for kind, iv := range vals {
			witID := fxt.WorkItemTypeByName(kind.String()).ID
			t.Run(kind.String(), func(t *testing.T) {
				// Handle cases where the conversion is supposed to work
				t.Run("legal", func(t *testing.T) {
					for _, expected := range iv.Valid {
						t.Run(spew.Sdump(expected), func(t *testing.T) {
							wi, err := s.repo.Create(s.Ctx, fxt.Spaces[0].ID, witID, map[string]interface{}{fieldName: expected}, fxt.Identities[0].ID)
							require.NoError(t, err, "expected no error when assigning this value to a '%s' field during work item creation: %#v", kind, spew.Sdump(expected))
							loadedWi, err := s.repo.LoadByID(s.Ctx, wi.ID)
							require.NoError(t, err)
							// compensate for errors when interpreting ambigous actual values
							actual := loadedWi.Fields[fieldName]
							if iv.Compensate != nil {
								actual = iv.Compensate(actual)
							}
							require.Equal(t, expected, actual, "expected no error when loading and comparing the workitem with a '%s': %#v", kind, spew.Sdump(expected))
						})
					}
				})
				t.Run("illegal", func(t *testing.T) {
					// Handle cases where the conversion is supposed to NOT work
					for _, expected := range iv.Invalid {
						t.Run(spew.Sdump(expected), func(t *testing.T) {
							_, err := s.repo.Create(s.Ctx, fxt.Spaces[0].ID, witID, map[string]interface{}{fieldName: expected}, fxt.Identities[0].ID)
							assert.NotNil(t, err, "expected an error when assigning this value to a '%s' field during work item creation: %#v", kind, spew.Sdump(expected))
						})
					}
				})
			})
		}
	})
}

func (s *workItemRepoBlackBoxTest) TestCheckExists() {
	s.T().Run("work item exists", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))
		// when
		err := s.repo.CheckExists(s.Ctx, fxt.WorkItems[0].ID.String())
		// then
		require.NoError(t, err)
	})

	s.T().Run("work item doesn't exist", func(t *testing.T) {
		// when
		err := s.repo.CheckExists(s.Ctx, uuid.NewV4().String())
		// then
		require.IsType(t, errors.NotFoundError{}, err)
	})
}

func (s *workItemRepoBlackBoxTest) TestGetCountsPerIteration() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		testFxt := tf.NewTestFixture(t, s.DB, tf.Iterations(2), tf.WorkItems(5, func(fxt *tf.TestFixture, idx int) error {
			wi := fxt.WorkItems[idx]
			wi.Fields[workitem.SystemIteration] = fxt.Iterations[0].ID.String()
			if idx < 3 {
				wi.Fields[workitem.SystemState] = workitem.SystemStateNew
			} else if idx >= 3 {
				wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
			}
			return nil
		}))

		// when
		countsMap, _ := s.repo.GetCountsPerIteration(s.Ctx, testFxt.Spaces[0].ID)
		// then
		require.Len(t, countsMap, 2)
		require.Contains(t, countsMap, testFxt.Iterations[0].ID.String())
		assert.Equal(t, 5, countsMap[testFxt.Iterations[0].ID.String()].Total)
		assert.Equal(t, 2, countsMap[testFxt.Iterations[0].ID.String()].Closed)
		require.Contains(t, countsMap, testFxt.Iterations[1].ID.String())
		assert.Equal(t, 0, countsMap[testFxt.Iterations[1].ID.String()].Total)
		assert.Equal(t, 0, countsMap[testFxt.Iterations[1].ID.String()].Closed)
	})
}

func (s *workItemRepoBlackBoxTest) TestLookupIDByNamedSpaceAndNumber() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))
		// when
		wiID, spaceID, err := s.repo.LookupIDByNamedSpaceAndNumber(s.Ctx, fxt.Identities[0].Username, fxt.Spaces[0].Name, fxt.WorkItems[0].Number)
		// then
		require.NoError(t, err)
		require.NotNil(t, wiID)
		assert.Equal(t, fxt.WorkItems[0].ID, *wiID)
		// TODO(xcoulon) can be removed once PR for #1452 is merged
		require.NotNil(t, spaceID)
		assert.Equal(t, fxt.WorkItems[0].SpaceID, *spaceID)
	})

	s.T().Run("not found", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))
		// when
		_, _, err := s.repo.LookupIDByNamedSpaceAndNumber(s.Ctx, "foo", fxt.Spaces[0].Name, fxt.WorkItems[0].Number)
		// then
		require.Error(s.T(), err)
		assert.IsType(s.T(), errors.NotFoundError{}, errs.Cause(err))
	})
}

// TestLoadBatchByID verifies that repo.LoadBatchByID returns distinct items
func (s *workItemRepoBlackBoxTest) TestLoadBatchByID() {
	fixtures := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(5))
	// include same ID multiple times in the list
	// Added only 2 distinct IDs in following list
	wis := []uuid.UUID{fixtures.WorkItems[1].ID, fixtures.WorkItems[1].ID, fixtures.WorkItems[2].ID, fixtures.WorkItems[2].ID}
	res, err := s.repo.LoadBatchByID(s.Ctx, wis) // pass duplicate IDs to fetch
	require.NoError(s.T(), err)
	assert.Len(s.T(), res, 2) // Only 2 distinct IDs should be returned
}

func (s *workItemRepoBlackBoxTest) TestLookupIDByNamedSpaceAndNumberStaleSpace() {
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(20, func(testf *tf.TestFixture, idx int) error {
		testf.WorkItems[idx].Fields[workitem.SystemState] = workitem.SystemStateNew
		testf.WorkItems[idx].Fields[workitem.SystemCreator] = testf.Identities[0].ID.String()
		return nil
	}))
	sp := *testFxt.Spaces[0]
	wi := *testFxt.WorkItems[0]
	in := *testFxt.Identities[0]
	wiID, spaceID, err := s.repo.LookupIDByNamedSpaceAndNumber(s.Ctx, in.Username, sp.Name, wi.Number)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), wiID)
	assert.Equal(s.T(), wi.ID, *wiID)
	require.NotNil(s.T(), spaceID)
	assert.Equal(s.T(), wi.SpaceID, *spaceID)

	// delete above space
	spaceRepo := space.NewRepository(s.DB)
	err = spaceRepo.Delete(s.Ctx, sp.ID)
	require.NoError(s.T(), err)

	testFxt2 := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1, func(testf *tf.TestFixture, idx int) error {
		testf.Spaces[0].Name = sp.Name
		testf.Spaces[0].OwnerID = in.ID
		return nil
	}), tf.WorkItems(20, func(testf *tf.TestFixture, idx int) error {
		testf.WorkItems[idx].Fields[workitem.SystemState] = workitem.SystemStateNew
		testf.WorkItems[idx].Fields[workitem.SystemCreator] = testf.Identities[0].ID.String()
		return nil
	}))
	sp2 := *testFxt2.Spaces[0]
	wi2 := *testFxt2.WorkItems[0]
	wiID2, spaceID2, err := s.repo.LookupIDByNamedSpaceAndNumber(s.Ctx, in.Username, sp2.Name, wi2.Number)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), wiID2)
	assert.Equal(s.T(), wi2.ID, *wiID2)
	require.NotNil(s.T(), spaceID2)
	assert.Equal(s.T(), wi2.SpaceID, *spaceID2)
}

// TestLoadByIteration verifies that repo.LoadByIteration returns only associated items
func (s *workItemRepoBlackBoxTest) TestLoadByIteration() {
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.Iterations(3, tf.SetIterationNames("root", "one", "two")),
		tf.WorkItems(5, func(fxt *tf.TestFixture, idx int) error {
			wi := fxt.WorkItems[idx]
			if idx < 3 {
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("one").ID.String()
			} else {
				// set root iteration to WI
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("root").ID.String()
			}
			return nil
		}))
	// Fetch work items for root iteration - should be 2
	wiInRootIteration, err := s.repo.LoadByIteration(s.Ctx, fxt.IterationByName("root").ID) // pass duplicate IDs to fetch
	require.NoError(s.T(), err)
	assert.Len(s.T(), wiInRootIteration, 2)

	// Fetch work items for "one"" iteration - should be 3
	wiInOneIteration, err := s.repo.LoadByIteration(s.Ctx, fxt.IterationByName("one").ID) // pass duplicate IDs to fetch
	require.NoError(s.T(), err)
	assert.Len(s.T(), wiInOneIteration, 3)

	// Fetch work items for "two" iteration - should be 0
	wiInTwoIteration, err := s.repo.LoadByIteration(s.Ctx, fxt.IterationByName("two").ID) // pass duplicate IDs to fetch
	require.NoError(s.T(), err)
	assert.Empty(s.T(), wiInTwoIteration)
}

func (s *workItemRepoBlackBoxTest) TestConcurrentWorkItemCreations() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment())
	type Report struct {
		id       int
		total    int
		failures int
	}
	routines := 10
	itemsPerRoutine := 50
	reports := make([]Report, routines)
	// when running concurrent go routines simultaneously
	var wg sync.WaitGroup
	for i := 0; i < routines; i++ {
		wg.Add(1)
		// in each go rountine, run 10 creations
		go func(routineID int) {
			defer wg.Done()
			report := Report{id: routineID}
			for j := 0; j < itemsPerRoutine; j++ {
				fields := map[string]interface{}{
					workitem.SystemTitle: uuid.NewV4().String(),
					workitem.SystemState: workitem.SystemStateNew,
				}
				if _, err := s.repo.Create(context.Background(), fxt.Spaces[0].ID, fxt.WorkItemTypes[0].ID, fields, fxt.Identities[0].ID); err != nil {
					s.T().Logf("Creation failed: %s", err.Error())
					report.failures++
				}
				report.total++
			}
			reports[routineID] = report
		}(i)
	}
	wg.Wait()
	// then
	// wait for all items to be created
	for _, report := range reports {
		s.T().Logf("Routine #%d done: %d creations, including %d failure(s)\n", report.id, report.total, report.failures)
		assert.Equal(s.T(), itemsPerRoutine, report.total)
		assert.Equal(s.T(), 0, report.failures)
	}
}
