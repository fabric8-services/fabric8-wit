package models_test

import (
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/models"
	satoriuuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

func TestRunProjectRepoBBTest(t *testing.T) {
	suite.Run(t, &projectRepoBBTest{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

type projectRepoBBTest struct {
	gormsupport.DBTestSuite
	undoScript *gormsupport.DBScript
	repo       *models.GormProjectRepository
}

func (test *projectRepoBBTest) SetupTest() {
	test.undoScript = &gormsupport.DBScript{}
	test.repo = models.NewProjectRepository(test.DB)
}

func (test *projectRepoBBTest) TestSave() {
	version := 0
	name := "bla"
	p := app.ProjectData{
		ID: satoriuuid.NewV4().String(),
		Attributes: &app.ProjectAttributes{
			Version: &version,
			Name:    &name,
		},
	}
	_, err := test.repo.Save(context.Background(), p)
	if err == nil {
		test.repo.Delete(context.Background(), p.ID)
		test.T().Fatal("Save succeded for new project")
	} else {
		test.T().Logf("got expected error: %v", err)
	}
}
