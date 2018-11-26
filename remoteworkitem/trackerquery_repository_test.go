package remoteworkitem_test

import (
	"net/http"
	"net/url"
	"testing"

	"context"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	uuid "github.com/satori/go.uuid"

	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestTrackerQueryRepository struct {
	gormtestsupport.DBTestSuite

	trackerRepo remoteworkitem.TrackerRepository
	queryRepo   remoteworkitem.TrackerQueryRepository
}

func TestRunTrackerQueryRepository(t *testing.T) {
	suite.Run(t, &TestTrackerQueryRepository{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (test *TestTrackerQueryRepository) SetupTest() {
	test.DBTestSuite.SetupTest()
	test.trackerRepo = remoteworkitem.NewTrackerRepository(test.DB)
	test.queryRepo = remoteworkitem.NewTrackerQueryRepository(test.DB)
}

func (test *TestTrackerQueryRepository) TestTrackerQueryCreate() {
	t := test.T()
	resource.Require(t, resource.Database)

	t.Run("tracker query create - fail", func(t *testing.T) {
		req := &http.Request{Host: "localhost"}
		params := url.Values{}
		ctx := goa.NewContext(context.Background(), nil, req, params)

		testFxt := tf.NewTestFixture(t, test.DB, tf.Spaces(1))

		tq := remoteworkitem.TrackerQuery{
			Query:     "abc",
			Schedule:  "xyz",
			TrackerID: uuid.NewV4(),
			SpaceID:   testFxt.Spaces[0].ID,
		}
		err := test.queryRepo.Create(ctx, &tq)
		require.Error(t, err)
		assert.IsType(t, errors.InternalError{}, err)
	})

	t.Run("tracker query create - success", func(t *testing.T) {
		req := &http.Request{Host: "localhost"}
		params := url.Values{}
		ctx := goa.NewContext(context.Background(), nil, req, params)

		tracker := remoteworkitem.Tracker{
			URL:  "http://issues.jboss.com",
			Type: remoteworkitem.ProviderJira,
		}
		err := test.trackerRepo.Create(ctx, &tracker)
		testFxt := tf.NewTestFixture(t, test.DB, tf.Spaces(1))

		tq := remoteworkitem.TrackerQuery{
			Query:     "abc",
			Schedule:  "xyz",
			TrackerID: tracker.ID,
			SpaceID:   testFxt.Spaces[0].ID,
		}
		err = test.queryRepo.Create(ctx, &tq)
		require.NoError(t, err)

		tq2, err := test.queryRepo.Load(ctx, tq.ID)
		require.NoError(t, err)
		assert.Equal(t, tq.ID, tq2.ID)
	})

}

func (test *TestTrackerQueryRepository) TestExistsTrackerQuery() {
	t := test.T()
	resource.Require(t, resource.Database)

	t.Run("tracker query exists", func(t *testing.T) {
		// given
		req := &http.Request{Host: "localhost"}
		params := url.Values{}
		ctx := goa.NewContext(context.Background(), nil, req, params)
		testFxt := tf.NewTestFixture(t, test.DB, tf.Spaces(1))

		tracker := remoteworkitem.Tracker{
			URL:  "http://issues.jboss.com",
			Type: remoteworkitem.ProviderJira,
		}
		err := test.trackerRepo.Create(ctx, &tracker)
		require.NoError(t, err)

		query := remoteworkitem.TrackerQuery{
			Query:     "abc",
			Schedule:  "xyz",
			TrackerID: tracker.ID,
			SpaceID:   testFxt.Spaces[0].ID,
		}
		err = test.queryRepo.Create(ctx, &query)
		require.NoError(t, err)

		err = test.queryRepo.CheckExists(ctx, query.ID)
		require.NoError(t, err)
	})

	t.Run("tracker query doesn't exist", func(t *testing.T) {
		req := &http.Request{Host: "localhost"}
		params := url.Values{}
		ctx := goa.NewContext(context.Background(), nil, req, params)

		err := test.queryRepo.CheckExists(ctx, uuid.NewV4())
		require.Error(t, err)
		require.IsType(t, errors.NotFoundError{}, err)
	})

}

func (test *TestTrackerQueryRepository) TestTrackerQuerySave() {
	t := test.T()
	resource.Require(t, resource.Database)
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx := goa.NewContext(context.Background(), nil, req, params)
	testFxt := tf.NewTestFixture(t, test.DB, tf.Spaces(1))

	query, err := test.queryRepo.Load(ctx, uuid.NewV4())
	require.Nil(t, query)
	require.Error(t, err)
	assert.IsType(t, errors.NotFoundError{}, err)

	tracker := remoteworkitem.Tracker{
		URL:  "http://issues.jboss.com",
		Type: remoteworkitem.ProviderJira,
	}
	err = test.trackerRepo.Create(ctx, &tracker)
	tracker2 := remoteworkitem.Tracker{
		URL:  "http://api.github.com",
		Type: remoteworkitem.ProviderGithub,
	}
	err = test.trackerRepo.Create(ctx, &tracker2)

	query1 := remoteworkitem.TrackerQuery{
		Query:     "abc",
		Schedule:  "xyz",
		TrackerID: tracker.ID,
		SpaceID:   testFxt.Spaces[0].ID,
	}
	err = test.queryRepo.Create(ctx, &query1)
	query2, err := test.queryRepo.Load(ctx, query1.ID)
	require.NoError(t, err)
	assert.Equal(t, query1.ID, query2.ID)

	query1.Query = "def"
	query1.Schedule = "rwd"
	query3, err := test.queryRepo.Save(ctx, query1)
	require.NoError(t, err)
	assert.Equal(t, query1.ID, query3.ID)

	err = test.trackerRepo.Delete(ctx, uuid.NewV4())
	assert.NotNil(t, err)

	query1.TrackerID = uuid.NewV4()
	query4, err := test.queryRepo.Save(ctx, query1)
	require.Nil(t, query4)
	require.Error(t, err)
	assert.IsType(t, errors.NotFoundError{}, err)
}

func (test *TestTrackerQueryRepository) TestTrackerQueryDelete() {
	t := test.T()
	resource.Require(t, resource.Database)

	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx := goa.NewContext(context.Background(), nil, req, params)
	testFxt := tf.NewTestFixture(t, test.DB, tf.Spaces(1))

	err := test.queryRepo.Delete(ctx, uuid.NewV4())
	require.Error(t, err)
	assert.IsType(t, errors.NotFoundError{}, err)

	tracker := remoteworkitem.Tracker{
		URL:  "http://api.github.com",
		Type: remoteworkitem.ProviderGithub,
	}
	err = test.trackerRepo.Create(ctx, &tracker)
	tq := &remoteworkitem.TrackerQuery{
		Query:     "is:open is:issue user:arquillian author:aslakknutsen",
		Schedule:  "15 * * * * *",
		TrackerID: tracker.ID,
		SpaceID:   testFxt.Spaces[0].ID,
	}
	err = test.queryRepo.Create(ctx, tq)
	require.NoError(t, err)
	err = test.queryRepo.Delete(ctx, tq.ID)
	require.NoError(t, err)

	tq, err = test.queryRepo.Load(ctx, tq.ID)
	require.Error(t, err)
	assert.IsType(t, errors.NotFoundError{}, err)

	tq, err = test.queryRepo.Load(ctx, uuid.NewV4())
	require.Error(t, err)
	assert.IsType(t, errors.NotFoundError{}, err)
}

func (test *TestTrackerQueryRepository) TestTrackerQueryList() {
	t := test.T()
	resource.Require(t, resource.Database)

	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx := goa.NewContext(context.Background(), nil, req, params)
	testFxt := tf.NewTestFixture(t, test.DB, tf.Spaces(1))

	trackerqueries1, _ := test.queryRepo.List(ctx)

	// create tracker
	tracker1 := remoteworkitem.Tracker{
		URL:  "http://api.github.com",
		Type: remoteworkitem.ProviderGithub,
	}
	err := test.trackerRepo.Create(ctx, &tracker1)
	require.NoError(t, err)

	// create tracker queries
	tq1 := remoteworkitem.TrackerQuery{
		Query:     "is:open is:issue user:arquillian author:aslakknutsen",
		Schedule:  "15 * * * * *",
		TrackerID: tracker1.ID,
		SpaceID:   testFxt.Spaces[0].ID,
	}
	err = test.queryRepo.Create(ctx, &tq1)
	require.NoError(t, err)

	tq2 := remoteworkitem.TrackerQuery{
		Query:     "is:open is:issue user:arquillian",
		Schedule:  "15 * * * * *",
		TrackerID: tracker1.ID,
		SpaceID:   testFxt.Spaces[0].ID,
	}
	err = test.queryRepo.Create(ctx, &tq2)
	require.NoError(t, err)

	tracker2 := remoteworkitem.Tracker{
		URL:  "http://issues.jboss.com",
		Type: remoteworkitem.ProviderJira,
	}
	err = test.trackerRepo.Create(ctx, &tracker2)
	require.NoError(t, err)

	tq3 := remoteworkitem.TrackerQuery{
		Query:     "project = ARQ AND text ~ 'arquillian'",
		Schedule:  "15 * * * * *",
		TrackerID: tracker2.ID,
		SpaceID:   testFxt.Spaces[0].ID,
	}
	err = test.queryRepo.Create(ctx, &tq3)
	require.NoError(t, err)

	tq4 := remoteworkitem.TrackerQuery{
		Query:     "project = ARQ AND text ~ 'javadoc'",
		Schedule:  "15 * * * * *",
		TrackerID: tracker2.ID,
		SpaceID:   testFxt.Spaces[0].ID,
	}
	err = test.queryRepo.Create(ctx, &tq4)
	require.NoError(t, err)

	trackerqueries2, _ := test.queryRepo.List(ctx)
	assert.Equal(t, len(trackerqueries1)+4, len(trackerqueries2))
	trackerqueries3, _ := test.queryRepo.List(ctx)
	require.True(t, len(trackerqueries3) >= 2)
	require.True(t, len(trackerqueries2) >= 2)
	assert.Equal(t, trackerqueries2[1], trackerqueries3[1])
}
