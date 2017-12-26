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

func (s *TestQueryRepository) TestCreateQuery() {
	t := s.T()
	resource.Require(t, resource.Database)
	repo := query.NewQueryRepository(s.DB)
	t.Run("success", func(t *testing.T) {
		title := "My WI for sprin #101"
		qs := `{"hello": "worold"}`
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))
		q := query.Query{
			Title:   title,
			Fields:  qs,
			SpaceID: fxt.Spaces[0].ID,
			Creator: fxt.Identities[0].ID,
		}
		// when
		err := repo.Create(context.Background(), &q)
		require.Nil(t, err)
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

	t.Run("fail", func(t *testing.T) {
		t.Run("empty title", func(t *testing.T) {
			title := ""
			qs := `{"hello": "worold"}`
			// given
			fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))
			q := query.Query{
				Title:   title,
				Fields:  qs,
				SpaceID: fxt.Spaces[0].ID,
			}
			// when
			err := repo.Create(context.Background(), &q)
			// then
			require.NotNil(t, err)
		})
		t.Run("invalid query json", func(t *testing.T) {
			title := "My WI for sprin #101"
			qs := "non-json query"
			// given
			fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))
			q := query.Query{
				Title:   title,
				Fields:  qs,
				SpaceID: fxt.Spaces[0].ID,
			}
			// when
			err := repo.Create(context.Background(), &q)
			// then
			require.NotNil(t, err)
		})
	})

}

func (s *TestQueryRepository) TestListQuery() {
	t := s.T()
	resource.Require(t, resource.Database)
	repo := query.NewQueryRepository(s.DB)
	t.Run("success", func(t *testing.T) {
		t.Run("by spaceID", func(t *testing.T) {

			// given
			fxt := tf.NewTestFixture(s.T(), s.DB,
				tf.Spaces(1), tf.Queries(3, tf.SetQueryTitles("q1", "q2", "q3")))
			// when
			qList, err := repo.List(context.Background(), fxt.Spaces[0].ID)
			// then
			require.Nil(t, err)
			mustHave := map[string]struct{}{
				"q1": {},
				"q2": {},
				"q3": {},
			}
			for _, q := range qList {
				delete(mustHave, q.Title)
			}
			assert.Empty(s.T(), mustHave)
		})
		t.Run("by spaceID and creator", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(s.T(), s.DB,
				tf.Spaces(1), tf.Queries(3, tf.SetQueryTitles("q1", "q2", "q3")))
			// when
			qList, err := repo.ListByCreator(context.Background(), fxt.Spaces[0].ID, fxt.Identities[0].ID)
			// then
			require.Nil(t, err)
			mustHave := map[string]struct{}{
				"q1": {},
				"q2": {},
				"q3": {},
			}
			for _, q := range qList {
				delete(mustHave, q.Title)
			}
			assert.Empty(s.T(), mustHave)
		})

	})
}

func (s *TestQueryRepository) TestShowQuery() {
	t := s.T()
	resource.Require(t, resource.Database)
	repo := query.NewQueryRepository(s.DB)
	t.Run("success", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB,
			tf.Spaces(1), tf.Queries(1, tf.SetQueryTitles("q1")))
		// when
		q, err := repo.Load(context.Background(), fxt.QueryByTitle("q1").ID)
		// then
		require.Nil(t, err)
		assert.Equal(s.T(), "q1", q.Title)
	})
	t.Run("fail", func(t *testing.T) {
		_, err := repo.Load(context.Background(), uuid.NewV4())
		require.NotNil(t, err)
	})
}
func (s *TestQueryRepository) TestDeleteQuery() {
	t := s.T()
	resource.Require(t, resource.Database)
	repo := query.NewQueryRepository(s.DB)
	t.Run("success", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB,
			tf.Spaces(1), tf.Queries(1, tf.SetQueryTitles("q1")))
		// when
		err := repo.Delete(context.Background(), fxt.QueryByTitle("q1").ID)
		// then
		require.Nil(t, err)
	})
}
