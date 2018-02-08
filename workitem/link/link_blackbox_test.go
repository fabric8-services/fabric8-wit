package link_test

import (
	"testing"

	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

// TestWorkItemLink_Equal Tests equality of two work item links
func TestWorkItemLink_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := link.WorkItemLink{
		ID:         uuid.NewV4(),
		SourceID:   uuid.NewV4(),
		TargetID:   uuid.NewV4(),
		LinkTypeID: uuid.NewV4(),
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
	b.ID = uuid.NewV4()
	require.False(t, a.Equal(b))

	// Test Version
	b = a
	b.Version += 1
	require.False(t, a.Equal(b))

	// Test SourceID
	b = a
	b.SourceID = uuid.NewV4()
	require.False(t, a.Equal(b))

	// Test TargetID
	b = a
	b.TargetID = uuid.NewV4()
	require.False(t, a.Equal(b))

	// Test LinkTypeID
	b = a
	b.LinkTypeID = uuid.NewV4()
	require.False(t, a.Equal(b))
}

func TestWorkItemLinkCheckValidForCreation(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := link.WorkItemLink{
		ID:         uuid.NewV4(),
		SourceID:   uuid.NewV4(),
		TargetID:   uuid.NewV4(),
		LinkTypeID: uuid.NewV4(),
	}

	// Check valid
	b := a
	require.Nil(t, b.CheckValidForCreation())

	// Check empty LinkTypeID
	b = a
	b.LinkTypeID = uuid.Nil
	require.NotNil(t, b.CheckValidForCreation())
}

func TestWorkItemLinkList(t *testing.T) {
	t.Parallel()

	x := uuid.NewV4()
	y := uuid.NewV4()
	z := uuid.NewV4()

	linkTypeID1 := uuid.NewV4()
	linkTypeID2 := uuid.NewV4()

	a := link.WorkItemLink{
		ID:         uuid.NewV4(),
		SourceID:   x,
		TargetID:   y,
		LinkTypeID: linkTypeID1,
	}

	b := link.WorkItemLink{
		ID:         uuid.NewV4(),
		SourceID:   y,
		TargetID:   z,
		LinkTypeID: linkTypeID1,
	}

	c := link.WorkItemLink{
		ID:         uuid.NewV4(),
		SourceID:   x,
		TargetID:   z,
		LinkTypeID: linkTypeID1,
	}

	d := link.WorkItemLink{
		ID:         uuid.NewV4(),
		SourceID:   z,
		TargetID:   y,
		LinkTypeID: linkTypeID2,
	}

	list := link.WorkItemLinkList{a, b, c, d}

	t.Run("GetParentIDOf", func(t *testing.T) {
		require.Equal(t, x, list.GetParentIDOf(y, linkTypeID1))
	})

	t.Run("GetDistinctListOfTargetIDs", func(t *testing.T) {
		toBeFound := id.Slice{y, z}.ToMap()
		actual := list.GetDistinctListOfTargetIDs(linkTypeID1)
		for _, ID := range actual {
			_, ok := toBeFound[ID]
			require.True(t, ok, "found unexpected ID: %s", ID)
			delete(toBeFound, ID)
		}
		require.Empty(t, toBeFound, "failed to find these IDs: %+v", toBeFound)
	})
}
