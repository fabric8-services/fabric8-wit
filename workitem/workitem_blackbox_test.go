package workitem_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/numbersequence"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestWorkItem_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	spaceID := uuid.NewV4()
	a := workitem.WorkItemStorage{
		HumanFriendlyNumber: numbersequence.NewHumanFriendlyNumber(spaceID, workitem.WorkItemStorage{}.TableName(), 1),
		ID:                  uuid.NewV4(),
		Type:                uuid.NewV4(),
		Version:             0,
		Fields: workitem.Fields{
			"foo": "bar",
		},
		SpaceID: spaceID,
	}

	t.Run("no equaler", func(t *testing.T) {
		t.Parallel()
		b := convert.DummyEqualer{}
		assert.False(t, a.Equal(b))
	})

	t.Run("lifecycle", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
		assert.False(t, a.Equal(b))
	})

	t.Run("type", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Type = uuid.NewV4()
		assert.False(t, a.Equal(b))
	})

	t.Run("version", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Version++
		assert.False(t, a.Equal(b))
	})

	t.Run("id", func(t *testing.T) {
		t.Parallel()
		b := a
		b.ID = uuid.NewV4()
		assert.False(t, a.Equal(b))
	})

	t.Run("number", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Number = 42
		assert.False(t, a.Equal(b))
	})

	t.Run("fields", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Fields = workitem.Fields{}
		assert.False(t, a.Equal(b))
	})

	t.Run("space", func(t *testing.T) {
		t.Parallel()
		b := a
		b.SpaceID = uuid.NewV4()
		assert.False(t, a.Equal(b))
	})

	t.Run("equality", func(t *testing.T) {
		t.Parallel()
		b := workitem.WorkItemStorage{
			HumanFriendlyNumber: numbersequence.NewHumanFriendlyNumber(spaceID, workitem.WorkItemStorage{}.TableName(), 1),
			ID:                  a.ID,
			Type:                a.Type,
			Version:             0,
			Fields: workitem.Fields{
				"foo": "bar",
			},
			SpaceID: spaceID,
		}
		assert.True(t, a.Equal(b))
		assert.True(t, b.Equal(a))
	})
}
