package remoteworkitem_test

import (
	"testing"

	"context"

	errors "github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	uuid "github.com/satori/go.uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestTrackerRepository struct {
	gormtestsupport.DBTestSuite
	repo remoteworkitem.TrackerRepository
}

func TestRunTrackerRepository(t *testing.T) {
	suite.Run(t, &TestTrackerRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (test *TestTrackerRepository) SetupTest() {
	test.DBTestSuite.SetupTest()
	test.repo = remoteworkitem.NewTrackerRepository(test.DB)
}

func (test *TestTrackerRepository) TestTrackerCreate() {
	t := test.T()
	resource.Require(t, resource.Database)

	tracker := remoteworkitem.Tracker{
		URL:  "url",
		Type: "type",
	}

	err := test.repo.Create(context.Background(), &tracker)
	assert.IsType(t, remoteworkitem.BadParameterError{}, err)

	tracker = remoteworkitem.Tracker{
		URL:  "http://api.github.com",
		Type: remoteworkitem.ProviderGithub,
	}

	err = test.repo.Create(context.Background(), &tracker)
	require.NoError(t, err)

	tracker2, err := test.repo.Load(context.Background(), tracker.ID)
	require.NoError(t, err)
	assert.NotNil(t, tracker2)
	assert.Equal(t, "http://api.github.com", tracker2.URL)
	assert.Equal(t, remoteworkitem.ProviderGithub, tracker2.Type)
}

func (test *TestTrackerRepository) TestExistsTracker() {
	t := test.T()
	resource.Require(t, resource.Database)
	githubTrackerURL := "https://api.github.com/"
	t.Run("tracker exists", func(t *testing.T) {
		t.Parallel()
		fxt := tf.NewTestFixture(t, test.DB, tf.Trackers(1))
		require.NotNil(t, fxt.Trackers[0])
		assert.Equal(t, githubTrackerURL, fxt.Trackers[0].URL)
		assert.Equal(t, remoteworkitem.ProviderGithub, fxt.Trackers[0].Type)

		err := test.repo.CheckExists(context.Background(), fxt.Trackers[0].ID.String())
		require.NoError(t, err)
	})

	t.Run("tracker doesn't exist", func(t *testing.T) {
		t.Parallel()
		err := test.repo.CheckExists(context.Background(), uuid.NewV4().String())
		require.IsType(t, errors.NotFoundError{}, err)
	})

}

func (test *TestTrackerRepository) TestTrackerSave() {
	t := test.T()
	resource.Require(t, resource.Database)

	tracker1, err := test.repo.Save(context.Background(), &remoteworkitem.Tracker{})
	assert.IsType(t, errors.NotFoundError{}, err)
	assert.Nil(t, tracker1)

	fxt := tf.NewTestFixture(t, test.DB, tf.Trackers(1))
	fxt.Trackers[0].Type = "blabla"
	tracker3, err := test.repo.Save(context.Background(), fxt.Trackers[0])
	assert.IsType(t, errors.BadParameterError{}, err)
	assert.Nil(t, tracker3)

	tracker4 := &remoteworkitem.Tracker{
		ID:   uuid.NewV4(),
		URL:  "random1",
		Type: remoteworkitem.ProviderJira,
	}
	tracker4, err = test.repo.Save(context.Background(), tracker4)
	assert.IsType(t, errors.NotFoundError{}, err)

	tracker5 := &remoteworkitem.Tracker{
		ID: uuid.FromStringOrNil("e0022d1-ad23-4f1b-9ee2-93f5d9269d1e"),
	}
	tracker5, err = test.repo.Save(context.Background(), tracker5)
	assert.IsType(t, errors.NotFoundError{}, err)
	assert.Nil(t, tracker5)

	tracker6 := &remoteworkitem.Tracker{
		ID: uuid.NewV4(),
	}
	tracker7, err := test.repo.Save(context.Background(), tracker6)
	assert.IsType(t, errors.NotFoundError{}, err)
	assert.Nil(t, tracker7)
}

func (test *TestTrackerRepository) TestTrackerDelete() {
	t := test.T()
	resource.Require(t, resource.Database)

	err := test.repo.Delete(context.Background(), uuid.NewV4())
	assert.IsType(t, errors.NotFoundError{}, err)

	// guard against other test leaving stuff behind
	err = test.repo.Delete(context.Background(), uuid.NewV4())

	err = test.repo.Delete(context.Background(), uuid.NewV4())
	assert.IsType(t, errors.NotFoundError{}, err)

	fxt := tf.NewTestFixture(t, test.DB, tf.Trackers(1))
	err = test.repo.Delete(context.Background(), fxt.Trackers[0].ID)
	require.NoError(t, err)

	tracker2, err := test.repo.Load(context.Background(), fxt.Trackers[0].ID)
	assert.IsType(t, errors.NotFoundError{}, err)
	assert.Nil(t, tracker2)

	tracker3, err := test.repo.Load(context.Background(), uuid.NewV4())
	assert.IsType(t, errors.NotFoundError{}, err)
	assert.Nil(t, tracker3)
}

func (test *TestTrackerRepository) TestTrackerList() {
	t := test.T()
	resource.Require(t, resource.Database)

	trackers, _ := test.repo.List(context.Background())

	tf.NewTestFixture(t, test.DB, tf.Trackers(4))

	trackers2, _ := test.repo.List(context.Background())

	assert.Equal(t, len(trackers)+4, len(trackers2))

	trackers3, _ := test.repo.List(context.Background())
	assert.Equal(t, trackers2[0].ID, trackers3[0].ID)
}
