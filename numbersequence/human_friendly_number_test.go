package numbersequence_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/numbersequence"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestHumanFriendlyNumber_Equal(t *testing.T) {
	t.Parallel()
	spaceID := uuid.NewV4()
	a := numbersequence.NewHumanFriendlyNumber(spaceID, "work_items", 1)
	t.Run("types", func(t *testing.T) {
		t.Parallel()
		b := convert.DummyEqualer{}
		assert.False(t, a.Equal(b))
	})
	t.Run("equality", func(t *testing.T) {
		t.Parallel()
		b := a
		assert.True(t, a.Equal(b))
	})
	t.Run("number", func(t *testing.T) {
		t.Parallel()
		b := numbersequence.NewHumanFriendlyNumber(spaceID, "work_items", 567)
		assert.False(t, a.Equal(b))
	})
	t.Run("table name", func(t *testing.T) {
		t.Parallel()
		b := numbersequence.NewHumanFriendlyNumber(spaceID, "iterations", 1)
		assert.False(t, a.Equal(b))
	})
	t.Run("space id", func(t *testing.T) {
		t.Parallel()
		b := numbersequence.NewHumanFriendlyNumber(uuid.NewV4(), "work_items", 1)
		assert.False(t, a.Equal(b))
	})
}

type humanFriendlyNumberSuite struct {
	gormtestsupport.DBTestSuite
}

func TestHumanFriendlyNumberSuite(t *testing.T) {
	suite.Run(t, &humanFriendlyNumberSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

const tableName1 = "human_friendly_number_test1"
const tableName2 = "human_friendly_number_test2"

// SetupTest implements suite.SetupTest
func (s *humanFriendlyNumberSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	// Prepare a table for our model test structures to persist their data in.
	db := s.DB.Exec(fmt.Sprintf(`
		DROP TABLE IF EXISTS %[1]q;
		DROP TABLE IF EXISTS %[2]q;
		CREATE TABLE %[1]q ( id uuid PRIMARY KEY, number int, message text );
		CREATE TABLE %[2]q ( id uuid PRIMARY KEY, number int, message text );
	`, tableName1, tableName2))
	require.NoError(s.T(), db.Error)
}

type testStruct1 struct {
	numbersequence.HumanFriendlyNumber
	ID      uuid.UUID `json:"id" gorm:"primary_key"`
	Message string    `json:"message"`
}

func (s testStruct1) TableName() string {
	return tableName1
}

type testStruct2 struct {
	numbersequence.HumanFriendlyNumber
	ID      uuid.UUID `json:"id" gorm:"primary_key"`
	Message string    `json:"message"`
}

func (s testStruct2) TableName() string {
	return tableName2
}

// TestBeforeCreate tests that two model types (testStruct1 and
// testStruct2) get numbers upon creation of model. Numbers are partitioned by
// space and table name.
func (s *humanFriendlyNumberSuite) TestBeforeCreate() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(2))
	db := s.DB

	// create models of two types in two spaces and check that the numbers are
	// assigned as expected.
	data1 := []testStruct1{
		{ID: uuid.NewV4(), Message: "first", HumanFriendlyNumber: numbersequence.NewHumanFriendlyNumber(fxt.Spaces[0].ID, tableName1)},
		{ID: uuid.NewV4(), Message: "second", HumanFriendlyNumber: numbersequence.NewHumanFriendlyNumber(fxt.Spaces[0].ID, tableName1)},
		{ID: uuid.NewV4(), Message: "third", HumanFriendlyNumber: numbersequence.NewHumanFriendlyNumber(fxt.Spaces[1].ID, tableName1)},
		{ID: uuid.NewV4(), Message: "fourth", HumanFriendlyNumber: numbersequence.NewHumanFriendlyNumber(fxt.Spaces[1].ID, tableName1)},
		{ID: uuid.NewV4(), Message: "fifth", HumanFriendlyNumber: numbersequence.NewHumanFriendlyNumber(fxt.Spaces[1].ID, tableName1)},
	}
	data2 := []testStruct2{
		{ID: uuid.NewV4(), Message: "first", HumanFriendlyNumber: numbersequence.NewHumanFriendlyNumber(fxt.Spaces[0].ID, tableName2)},
		{ID: uuid.NewV4(), Message: "second", HumanFriendlyNumber: numbersequence.NewHumanFriendlyNumber(fxt.Spaces[0].ID, tableName2)},
		{ID: uuid.NewV4(), Message: "third", HumanFriendlyNumber: numbersequence.NewHumanFriendlyNumber(fxt.Spaces[1].ID, tableName2)},
		{ID: uuid.NewV4(), Message: "fourth", HumanFriendlyNumber: numbersequence.NewHumanFriendlyNumber(fxt.Spaces[1].ID, tableName2)},
		{ID: uuid.NewV4(), Message: "fifth", HumanFriendlyNumber: numbersequence.NewHumanFriendlyNumber(fxt.Spaces[1].ID, tableName2)},
	}
	expectedNumbers := []int{1, 2, 1, 2, 3}
	for i := range data1 {
		// when
		db = db.Create(&data1[i])
		// then
		require.NoError(s.T(), db.Error)
		require.Equal(s.T(), expectedNumbers[i], data1[i].Number, "data1 item #%d should have number %d but has %d", i, expectedNumbers[i], data1[i].Number)
		// when
		db = db.Create(&data2[i])
		// then
		require.NoError(s.T(), db.Error)
		require.Equal(s.T(), expectedNumbers[i], data2[i].Number, "data2 item #%d should have number %d but has %d", i, expectedNumbers[i], data2[i].Number)
	}
}

// TestBeforeUpdate tests that you cannot change an already given number on a
// model when you update that model.
func (s *humanFriendlyNumberSuite) TestBeforeUpdate() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))
	db := s.DB
	model := testStruct1{ID: uuid.NewV4(), Message: "first", HumanFriendlyNumber: numbersequence.NewHumanFriendlyNumber(fxt.Spaces[0].ID, tableName1)}
	db = db.Create(&model)
	require.NoError(s.T(), db.Error)
	s.T().Run("update message", func(t *testing.T) {
		// when updating the message
		model.Message = "new message"
		db = db.Save(&model)
		// then the number should stay the same
		require.NoError(t, db.Error)
		require.Equal(t, 1, model.Number)
		require.Equal(t, "new message", model.Message)
	})
	s.T().Run("update number", func(t *testing.T) {
		// when updating the message and the number
		model.Message = "new message 2"
		model.Number = 2
		db = db.Save(&model)
		// then there should be no rows affected because we're not supposed to
		// update the number
		require.NoError(t, db.Error)
		require.Equal(t, int64(0), db.RowsAffected)
	})
}

func (s *humanFriendlyNumberSuite) TestConcurrentCreate() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))
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
				model := testStruct1{
					ID:                  uuid.NewV4(),
					Message:             "model " + uuid.NewV4().String(),
					HumanFriendlyNumber: numbersequence.NewHumanFriendlyNumber(fxt.Spaces[0].ID, tableName1),
				}
				db := s.DB.Create(&model)
				if db.Error != nil {
					s.T().Logf("creation failed: %+v", db.Error.Error())
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
		fmt.Printf("Routine #%d done: %d creations, including %d failure(s)\n", report.id, report.total, report.failures)
		assert.Equal(s.T(), itemsPerRoutine, report.total)
		assert.Equal(s.T(), 0, report.failures)
	}

	// check that the created models have the correct numbers
	toBeFound := map[int]struct{}{}
	for i := 0; i < routines*itemsPerRoutine; i++ {
		toBeFound[i+1] = struct{}{}
	}
	models := []testStruct1{}
	db := s.DB.Find(&models)
	require.NoError(s.T(), db.Error)

	for _, m := range models {
		_, ok := toBeFound[m.Number]
		assert.True(s.T(), ok, "unexpected number found: %d", m.Number)
		delete(toBeFound, m.Number)
	}
	require.Empty(s.T(), toBeFound, "failed to find these numbers: %+v", toBeFound)
}
