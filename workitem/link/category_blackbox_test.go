package link_test

import (
	"testing"

	"time"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/test/resource"
	"github.com/almighty/almighty-core/util"
	"github.com/almighty/almighty-core/workitem/link"
	satoriuuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

// TestWorkItemType_Equal Tests equality of two work item link categories
func TestWorkItemLinkCategory_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	description := "An example description"
	a := link.WorkItemLinkCategory{
		ID:          satoriuuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573e231"),
		Name:        "Example work item link category",
		Description: &description,
		Version:     0,
	}

	// Test types
	b := util.DummyEqualer{}
	require.False(t, a.Equal(b))

	// Test lifecycle
	c := a
	c.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
	require.False(t, a.Equal(c))

	// Test version
	c = a
	c.Version += 1
	require.False(t, a.Equal(c))

	// Test name
	c = a
	c.Name = "bar"
	require.False(t, a.Equal(c))

	// Test description
	otherDescription := "bar"
	c = a
	c.Description = &otherDescription
	require.False(t, a.Equal(c))

	// Test equality
	c = a
	require.True(t, a.Equal(c))

	// Test ID
	c = a
	c.ID = satoriuuid.FromStringOrNil("33371e36-871b-43a6-9166-0c4bd573e333")
	require.False(t, a.Equal(c))

	// Test when one Description is nil
	c = a
	c.Description = nil
	require.False(t, a.Equal(c))
}

func TestWorkItemLinkCategory_ConvertLinkCategoryFromModel(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	description := "An example description"
	m := link.WorkItemLinkCategory{
		ID:          satoriuuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573e231"),
		Name:        "Example work item link category",
		Description: &description,
		Version:     0,
	}

	expected := app.WorkItemLinkCategorySingle{
		Data: &app.WorkItemLinkCategoryData{
			Type: link.EndpointWorkItemLinkCategories,
			ID:   &m.ID,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &m.Name,
				Description: m.Description,
				Version:     &m.Version,
			},
		},
	}

	actual := link.ConvertLinkCategoryFromModel(m)
	require.Equal(t, expected.Data.Type, actual.Data.Type)
	require.Equal(t, *expected.Data.ID, *actual.Data.ID)
	require.Equal(t, *expected.Data.Attributes.Name, *actual.Data.Attributes.Name)
	require.Equal(t, *expected.Data.Attributes.Description, *actual.Data.Attributes.Description)
	require.Equal(t, *expected.Data.Attributes.Version, *actual.Data.Attributes.Version)
}
