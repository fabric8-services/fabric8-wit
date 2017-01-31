package comment_test

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/comment"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/resource"
	uuid "github.com/satori/go.uuid"
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

func (test *TestCommentRepository) TestCreateComment() {
	t := test.T()
	resource.Require(t, resource.Database)

	repo := comment.NewCommentRepository(test.DB)

	c := &comment.Comment{
		ParentID:  "A",
		Body:      "Test A",
		CreatedBy: uuid.NewV4(),
	}

	repo.Create(context.Background(), c)
	if c.ID == uuid.Nil {
		t.Errorf("Comment was not created, ID nil")
	}

	if c.CreatedAt.After(time.Now()) {
		t.Errorf("Comment was not created, CreatedAt after Now()?")
	}
}

func (test *TestCommentRepository) TestSaveComment() {
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
	repo.Save(context.Background(), c)

	offset := 0
	limit := 1
	cl, _, err := repo.List(context.Background(), parentID, &offset, &limit)
	if err != nil {
		t.Error("Failed to List", err.Error())
	}

	if len(cl) != 1 {
		t.Error("List returned more then expected based on parentID")
	}

	c1 := cl[0]
	if c1.Body != "Test AB" {
		t.Error("List returned unexpected comment")
	}

}

func (test *TestCommentRepository) TestCountComments() {
	t := test.T()
	resource.Require(t, resource.Database)

	repo := comment.NewCommentRepository(test.DB)

	parentID := "A"
	body := "Test A"

	cs := []*comment.Comment{
		&comment.Comment{
			ParentID:  parentID,
			Body:      body,
			CreatedBy: uuid.NewV4(),
		},
		&comment.Comment{
			ParentID:  "B",
			Body:      "Test B",
			CreatedBy: uuid.NewV4(),
		},
	}

	for _, c := range cs {
		err := repo.Create(context.Background(), c)
		if err != nil {
			t.Error("Failed to Create", err.Error())
		}

	}

	count, err := repo.Count(context.Background(), parentID)
	if err != nil {
		t.Error("Failed to Count", err.Error())
	}

	if count != 1 {
		t.Error("expected count is 1 but got:", count)
	}
}

func (test *TestCommentRepository) TestListComments() {
	t := test.T()
	resource.Require(t, resource.Database)

	repo := comment.NewCommentRepository(test.DB)

	parentID := "A"
	body := "Test A"

	cs := []*comment.Comment{
		&comment.Comment{
			ParentID:  parentID,
			Body:      body,
			CreatedBy: uuid.NewV4(),
		},
		&comment.Comment{
			ParentID:  "B",
			Body:      "Test B",
			CreatedBy: uuid.NewV4(),
		},
	}

	for _, c := range cs {
		repo.Create(context.Background(), c)
	}

	offset := 0
	limit := 1
	cl, _, err := repo.List(context.Background(), parentID, &offset, &limit)
	if err != nil {
		t.Error("Failed to List", err.Error())
	}

	if len(cl) != 1 {
		t.Error("List returned more then expected based on parentID")
	}

	c := cl[0]
	if c.Body != body {
		t.Error("List returned unexpected comment")
	}
}

func (test *TestCommentRepository) TestListCommentsWrongOffset() {
	t := test.T()
	resource.Require(t, resource.Database)

	repo := comment.NewCommentRepository(test.DB)

	parentID := "A"
	body := "Test A"

	cs := []*comment.Comment{
		&comment.Comment{
			ParentID:  parentID,
			Body:      body,
			CreatedBy: uuid.NewV4(),
		},
		&comment.Comment{
			ParentID:  "B",
			Body:      "Test B",
			CreatedBy: uuid.NewV4(),
		},
	}

	for _, c := range cs {
		repo.Create(context.Background(), c)
	}

	offset := -1
	limit := 1
	_, _, err := repo.List(context.Background(), parentID, &offset, &limit)
	if err == nil {
		t.Error("Expected an error to List")
	}

}

func (test *TestCommentRepository) TestListCommentsWrongLimit() {
	t := test.T()
	resource.Require(t, resource.Database)

	repo := comment.NewCommentRepository(test.DB)

	parentID := "A"
	body := "Test A"

	cs := []*comment.Comment{
		&comment.Comment{
			ParentID:  parentID,
			Body:      body,
			CreatedBy: uuid.NewV4(),
		},
		&comment.Comment{
			ParentID:  "B",
			Body:      "Test B",
			CreatedBy: uuid.NewV4(),
		},
	}

	for _, c := range cs {
		repo.Create(context.Background(), c)
	}

	offset := 0
	limit := -1
	_, _, err := repo.List(context.Background(), parentID, &offset, &limit)
	if err == nil {
		t.Error("Expected an error to List")
	}

}

func (test *TestCommentRepository) TestLoadComment() {
	t := test.T()
	resource.Require(t, resource.Database)

	repo := comment.NewCommentRepository(test.DB)

	c := &comment.Comment{
		ParentID:  "A",
		Body:      "Test A",
		CreatedBy: uuid.NewV4(),
	}

	repo.Create(context.Background(), c)

	l, err := repo.Load(context.Background(), c.ID)
	if err != nil {
		t.Error("Error loading comment")
	}
	if l.ID != c.ID {
		t.Errorf("Loaded comment was not same as requested")
	}
	if l.Body != c.Body {
		t.Error("Loaded comment has different body")
	}
}
