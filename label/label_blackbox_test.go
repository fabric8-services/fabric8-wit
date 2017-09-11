package label_test

import (
	"context"
	"testing"
	"time"

	errs "github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
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
	clean func()
}

func TestRunLabelRepository(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestLabelRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *TestLabelRepository) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

func (s *TestLabelRepository) TearDownTest() {
	s.clean()
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
	require.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "labels_name_check")
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
	require.False(s.T(), l.CreatedAt.After(time.Now()), "Label was not created, CreatedAt after Now()")
	require.False(s.T(), l.UpdatedAt.After(time.Now()), "Label was not created, UpdatedAt after Now()")
	require.Nil(s.T(), l.DeletedAt)

	err := repo.Create(context.Background(), &l)
	require.NotNil(s.T(), err)
	_, ok := errors.Cause(err).(errs.DataConflictError)
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
	require.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "labels_text_color_check")

	l2 := label.Label{
		SpaceID:         testFxt.Spaces[0].ID,
		Name:            name,
		BackgroundColor: "#yyppww",
	}
	err = repo.Create(context.Background(), &l2)
	require.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "labels_background_color_check")
}

func (s *TestLabelRepository) TestListLabelBySpace() {
	n := 3
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Labels(n))

	labelList, err := label.NewLabelRepository(s.DB).List(context.Background(), testFxt.Spaces[0].ID)
	require.Nil(s.T(), err)
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
	lbl, err := label.NewLabelRepository(s.DB).Load(context.Background(), testFxt.Spaces[0].ID, testFxt.Labels[0].ID)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), lbl)
	assert.Equal(s.T(), testFxt.Labels[0].Name, lbl.Name)
}
