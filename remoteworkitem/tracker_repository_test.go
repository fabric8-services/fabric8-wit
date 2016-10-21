package remoteworkitem

import (
	"errors"
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/transaction"
	"github.com/stretchr/testify/assert"
)

func TestTrackerCreate(t *testing.T) {
	doWithTrackerRepository(t, func(ctx context.Context, trackerRepo TrackerRepository) {
		tracker, err := trackerRepo.Create(ctx, "gugus", "dada")
		assert.IsType(t, BadParameterError{}, err)
		assert.Nil(t, tracker)

		tracker, err = trackerRepo.Create(ctx, "gugus", ProviderGithub)
		assert.Nil(t, err)
		assert.NotNil(t, tracker)
		assert.Equal(t, "gugus", tracker.URL)
		assert.Equal(t, ProviderGithub, tracker.Type)

		tracker2, err := trackerRepo.Load(ctx, tracker.ID)
		assert.Nil(t, err)
		assert.NotNil(t, tracker2)
	})
}

func TestTrackerSave(t *testing.T) {
	doWithTrackerRepository(t, func(ctx context.Context, trackerRepo TrackerRepository) {
		tracker, err := trackerRepo.Save(ctx, app.Tracker{})
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, tracker)

		tracker, _ = trackerRepo.Create(ctx, "gugus", ProviderGithub)
		tracker.Type = "blabla"
		tracker2, err := trackerRepo.Save(ctx, *tracker)
		assert.IsType(t, BadParameterError{}, err)
		assert.Nil(t, tracker2)

		tracker.Type = ProviderJira
		tracker.URL = "blabla"
		tracker, err = trackerRepo.Save(ctx, *tracker)
		assert.Equal(t, ProviderJira, tracker.Type)
		assert.Equal(t, "blabla", tracker.URL)

		tracker.ID = "10000"
		tracker2, err = trackerRepo.Save(ctx, *tracker)
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, tracker2)

		tracker.ID = "asdf"
		tracker2, err = trackerRepo.Save(ctx, *tracker)
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, tracker2)

	})
}

func TestTrackerDelete(t *testing.T) {
	doWithTrackerRepository(t, func(ctx context.Context, trackerRepo TrackerRepository) {
		err := trackerRepo.Delete(ctx, "asdf")
		assert.IsType(t, NotFoundError{}, err)

		// guard against other test leaving stuff behind
		err = trackerRepo.Delete(ctx, "10000")

		err = trackerRepo.Delete(ctx, "10000")
		assert.IsType(t, NotFoundError{}, err)

		tracker, _ := trackerRepo.Create(ctx, "gugus", ProviderGithub)
		err = trackerRepo.Delete(ctx, tracker.ID)
		assert.Nil(t, err)

		tracker, err = trackerRepo.Load(ctx, tracker.ID)
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, tracker)

		tracker, err = trackerRepo.Load(ctx, "xyz")
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, tracker)
	})
}

func TestTrackerList(t *testing.T) {
	doWithTrackerRepository(t, func(ctx context.Context, trackerRepo TrackerRepository) {
		trackers, _ := trackerRepo.List(ctx, criteria.Literal(true), nil, nil)

		trackerRepo.Create(ctx, "gugus", ProviderGithub)
		trackerRepo.Create(ctx, "dada", ProviderJira)
		trackerRepo.Create(ctx, "blabla", ProviderJira)
		trackerRepo.Create(ctx, "xoxo", ProviderGithub)

		trackers2, _ := trackerRepo.List(ctx, criteria.Literal(true), nil, nil)

		assert.Equal(t, len(trackers)+4, len(trackers2))
		start, len := 1, 1

		trackers3, _ := trackerRepo.List(ctx, criteria.Literal(true), &start, &len)
		assert.Equal(t, trackers2[1], trackers3[0])

	})
}

func doWithTrackerRepository(t *testing.T, todo func(context.Context, TrackerRepository)) {
	doWithTransaction(t, func(ctx context.Context) {
		trackerRepo := NewTrackerRepository()
		todo(ctx, trackerRepo)
	})

}

func doWithTransaction(t *testing.T, todo func(context.Context)) {

	resource.Require(t, resource.Database)
	ts := models.NewGormTransactionSupport(db)
	transaction.Do(ts, context.Background(), func(ctx context.Context) error {
		todo(ctx)
		return errors.New("force rollback")
	})
}
