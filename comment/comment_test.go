package comment_test

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/comment"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestCommentRepository struct {
	gormsupport.DBTestSuite

	clean func()
}

func TestRunCommentRepository(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestCommentRepository{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

func (test *TestCommentRepository) SetupTest() {
	test.clean = cleaner.DeleteCreatedEntities(test.DB)
}

func (test *TestCommentRepository) TearDownTest() {
	test.clean()
}

func newComment(parentID, body, markup string) *comment.Comment {
	return &comment.Comment{
		ParentID:  parentID,
		Body:      body,
		Markup:    markup,
		CreatedBy: uuid.NewV4(),
	}
}

func (test *TestCommentRepository) createComment(c *comment.Comment) {
	repo := comment.NewCommentRepository(test.DB)
	err := repo.Create(context.Background(), c)
	require.Nil(test.T(), err)
}

func (test *TestCommentRepository) createComments(comments []*comment.Comment) {
	repo := comment.NewCommentRepository(test.DB)
	for _, comment := range comments {
		err := repo.Create(context.Background(), comment)
		require.Nil(test.T(), err)
	}
}

func (test *TestCommentRepository) TestCreateCommentWithMarkup() {
	// given
	repo := comment.NewCommentRepository(test.DB)
	comment := newComment("A", "Test A", rendering.SystemMarkupMarkdown)
	// when
	repo.Create(context.Background(), comment)
	// then
	assert.NotNil(test.T(), comment.ID, "Comment was not created, ID nil")
	require.NotNil(test.T(), comment.CreatedAt, "Comment was not created?")
	assert.False(test.T(), comment.CreatedAt.After(time.Now()), "Comment was not created, CreatedAt after Now()?")
}

func (test *TestCommentRepository) TestCreateCommentWithoutMarkup() {
	// given
	repo := comment.NewCommentRepository(test.DB)
	comment := newComment("A", "Test A", "")
	// when
	repo.Create(context.Background(), comment)
	// then
	assert.NotNil(test.T(), comment.ID, "Comment was not created, ID nil")
	require.NotNil(test.T(), comment.CreatedAt, "Comment was not created?")
	assert.False(test.T(), comment.CreatedAt.After(time.Now()), "CreatedAt after Now()?")
	assert.Equal(test.T(), rendering.SystemMarkupDefault, comment.Markup)
}

func (test *TestCommentRepository) TestSaveCommentWithMarkup() {
	// given
	repo := comment.NewCommentRepository(test.DB)
	comment := newComment("A", "Test A", rendering.SystemMarkupPlainText)
	test.createComment(comment)
	assert.NotNil(test.T(), comment.ID, "Comment was not created, ID nil")
	// when
	comment.Body = "Test AB"
	comment.Markup = rendering.SystemMarkupMarkdown
	repo.Save(context.Background(), comment)
	offset := 0
	limit := 1
	comments, _, err := repo.List(context.Background(), comment.ParentID, &offset, &limit)
	// then
	require.Nil(test.T(), err)
	require.Equal(test.T(), 1, len(comments), "List returned more then expected based on parentID")
	assert.Equal(test.T(), "Test AB", comments[0].Body)
	assert.Equal(test.T(), rendering.SystemMarkupMarkdown, comments[0].Markup)
}

func (test *TestCommentRepository) TestSaveCommentWithoutMarkup() {
	// given
	repo := comment.NewCommentRepository(test.DB)
	comment := newComment("A", "Test A", rendering.SystemMarkupMarkdown)
	test.createComment(comment)
	assert.NotNil(test.T(), comment.ID, "Comment was not created, ID nil")
	// when
	comment.Body = "Test AB"
	comment.Markup = ""
	repo.Save(context.Background(), comment)
	offset := 0
	limit := 1
	comments, _, err := repo.List(context.Background(), comment.ParentID, &offset, &limit)
	// then
	require.Nil(test.T(), err)
	require.Equal(test.T(), 1, len(comments), "List returned more then expected based on parentID")
	assert.Equal(test.T(), "Test AB", comments[0].Body)
	assert.Equal(test.T(), rendering.SystemMarkupPlainText, comments[0].Markup)
}

func (test *TestCommentRepository) TestDeleteComment() {
	defer cleaner.DeleteCreatedEntities(test.DB)()

	t := test.T()
	resource.Require(t, resource.Database)

	repo := comment.NewCommentRepository(test.DB)

	parentID := "AA"
	c := &comment.Comment{
		ParentID:  parentID,
		Body:      "Test AA",
		CreatedBy: uuid.NewV4(),
	}

	repo.Create(context.Background(), c)
	if c.ID == uuid.Nil {
		t.Errorf("Comment was not created, ID nil")
	}

	c.Body = "Test AB"
	err := repo.Delete(context.Background(), c)

	if err != nil {
		t.Error("Failed to Delete", err.Error())
	}

}

func (test *TestCommentRepository) TestCountComments() {
	// given
	repo := comment.NewCommentRepository(test.DB)
	parentID := "A"
	comment1 := newComment("A", "Test A", rendering.SystemMarkupMarkdown)
	comment2 := newComment("B", "Test B", rendering.SystemMarkupMarkdown)
	comments := []*comment.Comment{comment1, comment2}
	test.createComments(comments)
	// when
	count, err := repo.Count(context.Background(), parentID)
	// then
	require.Nil(test.T(), err)
	assert.Equal(test.T(), 1, count)
}

func (test *TestCommentRepository) TestListComments() {
	// given
	repo := comment.NewCommentRepository(test.DB)
	comment1 := newComment("A", "Test A", rendering.SystemMarkupMarkdown)
	comment2 := newComment("B", "Test B", rendering.SystemMarkupMarkdown)
	comments := []*comment.Comment{comment1, comment2}
	test.createComments(comments)
	// when
	offset := 0
	limit := 1
	comments, _, err := repo.List(context.Background(), comment1.ParentID, &offset, &limit)
	// then
	require.Nil(test.T(), err)
	require.Equal(test.T(), 1, len(comments))
	assert.Equal(test.T(), comment1.Body, comments[0].Body)
}

func (test *TestCommentRepository) TestListCommentsWrongOffset() {
	// given
	repo := comment.NewCommentRepository(test.DB)
	comment1 := newComment("A", "Test A", rendering.SystemMarkupMarkdown)
	comment2 := newComment("B", "Test B", rendering.SystemMarkupMarkdown)
	comments := []*comment.Comment{comment1, comment2}
	test.createComments(comments)
	// when
	offset := -1
	limit := 1
	_, _, err := repo.List(context.Background(), comment1.ParentID, &offset, &limit)
	// then
	assert.NotNil(test.T(), err)
}

func (test *TestCommentRepository) TestListCommentsWrongLimit() {
	// given
	repo := comment.NewCommentRepository(test.DB)
	comment1 := newComment("A", "Test A", rendering.SystemMarkupMarkdown)
	comment2 := newComment("B", "Test B", rendering.SystemMarkupMarkdown)
	comments := []*comment.Comment{comment1, comment2}
	test.createComments(comments)
	// when
	offset := 0
	limit := -1
	_, _, err := repo.List(context.Background(), comment1.ParentID, &offset, &limit)
	// then
	assert.NotNil(test.T(), err)
}

func (test *TestCommentRepository) TestLoadComment() {
	// given
	repo := comment.NewCommentRepository(test.DB)
	comment := newComment("A", "Test A", rendering.SystemMarkupMarkdown)
	test.createComment(comment)
	// when
	loadedComment, err := repo.Load(context.Background(), comment.ID)
	require.Nil(test.T(), err)
	assert.Equal(test.T(), comment.ID, loadedComment.ID)
	assert.Equal(test.T(), comment.Body, loadedComment.Body)
}
