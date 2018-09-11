package link_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

// TestWorkItemType_Equal Tests equality of two work item link types
func TestWorkItemLinkType_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	description := "An example description"
	a := link.WorkItemLinkType{
		ID:                 uuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573e231"),
		Name:               "Example work item link category",
		Description:        &description,
		Topology:           link.TopologyNetwork,
		Version:            0,
		ForwardName:        "blocks",
		ForwardDescription: ptr.String("description for forward direction"),
		ReverseName:        "blocked by",
		ReverseDescription: ptr.String("description for reverse direction"),
		LinkCategoryID:     uuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573eAAA"),
		SpaceTemplateID:    uuid.FromStringOrNil("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
	}

	t.Run("equality", func(t *testing.T) {
		b := a
		require.True(t, a.Equal(b))
	})

	t.Run("types", func(t *testing.T) {
		b := convert.DummyEqualer{}
		require.False(t, a.Equal(b))
	})

	t.Run("ID", func(t *testing.T) {
		b := a
		b.ID = uuid.FromStringOrNil("CCC71e36-871b-43a6-9166-0c4bd573eCCC")
		require.False(t, a.Equal(b))
	})

	t.Run("version", func(t *testing.T) {
		b := a
		b.Version += 1
		require.False(t, a.Equal(b))
	})

	t.Run("name", func(t *testing.T) {
		b := a
		b.Name = "bar"
		require.False(t, a.Equal(b))
	})

	t.Run("description", func(t *testing.T) {
		b := a
		b.Description = ptr.String("bar")
		require.False(t, a.Equal(b))
	})

	t.Run("topology", func(t *testing.T) {
		b := a
		b.Topology = link.TopologyTree
		require.False(t, a.Equal(b))
	})

	t.Run("forward name", func(t *testing.T) {
		b := a
		b.ForwardName = "go, go, go!"
		require.False(t, a.Equal(b))
	})

	t.Run("forward description", func(t *testing.T) {
		b := a
		b.ForwardDescription = ptr.String("another forward description")
		require.False(t, a.Equal(b))
	})

	t.Run("reverse name", func(t *testing.T) {
		b := a
		b.ReverseName = "backup, backup!"
		require.False(t, a.Equal(b))
	})

	t.Run("reverse description", func(t *testing.T) {
		b := a
		b.ReverseDescription = ptr.String("another reverse description")
		require.False(t, a.Equal(b))
	})

	t.Run("link category", func(t *testing.T) {
		b := a
		b.LinkCategoryID = uuid.FromStringOrNil("aaa71e36-871b-43a6-9166-0c4bd573eCCC")
		require.False(t, a.Equal(b))
	})

	t.Run("space template", func(t *testing.T) {
		b := a
		b.SpaceTemplateID = uuid.FromStringOrNil("aaa71e36-871b-43a6-9166-0v5ce684dBBB")
		require.False(t, a.Equal(b))
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
		LinkCategoryID:  uuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573eAAA"),
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

	t.Run("empty link cat ID", func(t *testing.T) {
		b := a
		b.LinkCategoryID = uuid.Nil
		require.NotNil(t, b.CheckValidForCreation())
	})

	t.Run("empty space template ID", func(t *testing.T) {
		b := a
		b.SpaceTemplateID = uuid.Nil
		require.NotNil(t, b.CheckValidForCreation())
	})
}
