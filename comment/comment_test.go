package comment_test

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/comment"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/resource"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/suite"
)

type TestCommentRepository struct {
	gormsupport.DBTestSuite

	clean func()
}

func TestRunCommentRepository(t *testing.T) {
	suite.Run(t, &TestCommentRepository{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

func (test *TestCommentRepository) SetupTest() {
	test.clean = gormsupport.DeleteCreatedEntities(test.DB)
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

	cl, err := repo.List(context.Background(), parentID)
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
