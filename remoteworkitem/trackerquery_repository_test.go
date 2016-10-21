package remoteworkitem

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/stretchr/testify/assert"
)

func TestTrackerQueryCreate(t *testing.T) {
	doWithTrackerRepositories(t, func(ctx context.Context, trackerRepo TrackerRepository, queryRepo TrackerQueryRepository) {
		query, err := queryRepo.Create(ctx, "abc", "xyz", "lmn")
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, query)

		tracker, err := trackerRepo.Create(ctx, "gugus", ProviderJira)
		query, err = queryRepo.Create(ctx, "abc", "xyz", tracker.ID)
		assert.Nil(t, err)
		assert.Equal(t, "abc", query.Query)
		assert.Equal(t, "xyz", query.Schedule)

		query2, err := queryRepo.Load(ctx, query.ID)
		assert.Nil(t, err)
		assert.Equal(t, query, query2)
	})
}

func TestTrackerQuerySave(t *testing.T) {
	doWithTrackerRepositories(t, func(ctx context.Context, trackerRepo TrackerRepository, queryRepo TrackerQueryRepository) {

		query, err := queryRepo.Load(ctx, "abcd")
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, query)

		tracker, err := trackerRepo.Create(ctx, "gugus", ProviderJira)
		tracker2, err := trackerRepo.Create(ctx, "theother", ProviderGithub)
		query, err = queryRepo.Create(ctx, "abc", "xyz", tracker.ID)
		query2, err := queryRepo.Load(ctx, query.ID)
		assert.Nil(t, err)
		assert.Equal(t, query, query2)

		query.Query = "after"
		query.Schedule = "the"
		query.TrackerID = tracker2.ID
		if err != nil {
			t.Errorf("could not convert id: %s", tracker2.ID)
		}

		query2, err = queryRepo.Save(ctx, *query)
		assert.Nil(t, err)
		assert.Equal(t, query, query2)

		trackerRepo.Delete(ctx, "10000")

		query.TrackerID = "10000"
		query2, err = queryRepo.Save(ctx, *query)
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, query2)
	})
}

func TestTrackerQueryDelete(t *testing.T) {
	doWithTrackerRepositories(t, func(ctx context.Context, trackerRepo TrackerRepository, queryRepo TrackerQueryRepository) {
		_, err := queryRepo.Load(ctx, "asdf")
		assert.IsType(t, NotFoundError{}, err)

		_, err = queryRepo.Load(ctx, "100000")
		assert.IsType(t, NotFoundError{}, err)
	})
}

func doWithTrackerRepositories(t *testing.T, todo func(context.Context, TrackerRepository, TrackerQueryRepository)) {
	doWithTransaction(t, func(ctx context.Context) {
		trackerRepo := NewTrackerRepository()
		queryRepo := NewTrackerQueryRepository()
		todo(ctx, trackerRepo, queryRepo)
	})

}
