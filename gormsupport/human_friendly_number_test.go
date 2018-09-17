package gormsupport

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/convert"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestHumanFriendlyNumber_Equal(t *testing.T) {
	t.Parallel()
	a := HumanFriendlyNumber{
		Number:    1,
		spaceID:   uuid.NewV4(),
		tableName: "work_items",
	}
	t.Run("equality", func(t *testing.T) {
		t.Parallel()
		b := convert.DummyEqualer{}
		assert.False(t, a.Equal(b))
	})
	t.Run("equality", func(t *testing.T) {
		t.Parallel()
		b := a
		assert.True(t, a.Equal(b))
	})
	t.Run("number", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Number = 567
		assert.False(t, a.Equal(b))
	})
	t.Run("table name", func(t *testing.T) {
		t.Parallel()
		b := a
		b.tableName = "iterations"
		assert.False(t, a.Equal(b))
	})
	t.Run("space id", func(t *testing.T) {
		t.Parallel()
		b := a
		b.spaceID = uuid.NewV4()
		assert.False(t, a.Equal(b))
	})
}
