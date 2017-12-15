package comment_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/comment"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestCommentRepository struct {
	gormtestsupport.DBTestSuite
	repo comment.Repository
}

func TestRunCommentRepository(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestCommentRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *TestCommentRepository) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = comment.NewRepository(s.DB)
}

func newComment(parentID uuid.UUID, body, markup string) *comment.Comment {
	return &comment.Comment{
		ParentID: parentID,
		Body:     body,
		Markup:   markup,
		Creator:  uuid.NewV4(),
	}
}

func (s *TestCommentRepository) createComment(c *comment.Comment, creator uuid.UUID) {
	err := s.repo.Create(s.Ctx, c, creator)
	require.NoError(s.T(), err)
}

func (s *TestCommentRepository) createComments(comments []*comment.Comment, creator uuid.UUID) {
	for _, c := range comments {
		s.createComment(c, creator)
	}
}

func (s *TestCommentRepository) TestCreateCommentWithMarkup() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Identities(1))
	comment := newComment(uuid.NewV4(), "Test A", rendering.SystemMarkupMarkdown)
	// when
	s.repo.Create(s.Ctx, comment, fxt.Identities[0].ID)
	// then
	assert.NotNil(s.T(), comment.ID, "Comment was not created, ID nil")
	require.NotNil(s.T(), comment.CreatedAt, "Comment was not created?")
	assert.False(s.T(), comment.CreatedAt.After(time.Now()), "Comment was not created, CreatedAt after Now()?")
}

func (s *TestCommentRepository) TestCreateCommentWithoutMarkup() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Identities(1))
	comment := newComment(uuid.NewV4(), "Test A", "")
	// when
	s.repo.Create(s.Ctx, comment, fxt.Identities[0].ID)
	// then
	assert.NotNil(s.T(), comment.ID, "Comment was not created, ID nil")
	require.NotNil(s.T(), comment.CreatedAt, "Comment was not created?")
	assert.False(s.T(), comment.CreatedAt.After(time.Now()), "CreatedAt after Now()?")
	assert.Equal(s.T(), rendering.SystemMarkupDefault, comment.Markup)
}

func (s *TestCommentRepository) TestSaveCommentWithMarkup() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Comments(1, func(fxt *tf.TestFixture, idx int) error {
		fxt.Comments[idx].Markup = rendering.SystemMarkupPlainText
		return nil
	}))
	comment := fxt.Comments[0]
	// when
	comment.Body = "Test AB"
	comment.Markup = rendering.SystemMarkupMarkdown
	s.repo.Save(s.Ctx, comment, comment.Creator)
	offset := 0
	limit := 1
	comments, _, err := s.repo.List(s.Ctx, comment.ParentID, &offset, &limit)
	// then
	require.NoError(s.T(), err)
	require.Len(s.T(), comments, 1)
	assert.Equal(s.T(), "Test AB", comments[0].Body)
	assert.Equal(s.T(), rendering.SystemMarkupMarkdown, comments[0].Markup)
}

func (s *TestCommentRepository) TestSaveCommentWithoutMarkup() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Comments(1, func(fxt *tf.TestFixture, idx int) error {
		fxt.Comments[idx].Markup = rendering.SystemMarkupMarkdown
		return nil
	}))
	comment := fxt.Comments[0]
	// when
	comment.Body = "Test AB"
	comment.Markup = ""
	s.repo.Save(s.Ctx, comment, comment.Creator)
	offset := 0
	limit := 1
	comments, _, err := s.repo.List(s.Ctx, comment.ParentID, &offset, &limit)
	// then
	require.NoError(s.T(), err)
	require.Len(s.T(), comments, 1)
	assert.Equal(s.T(), "Test AB", comments[0].Body)
	assert.Equal(s.T(), rendering.SystemMarkupPlainText, comments[0].Markup)
}

func (s *TestCommentRepository) TestDeleteComment() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Comments(1, func(fxt *tf.TestFixture, idx int) error {
		fxt.Comments[idx].Markup = rendering.SystemMarkupPlainText
		return nil
	}))
	c := fxt.Comments[0]
	// when
	err := s.repo.Delete(s.Ctx, c.ID, c.Creator)
	// then
	require.NoError(s.T(), err)
}

func (s *TestCommentRepository) TestCountComments() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(2), tf.Comments(2, func(fxt *tf.TestFixture, idx int) error {
		switch idx {
		case 0:
			fxt.Comments[idx].ParentID = fxt.WorkItems[0].ID
		case 1:
			fxt.Comments[idx].ParentID = fxt.WorkItems[1].ID
		}
		return nil
	}))
	// when
	count, err := s.repo.Count(s.Ctx, fxt.WorkItems[0].ID)
	// then
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 1, count)
}

func (s *TestCommentRepository) TestListComments() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Comments(2))
	// when
	offset := 0
	limit := 1
	resultComments, _, err := s.repo.List(s.Ctx, fxt.Comments[0].ParentID, &offset, &limit)
	// then
	require.NoError(s.T(), err)
	require.Len(s.T(), resultComments, 1)
	assert.Equal(s.T(), fxt.Comments[0].Body, resultComments[0].Body)
}

func (s *TestCommentRepository) TestListCommentsWrongOffset() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Comments(2))
	// when
	offset := -1
	limit := 1
	_, _, err := s.repo.List(s.Ctx, fxt.Comments[0].ParentID, &offset, &limit)
	// then
	require.Error(s.T(), err)
}

func (s *TestCommentRepository) TestListCommentsWrongLimit() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Comments(2))
	// when
	offset := 0
	limit := -1
	_, _, err := s.repo.List(s.Ctx, fxt.Comments[0].ParentID, &offset, &limit)
	// then
	require.Error(s.T(), err)
}

func (s *TestCommentRepository) TestLoadComment() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Comments(1))
	// when
	loadedComment, err := s.repo.Load(s.Ctx, fxt.Comments[0].ID)
	// then
	require.NoError(s.T(), err)
	assert.Equal(s.T(), fxt.Comments[0].ID, loadedComment.ID)
	assert.Equal(s.T(), fxt.Comments[0].Body, loadedComment.Body)
}

func (s *TestCommentRepository) TestExistsComment() {
	s.T().Run("comment exists", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.Comments(1))
		// when
		err := s.repo.CheckExists(s.Ctx, fxt.Comments[0].ID.String())
		// then
		require.NoError(t, err)
	})

	s.T().Run("comment doesn't exist", func(t *testing.T) {
		// when
		err := s.repo.CheckExists(s.Ctx, uuid.NewV4().String())
		// then
		require.IsType(t, errors.NotFoundError{}, err)
	})
}
