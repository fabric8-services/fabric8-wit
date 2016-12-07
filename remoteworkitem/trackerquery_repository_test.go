package remoteworkitem

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/gormsupport"
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
		err := queryRepo.Delete(context.Background(), "asdf")
		assert.IsType(t, NotFoundError{}, err)

		tracker, _ := trackerRepo.Create(context.Background(), "http://api.github.com", ProviderGithub)
		tq, _ := queryRepo.Create(context.Background(), "is:open is:issue user:arquillian author:aslakknutsen", "15 * * * * *", tracker.ID)
		err = queryRepo.Delete(context.Background(), tq.ID)
		assert.Nil(t, err)

		tq, err = queryRepo.Load(context.Background(), tq.ID)
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, tq)

		tq, err = queryRepo.Load(context.Background(), "100000")
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, tq)
	})
}

func TestTrackerQueryList(t *testing.T) {
	doWithTrackerRepositories(t, func(trackerRepo application.TrackerRepository, queryRepo application.TrackerQueryRepository) {
		trackerqueries1, _ := queryRepo.List(context.Background())

		tracker1, _ := trackerRepo.Create(context.Background(), "http://api.github.com", ProviderGithub)
		queryRepo.Create(context.Background(), "is:open is:issue user:arquillian author:aslakknutsen", "15 * * * * *", tracker1.ID)
		queryRepo.Create(context.Background(), "is:close is:issue user:arquillian author:aslakknutsen", "", tracker1.ID)

		tracker2, _ := trackerRepo.Create(context.Background(), "http://issues.jboss.com", ProviderJira)
		queryRepo.Create(context.Background(), "project = ARQ AND text ~ 'arquillian'", "15 * * * * *", tracker2.ID)
		queryRepo.Create(context.Background(), "project = ARQ AND text ~ 'javadoc'", "15 * * * * *", tracker2.ID)

		trackerqueries2, _ := queryRepo.List(context.Background())
		assert.Equal(t, len(trackerqueries1)+4, len(trackerqueries2))
		trackerqueries3, _ := queryRepo.List(context.Background())
		assert.Equal(t, trackerqueries2[1], trackerqueries3[1])
	})
}

func doWithTrackerRepositories(t *testing.T, todo func(trackerRepo application.TrackerRepository, queryRepo application.TrackerQueryRepository)) {
	doWithTransaction(t, func(db *gorm.DB) {
		defer gormsupport.DeleteCreatedEntities(db)()
		trackerRepo := NewTrackerRepository(db)
		queryRepo := NewTrackerQueryRepository(db)
		todo(trackerRepo, queryRepo)
	})

}
