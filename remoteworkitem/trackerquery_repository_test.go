package remoteworkitem

import (
	"net/http"
	"net/url"
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/space"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
)

func TestTrackerQueryCreate(t *testing.T) {
	doWithTrackerRepositories(t, func(trackerRepo application.TrackerRepository, queryRepo application.TrackerQueryRepository) {
		req := &http.Request{Host: "localhost"}
		params := url.Values{}
		ctx := goa.NewContext(context.Background(), nil, req, params)

		query, err := queryRepo.Create(ctx, "abc", "xyz", "lmn", space.SystemSpace)
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, query)

		tracker, err := trackerRepo.Create(ctx, "http://issues.jboss.com", ProviderJira)
		query, err = queryRepo.Create(ctx, "abc", "xyz", tracker.ID, space.SystemSpace)
		assert.Nil(t, err)
		assert.Equal(t, "abc", query.Query)
		assert.Equal(t, "xyz", query.Schedule)

		query2, err := queryRepo.Load(ctx, query.ID)
		assert.Nil(t, err)
		assert.Equal(t, query, query2)
	})
}

func TestTrackerQuerySave(t *testing.T) {
	doWithTrackerRepositories(t, func(trackerRepo application.TrackerRepository, queryRepo application.TrackerQueryRepository) {
		req := &http.Request{Host: "localhost"}
		params := url.Values{}
		ctx := goa.NewContext(context.Background(), nil, req, params)

		query, err := queryRepo.Load(ctx, "abcd")
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, query)

		tracker, err := trackerRepo.Create(ctx, "http://issues.jboss.com", ProviderJira)
		tracker2, err := trackerRepo.Create(ctx, "http://api.github.com", ProviderGithub)
		query, err = queryRepo.Create(ctx, "abc", "xyz", tracker.ID, space.SystemSpace)
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
	doWithTrackerRepositories(t, func(trackerRepo application.TrackerRepository, queryRepo application.TrackerQueryRepository) {
		req := &http.Request{Host: "localhost"}
		params := url.Values{}
		ctx := goa.NewContext(context.Background(), nil, req, params)

		err := queryRepo.Delete(ctx, "asdf")
		assert.IsType(t, NotFoundError{}, err)

		tracker, _ := trackerRepo.Create(ctx, "http://api.github.com", ProviderGithub)
		tq, _ := queryRepo.Create(ctx, "is:open is:issue user:arquillian author:aslakknutsen", "15 * * * * *", tracker.ID, space.SystemSpace)
		err = queryRepo.Delete(ctx, tq.ID)
		assert.Nil(t, err)

		tq, err = queryRepo.Load(ctx, tq.ID)
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, tq)

		tq, err = queryRepo.Load(ctx, "100000")
		assert.IsType(t, NotFoundError{}, err)
		assert.Nil(t, tq)
	})
}

func TestTrackerQueryList(t *testing.T) {
	doWithTrackerRepositories(t, func(trackerRepo application.TrackerRepository, queryRepo application.TrackerQueryRepository) {
		req := &http.Request{Host: "localhost"}
		params := url.Values{}
		ctx := goa.NewContext(context.Background(), nil, req, params)

		trackerqueries1, _ := queryRepo.List(ctx)

		tracker1, _ := trackerRepo.Create(ctx, "http://api.github.com", ProviderGithub)
		queryRepo.Create(ctx, "is:open is:issue user:arquillian author:aslakknutsen", "15 * * * * *", tracker1.ID, space.SystemSpace)
		queryRepo.Create(ctx, "is:close is:issue user:arquillian author:aslakknutsen", "15 * * * * *", tracker1.ID, space.SystemSpace)

		tracker2, _ := trackerRepo.Create(ctx, "http://issues.jboss.com", ProviderJira)
		queryRepo.Create(ctx, "project = ARQ AND text ~ 'arquillian'", "15 * * * * *", tracker2.ID, space.SystemSpace)
		queryRepo.Create(ctx, "project = ARQ AND text ~ 'javadoc'", "15 * * * * *", tracker2.ID, space.SystemSpace)

		trackerqueries2, _ := queryRepo.List(ctx)
		assert.Equal(t, len(trackerqueries1)+4, len(trackerqueries2))
		trackerqueries3, _ := queryRepo.List(ctx)
		assert.Equal(t, trackerqueries2[1], trackerqueries3[1])
	})
}

func doWithTrackerRepositories(t *testing.T, todo func(trackerRepo application.TrackerRepository, queryRepo application.TrackerQueryRepository)) {
	doWithTransaction(t, func(db *gorm.DB) {
		defer cleaner.DeleteCreatedEntities(db)()
		trackerRepo := NewTrackerRepository(db)
		queryRepo := NewTrackerQueryRepository(db)
		todo(trackerRepo, queryRepo)
	})

}
