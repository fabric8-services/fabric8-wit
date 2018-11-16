package link_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/gormsupport"

	"github.com/fabric8-services/fabric8-common/id"
	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

// TestWorkItemLink_Equal Tests equality of two work item links
func TestWorkItemLink_EqualAndEqualValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	now := time.Now()
	a := link.WorkItemLink{
		ID:         uuid.NewV4(),
		SourceID:   uuid.NewV4(),
		TargetID:   uuid.NewV4(),
		LinkTypeID: uuid.NewV4(),
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

	t.Run("source", func(t *testing.T) {
		t.Parallel()
		b := a
		b.SourceID = uuid.NewV4()
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})

	t.Run("target", func(t *testing.T) {
		t.Parallel()
		b := a
		b.TargetID = uuid.NewV4()
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})

	t.Run("link type", func(t *testing.T) {
		t.Parallel()
		b := a
		b.LinkTypeID = uuid.NewV4()
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})
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
