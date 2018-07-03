package controller_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/fabric8-services/fabric8-wit/spacetemplate"

	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/suite"
)

type workItemTypeGroupSuite struct {
	gormtestsupport.DBTestSuite
	svc            *goa.Service
	typeGroupCtrl  *WorkItemTypeGroupController
	typeGroupsCtrl *WorkItemTypeGroupsController
	testDir        string
}

func TestWorkItemTypeGroupSuite(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemTypeGroupSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

// The SetupTest method will be run before every test in the suite.
func (s *workItemTypeGroupSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.svc = testsupport.ServiceAsUser("WITG-Service", testsupport.TestIdentity)
	s.typeGroupCtrl = NewWorkItemTypeGroupController(s.svc, s.GormDB)
	s.typeGroupsCtrl = NewWorkItemTypeGroupsController(s.svc, s.GormDB)
	s.testDir = filepath.Join("test-files", "work_item_type_group")
}

func (s *workItemTypeGroupSuite) TestList() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypeGroups(3))
		testData := map[string]uuid.UUID{
			"generated_template": fxt.SpaceTemplates[0].ID,
			"base_template":      spacetemplate.SystemBaseTemplateID,
			"legacy_template":    spacetemplate.SystemLegacyTemplateID,
			"scrum_template":     spacetemplate.SystemScrumTemplateID,
		}
		// when
		for name, spaceTemplateID := range testData {
			t.Run(name, func(t *testing.T) {
				res, groups := test.ListWorkItemTypeGroupsOK(t, nil, s.svc, s.typeGroupsCtrl, spaceTemplateID)
				// then
				if spaceTemplateID == fxt.SpaceTemplates[0].ID {
					compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", fmt.Sprintf("ok_%s.payload.golden.json", name)), groups)
					compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", fmt.Sprintf("ok_%s.headers.golden.json", name)), res.Header())
				} else {
					compareWithGoldenAgnosticTime(t, filepath.Join(s.testDir, "list", fmt.Sprintf("ok_%s.payload.golden.json", name)), groups)
					compareWithGoldenAgnosticTime(t, filepath.Join(s.testDir, "list", fmt.Sprintf("ok_%s.headers.golden.json", name)), res.Header())
				}
			})
		}
	})
	s.T().Run("not found", func(t *testing.T) {
		// given
		sapcetemplateID := uuid.NewV4()
		// when
		res, jerrs := test.ListWorkItemTypeGroupsNotFound(t, nil, s.svc, s.typeGroupsCtrl, sapcetemplateID)
		// then
		ignoreMe := "IGNOREME"
		jerrs.Errors[0].ID = &ignoreMe
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_found.errors.golden.json"), jerrs)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_found.headers.golden.json"), res.Header())
	})
}

func (s *workItemTypeGroupSuite) TestShow() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypeGroups(1))
		// when
		res, group := test.ShowWorkItemTypeGroupOK(t, nil, s.svc, s.typeGroupCtrl, fxt.WorkItemTypeGroups[0].ID)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.witg.golden.json"), group)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.headers.golden.json"), res.Header())
	})
	s.T().Run("not found", func(t *testing.T) {
		// given
		typeGroupID := uuid.NewV4()
		// when
		res, jerrs := test.ShowWorkItemTypeGroupNotFound(t, nil, s.svc, s.typeGroupCtrl, typeGroupID)
		// then
		ignoreMe := "IGNOREME"
		jerrs.Errors[0].ID = &ignoreMe
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_found.errors.golden.json"), jerrs)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_found.headers.golden.json"), res.Header())
	})
}
