package importer_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	"github.com/fabric8-services/fabric8-wit/spacetemplate/importer"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	uuid "github.com/satori/go.uuid"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_FromString(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	t.Run("valid", func(t *testing.T) {
		t.Run("minimal", func(t *testing.T) {
			t.Parallel()
			// given: valid empty template
			yaml := `
space_template:
  id: "038e4d23-4e52-45e7-b0d9-5d736109845f"
  name: "foo"
  description: "bar"
work_item_types:
work_item_link_types:
- id: "d45e7af0-d88b-4777-a180-581b068063c6"
  topology: "tree"`
			// when
			templ, err := importer.FromString(yaml)
			// then
			require.NoError(t, err)
			require.NotNil(t, templ)
			require.Equal(t, uuid.FromStringOrNil("038e4d23-4e52-45e7-b0d9-5d736109845f"), templ.Template.ID)
			require.Equal(t, "foo", templ.Template.Name)
			require.NotNil(t, templ.Template.Description)
			require.Equal(t, "bar", *templ.Template.Description)
			require.NoError(t, templ.Validate())
		})
	})

	t.Run("invalid", func(t *testing.T) {

		t.Run("empty name", func(t *testing.T) {
			t.Parallel()
			// given: valid empty template
			yaml := `
space_template:
  id: "038e4d23-4e52-45e7-b0d9-5d736109845f"
  description: "bar"`
			// when
			_, err := importer.FromString(yaml)
			require.Error(t, err)
		})
	})
}

func Test_ImportHelper_Validate(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	t.Run("valid", func(t *testing.T) {
		t.Run("legacy template", func(t *testing.T) {
			t.Parallel()
			// given
			templ, err := importer.LegacyTemplate()
			// then
			require.NoError(t, err)
			require.Equal(t, spacetemplate.SystemLegacyTemplateID, templ.Template.ID)
			witsToBeFound := id.Map{
				workitem.SystemTask:             {},
				workitem.SystemValueProposition: {},
				workitem.SystemFundamental:      {},
				workitem.SystemExperience:       {},
				workitem.SystemFeature:          {},
				workitem.SystemScenario:         {},
				workitem.SystemBug:              {},
				workitem.SystemPapercuts:        {},
			}
			for _, wit := range templ.WITs {
				delete(witsToBeFound, wit.ID)
			}
			require.Len(t, witsToBeFound, 0, "these work item types where not found in the legacy template: %+v", witsToBeFound)
			require.NoError(t, templ.Validate())
		})

		t.Run("scrum template", func(t *testing.T) {
			t.Parallel()
			// given
			templ, err := importer.ScrumTemplate()
			// then
			require.NoError(t, err)
			require.Equal(t, spacetemplate.SystemScrumTemplateID, templ.Template.ID)
			witsToBeFound := map[string]struct{}{
				"Scrum Common Type":    {},
				"Bug":                  {},
				"Task":                 {},
				"Epic":                 {},
				"Feature":              {},
				"Impediment":           {},
				"Product Backlog Item": {},
			}
			for _, wit := range templ.WITs {
				_, ok := witsToBeFound[wit.Name]
				require.True(t, ok, "found unexpected work item type: %s", wit.Name)
				delete(witsToBeFound, wit.Name)
			}
			require.Len(t, witsToBeFound, 0, "these work item types where not found in the scrum template: %+v", witsToBeFound)
			require.NoError(t, templ.Validate())
		})

		t.Run("test template", func(t *testing.T) {
			t.Parallel()
			// given
			spaceTemplateID := uuid.NewV4()
			witID := uuid.NewV4()
			wiltID := uuid.NewV4()
			witgID := uuid.NewV4()

			yaml := getValidTestTemplate(spaceTemplateID, witID, wiltID, witgID)
			// when
			actual, err := importer.FromString(yaml)
			// then
			require.NoError(t, err)
			require.Equal(t, spaceTemplateID, actual.Template.ID)
			require.NoError(t, actual.Validate())
			expected := getValidTestTemplateParsed(t, spaceTemplateID, witID, wiltID, witgID)
			assert.True(t, expected.Equal(*actual))
			assert.Equal(t, expected.String(), actual.String())
			checkDiff(t, expected, *actual)
		})
	})
	t.Run("invalid", func(t *testing.T) {
		t.Run("invalid space template ID on WIT", func(t *testing.T) {
			t.Parallel()
			// given: valid empty template
			spaceTemplateID := uuid.NewV4()
			templ := getValidTestTemplateParsed(t, spaceTemplateID, uuid.NewV4(), uuid.NewV4(), uuid.NewV4())
			// when
			templ.WITs[0].SpaceTemplateID = uuid.NewV4()
			// then
			require.Error(t, templ.Validate())
		})

		t.Run("invalid space template ID on WILT", func(t *testing.T) {
			t.Parallel()
			// given: valid empty template
			spaceTemplateID := uuid.NewV4()
			templ := getValidTestTemplateParsed(t, spaceTemplateID, uuid.NewV4(), uuid.NewV4(), uuid.NewV4())
			// when
			templ.WILTs[0].SpaceTemplateID = uuid.NewV4()
			// then
			require.Error(t, templ.Validate())
		})
	})
}

func checkDiff(t *testing.T, expected, actual importer.ImportHelper) {
	expectedStr := expected.String()
	actualStr := actual.String()
	if expectedStr != actualStr {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(expectedStr, actualStr, false)
		t.Errorf("mismatch of expected and actual template string:\n %s \n", dmp.DiffPrettyText(diffs))
	}
}

// getValidTestTemplate returns a test template in unparsed format. See
// getValidTestTemplateParsed() for the parsed representation of this template
func getValidTestTemplate(spaceTemplateID, witID, wiltID, witgID uuid.UUID) string {
	return `
space_template:
  id: "` + spaceTemplateID.String() + `"
  name: test template
  description: test template description
work_item_types:
- id: "` + witID.String() + `"
  description: a basic work item type
  name: Bug
  icon: fa fa-bug
  extends: "` + workitem.SystemPlannerItem.String() + `"
  child_types:
  - "` + workitem.SystemPlannerItem.String() + `"
  fields:
    title:
      label: Title
      description: The title of the bug
      required: yes
      type:
        kind: string
    state:
      label: State
      description: The state of the bug
      required: NO
      type:
        simple_type:
          kind: enum
        base_type:
          kind: string
        values:
          - new
          - closed
    priority:
      label: Priority
      description: The priority of the bug
      required: NO
      type:
        simple_type:
          kind: list
        component_type:
          kind: integer
work_item_link_types:
- id: "` + wiltID.String() + `"
  name: Blocker
  description: work item blocks another one
  forward_name: blocks
  reverse_name: blocked by
  topology: tree
  link_category_id: "2F24724F-797C-4073-8B16-4BB8CE9E84A6"
work_item_type_groups:
- name: Scenarios
  id: "` + witgID.String() + `"
  type_list:
    - "` + witID.String() + `"
  bucket: portfolio
  icon: fa fa-suitcase
`
}

// getValidTestTemplateParsed returns the expected parsed representation of the
// getValidTestTemplate string
func getValidTestTemplateParsed(t *testing.T, spaceTemplateID, witID, wiltID uuid.UUID, witgID uuid.UUID) importer.ImportHelper {
	expected := importer.ImportHelper{
		Template: spacetemplate.SpaceTemplate{
			ID:          spaceTemplateID,
			Name:        "test template",
			Description: ptr.String("test template description"),
		},
		WITs: []*workitem.WorkItemType{
			{
				ID:              witID,
				SpaceTemplateID: spaceTemplateID,
				Name:            "Bug",
				Description:     ptr.String("a basic work item type"),
				Icon:            "fa fa-bug",
				Extends:         workitem.SystemPlannerItem,
				ChildTypeIDs: []uuid.UUID{
					workitem.SystemPlannerItem,
				},
				Fields: map[string]workitem.FieldDefinition{
					"title": {
						Label:       "Title",
						Description: "The title of the bug",
						Required:    true,
						Type: workitem.SimpleType{
							Kind: workitem.KindString,
						},
					},
					"state": {
						Label:       "State",
						Description: "The state of the bug",
						Required:    false,
						Type: workitem.EnumType{
							SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
							BaseType:   workitem.SimpleType{Kind: workitem.KindString},
							// TODO(kwk): Once we parse values, fill them in here
							Values: []interface{}{
								"new",
								"closed",
							},
						},
					},
					"priority": {
						Label:       "Priority",
						Description: "The priority of the bug",
						Required:    false,
						Type: workitem.ListType{
							SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
							ComponentType: workitem.SimpleType{Kind: workitem.KindInteger},
						},
					},
				},
			},
		},
		WILTs: []*link.WorkItemLinkType{
			{
				ID:              wiltID,
				SpaceTemplateID: spaceTemplateID,
				Name:            "Blocker",
				Description:     ptr.String("work item blocks another one"),
				ForwardName:     "blocks",
				ReverseName:     "blocked by",
				Topology:        "tree",
				LinkCategoryID:  link.SystemWorkItemLinkCategoryUserID,
			},
		},
		WITGs: []*workitem.WorkItemTypeGroup{
			{
				ID:              witgID,
				Name:            "Scenarios",
				Bucket:          workitem.BucketPortfolio,
				Icon:            "fa fa-suitcase",
				SpaceTemplateID: spaceTemplateID,
				TypeList: []uuid.UUID{
					witID,
				},
			},
		},
	}
	return expected
}

func Test_ImportHelperEqual(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	expectedID := uuid.FromStringOrNil("89821c14-1051-4f91-a879-a46f7d66c05c")
	expected := importer.ImportHelper{
		Template: spacetemplate.SpaceTemplate{
			ID:          expectedID,
			Name:        "empty template " + expectedID.String(),
			Description: ptr.String("description for empty template " + expectedID.String()),
		},
	}

	t.Run("different types", func(t *testing.T) {
		t.Parallel()
		assert.False(t, expected.Equal(convert.DummyEqualer{}))
	})
	t.Run("equalness", func(t *testing.T) {
		t.Parallel()
		actual := expected
		assert.True(t, expected.Equal(actual))
	})
	t.Run("different WITs", func(t *testing.T) {
		t.Parallel()
		actual := expected
		actual.WITs = append(actual.WITs, &workitem.WorkItemType{})
		assert.False(t, expected.Equal(actual))
	})
	t.Run("different WILTs", func(t *testing.T) {
		t.Parallel()
		actual := expected
		actual.WILTs = append(actual.WILTs, &link.WorkItemLinkType{})
		assert.False(t, expected.Equal(actual))
	})
	t.Run("different WITGs", func(t *testing.T) {
		t.Parallel()
		actual := expected
		actual.WITGs = append(actual.WITGs, &workitem.WorkItemTypeGroup{})
		assert.False(t, expected.Equal(actual))
	})
}
