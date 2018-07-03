package controller

import (
	"context"
	"testing"

	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestParseInts(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	integers, err := parseInts(nil)
	assert.Equal(t, nil, err)
	assert.Equal(t, []int{}, integers)

	str := "1, 2, foo"
	_, err = parseInts(&str)
	assert.NotNil(t, err)
}

func TestParseLimit(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// Test parsing error in parseInts
	str := "1000, foo"
	integers, length, err := parseLimit(&str)
	assert.NotNil(t, err)
	assert.Equal(t, 0, length)
	assert.Nil(t, integers)

	// Test length = 1
	str = "1000"
	integers, length, err = parseLimit(&str)
	require.NoError(t, err)
	assert.Equal(t, 1000, length)
	assert.Nil(t, integers)

	// Test empty string
	str = ""
	integers, length, err = parseLimit(&str)
	require.NoError(t, err)
	assert.Equal(t, 100, length)
	assert.Nil(t, integers)
}

func TestSetPagingLinks(t *testing.T) {
	links := &app.PagingLinks{}
	setPagingLinks(links, "", 0, 0, 1, 0)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.First)
	assert.Equal(t, "?page[offset]=0&page[limit]=0", *links.Last)
	assert.Nil(t, links.Next)
	assert.Nil(t, links.Prev)

	setPagingLinks(links, "prefix", 0, 0, 1, 0)
	assert.Equal(t, "prefix?page[offset]=0&page[limit]=1", *links.First)
	assert.Equal(t, "prefix?page[offset]=0&page[limit]=0", *links.Last)
	assert.Nil(t, links.Next)
	assert.Nil(t, links.Prev)

	setPagingLinks(links, "", 0, 0, 1, 1)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.First)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Last)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Next)
	assert.Nil(t, links.Prev)

	setPagingLinks(links, "", 0, 1, 1, 0)
	assert.Equal(t, "?page[offset]=0&page[limit]=0", *links.First)
	assert.Equal(t, "?page[offset]=0&page[limit]=0", *links.Last)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Next)
	assert.Nil(t, links.Prev)

	setPagingLinks(links, "", 0, 1, 1, 1)
	assert.Equal(t, "?page[offset]=0&page[limit]=0", *links.First)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Last)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Next)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Prev)

	setPagingLinks(links, "", 0, 2, 1, 1)
	assert.Equal(t, "?page[offset]=0&page[limit]=0", *links.First)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Last)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Next)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Prev)

	setPagingLinks(links, "", 0, 3, 4, 4)
	assert.Equal(t, "?page[offset]=0&page[limit]=3", *links.First)
	assert.Equal(t, "?page[offset]=3&page[limit]=4", *links.Last)
	assert.Equal(t, "?page[offset]=3&page[limit]=4", *links.Next)
	assert.Equal(t, "?page[offset]=0&page[limit]=3", *links.Prev)
}

type TestWorkItemREST struct {
	gormtestsupport.DBTestSuite
}

func TestRunWorkItemREST(t *testing.T) {
	suite.Run(t, &TestWorkItemREST{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (rest *TestWorkItemREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
}

func prepareWI2(attributes map[string]interface{}, witID, spaceID uuid.UUID) app.WorkItem {
	spaceRelatedURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(spaceID.String()))
	witRelatedURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.WorkitemtypeHref(witID.String()))
	return app.WorkItem{
		Type: "workitems",
		Relationships: &app.WorkItemRelationships{
			BaseType: &app.RelationBaseType{
				Data: &app.BaseTypeData{
					Type: "workitemtypes",
					ID:   witID,
				},
				Links: &app.GenericLinks{
					Self:    &witRelatedURL,
					Related: &witRelatedURL,
				},
			},
			Space: app.NewSpaceRelation(spaceID, spaceRelatedURL),
		},
		Attributes: attributes,
	}
}

func (rest *TestWorkItemREST) TestConvertJSONAPIToWorkItemWithLegacyDescription() {
	t := rest.T()
	resource.Require(t, resource.Database)
	//given
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.CreateWorkItemEnvironment())
	attributes := map[string]interface{}{
		workitem.SystemTitle:       "title",
		workitem.SystemDescription: "description",
	}
	source := prepareWI2(attributes, fxt.WorkItemTypes[0].ID, fxt.Spaces[0].ID)
	target := &workitem.WorkItem{Fields: map[string]interface{}{}}
	err := application.Transactional(rest.GormDB, func(app application.Application) error {
		return ConvertJSONAPIToWorkItem(context.Background(), "", app, source, target, fxt.WorkItemTypes[0].ID, fxt.Spaces[0].ID)
	})
	// assert
	require.NoError(t, err)
	require.NotNil(t, target)
	require.NotNil(t, target.Fields)
	require.True(t, uuid.Equal(source.Relationships.BaseType.Data.ID, target.Type))
	expectedDescription := rendering.NewMarkupContentFromLegacy("description")
	assert.Equal(t, expectedDescription, target.Fields[workitem.SystemDescription])

}

func (rest *TestWorkItemREST) TestConvertJSONAPIToWorkItemWithDescriptionContentNoMarkup() {
	t := rest.T()
	resource.Require(t, resource.Database)
	//given
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.CreateWorkItemEnvironment())
	attributes := map[string]interface{}{
		workitem.SystemTitle:       "title",
		workitem.SystemDescription: rendering.NewMarkupContentFromLegacy("description"),
	}
	source := prepareWI2(attributes, fxt.WorkItemTypes[0].ID, fxt.Spaces[0].ID)
	target := &workitem.WorkItem{Fields: map[string]interface{}{}}
	err := application.Transactional(rest.GormDB, func(app application.Application) error {
		return ConvertJSONAPIToWorkItem(context.Background(), "", app, source, target, fxt.WorkItemTypes[0].ID, fxt.Spaces[0].ID)
	})
	require.NoError(t, err)
	require.NotNil(t, target)
	require.NotNil(t, target.Fields)
	require.True(t, uuid.Equal(source.Relationships.BaseType.Data.ID, target.Type))
	expectedDescription := rendering.NewMarkupContentFromLegacy("description")
	assert.Equal(t, expectedDescription, target.Fields[workitem.SystemDescription])
}

func (rest *TestWorkItemREST) TestConvertJSONAPIToWorkItemWithDescriptionContentAndMarkup() {
	t := rest.T()
	resource.Require(t, resource.Database)
	//given
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.CreateWorkItemEnvironment())
	attributes := map[string]interface{}{
		workitem.SystemTitle:       "title",
		workitem.SystemDescription: rendering.NewMarkupContent("description", rendering.SystemMarkupMarkdown),
	}
	source := prepareWI2(attributes, fxt.WorkItemTypes[0].ID, fxt.Spaces[0].ID)
	target := &workitem.WorkItem{Fields: map[string]interface{}{}}
	err := application.Transactional(rest.GormDB, func(app application.Application) error {
		return ConvertJSONAPIToWorkItem(context.Background(), "", app, source, target, fxt.WorkItemTypes[0].ID, fxt.Spaces[0].ID)
	})
	require.NoError(t, err)
	require.NotNil(t, target)
	require.NotNil(t, target.Fields)
	require.True(t, uuid.Equal(source.Relationships.BaseType.Data.ID, target.Type))
	expectedDescription := rendering.NewMarkupContent("description", rendering.SystemMarkupMarkdown)
	assert.Equal(t, expectedDescription, target.Fields[workitem.SystemDescription])
}

func (rest *TestWorkItemREST) TestConvertJSONAPIToWorkItemWithTitle() {
	t := rest.T()
	resource.Require(t, resource.Database)
	//given
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.CreateWorkItemEnvironment())
	title := "title"
	attributes := map[string]interface{}{
		workitem.SystemTitle: title,
	}
	source := prepareWI2(attributes, fxt.WorkItemTypes[0].ID, fxt.Spaces[0].ID)
	target := &workitem.WorkItem{Fields: map[string]interface{}{}}
	err := application.Transactional(rest.GormDB, func(app application.Application) error {
		return ConvertJSONAPIToWorkItem(context.Background(), "", app, source, target, fxt.WorkItemTypes[0].ID, fxt.Spaces[0].ID)
	})
	require.NoError(t, err)
	require.NotNil(t, target)
	require.NotNil(t, target.Fields)
	require.True(t, uuid.Equal(source.Relationships.BaseType.Data.ID, target.Type))
	assert.Equal(t, title, target.Fields[workitem.SystemTitle])
}

func (rest *TestWorkItemREST) TestConvertJSONAPIToWorkItemWithMissingTitle() {
	t := rest.T()
	resource.Require(t, resource.Database)
	// given
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.CreateWorkItemEnvironment())
	attributes := map[string]interface{}{}
	source := prepareWI2(attributes, fxt.WorkItemTypes[0].ID, fxt.Spaces[0].ID)
	target := &workitem.WorkItem{Fields: map[string]interface{}{}}
	// when
	err := application.Transactional(rest.GormDB, func(app application.Application) error {
		return ConvertJSONAPIToWorkItem(context.Background(), "", app, source, target, fxt.WorkItemTypes[0].ID, fxt.Spaces[0].ID)
	})
	// then: no error expected at this level, even though the title is missing
	require.NoError(t, err)
}

func (rest *TestWorkItemREST) TestConvertJSONAPIToWorkItemWithEmptyTitle() {
	t := rest.T()
	resource.Require(t, resource.Database)
	// given
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.CreateWorkItemEnvironment())
	attributes := map[string]interface{}{
		workitem.SystemTitle: "",
	}
	source := prepareWI2(attributes, fxt.WorkItemTypes[0].ID, fxt.Spaces[0].ID)
	target := &workitem.WorkItem{Fields: map[string]interface{}{}}
	// when
	err := application.Transactional(rest.GormDB, func(app application.Application) error {
		return ConvertJSONAPIToWorkItem(context.Background(), "", app, source, target, fxt.WorkItemTypes[0].ID, fxt.Spaces[0].ID)
	})
	// then: no error expected at this level, even though the title is missing
	require.NoError(t, err)
}

func (rest *TestWorkItemREST) TestConvertWorkItem() {
	request := &http.Request{Host: "localhost"}
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.CreateWorkItemEnvironment())

	rest.T().Run("with description", func(t *testing.T) {
		wi := workitem.WorkItem{
			Type:    fxt.WorkItemTypes[0].ID,
			SpaceID: fxt.Spaces[0].ID,
			Fields: map[string]interface{}{
				workitem.SystemTitle:       "title",
				workitem.SystemDescription: "description",
			},
		}
		wi2, err := ConvertWorkItem(request, *fxt.WorkItemTypes[0], wi)
		require.NoError(t, err)
		assert.Equal(t, "title", wi2.Attributes[workitem.SystemTitle])
		assert.Equal(t, "description", wi2.Attributes[workitem.SystemDescription])
	})
	rest.T().Run("without description", func(t *testing.T) {
		request := &http.Request{Host: "localhost"}
		wi := workitem.WorkItem{
			Type:    fxt.WorkItemTypes[0].ID,
			SpaceID: fxt.Spaces[0].ID,
			Fields: map[string]interface{}{
				workitem.SystemTitle: "title",
			},
		}
		wi2, err := ConvertWorkItem(request, *fxt.WorkItemTypes[0], wi)
		require.NoError(t, err)
		assert.Equal(t, "title", wi2.Attributes[workitem.SystemTitle])
		assert.Nil(t, wi2.Attributes[workitem.SystemDescription])
	})
}

func (rest *TestWorkItemREST) TestConvertWorkItems() {
	rest.T().Run("ok", func(t *testing.T) {
		// given
		request := &http.Request{Host: "localhost"}
		fxt := tf.NewTestFixture(t, rest.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(3))
		wis := []workitem.WorkItem{*fxt.WorkItems[0], *fxt.WorkItems[1], *fxt.WorkItems[2]}
		wits, err := loadWorkItemTypesFromPtrArr(rest.Ctx, rest.GormDB, fxt.WorkItems)
		require.NoError(t, err)
		// when
		convertedWIs, err := ConvertWorkItems(request, wits, wis)
		require.NoError(t, err)
		for i, converted := range convertedWIs {
			require.Equal(t, fxt.WorkItems[i].ID, *converted.ID)
			require.Equal(t, fxt.WorkItems[i].Fields[workitem.SystemTitle], converted.Attributes[workitem.SystemTitle])
			require.Equal(t, fxt.WorkItems[i].Fields[workitem.SystemDescription], converted.Attributes[workitem.SystemDescription])
		}
	})
	rest.T().Run("length mismatch", func(t *testing.T) {
		// given
		request := &http.Request{Host: "localhost"}
		fxt := tf.NewTestFixture(t, rest.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(3))
		wis := []workitem.WorkItem{*fxt.WorkItems[0], *fxt.WorkItems[1], *fxt.WorkItems[2]}
		wits := []workitem.WorkItemType{}
		// when
		_, err := ConvertWorkItems(request, wits, wis)
		require.Error(t, err)
		require.Contains(t, err.Error(), "length mismatch")
	})
}

func (rest *TestWorkItemREST) TestLoadWorkItemTypes() {
	fxt := tf.NewTestFixture(rest.T(), rest.DB,
		tf.WorkItemTypes(3),
		tf.WorkItems(3, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Type = fxt.WorkItemTypes[idx].ID
			return nil
		}),
	)
	rest.T().Run("from normal array of work items", func(t *testing.T) {
		t.Run("not empty", func(t *testing.T) {
			// given
			wis := []workitem.WorkItem{*fxt.WorkItems[0], *fxt.WorkItems[1], *fxt.WorkItems[2]}
			// when
			wits, err := loadWorkItemTypesFromArr(rest.Ctx, rest.GormDB, wis)
			// then
			require.NoError(t, err)
			toBeFound := id.Slice{fxt.WorkItemTypes[0].ID, fxt.WorkItemTypes[1].ID, fxt.WorkItemTypes[2].ID}.ToMap()
			for _, wit := range wits {
				_, ok := toBeFound[wit.ID]
				require.True(t, ok, "found unexpected work item type: %s (%s)", wit.ID, wit.Name)
				delete(toBeFound, wit.ID)
			}
			require.Empty(t, toBeFound, "failed to find these work item types: %+v", toBeFound)
		})
		t.Run("empty", func(t *testing.T) {
			// given
			wis := []workitem.WorkItem{}
			// when
			wits, err := loadWorkItemTypesFromArr(rest.Ctx, rest.GormDB, wis)
			// then
			require.NoError(t, err)
			require.Empty(t, wits)
		})
		t.Run("nil", func(t *testing.T) {
			// given
			var wis []workitem.WorkItem
			// when
			wits, err := loadWorkItemTypesFromArr(rest.Ctx, rest.GormDB, wis)
			// then
			require.NoError(t, err)
			require.Empty(t, wits)
		})
	})
	rest.T().Run("from pointer array of work items", func(t *testing.T) {
		t.Run("not empty", func(t *testing.T) {
			// given
			wis := []*workitem.WorkItem{fxt.WorkItems[0], fxt.WorkItems[1], fxt.WorkItems[2]}
			// when
			wits, err := loadWorkItemTypesFromPtrArr(rest.Ctx, rest.GormDB, wis)
			// then
			require.NoError(t, err)
			toBeFound := id.Slice{fxt.WorkItemTypes[0].ID, fxt.WorkItemTypes[1].ID, fxt.WorkItemTypes[2].ID}.ToMap()
			for _, wit := range wits {
				_, ok := toBeFound[wit.ID]
				require.True(t, ok, "found unexpected work item type: %s (%s)", wit.ID, wit.Name)
				delete(toBeFound, wit.ID)
			}
			require.Empty(t, toBeFound, "failed to find these work item types: %+v", toBeFound)
		})
		t.Run("empty", func(t *testing.T) {
			// given
			wis := []*workitem.WorkItem{}
			// when
			wits, err := loadWorkItemTypesFromPtrArr(rest.Ctx, rest.GormDB, wis)
			// then
			require.NoError(t, err)
			require.Empty(t, wits)
		})
		t.Run("nil", func(t *testing.T) {
			// given
			var wis []*workitem.WorkItem
			// when
			wits, err := loadWorkItemTypesFromPtrArr(rest.Ctx, rest.GormDB, wis)
			// then
			require.NoError(t, err)
			require.Empty(t, wits)
		})
	})
}
