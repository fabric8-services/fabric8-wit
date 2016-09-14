package models_test

import (
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestLifecycle_Equal(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	now := time.Now()
	nowPlus := time.Now().Add(time.Duration(1000))

	a := models.Lifecycle{
		CreatedAt: now,
		UpdatedAt: now,
		DeletedAt: nil,
	}

	// Test for type difference
	b := models.DummyEqualer{}
	assert.False(t, a.Equal(b))

	// Test CreateAt difference
	c := models.Lifecycle{
		CreatedAt: nowPlus,
		UpdatedAt: now,
		DeletedAt: nil,
	}
	assert.False(t, a.Equal(c))

	// Test UpdatedAt difference
	d := models.Lifecycle{
		CreatedAt: now,
		UpdatedAt: nowPlus,
		DeletedAt: nil,
	}
	assert.False(t, a.Equal(d))

	// Test DeletedAt (one is not nil, the other is) difference
	e := models.Lifecycle{
		CreatedAt: now,
		UpdatedAt: now,
		DeletedAt: &now,
	}
	assert.False(t, a.Equal(e))

	// Test DeletedAt (both are not nil) difference
	g := models.Lifecycle{
		CreatedAt: now,
		UpdatedAt: nowPlus,
		DeletedAt: &now,
	}
	h := models.Lifecycle{
		CreatedAt: now,
		UpdatedAt: nowPlus,
		DeletedAt: &nowPlus,
	}
	assert.False(t, g.Equal(h))
}
