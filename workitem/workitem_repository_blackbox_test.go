package workitem_test

import (
	"fmt"
	"math"
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
		require.Nil(t, err)
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
		require.Nil(s.T(), err)
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
	s.T().Run("ok - save assignees", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemAssignees] = []string{"A", "B"}
			return nil
		}))
		// when
		wi, err := s.repo.LoadByID(s.Ctx, fxt.WorkItems[0].ID)
		// then
		require.Nil(t, err)
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
		require.Nil(t, err)
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
		require.Nil(t, err)
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
		require.Nil(t, err)
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
		require.NotNil(t, err)
	})

	s.T().Run("field types", func(t *testing.T) {
		vals := getFieldTypeTestData(t)
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
						wi, err := s.repo.Create(s.Ctx, fxt.Spaces[0].ID, witID, map[string]interface{}{fieldName: expected}, fxt.Identities[0].ID)
						assert.Nil(t, err, "expected no error when assigning this value to a '%s' field during work item creation: %#v", kind, spew.Sdump(expected))
						loadedWi, err := s.repo.LoadByID(s.Ctx, wi.ID)
						require.Nil(t, err)
						// compensate for errors when interpreting ambigous actual values
						actual := loadedWi.Fields[fieldName]
						if iv.Compensate != nil {
							actual = iv.Compensate(actual)
						}
						require.Equal(t, expected, actual, "expected no error when loading and comparing the workitem with a '%s': %#v", kind, spew.Sdump(expected))
					}
				})
				t.Run("illegal", func(t *testing.T) {
					// Handle cases where the conversion is supposed to NOT work
					for _, expected := range iv.Invalid {
						_, err := s.repo.Create(s.Ctx, fxt.Spaces[0].ID, witID, map[string]interface{}{fieldName: expected}, fxt.Identities[0].ID)
						assert.NotNil(t, err, "expected an error when assigning this value to a '%s' field during work item creation: %#v", kind, spew.Sdump(expected))
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
		require.Nil(t, err)
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
		require.Nil(t, err)
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
		require.NotNil(s.T(), err)
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
	require.Nil(s.T(), err)
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
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wiID)
	assert.Equal(s.T(), wi.ID, *wiID)
	require.NotNil(s.T(), spaceID)
	assert.Equal(s.T(), wi.SpaceID, *spaceID)

	// delete above space
	spaceRepo := space.NewRepository(s.DB)
	err = spaceRepo.Delete(s.Ctx, sp.ID)
	require.Nil(s.T(), err)

	testFxt2 := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1, func(testf *tf.TestFixture, idx int) error {
		testf.Spaces[0].Name = sp.Name
		testf.Spaces[0].OwnerId = in.ID
		return nil
	}), tf.WorkItems(20, func(testf *tf.TestFixture, idx int) error {
		testf.WorkItems[idx].Fields[workitem.SystemState] = workitem.SystemStateNew
		testf.WorkItems[idx].Fields[workitem.SystemCreator] = testf.Identities[0].ID.String()
		return nil
	}))
	sp2 := *testFxt2.Spaces[0]
	wi2 := *testFxt2.WorkItems[0]
	wiID2, spaceID2, err := s.repo.LookupIDByNamedSpaceAndNumber(s.Ctx, in.Username, sp2.Name, wi2.Number)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wiID2)
	assert.Equal(s.T(), wi2.ID, *wiID2)
	require.NotNil(s.T(), spaceID2)
	assert.Equal(s.T(), wi2.SpaceID, *spaceID2)
}

// given a bunch of tests with expected error results for each work item
// type field kind, a work item type for each kind...
type validInvalid struct {
	Valid   []interface{}
	Invalid []interface{}
	// When the actual value is a zero (0), it will be interpreted as a
	// float64 rather than an int. To compensate for that ambiguity, a
	// kind can opt-in to provide an construction function that returns
	// the correct value.
	Compensate func(interface{}) interface{}
}

// getFieldTypeTestData returns a list of legal and illegal values to be used
// with a given field type (here: the map key).
func getFieldTypeTestData(t *testing.T) map[workitem.Kind]validInvalid {
	// helper function to convert a string into a duration and handling the
	// error
	validDuration := func(s string) time.Duration {
		d, err := time.ParseDuration(s)
		if err != nil {
			require.Nil(t, err, "we expected the duration to be valid: %s", s)
		}
		return d
	}

	return map[workitem.Kind]validInvalid{
		workitem.KindString: {
			Valid: []interface{}{
				"foo",
			},
			Invalid: []interface{}{
				"", // NOTE: an empty string is not allowed in a required field.
				nil,
				0,
				true,
				0.1,
			},
		},
		workitem.KindUser: {
			Valid: []interface{}{
				"jane doe", // TODO(kwk): do we really allow usernames with spaces?
				"",         // TODO(kwk): do we really allow empty usernames?
			},
			Invalid: []interface{}{
				nil,
				0,
				true,
				0.1,
			},
		},
		workitem.KindIteration: {
			Valid: []interface{}{
				"some iteration name",
				"", // TODO(kwk): do we really allow empty iteration names?
			},
			Invalid: []interface{}{
				nil,
				0,
				true,
				0.1,
			},
		},
		workitem.KindArea: {
			Valid: []interface{}{
				"some are name",
				"", // TODO(kwk): do we really allow empty area names?
			},
			Invalid: []interface{}{
				nil,
				0,
				true,
				0.1,
			},
		},
		workitem.KindLabel: {
			Valid: []interface{}{
				"some label name",
				"", // TODO(kwk): do we really allow empty label names?
			},
			Invalid: []interface{}{
				nil,
				0,
				true,
				0.1,
			},
		},
		workitem.KindURL: {
			Valid: []interface{}{
				"127.0.0.1",
				"http://www.openshift.io",
				"openshift.io",
				"ftp://url.with.port.and.different.protocol.port.and.parameters.com:8080/fooo?arg=bar&key=value",
			},
			Invalid: []interface{}{
				0,
				"", // NOTE: An empty URL is not allowed when the field is required (see simple_type.go:53)
				"http://url with whitespace.com",
				"http://www.example.com/foo bar",
				"localhost", // TODO(kwk): shall we disallow localhost?
				"foo",
			},
		},
		workitem.KindInteger: {
			// Compensate for wrong interpretation of 0
			Compensate: func(in interface{}) interface{} {
				v := in.(float64) // NOTE: float64 is correct here because a 0 will first and foremost be treated as float64
				if v != math.Trunc(v) {
					panic(fmt.Sprintf("value is not a whole number %v", v))
				}
				return int(v)
			},
			Valid: []interface{}{
				0,
				333,
				-100,
			},
			Invalid: []interface{}{
				1e2,
				nil,
				"",
				"foo",
				0.1,
				true,
				false,
			},
		},
		workitem.KindFloat: {
			Valid: []interface{}{
				0.1,
				-1111.0,
				+555.0,
			},
			Invalid: []interface{}{
				1,
				0,
				"string",
			},
		},
		workitem.KindBoolean: {
			Valid: []interface{}{
				true,
				false,
			},
			Invalid: []interface{}{
				nil,
				0,
				1,
				"",
				"yes",
				"no",
				"0",
				"1",
				"true",
				"false",
			},
		},
		workitem.KindDuration: {
			// Compensate for wrong interpretation of 0
			Compensate: func(in interface{}) interface{} {
				i := in.(float64)
				return time.Duration(int64(i))
			},
			Valid: []interface{}{
				validDuration("0"),
				validDuration("300ms"),
				validDuration("-1.5h"),
				// 0, // TODO(kwk): should work because an untyped integer constant can be converted to time.Duration's underlying type: int64
			},
			Invalid: []interface{}{
				// 0, // TODO(kwk): 0 doesn't fit in legal nor illegal
				nil,
				"1e2",
				"4000",
			},
		},
		workitem.KindInstant: {
			// Compensate for wrong interpretation of location value and default to UTC
			Compensate: func(in interface{}) interface{} {
				v := in.(time.Time)
				return v.UTC()
			},
			Valid: []interface{}{
				// NOTE: If we don't use UTC(), the unmarshalled JSON will
				// have a different time zone (read up on JSON an time
				// location if you don't believe me).
				func() interface{} {
					v, err := time.Parse("02 Jan 06 15:04 -0700", "02 Jan 06 15:04 -0700")
					require.Nil(t, err)
					return v.UTC()
				}(),
				// time.Now().UTC(), // TODO(kwk): Somehow this fails due to different nsec
			},
			Invalid: []interface{}{
				time.Now().String(),
				time.Now().UTC().String(),
				"2017-09-27 13:40:48.099780356 +0200 CEST", // NOTE: looks like a time.Time but is a string
				"",
				0,
				333,
				100,
				1e2,
				nil,
				"foo",
				0.1,
				true,
				false,
			},
		},
		workitem.KindMarkup: {
			Valid: []interface{}{
				rendering.MarkupContent{Content: "plain text", Markup: rendering.SystemMarkupPlainText},
				rendering.MarkupContent{Content: "default", Markup: rendering.SystemMarkupDefault},
				rendering.MarkupContent{Content: "# markdown", Markup: rendering.SystemMarkupMarkdown},
			},
			Invalid: []interface{}{
				0,
				rendering.MarkupContent{Content: "jira", Markup: rendering.SystemMarkupJiraWiki}, // TODO(kwk): not supported yet
				rendering.MarkupContent{Content: "", Markup: ""},                                 // NOTE: We allow allow empty strings
				rendering.MarkupContent{Content: "foo", Markup: "unknown markup type"},
				"",
				"foo",
			},
		},
		workitem.KindCodebase: {
			Valid: []interface{}{
				codebase.Content{
					Repository: "git://github.com/ember-cli/ember-cli.git#ff786f9f",
					Branch:     "foo",
					FileName:   "bar.js",
					LineNumber: 10,
					CodebaseID: "dunno",
				},
			},
			Invalid: []interface{}{
				// empty repository (see codebase.Content.IsValid())
				codebase.Content{
					Repository: "",
					Branch:     "foo",
					FileName:   "bar.js",
					LineNumber: 10,
					CodebaseID: "dunno",
				},
				// invalid repository URL (see codebase.Content.IsValid())
				codebase.Content{
					Repository: "/path/to/repo.git/",
					Branch:     "foo",
					FileName:   "bar.js",
					LineNumber: 10,
					CodebaseID: "dunno",
				},
				"",
				0,
				333,
				100,
				1e2,
				nil,
				"foo",
				0.1,
				true,
				false,
			},
		},
		//workitem.KindEnum:  {}, // TODO(kwk): Add test for workitem.KindEnum
		//workitem.KindList:  {}, // TODO(kwk): Add test for workitem.KindList
	}
}
