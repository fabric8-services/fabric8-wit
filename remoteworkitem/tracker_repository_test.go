package remoteworkitem

import (
	"testing"

	"context"

	errors "github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	uuid "github.com/satori/go.uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestTrackerRepository struct {
	gormtestsupport.DBTestSuite

	repo TrackerRepository

	clean func()
}

func TestRunTrackerRepository(t *testing.T) {
	suite.Run(t, &TestTrackerRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (test *TestTrackerRepository) SetupTest() {
	test.repo = NewTrackerRepository(test.DB)
	test.clean = cleaner.DeleteCreatedEntities(test.DB)
}

func (test *TestTrackerRepository) TearDownTest() {
	test.clean()
}

func (test *TestTrackerRepository) TestTrackerCreate() {
	t := test.T()
	resource.Require(t, resource.Database)

	tracker := Tracker{
		URL:  "url",
		Type: "type",
	}

	err := test.repo.Create(context.Background(), &tracker)
	assert.IsType(t, BadParameterError{}, err)

	tracker = Tracker{
		URL:  "http://api.github.com",
		Type: ProviderGithub,
	}

	err = test.repo.Create(context.Background(), &tracker)
	assert.Nil(t, err)

	tracker2, err := test.repo.Load(context.Background(), tracker.ID)
	assert.Nil(t, err)
	assert.NotNil(t, tracker2)
	assert.Equal(t, "http://api.github.com", tracker2.URL)
	assert.Equal(t, ProviderGithub, tracker2.Type)
}

func (test *TestTrackerRepository) TestExistsTracker() {
	t := test.T()
	resource.Require(t, resource.Database)

	t.Run("tracker exists", func(t *testing.T) {
		t.Parallel()
		tracker := Tracker{
			URL:  "http://api.github.com",
			Type: ProviderGithub,
		}
		err := test.repo.Create(context.Background(), &tracker)
		assert.Nil(t, err)
		require.NotNil(t, tracker)
		assert.Equal(t, "http://api.github.com", tracker.URL)
		assert.Equal(t, ProviderGithub, tracker.Type)

		err = test.repo.CheckExists(context.Background(), tracker.ID.String())
		assert.Nil(t, err)
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

	tracker1, err := test.repo.Save(context.Background(), &Tracker{})
	assert.IsType(t, errors.NotFoundError{}, err)
	assert.Nil(t, tracker1)

	tracker2 := &Tracker{
		URL:  "http://api.github.com",
		Type: ProviderGithub,
	}

	err = test.repo.Create(context.Background(), tracker2)
	tracker2.Type = "blabla"
	tracker3, err := test.repo.Save(context.Background(), tracker2)
	assert.IsType(t, errors.BadParameterError{}, err)
	assert.Nil(t, tracker3)

	tracker4 := &Tracker{
		ID:   uuid.NewV4(),
		URL:  "random1",
		Type: ProviderJira,
	}
	tracker4, err = test.repo.Save(context.Background(), tracker4)
	assert.IsType(t, errors.NotFoundError{}, err)

	tracker5 := &Tracker{
		ID: uuid.FromStringOrNil("e0022d1-ad23-4f1b-9ee2-93f5d9269d1e"),
	}
	tracker5, err = test.repo.Save(context.Background(), tracker5)
	assert.IsType(t, errors.NotFoundError{}, err)
	assert.Nil(t, tracker5)

	tracker6 := &Tracker{
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

	tracker := Tracker{
		URL:  "http://api.github.com",
		Type: ProviderGithub,
	}
	err = test.repo.Create(context.Background(), &tracker)
	err = test.repo.Delete(context.Background(), tracker.ID)
	assert.Nil(t, err)

	tracker2, err := test.repo.Load(context.Background(), tracker.ID)
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

	tracker1 := Tracker{
		URL:  "http://api.github.com",
		Type: ProviderGithub,
	}
	tracker2 := Tracker{
		URL:  "http://api.github.com",
		Type: ProviderGithub,
	}
	tracker3 := Tracker{
		URL:  "http://issues.jboss.com",
		Type: ProviderJira,
	}
	tracker4 := Tracker{
		URL:  "http://issues.jboss.com",
		Type: ProviderJira,
	}
	test.repo.Create(context.Background(), &tracker1)
	test.repo.Create(context.Background(), &tracker2)
	test.repo.Create(context.Background(), &tracker3)
	test.repo.Create(context.Background(), &tracker4)

	trackers2, _ := test.repo.List(context.Background())

	assert.Equal(t, len(trackers)+4, len(trackers2))

	trackers3, _ := test.repo.List(context.Background())
	assert.Equal(t, trackers2[0].ID, trackers3[0].ID)
}
