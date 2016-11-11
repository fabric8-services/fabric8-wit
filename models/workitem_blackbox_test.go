package models_test

import (
	"testing"
	"time"

	"github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

func TestWorkItem_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := models.WorkItem{
		ID:      0,
		Type:    "foo",
		Version: 0,
		Fields: models.Fields{
			"foo": "bar",
		},
	}

	// Test type difference
	b := convert.DummyEqualer{}
	assert.False(t, a.Equal(b))

	// Test lifecycle difference
	c := a
	c.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
	assert.False(t, a.Equal(c))

	// Test type difference
	d := a
	d.Type = "something else"
	assert.False(t, a.Equal(d))

	// Test version difference
	e := a
	e.Version += 1
	assert.False(t, a.Equal(e))

	// Test version difference
	f := a
	f.Version += 1
	assert.False(t, a.Equal(f))

	// Test ID difference
	g := a
	g.ID = 1
	assert.False(t, a.Equal(g))

	// Test fields difference
	h := a
	h.Fields = models.Fields{}
	assert.False(t, a.Equal(h))
}
