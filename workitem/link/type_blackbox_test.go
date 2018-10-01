package link_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/gormsupport"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

// TestWorkItemType_Equal Tests equality of two work item link types
func TestWorkItemLinkType_EqualAndEqualValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	now := time.Now()
	description := "An example description"
	a := link.WorkItemLinkType{
		ID:                 uuid.NewV4(),
		Name:               "Example work item link category",
		Description:        &description,
		Topology:           link.TopologyNetwork,
		Version:            0,
		ForwardName:        "blocks",
		ForwardDescription: ptr.String("description for forward direction"),
		ReverseName:        "blocked by",
		ReverseDescription: ptr.String("description for reverse direction"),
		SpaceTemplateID:    uuid.NewV4(),
		Lifecycle: gormsupport.Lifecycle{
			CreatedAt: now,
			UpdatedAt: now,
			DeletedAt: nil,
		},
	}

	t.Run("equality", func(t *testing.T) {
		t.Parallel()
		b := a
		require.True(t, a.Equal(b))
		require.True(t, a.EqualValue(b))
	})

	t.Run("types", func(t *testing.T) {
		t.Parallel()
		b := convert.DummyEqualer{}
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})

	t.Run("ID", func(t *testing.T) {
		t.Parallel()
		b := a
		b.ID = uuid.NewV4()
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

	t.Run("lifecycle", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Lifecycle.CreatedAt = time.Now().Add(time.Hour * 10)
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

	t.Run("topology", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Topology = link.TopologyTree
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})

	t.Run("forward name", func(t *testing.T) {
		t.Parallel()
		b := a
		b.ForwardName = "go, go, go!"
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})

	t.Run("forward description", func(t *testing.T) {
		t.Parallel()
		b := a
		b.ForwardDescription = ptr.String("another forward description")
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})

	t.Run("reverse name", func(t *testing.T) {
		t.Parallel()
		b := a
		b.ReverseName = "backup, backup!"
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})

	t.Run("reverse description", func(t *testing.T) {
		t.Parallel()
		b := a
		b.ReverseDescription = ptr.String("my new description")
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})

	t.Run("space template", func(t *testing.T) {
		t.Parallel()
		b := a
		b.SpaceTemplateID = uuid.NewV4()
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})
}

func TestWorkItemLinkTypeCheckValidForCreation(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	description := "An example description"
	a := link.WorkItemLinkType{
		ID:              uuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573e231"),
		Name:            "Example work item link category",
		Description:     &description,
		Topology:        link.TopologyNetwork,
		Version:         0,
		ForwardName:     "blocks",
		ReverseName:     "blocked by",
		SpaceTemplateID: uuid.FromStringOrNil("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
	}

	t.Run("check valid", func(t *testing.T) {
		b := a
		require.Nil(t, b.CheckValidForCreation())
	})

	t.Run("check empty name", func(t *testing.T) {
		b := a
		b.Name = ""
		require.NotNil(t, b.CheckValidForCreation())
	})

	t.Run("empty forward name", func(t *testing.T) {
		b := a
		b.ForwardName = ""
		require.NotNil(t, b.CheckValidForCreation())
	})

	t.Run("empty reverse name", func(t *testing.T) {
		b := a
		b.ReverseName = ""
		require.NotNil(t, b.CheckValidForCreation())
	})

	t.Run("empty topology", func(t *testing.T) {
		b := a
		b.Topology = link.Topology("")
		require.NotNil(t, b.CheckValidForCreation())
	})

	t.Run("empty space template ID", func(t *testing.T) {
		b := a
		b.SpaceTemplateID = uuid.Nil
		require.NotNil(t, b.CheckValidForCreation())
	})
}
