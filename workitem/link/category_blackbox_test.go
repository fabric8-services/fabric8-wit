package link_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/ptr"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

// TestWorkItemType_Equal Tests equality of two work item link categories
func TestWorkItemLinkCategory_EqualAndEqualValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	now := time.Now()
	description := "An example description"
	a := link.WorkItemLinkCategory{
		ID:          uuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573e231"),
		Name:        "Example work item link category",
		Description: &description,
		Version:     0,
		Lifecycle: gormsupport.Lifecycle{
			CreatedAt: now,
			UpdatedAt: now,
			DeletedAt: nil,
		},
	}

	t.Run("types", func(t *testing.T) {
		t.Parallel()
		b := convert.DummyEqualer{}
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})

	t.Run("version", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Version += 1
		require.False(t, a.Equal(b))
		require.True(t, a.EqualValue(b))
	})

	t.Run("name", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Name = "bar"
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})

	t.Run("description", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Description = ptr.String("bar")
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})

	t.Run("equality", func(t *testing.T) {
		t.Parallel()
		b := a
		require.True(t, a.Equal(b))
		require.True(t, a.EqualValue(b))
	})

	t.Run("id", func(t *testing.T) {
		t.Parallel()
		b := a
		b.ID = uuid.NewV4()
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})

	t.Run("lifecycle", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Lifecycle.CreatedAt = time.Now().Add(time.Hour * 10)
		require.False(t, a.Equal(b))
		require.True(t, a.EqualValue(b))
	})
}
