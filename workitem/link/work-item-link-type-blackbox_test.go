package link_test

import (
	"testing"

	"time"

	"github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/workitem"
	"github.com/almighty/almighty-core/workitem/link"
	satoriuuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

// TestWorkItemType_Equal Tests equality of two work item link types
func TestWorkItemLinkType_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	description := "An example description"
	a := link.WorkItemLinkType{
		ID:             satoriuuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573e231"),
		Name:           "Example work item link category",
		Description:    &description,
		Topology:       "network",
		Version:        0,
		SourceTypeName: workitem.SystemBug,
		TargetTypeName: workitem.SystemUserStory,
		ForwardName:    "blocks",
		ReverseName:    "blocked by",
		LinkCategoryID: satoriuuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573eAAA"),
	}

	// Test equality
	b := a
	require.True(t, a.Equal(b))

	// Test types
	c := convert.DummyEqualer{}
	require.False(t, a.Equal(c))

	// Test lifecycle
	b = a
	b.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
	require.False(t, a.Equal(b))

	// Test ID
	b = a
	b.ID = satoriuuid.FromStringOrNil("CCC71e36-871b-43a6-9166-0c4bd573eCCC")
	require.False(t, a.Equal(b))

	// Test Version
	b = a
	b.Version += 1
	require.False(t, a.Equal(b))

	// Test Name
	b = a
	b.Name = "bar"
	require.False(t, a.Equal(b))

	// Test Description
	otherDescription := "bar"
	b = a
	b.Description = &otherDescription
	require.False(t, a.Equal(b))

	// Test Topology
	b = a
	b.Topology = "tree"
	require.False(t, a.Equal(b))

	// Test SourceTypeName
	b = a
	b.SourceTypeName = "foobar"
	require.False(t, a.Equal(b))

	// Test TargetTypeName
	b = a
	b.TargetTypeName = "fooooobar"
	require.False(t, a.Equal(b))

	// Test ForwardName
	b = a
	b.ForwardName = "go, go, go!"
	require.False(t, a.Equal(b))

	// Test ReverseName
	b = a
	b.ReverseName = "backup, backup!"
	require.False(t, a.Equal(b))

	// Test LinkCategoryID
	b = a
	b.LinkCategoryID = satoriuuid.FromStringOrNil("aaa71e36-871b-43a6-9166-0c4bd573eCCC")
	require.False(t, a.Equal(b))
}

func TestWorkItemLinkTypeCheckValidForCreation(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	description := "An example description"
	a := link.WorkItemLinkType{
		ID:             satoriuuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573e231"),
		Name:           "Example work item link category",
		Description:    &description,
		Topology:       link.TopologyNetwork,
		Version:        0,
		SourceTypeName: workitem.SystemBug,
		TargetTypeName: workitem.SystemUserStory,
		ForwardName:    "blocks",
		ReverseName:    "blocked by",
		LinkCategoryID: satoriuuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573eAAA"),
	}

	// Check valid
	b := a
	require.Nil(t, b.CheckValidForCreation())

	// Check empty Name
	b = a
	b.Name = ""
	require.NotNil(t, b.CheckValidForCreation())

	// Check empty SourceTypeName
	b = a
	b.SourceTypeName = ""
	require.NotNil(t, b.CheckValidForCreation())

	// Check empty TargetTypeName
	b = a
	b.TargetTypeName = ""
	require.NotNil(t, b.CheckValidForCreation())

	// Check empty ForwardName
	b = a
	b.ForwardName = ""
	require.NotNil(t, b.CheckValidForCreation())

	// Check empty ReverseName
	b = a
	b.ReverseName = ""
	require.NotNil(t, b.CheckValidForCreation())

	// Check empty Topology
	b = a
	b.Topology = ""
	require.NotNil(t, b.CheckValidForCreation())

	// Check empty LinkCategoryID
	b = a
	b.LinkCategoryID = satoriuuid.Nil
	require.NotNil(t, b.CheckValidForCreation())
}
