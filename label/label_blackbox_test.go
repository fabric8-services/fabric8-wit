package label_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	errs "github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/label"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestLabelRepository struct {
	gormtestsupport.DBTestSuite
}

func TestRunLabelRepository(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestLabelRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *TestLabelRepository) TestCreateLabel() {
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))
	repo := label.NewLabelRepository(s.DB)
	name := "TestCreateLabel"
	l := label.Label{
		SpaceID: testFxt.Spaces[0].ID,
		Name:    name,
	}
	repo.Create(context.Background(), &l)
	require.NotEqual(s.T(), uuid.Nil, l.ID)
	require.Equal(s.T(), "#000000", l.TextColor)
	require.Equal(s.T(), "#FFFFFF", l.BackgroundColor)
	require.Equal(s.T(), "#000000", l.BorderColor)
	require.False(s.T(), l.CreatedAt.After(time.Now()), "Label was not created, CreatedAt after Now()")
	require.False(s.T(), l.UpdatedAt.After(time.Now()), "Label was not created, UpdatedAt after Now()")
	require.Nil(s.T(), l.DeletedAt)
}

func (s *TestLabelRepository) TestCreateLabelWithEmptyName() {
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))
	repo := label.NewLabelRepository(s.DB)
	name := ""
	l := label.Label{
		SpaceID: testFxt.Spaces[0].ID,
		Name:    name,
	}
	err := repo.Create(context.Background(), &l)
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "label name cannot be empty string")
}

func (s *TestLabelRepository) TestCreateLabelWithSameName() {
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))
	repo := label.NewLabelRepository(s.DB)
	name := "TestCreateLabel"
	l := label.Label{
		SpaceID: testFxt.Spaces[0].ID,
		Name:    name,
	}
	repo.Create(context.Background(), &l)
	require.NotEqual(s.T(), uuid.Nil, l.ID)
	require.Equal(s.T(), "#000000", l.TextColor)
	require.Equal(s.T(), "#FFFFFF", l.BackgroundColor)
	require.Equal(s.T(), "#000000", l.BorderColor)
	require.False(s.T(), l.CreatedAt.After(time.Now()), "Label was not created, CreatedAt after Now()")
	require.False(s.T(), l.UpdatedAt.After(time.Now()), "Label was not created, UpdatedAt after Now()")
	require.Nil(s.T(), l.DeletedAt)

	err := repo.Create(context.Background(), &l)
	require.Error(s.T(), err)
	_, ok := errors.Cause(err).(errs.DataConflictError)
	assert.Contains(s.T(), err.Error(), "label already exists with name = TestCreateLabel")
	assert.True(s.T(), ok)
}

func (s *TestLabelRepository) TestCreateLabelWithWrongColorCode() {
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))
	repo := label.NewLabelRepository(s.DB)
	name := "TestCreateLabel"
	l := label.Label{
		SpaceID:   testFxt.Spaces[0].ID,
		Name:      name,
		TextColor: "#yyppww",
	}
	err := repo.Create(context.Background(), &l)
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "labels_text_color_check")

	l2 := label.Label{
		SpaceID:         testFxt.Spaces[0].ID,
		Name:            name,
		BackgroundColor: "#yyppww",
	}
	err = repo.Create(context.Background(), &l2)
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "labels_background_color_check")
}

func (s *TestLabelRepository) TestSave() {
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Labels(1))
	repo := label.NewLabelRepository(s.DB)

	s.T().Run("success - save label", func(t *testing.T) {
		l := testFxt.Labels[0]
		l.Name = "severity/p5"
		l.TextColor = "#778899"
		l.BackgroundColor = "#445566"
		l.BorderColor = "#112233"

		lbl, err := repo.Save(context.Background(), *l)
		require.NoError(t, err)
		assert.Equal(t, l.Name, lbl.Name)
		assert.Equal(t, l.TextColor, lbl.TextColor)
		assert.Equal(t, l.BackgroundColor, lbl.BackgroundColor)
		assert.Equal(t, l.BorderColor, lbl.BorderColor)
	})

	s.T().Run("empty name", func(t *testing.T) {
		l := testFxt.Labels[0]
		l.Name = ""
		l.TextColor = "#778899"
		l.BackgroundColor = "#445566"
		l.BorderColor = "#112233"

		_, err := repo.Save(context.Background(), *l)
		require.Error(t, err)
		_, ok := errors.Cause(err).(errs.BadParameterError)
		assert.Contains(t, err.Error(), "label name cannot be empty string")
		assert.True(t, ok)
	})

	s.T().Run("non-existing label", func(t *testing.T) {
		fakeID := uuid.NewV4()
		fakeLabel := label.Label{
			ID:   fakeID,
			Name: "Some name",
		}
		repo := label.NewLabelRepository(s.DB)
		_, err := repo.Save(context.Background(), fakeLabel)
		require.Error(t, err)
		assert.Equal(t, reflect.TypeOf(errs.NotFoundError{}), reflect.TypeOf(err))
	})
	s.T().Run("update label with same name", func(t *testing.T) {
		testFxt := tf.NewTestFixture(t, s.DB, tf.Labels(2))
		repo := label.NewLabelRepository(s.DB)
		testFxt.Labels[0].Name = testFxt.Labels[1].Name

		_, err := repo.Save(context.Background(), *testFxt.Labels[0])
		require.Error(t, err)
		_, ok := errors.Cause(err).(errs.DataConflictError)
		assert.Contains(t, err.Error(), "label already exists with name = label")
		assert.True(t, ok)
	})
}

func (s *TestLabelRepository) TestListLabelBySpace() {
	n := 3
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Labels(n))

	labelList, err := label.NewLabelRepository(s.DB).List(context.Background(), testFxt.Spaces[0].ID)
	require.NoError(s.T(), err)
	require.Len(s.T(), labelList, n)

	labelIDs := map[uuid.UUID]struct{}{}
	for _, l := range testFxt.Labels {
		labelIDs[l.ID] = struct{}{}
	}
	for _, l := range labelList {
		delete(labelIDs, l.ID)
	}
	require.Empty(s.T(), labelIDs, "not all labels were found")
}

func (s *TestLabelRepository) TestLoadLabel() {
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Labels(1))
	lbl, err := label.NewLabelRepository(s.DB).Load(context.Background(), testFxt.Labels[0].ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), lbl)
	assert.Equal(s.T(), testFxt.Labels[0].Name, lbl.Name)
}
