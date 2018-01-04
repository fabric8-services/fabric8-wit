package query_test

import (
	"context"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/query"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestQueryRepository struct {
	gormtestsupport.DBTestSuite
}

func TestRunQueryRepository(t *testing.T) {
	suite.Run(t, &TestQueryRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *TestQueryRepository) TestCreate() {
	resource.Require(s.T(), resource.Database)
	repo := query.NewQueryRepository(s.DB)
	s.T().Run("success", func(t *testing.T) {
		title := "My WI for sprint #101"
		qs := `{"hello": "world"}`
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))
		q := query.Query{
			Title:   title,
			Fields:  qs,
			SpaceID: fxt.Spaces[0].ID,
			Creator: fxt.Identities[0].ID,
		}
		// when
		err := repo.Create(context.Background(), &q)
		require.NoError(t, err)
		// then
		if q.ID == uuid.Nil {
			t.Errorf("Query was not created, ID nil")
		}
		if q.CreatedAt.After(time.Now()) {
			t.Errorf("Query was not created, CreatedAt after Now()?")
		}
		assert.Equal(t, title, q.Title)
		assert.Equal(t, qs, q.Fields)
	})

	s.T().Run("fail", func(t *testing.T) {
		t.Run("empty title", func(t *testing.T) {
			title := ""
			qs := `{"hello": "world"}`
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))
			q := query.Query{
				Title:   title,
				Fields:  qs,
				SpaceID: fxt.Spaces[0].ID,
			}
			// when
			err := repo.Create(context.Background(), &q)
			// then
			require.Error(t, err)
		})
		t.Run("invalid query json", func(t *testing.T) {
			title := "My WI for sprint #101"
			qs := "non-json query"
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))
			q := query.Query{
				Title:   title,
				Fields:  qs,
				SpaceID: fxt.Spaces[0].ID,
			}
			// when
			err := repo.Create(context.Background(), &q)
			// then
			require.Error(t, err)
		})
	})

}

func (s *TestQueryRepository) TestList() {
	resource.Require(s.T(), resource.Database)
	repo := query.NewQueryRepository(s.DB)
	s.T().Run("success", func(t *testing.T) {
		t.Run("by spaceID", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB,
				tf.Spaces(1), tf.Queries(3, tf.SetQueryTitles("q1", "q2", "q3")))
			// when
			qList, err := repo.List(context.Background(), fxt.Spaces[0].ID)
			// then
			require.NoError(t, err)
			mustHave := map[string]struct{}{
				"q1": {},
				"q2": {},
				"q3": {},
			}
			for _, q := range qList {
				delete(mustHave, q.Title)
			}
			assert.Empty(t, mustHave)
		})
		t.Run("by spaceID and creator", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB,
				tf.Spaces(1), tf.Queries(3, tf.SetQueryTitles("q1", "q2", "q3")))
			// when
			qList, err := repo.ListByCreator(context.Background(), fxt.Spaces[0].ID, fxt.Identities[0].ID)
			// then
			require.NoError(t, err)
			mustHave := map[string]struct{}{
				"q1": {},
				"q2": {},
				"q3": {},
			}
			for _, q := range qList {
				delete(mustHave, q.Title)
			}
			assert.Empty(t, mustHave)
		})

	})
}

func (s *TestQueryRepository) TestShow() {
	resource.Require(s.T(), resource.Database)
	repo := query.NewQueryRepository(s.DB)
	s.T().Run("success", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.Spaces(1), tf.Queries(1, tf.SetQueryTitles("q1")))
		// when
		q, err := repo.Load(context.Background(), fxt.QueryByTitle("q1").ID, fxt.Spaces[0].ID)
		// then
		require.NoError(t, err)
		assert.Equal(t, "q1", q.Title)
	})
	s.T().Run("fail", func(t *testing.T) {
		_, err := repo.Load(context.Background(), uuid.NewV4(), uuid.NewV4())
		require.Error(t, err)
	})
}
func (s *TestQueryRepository) TestDelete() {
	resource.Require(s.T(), resource.Database)
	repo := query.NewQueryRepository(s.DB)
	s.T().Run("success", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.Spaces(1), tf.Queries(1, tf.SetQueryTitles("q1")))
		// when
		err := repo.Delete(context.Background(), fxt.QueryByTitle("q1").ID)
		// then
		require.NoError(t, err)
	})
}

func (s *TestQueryRepository) TestDuplicate() {
	resource.Require(s.T(), resource.Database)
	repo := query.NewQueryRepository(s.DB)
	s.T().Run("true (exact same content)", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.Queries(1))
		q1 := fxt.Queries[0]
		q := query.Query{
			Title:   q1.Title,
			Fields:  q1.Fields,
			SpaceID: q1.SpaceID,
			Creator: q1.Creator,
		}
		// when
		dup, err := repo.IsDuplicate(context.Background(), &q)
		require.Error(t, err)
		require.True(t, dup)
	})
	s.T().Run("false", func(t *testing.T) {
		t.Run("different title", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.Queries(1))
			q1 := fxt.Queries[0]
			q := query.Query{
				Title:   q1.Title + " random title",
				Fields:  q1.Fields,
				SpaceID: q1.SpaceID,
				Creator: q1.Creator,
			}
			// when
			dup, err := repo.IsDuplicate(context.Background(), &q)
			require.NoError(t, err)
			require.False(t, dup)
		})
		t.Run("different creator", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.Queries(1), tf.Identities(2))
			q1 := fxt.Queries[0]
			q := query.Query{
				Title:   q1.Title,
				Fields:  q1.Fields,
				SpaceID: q1.SpaceID,
				Creator: fxt.Identities[1].ID,
			}
			// when
			dup, err := repo.IsDuplicate(context.Background(), &q)
			require.NoError(t, err)
			require.False(t, dup)
		})
		t.Run("same user but in different space", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.Queries(1), tf.Spaces(2))
			q1 := fxt.Queries[0]
			q := query.Query{
				Title:   q1.Title,
				Fields:  q1.Fields,
				SpaceID: fxt.Spaces[1].ID,
				Creator: q1.Creator,
			}
			// when
			dup, err := repo.IsDuplicate(context.Background(), &q)
			require.NoError(t, err)
			require.False(t, dup)
		})
	})
}
