package link_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

// TestWorkItemType_Equal Tests equality of two work item link categories
func TestWorkItemLinkCategory_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	description := "An example description"
	a := link.WorkItemLinkCategory{
		ID:          uuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573e231"),
		Name:        "Example work item link category",
		Description: &description,
	}

	t.Run("types", func(t *testing.T) {
		// Test types
		b := convert.DummyEqualer{}
		require.False(t, a.Equal(b))
	})

	t.Run("version", func(t *testing.T) {
		c := a
		c.Version += 1
		require.False(t, a.Equal(c))
	})

	t.Run("name", func(t *testing.T) {
		c := a
		c.Name = "bar"
		require.False(t, a.Equal(c))
	})

	t.Run("description", func(t *testing.T) {
		otherDescription := "bar"
		c := a
		c.Description = &otherDescription
		require.False(t, a.Equal(c))
	})

	t.Run("equality", func(t *testing.T) {
		c := a
		require.True(t, a.Equal(c))
	})

	t.Run("id", func(t *testing.T) {
		c := a
		c.ID = uuid.FromStringOrNil("33371e36-871b-43a6-9166-0c4bd573e333")
		require.False(t, a.Equal(c))
	})

	t.Run("when one Description is nil", func(t *testing.T) {
		c := a
		c.Description = nil
		require.False(t, a.Equal(c))
	})
}
