package controller_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem/link"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestSuiteWorkItemLinkCategory(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemLinkCategorySuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

type workItemLinkCategorySuite struct {
	gormtestsupport.DBTestSuite
	linkCatCtrl *WorkItemLinkCategoryController
	svc         *goa.Service
	testDir     string
}

func (s *workItemLinkCategorySuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.svc = testsupport.ServiceAsUser("workItemLinkSpace-Service", testsupport.TestIdentity)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(s.svc, s.GormDB)
	s.testDir = filepath.Join("test-files", "work_item_link_category")
}

func createWorkItemLinkCategoryInRepo(t *testing.T, db application.DB, ctx context.Context, linkCat link.WorkItemLinkCategory) uuid.UUID {
	err := application.Transactional(db, func(appl application.Application) error {
		_, err := appl.WorkItemLinkCategories().Create(ctx, &linkCat)
		return err
	})
	require.NoError(t, err)
	return linkCat.ID
}

func (s *workItemLinkCategorySuite) TestShow() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinkCategories(1))
		// when
		res, cat := test.ShowWorkItemLinkCategoryOK(t, nil, nil, s.linkCatCtrl, fxt.WorkItemLinkCategories[0].ID)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.res.payload.golden.json"), cat)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.res.headers.golden.json"), res.Header())
	})
	s.T().Run("not found", func(t *testing.T) {
		// given
		id := uuid.NewV4()
		// when
		res, jerrs := test.ShowWorkItemLinkCategoryNotFound(t, nil, nil, s.linkCatCtrl, id)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_found.res.payload.golden.json"), jerrs)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_found.res.headers.golden.json"), res.Header())
	})
}

func (s *workItemLinkCategorySuite) TestList() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinkCategories(2))
		// when
		res, cats := test.ListWorkItemLinkCategoryOK(s.T(), nil, nil, s.linkCatCtrl)
		// then
		toBeFound := id.Slice{fxt.WorkItemLinkCategories[0].ID, fxt.WorkItemLinkCategories[1].ID}.ToMap()
		for _, cat := range cats.Data {
			delete(toBeFound, *cat.ID)
		}
		require.Empty(t, toBeFound, "failed to find these expected work item link categories: %v", toBeFound)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok.res.payload.golden.json"), cats)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok.res.headers.golden.json"), res.Header())
	})
}
