package workitem_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestWorkItem_EqualAndEqualValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	now := time.Now()
	spaceID := uuid.NewV4()
	a := workitem.WorkItemStorage{
		ID:      uuid.NewV4(),
		Number:  1,
		Type:    uuid.NewV4(),
		Version: 0,
		Fields: workitem.Fields{
			"foo": "bar",
		},
		SpaceID:                spaceID,
		ExecutionOrder:         111,
		RelationShipsChangedAt: ptr.Time(now),
	}

	t.Run("no equaler", func(t *testing.T) {
		t.Parallel()
		b := convert.DummyEqualer{}
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})

	t.Run("lifecycle", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
		assert.False(t, a.Equal(b))
		assert.True(t, a.EqualValue(b))
	})

	t.Run("type", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Type = uuid.NewV4()
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})

	t.Run("version", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Version += 1
		assert.False(t, a.Equal(b))
		assert.True(t, a.EqualValue(b))
	})

	t.Run("id", func(t *testing.T) {
		t.Parallel()
		b := a
		b.ID = uuid.NewV4()
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})

	t.Run("number", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Number = 42
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})

	t.Run("fields", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Fields = workitem.Fields{}
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})

	t.Run("space", func(t *testing.T) {
		t.Parallel()
		b := a
		b.SpaceID = uuid.NewV4()
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})

	t.Run("execution order", func(t *testing.T) {
		t.Parallel()
		b := a
		b.ExecutionOrder = 123
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})

	t.Run("relationships changed at", func(t *testing.T) {
		t.Parallel()
		b := a
		b.RelationShipsChangedAt = ptr.Time(time.Now().Add(time.Hour * 10))
		assert.False(t, a.Equal(b))
		assert.True(t, a.EqualValue(b))
	})

	t.Run("equality", func(t *testing.T) {
		t.Parallel()
		b := workitem.WorkItemStorage{
			ID:             a.ID,
			Type:           a.Type,
			Number:         1,
			ExecutionOrder: 111,
			Version:        0,
			Fields: workitem.Fields{
				"foo": "bar",
			},
			SpaceID:                spaceID,
			RelationShipsChangedAt: ptr.Time(now),
		}
		assert.True(t, a.Equal(b))
		assert.True(t, b.Equal(a))
		assert.True(t, a.EqualValue(b))
		assert.True(t, b.EqualValue(a))
	})
}
