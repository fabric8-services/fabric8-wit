package remoteworkitem

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/application"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
)

func TestTrackerQueryCreate(t *testing.T) {
	doWithTrackerRepositories(t, func(trackerRepo application.TrackerRepository, queryRepo application.TrackerQueryRepository) {
		query, err := queryRepo.Create(context.Background(), "abc", "xyz", "lmn")
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, query)

		tracker, err := trackerRepo.Create(context.Background(), "http://issues.jboss.com", ProviderJira)
		query, err = queryRepo.Create(context.Background(), "abc", "xyz", tracker.ID)
		assert.Nil(t, err)
		assert.Equal(t, "abc", query.Query)
		assert.Equal(t, "xyz", query.Schedule)

		query2, err := queryRepo.Load(context.Background(), query.ID)
		assert.Nil(t, err)
		assert.Equal(t, query, query2)
	})
}

func TestTrackerQuerySave(t *testing.T) {
	doWithTrackerRepositories(t, func(trackerRepo application.TrackerRepository, queryRepo application.TrackerQueryRepository) {

		query, err := queryRepo.Load(context.Background(), "abcd")
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, query)

		tracker, err := trackerRepo.Create(context.Background(), "http://issues.jboss.com", ProviderJira)
		tracker2, err := trackerRepo.Create(context.Background(), "http://api.github.com", ProviderGithub)
		query, err = queryRepo.Create(context.Background(), "abc", "xyz", tracker.ID)
		query2, err := queryRepo.Load(context.Background(), query.ID)
		assert.Nil(t, err)
		assert.Equal(t, query, query2)

		query.Query = "after"
		query.Schedule = "the"
		query.TrackerID = tracker2.ID
		if err != nil {
			t.Errorf("could not convert id: %s", tracker2.ID)
		}

		query2, err = queryRepo.Save(context.Background(), *query)
		assert.Nil(t, err)
		assert.Equal(t, query, query2)

		trackerRepo.Delete(context.Background(), "10000")

		query.TrackerID = "10000"
		query2, err = queryRepo.Save(context.Background(), *query)
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, query2)
	})
}

func TestTrackerQueryDelete(t *testing.T) {
	doWithTrackerRepositories(t, func(trackerRepo application.TrackerRepository, queryRepo application.TrackerQueryRepository) {
		_, err := queryRepo.Load(context.Background(), "asdf")
		assert.IsType(t, NotFoundError{}, err)

		_, err = queryRepo.Load(context.Background(), "100000")
		assert.IsType(t, NotFoundError{}, err)
	})
}

func doWithTrackerRepositories(t *testing.T, todo func(trackerRepo application.TrackerRepository, queryRepo application.TrackerQueryRepository)) {
	doWithTransaction(t, func(ts *gorm.DB) {
		trackerRepo := NewTrackerRepository(db)
		queryRepo := NewTrackerQueryRepository(db)
		todo(trackerRepo, queryRepo)
	})

}
