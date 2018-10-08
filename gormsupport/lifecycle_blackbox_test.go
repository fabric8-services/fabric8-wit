package gormsupport_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/assert"
)

func TestLifecycle_EqualAndEqualValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// given
	now := time.Now()
	a := gormsupport.Lifecycle{
		CreatedAt: now,
		UpdatedAt: now,
		DeletedAt: nil,
	}

	t.Run("equality", func(t *testing.T) {
		t.Parallel()
		assert.True(t, a.Equal(a))
		assert.True(t, a.EqualValue(a))
	})

	t.Run("type difference", func(t *testing.T) {
		t.Parallel()
		b := convert.DummyEqualer{}
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})

	t.Run("created at", func(t *testing.T) {
		t.Parallel()
		b := gormsupport.Lifecycle{
			CreatedAt: now.Add(time.Duration(1000)),
			UpdatedAt: now,
			DeletedAt: nil,
		}
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})

	t.Run("updated at", func(t *testing.T) {
		t.Parallel()
		b := gormsupport.Lifecycle{
			CreatedAt: now,
			UpdatedAt: now.Add(time.Duration(1000)),
			DeletedAt: nil,
		}
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})

	t.Run("deleted at", func(t *testing.T) {
		t.Parallel()
		b := gormsupport.Lifecycle{
			CreatedAt: now,
			UpdatedAt: now,
			DeletedAt: &now,
		}
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})
}
