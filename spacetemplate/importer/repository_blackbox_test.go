package importer_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	"github.com/fabric8-services/fabric8-wit/spacetemplate/importer"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	uuid "github.com/satori/go.uuid"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type repoSuite struct {
	gormtestsupport.DBTestSuite
	spaceTemplateRepo spacetemplate.Repository
	importerRepo      importer.Repository
	witRepo           workitem.WorkItemTypeRepository
	wiltRepo          link.WorkItemLinkTypeRepository
	witgRepo          workitem.WorkItemTypeGroupRepository
}

func TestRepository(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &repoSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *repoSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()

	s.spaceTemplateRepo = spacetemplate.NewRepository(s.DB)
	s.importerRepo = importer.NewRepository(s.DB)
	s.witRepo = workitem.NewWorkItemTypeRepository(s.DB)
	s.wiltRepo = link.NewWorkItemLinkTypeRepository(s.DB)
	s.witgRepo = workitem.NewWorkItemTypeGroupRepository(s.DB)
}

func diff(expectedStr, actualStr string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(expectedStr, actualStr, false)
	return dmp.DiffPrettyText(diffs)
}

func (s *repoSuite) TestImport() {
	// given
	spaceTemplateID := uuid.NewV4()
	witID := uuid.NewV4()
	wiltID := uuid.NewV4()
	witgID := uuid.NewV4()

	s.T().Run("valid", func(t *testing.T) {
		t.Run("test template", func(t *testing.T) {
			// when
			expected := getValidTestTemplateParsed(t, spaceTemplateID, witID, wiltID, witgID)
			_, err := s.importerRepo.Import(s.Ctx, expected)
			// then
			require.NoError(t, err)
			err = s.spaceTemplateRepo.CheckExists(s.Ctx, expected.Template.ID)
			assert.NoError(t, err)
			err = s.witRepo.CheckExists(s.Ctx, witID)
			assert.NoError(t, err)
			err = s.wiltRepo.CheckExists(s.Ctx, wiltID)
			assert.NoError(t, err)
			err = s.witgRepo.CheckExists(s.Ctx, witgID)
			assert.NoError(t, err)

			wit, err := s.witRepo.Load(s.Ctx, witID)
			require.NoError(t, err)
			t.Run("child type IDs set", func(t *testing.T) {
				assert.Equal(t, []uuid.UUID{workitem.SystemPlannerItem}, wit.ChildTypeIDs)
			})

			// Check that the work item type "bug" correctly extends the planner item type
			t.Run("WIT extends correctly", func(t *testing.T) {
				toBeFound := map[string]struct{}{
					"title":    {},
					"state":    {},
					"priority": {},
					// the following fields must be created because the WIT
					// extends the "planner item type".
					workitem.SystemArea:         {},
					workitem.SystemOrder:        {},
					workitem.SystemState:        {},
					workitem.SystemTitle:        {},
					workitem.SystemCreator:      {},
					workitem.SystemCodebase:     {},
					workitem.SystemAssignees:    {},
					workitem.SystemIteration:    {},
					workitem.SystemCreatedAt:    {},
					workitem.SystemUpdatedAt:    {},
					workitem.SystemDescription:  {},
					workitem.SystemRemoteItemID: {},
				}
				for field := range toBeFound {
					obj, ok := wit.Fields[field]
					require.True(t, ok, "field %s doesn't exist within work item type %s", field, witID)
					require.NotNil(t, obj)
				}
			})
		})
		t.Run("import existing template with changes", func(t *testing.T) {
			// Create fresh template
			spaceTemplateID := uuid.NewV4()
			witID := uuid.NewV4()
			wiltID := uuid.NewV4()
			witgID := uuid.NewV4()
			oldTempl := getValidTestTemplateParsed(t, spaceTemplateID, witID, wiltID, witgID)
			oldTempl.Template.Name = "old name for space template " + spaceTemplateID.String()
			oldTempl.Template.CanConstruct = true
			_, err := s.importerRepo.Import(s.Ctx, oldTempl)
			require.NoError(t, err)
			// Import it once more but this time with changes
			templ := getValidTestTemplateParsed(t, spaceTemplateID, witID, wiltID, witgID)
			templ.Template.Name = "new name for space template " + spaceTemplateID.String()
			templ.Template.CanConstruct = false
			templ.Template.Description = ptr.String("new description")
			templ.WITs[0].CanConstruct = !templ.WITs[0].CanConstruct
			templ.WITs[0].Name = "new name for WIT " + templ.WITs[0].ID.String()
			templ.WILTs[0].Name = "new name for WILT " + templ.WILTs[0].ID.String()
			templ.WITs[0].Fields["flavor"] = workitem.FieldDefinition{
				Required:    true,
				Label:       "Flavor",
				Description: "Put it in your soup",
				Type: workitem.SimpleType{
					Kind: workitem.KindString,
				},
			}
			templ.WITGs[0].Name = "Helmet"
			// when
			_, err = s.importerRepo.Import(s.Ctx, templ)
			// then
			require.NoError(t, err)
			t.Run("template name, description and can_construct has changed", func(t *testing.T) {
				st, err := s.spaceTemplateRepo.Load(s.Ctx, templ.Template.ID)
				require.NoError(t, err)
				require.Equal(t, templ.Template.Name, st.Name)
				require.Equal(t, templ.Template.Description, st.Description)
				require.Equal(t, templ.Template.CanConstruct, st.CanConstruct)
			})
			t.Run("WIT has changed", func(t *testing.T) {
				wit, err := s.witRepo.Load(s.Ctx, templ.WITs[0].ID)
				require.NoError(t, err)
				t.Run("name changed", func(t *testing.T) {
					require.Equal(t, templ.WITs[0].Name, wit.Name)
				})
				t.Run("can-constrcut changed", func(t *testing.T) {
					require.Equal(t, templ.WITs[0].CanConstruct, wit.CanConstruct)
				})
				t.Run("\"flavor\" field added", func(t *testing.T) {
					obj, ok := wit.Fields["flavor"]
					require.True(t, ok, "flavor field not found in %+v", wit.Fields)
					require.NotNil(t, obj)
				})
			})
			t.Run("WILT name has changed", func(t *testing.T) {
				wilt, err := s.wiltRepo.Load(s.Ctx, templ.WILTs[0].ID)
				require.NoError(t, err)
				require.Equal(t, templ.WILTs[0].Name, wilt.Name)
			})
			t.Run("WITG name has changed", func(t *testing.T) {
				witg, err := s.witgRepo.Load(s.Ctx, templ.WITGs[0].ID)
				require.NoError(t, err)
				require.Equal(t, templ.WITGs[0].Name, witg.Name)
			})
		})
	})
	s.T().Run("invalid", func(t *testing.T) {
		t.Run("change in field type", func(t *testing.T) {
			// Create fresh template
			spaceTemplateID := uuid.NewV4()
			witID := uuid.NewV4()
			wiltID := uuid.NewV4()
			witgID := uuid.NewV4()
			oldTempl := getValidTestTemplateParsed(t, spaceTemplateID, witID, wiltID, witgID)
			oldTempl.Template.Name = "old name for space template " + spaceTemplateID.String()
			_, err := s.importerRepo.Import(s.Ctx, oldTempl)
			require.NoError(t, err)
			// Import it once more but this time with changes
			templ := getValidTestTemplateParsed(t, spaceTemplateID, witID, wiltID, witgID)
			templ.WITs[0].Fields["title"] = workitem.FieldDefinition{
				Label:       "Title",
				Description: "The title of the bug",
				Required:    true,
				Type: workitem.SimpleType{
					Kind: workitem.KindInteger,
				},
			}
			// when
			_, err = s.importerRepo.Import(s.Ctx, templ)
			// then
			require.Error(t, err)
		})
		t.Run("WIT already exists", func(t *testing.T) {
			// given old space template with new name, new ID, and new WILT ID
			newWILTID := uuid.NewV4()
			new := getValidTestTemplateParsed(t, spaceTemplateID, witID, newWILTID, witgID)
			new.Template.Name = testsupport.CreateRandomValidTestName("test template")
			new.SetID(uuid.NewV4())
			// when
			_, err := s.importerRepo.Import(s.Ctx, new)
			require.Error(t, err)
		})
		t.Run("WILT already exists", func(t *testing.T) {
			// given old space template with new name, new ID, and new WIT ID
			newWITID := uuid.NewV4()
			new := getValidTestTemplateParsed(t, spaceTemplateID, newWITID, wiltID, witgID)
			new.Template.Name = testsupport.CreateRandomValidTestName("test template")
			new.SetID(uuid.NewV4())
			// when
			_, err := s.importerRepo.Import(s.Ctx, new)
			require.Error(t, err)
		})
		t.Run("violating unique name constraint", func(t *testing.T) {
			// given a template creation with some name
			firstTemplate := getValidTestTemplateParsed(t, uuid.NewV4(), uuid.NewV4(), uuid.NewV4(), uuid.NewV4())
			firstTemplate.Template.Name = "first template " + firstTemplate.Template.ID.String()
			_, err := s.importerRepo.Import(s.Ctx, firstTemplate)
			require.NoError(t, err)
			// when we try to create another template with the same name as
			// before
			expected := getValidTestTemplateParsed(t, uuid.NewV4(), uuid.NewV4(), uuid.NewV4(), uuid.NewV4())
			expected.Template.Name = firstTemplate.Template.Name
			actual, err := s.importerRepo.Import(s.Ctx, expected)
			// then the is not allowed
			require.Error(t, err)
			require.Nil(t, actual)
			isBadParameterError, e := errors.IsBadParameterError(err)
			require.True(t, isBadParameterError)
			require.Contains(t, e.Error(), "name")
			require.Contains(t, e.Error(), "unique")
		})
		t.Run("missing WIT from previous import", func(t *testing.T) {
			// when
			spaceTemplateID := uuid.NewV4()
			witID := uuid.NewV4()
			wiltID := uuid.NewV4()
			witgID := uuid.NewV4()
			firstTemplate := getValidTestTemplateParsed(t, spaceTemplateID, witID, wiltID, witgID)
			firstTemplate.Template.Name = testsupport.CreateRandomValidTestName("first template")
			// Define an additional WIT that we'll skip the next time we import
			// the space template
			firstTemplate.WITs = append(firstTemplate.WITs, &workitem.WorkItemType{
				ID:              uuid.NewV4(),
				SpaceTemplateID: spaceTemplateID,
				Name:            "Foo",
				Description:     ptr.String("Description of foo"),
				Icon:            "fa fa-bug",
				Extends:         workitem.SystemPlannerItem,
			})
			_, err := s.importerRepo.Import(s.Ctx, firstTemplate)
			require.NoError(t, err)
			// Now import the same template but leave out "foo" WIT
			firstTemplate.WITs = []*workitem.WorkItemType{firstTemplate.WITs[0]}
			_, err = s.importerRepo.Import(s.Ctx, firstTemplate)
			require.Error(t, err)
			require.Contains(t, err.Error(), "work item types to be imported must not remove these existing work item types")
		})
		t.Run("missing WILT from previous import", func(t *testing.T) {
			// when
			spaceTemplateID := uuid.NewV4()
			witID := uuid.NewV4()
			wiltID := uuid.NewV4()
			witgID := uuid.NewV4()
			firstTemplate := getValidTestTemplateParsed(t, spaceTemplateID, witID, wiltID, witgID)
			firstTemplate.Template.Name = testsupport.CreateRandomValidTestName("first template")
			// Define an additional WILT that we'll skip the next time we import
			// the space template
			firstTemplate.WILTs = append(firstTemplate.WILTs, &link.WorkItemLinkType{
				ID:              uuid.NewV4(),
				SpaceTemplateID: spaceTemplateID,
				Name:            "My Link Type",
				Description:     ptr.String("My Link Type description"),
				ForwardName:     "forward",
				ReverseName:     "backwards",
				Topology:        "tree",
				LinkCategoryID:  link.SystemWorkItemLinkCategoryUserID,
			})
			_, err := s.importerRepo.Import(s.Ctx, firstTemplate)
			require.NoError(t, err)
			// Now import the same template but leave out "myLinkType" WILT
			firstTemplate.WILTs = []*link.WorkItemLinkType{firstTemplate.WILTs[0]}
			_, err = s.importerRepo.Import(s.Ctx, firstTemplate)
			require.Error(t, err)
			require.Contains(t, err.Error(), "work item link types to be imported must not remove these existing work item link types")
		})
		t.Run("import existing template with removed \"title\" field in WIT", func(t *testing.T) {
			// Create fresh template
			spaceTemplateID := uuid.NewV4()
			witID := uuid.NewV4()
			wiltID := uuid.NewV4()
			witgID := uuid.NewV4()
			oldTempl := getValidTestTemplateParsed(t, spaceTemplateID, witID, wiltID, witgID)
			oldTempl.Template.Name = "old name for space template " + spaceTemplateID.String()
			_, err := s.importerRepo.Import(s.Ctx, oldTempl)
			require.NoError(t, err)
			// Import it once more but this time remove a field from a WIT
			templ := getValidTestTemplateParsed(t, spaceTemplateID, witID, wiltID, witgID)
			templ.Template.Name = oldTempl.Template.Name
			// The field to be removed needs to be explicitly defined in the WIT
			// and not in the WIT it extends.
			delete(templ.WITs[0].Fields, "title")
			// when
			_, err = s.importerRepo.Import(s.Ctx, templ)
			// then
			require.Error(t, err)
			require.Contains(t, err.Error(), "you must not remove these fields from the new work item type definition of")
		})
	})
}

func (s *repoSuite) TestExists() {
	// given
	spaceTemplateID := uuid.NewV4()
	witID := uuid.NewV4()
	wiltID := uuid.NewV4()
	witgID := uuid.NewV4()

	s.T().Run("existing template", func(t *testing.T) {
		// given
		expected := getValidTestTemplateParsed(t, spaceTemplateID, witID, wiltID, witgID)
		_, err := s.spaceTemplateRepo.Create(s.Ctx, expected.Template)
		require.NoError(t, err)
		// when
		err = s.spaceTemplateRepo.CheckExists(s.Ctx, spaceTemplateID)
		// then
		require.NoError(t, err)
	})

	s.T().Run("not existing template", func(t *testing.T) {
		// given
		notExistingID := uuid.NewV4()
		// when
		err := s.spaceTemplateRepo.CheckExists(s.Ctx, notExistingID)
		// then
		require.Error(t, err)
	})
}

func (s *repoSuite) TestLoad() {
	// given
	spaceTemplateID := uuid.NewV4()
	witID := uuid.NewV4()
	wiltID := uuid.NewV4()
	witgID := uuid.NewV4()

	s.T().Run("existing template", func(t *testing.T) {
		// given
		expected := getValidTestTemplateParsed(t, spaceTemplateID, witID, wiltID, witgID)
		_, err := s.spaceTemplateRepo.Create(s.Ctx, expected.Template)
		require.NoError(t, err)
		// when
		actual, err := s.spaceTemplateRepo.Load(s.Ctx, spaceTemplateID)
		// then
		require.NoError(t, err)
		require.NotNil(t, actual)
		require.Equal(t, expected.Template.ID, actual.ID)
		require.Equal(t, expected.Template.Name, actual.Name)
		require.Equal(t, expected.Template.Description, actual.Description)
	})

	s.T().Run("not existing template)", func(t *testing.T) {
		// when
		s, err := s.spaceTemplateRepo.Load(s.Ctx, uuid.NewV4())
		// then
		require.Error(t, err)
		require.Nil(t, s)
	})
}

func (s *repoSuite) TestRepository_List() {
	s.T().Run("empty or filled", func(t *testing.T) {
		// when
		spaceTemplates, err := s.spaceTemplateRepo.List(s.Ctx)
		// then expect zero or more space templates
		require.NoError(t, err)
		assert.True(t, len(spaceTemplates) >= 0)
	})

	s.T().Run("list 2 space templates", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(2))
		// when
		spaceTemplates, err := s.spaceTemplateRepo.List(s.Ctx)
		// then
		require.NoError(t, err)
		assert.True(t, len(spaceTemplates) >= 2)
		spaceTemplatesToBeFound := id.Map{
			fxt.SpaceTemplates[0].ID: {},
			fxt.SpaceTemplates[0].ID: {},
		}
		for _, st := range spaceTemplates {
			if _, ok := spaceTemplatesToBeFound[st.ID]; ok {
				delete(spaceTemplatesToBeFound, st.ID)
			}
		}
		require.Len(t, spaceTemplatesToBeFound, 0, "these space templates where not found", spaceTemplatesToBeFound)
	})
}
