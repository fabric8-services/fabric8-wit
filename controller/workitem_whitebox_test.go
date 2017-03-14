package controller

import (
	"fmt"
	"os"
	"testing"

	"net/http"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	config "github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/space"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

var DB *gorm.DB
var RwiScheduler *remoteworkitem.Scheduler
var configuration *config.ConfigurationData

func TestMain(m *testing.M) {
	var err error

	configuration, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	if _, c := os.LookupEnv(resource.Database); c != false {

		DB, err = gorm.Open("postgres", configuration.GetPostgresConfigString())

		if err != nil {
			panic("Failed to connect database: " + err.Error())
		}
		defer DB.Close()

		// Make sure the database is populated with the correct types (e.g. bug etc.)
		if configuration.GetPopulateCommonTypes() {
			ctx := migration.NewMigrationContext(context.Background())
			if err := models.Transactional(DB, func(tx *gorm.DB) error {
				return migration.PopulateCommonTypes(ctx, tx, workitem.NewWorkItemTypeRepository(tx))
			}); err != nil {
				panic(err.Error())
			}
		}

		// RemoteWorkItemScheduler now available for all other test cases
		RwiScheduler = remoteworkitem.NewScheduler(DB)
	}
	os.Exit(func() int {
		c := m.Run()
		RwiScheduler.Stop()
		return c
	}())
}

func TestNewWorkitemController(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	assert.Panics(t, func() {
		NewWorkitemController(goa.New("Test service"), nil)
	})
}

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
	assert.Nil(t, err)
	assert.Equal(t, 1000, length)
	assert.Nil(t, integers)

	// Test empty string
	str = ""
	integers, length, err = parseLimit(&str)
	assert.Nil(t, err)
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

func TestConvertWorkItemWithDescription(t *testing.T) {
	request := http.Request{Host: "localhost"}
	requestData := &goa.RequestData{Request: &request}
	// map[string]interface{}
	fields := map[string]interface{}{
		workitem.SystemTitle:       "title",
		workitem.SystemDescription: "description",
	}

	spaceSelfURL := rest.AbsoluteURL(requestData, app.SpaceHref(space.SystemSpace.String()))
	wi := app.WorkItem{
		Fields: fields,
		Relationships: &app.WorkItemRelationships{
			Space: space.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
		},
	}
	wi2 := ConvertWorkItem(requestData, &wi)
	assert.Equal(t, "title", wi2.Attributes[workitem.SystemTitle])
	assert.Equal(t, "description", wi2.Attributes[workitem.SystemDescription])
}

func TestConvertWorkItemWithoutDescription(t *testing.T) {
	request := http.Request{Host: "localhost"}
	requestData := &goa.RequestData{Request: &request}
	// map[string]interface{}
	fields := map[string]interface{}{
		workitem.SystemTitle: "title",
	}

	spaceSelfURL := rest.AbsoluteURL(requestData, app.SpaceHref(space.SystemSpace.String()))
	wi := app.WorkItem{
		Fields: fields,
		Relationships: &app.WorkItemRelationships{
			Space: space.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
		},
	}
	wi2 := ConvertWorkItem(requestData, &wi)
	assert.Equal(t, "title", wi2.Attributes[workitem.SystemTitle])
	assert.Nil(t, wi2.Attributes[workitem.SystemDescription])
}

func prepareWI2(attributes map[string]interface{}) app.WorkItem2 {
	spaceSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(space.SystemSpace.String()))
	return app.WorkItem2{
		Type: "workitems",
		Relationships: &app.WorkItemRelationships{
			BaseType: &app.RelationBaseType{
				Data: &app.BaseTypeData{
					Type: "workitemtypes",
					ID:   workitem.SystemBug,
				},
			},
			Space: space.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
		},
		Attributes: attributes,
	}
}

func TestConvertJSONAPIToWorkItemWithLegacyDescription(t *testing.T) {
	appl := new(application.Application)
	attributes := map[string]interface{}{
		workitem.SystemTitle:       "title",
		workitem.SystemDescription: "description",
	}
	source := prepareWI2(attributes)
	target := &app.WorkItem{Fields: map[string]interface{}{}}
	err := ConvertJSONAPIToWorkItem(*appl, source, target)
	require.Nil(t, err)
	require.NotNil(t, target)
	require.NotNil(t, target.Fields)
	require.True(t, uuid.Equal(source.Relationships.BaseType.Data.ID, target.Type))
	expectedDescription := rendering.NewMarkupContentFromLegacy("description")
	assert.Equal(t, expectedDescription, target.Fields[workitem.SystemDescription])
}

func TestConvertJSONAPIToWorkItemWithDescriptionContentNoMarkup(t *testing.T) {
	appl := new(application.Application)
	attributes := map[string]interface{}{
		workitem.SystemTitle:       "title",
		workitem.SystemDescription: rendering.NewMarkupContentFromLegacy("description"),
	}
	source := prepareWI2(attributes)
	target := &app.WorkItem{Fields: map[string]interface{}{}}
	err := ConvertJSONAPIToWorkItem(*appl, source, target)
	require.Nil(t, err)
	require.NotNil(t, target)
	require.NotNil(t, target.Fields)
	require.True(t, uuid.Equal(source.Relationships.BaseType.Data.ID, target.Type))
	expectedDescription := rendering.NewMarkupContentFromLegacy("description")
	assert.Equal(t, expectedDescription, target.Fields[workitem.SystemDescription])
}

func TestConvertJSONAPIToWorkItemWithDescriptionContentAndMarkup(t *testing.T) {
	appl := new(application.Application)
	attributes := map[string]interface{}{
		workitem.SystemTitle:       "title",
		workitem.SystemDescription: rendering.NewMarkupContent("description", rendering.SystemMarkupMarkdown),
	}
	source := prepareWI2(attributes)
	target := &app.WorkItem{Fields: map[string]interface{}{}}
	err := ConvertJSONAPIToWorkItem(*appl, source, target)
	require.Nil(t, err)
	require.NotNil(t, target)
	require.NotNil(t, target.Fields)
	require.True(t, uuid.Equal(source.Relationships.BaseType.Data.ID, target.Type))
	expectedDescription := rendering.NewMarkupContent("description", rendering.SystemMarkupMarkdown)
	assert.Equal(t, expectedDescription, target.Fields[workitem.SystemDescription])
}

func TestConvertJSONAPIToWorkItemWithTitle(t *testing.T) {
	title := "title"
	appl := new(application.Application)
	attributes := map[string]interface{}{
		workitem.SystemTitle: title,
	}
	source := prepareWI2(attributes)
	target := &app.WorkItem{Fields: map[string]interface{}{}}
	err := ConvertJSONAPIToWorkItem(*appl, source, target)
	require.Nil(t, err)
	require.NotNil(t, target)
	require.NotNil(t, target.Fields)
	require.True(t, uuid.Equal(source.Relationships.BaseType.Data.ID, target.Type))
	assert.Equal(t, title, target.Fields[workitem.SystemTitle])
}

func TestConvertJSONAPIToWorkItemWithMissingTitle(t *testing.T) {
	// given
	appl := new(application.Application)
	attributes := map[string]interface{}{}
	source := prepareWI2(attributes)
	target := &app.WorkItem{Fields: map[string]interface{}{}}
	// when
	err := ConvertJSONAPIToWorkItem(*appl, source, target)
	// then: no error expected at this level, even though the title is missing
	require.Nil(t, err)
}

func TestConvertJSONAPIToWorkItemWithEmptyTitle(t *testing.T) {
	// given
	appl := new(application.Application)
	attributes := map[string]interface{}{
		workitem.SystemTitle: "",
	}
	source := prepareWI2(attributes)
	target := &app.WorkItem{Fields: map[string]interface{}{}}
	// when
	err := ConvertJSONAPIToWorkItem(*appl, source, target)
	// then: no error expected at this level, even though the title is missing
	require.Nil(t, err)
}
