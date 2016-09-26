package remoteworkitem

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
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

		tracker.ID = "10000"
		tracker2, err = trackerRepo.Save(context.Background(), *tracker)
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, tracker2)

		tracker.ID = "asdf"
		tracker2, err = trackerRepo.Save(context.Background(), *tracker)
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, tracker2)

	})
}

func TestTrackerDelete(t *testing.T) {
	doWithTrackerRepository(t, func(trackerRepo TrackerRepository) {
		err := trackerRepo.Delete(context.Background(), "asdf")
		assert.IsType(t, NotFoundError{}, err)

		// guard against other test leaving stuff behind
		err = trackerRepo.Delete(context.Background(), "10000")

		err = trackerRepo.Delete(context.Background(), "10000")
		assert.IsType(t, NotFoundError{}, err)

		tracker, _ := trackerRepo.Create(context.Background(), "gugus", ProviderGithub)
		err = trackerRepo.Delete(context.Background(), tracker.ID)
		assert.Nil(t, err)

		tracker, err = trackerRepo.Load(context.Background(), tracker.ID)
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, tracker)

		tracker, err = trackerRepo.Load(context.Background(), "xyz")
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, tracker)
	})
}

func TestTrackerList(t *testing.T) {
	doWithTrackerRepository(t, func(trackerRepo TrackerRepository) {
		trackers, _ := trackerRepo.List(context.Background(), criteria.Literal(true), nil, nil)

		trackerRepo.Create(context.Background(), "gugus", ProviderGithub)
		trackerRepo.Create(context.Background(), "dada", ProviderJira)
		trackerRepo.Create(context.Background(), "blabla", ProviderJira)
		trackerRepo.Create(context.Background(), "xoxo", ProviderGithub)

		trackers2, _ := trackerRepo.List(context.Background(), criteria.Literal(true), nil, nil)

		assert.Equal(t, len(trackers)+4, len(trackers2))
		start, len := 1, 1

		trackers3, _ := trackerRepo.List(context.Background(), criteria.Literal(true), &start, &len)
		assert.Equal(t, trackers2[1], trackers3[0])

	})
}

func doWithTrackerRepository(t *testing.T, todo func(repo TrackerRepository)) {
	doWithTransaction(t, func(ts *models.GormTransactionSupport) {
		trackerRepo := NewTrackerRepository(ts)
		todo(trackerRepo)
	})

}

func doWithTransaction(t *testing.T, todo func(ts *models.GormTransactionSupport)) {
	resource.Require(t, resource.Database)
	ts := models.NewGormTransactionSupport(db)
	if err := ts.Begin(); err != nil {
		panic(err.Error())
	}
	defer ts.Rollback()
	todo(ts)
}
