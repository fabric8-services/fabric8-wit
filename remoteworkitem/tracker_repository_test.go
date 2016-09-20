package remoteworkitem

import (
	"context"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

func TestTrackerCreate(t *testing.T) {
	doWithTrackerRepository(t, func(trackerRepo TrackerRepository) {
		tracker, err := trackerRepo.Create(context.Background(), "gugus", "dada")
		assert.IsType(t, BadParameterError{}, err)
		assert.Nil(t, tracker)

		tracker, err = trackerRepo.Create(context.Background(), "gugus", ProviderGithub)
		assert.Nil(t, err)
		assert.NotNil(t, tracker)
		assert.Equal(t, "gugus", tracker.URL)
		assert.Equal(t, ProviderGithub, tracker.Type)

		tracker2, err := trackerRepo.Load(context.Background(), tracker.ID)
		assert.Nil(t, err)
		assert.NotNil(t, tracker2)
	})
}

func TestTrackerSave(t *testing.T) {
	doWithTrackerRepository(t, func(trackerRepo TrackerRepository) {
		tracker, err := trackerRepo.Save(context.Background(), app.Tracker{})
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, tracker)

		tracker, _ = trackerRepo.Create(context.Background(), "gugus", ProviderGithub)
		tracker.Type = "blabla"
		tracker2, err := trackerRepo.Save(context.Background(), *tracker)
		assert.IsType(t, BadParameterError{}, err)
		assert.Nil(t, tracker2)

		tracker.Type = ProviderJira
		tracker.URL = "blabla"
		tracker, err = trackerRepo.Save(context.Background(), *tracker)
		assert.Equal(t, ProviderJira, tracker.Type)
		assert.Equal(t, "blabla", tracker.URL)
	})
}

func testTrackerDelete(t *testing.T) {
	doWithTrackerRepository(t, func(trackerRepo TrackerRepository) {
		err := trackerRepo.Delete(context.Background(), "asdf")
		assert.IsType(t, NotFoundError{}, err)

		tracker, _ := trackerRepo.Create(context.Background(), "gugus", ProviderGithub)
		err = trackerRepo.Delete(context.Background(), tracker.ID)
		assert.Nil(t, err)

		tracker, err = trackerRepo.Load(context.Background(), tracker.ID)
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, tracker)
	})
}

func doWithTrackerRepository(t *testing.T, todo func(repo TrackerRepository)) {
	resource.Require(t, resource.Database)
	ts := models.NewGormTransactionSupport(db)
	trackerRepo := NewTrackerRepository(ts)
	if err := ts.Begin(); err != nil {
		panic(err.Error())
	}
	defer ts.Rollback()
	todo(trackerRepo)
}
