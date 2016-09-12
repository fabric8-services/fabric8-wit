package models

import (
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestWorkItemNotEqual(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	a := WorkItem{
		ID:      0,
		Type:    "foo",
		Version: 0,
		Fields: Fields{
			"foo": "bar",
		},
	}

	// Test type difference
	b := DummyEqualer{}
	assert.False(t, a.Equal(b))

	// Test lifecycle difference
	c := a
	c.Lifecycle = Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
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

	// Test fields difference
	g := a
	g.Fields = Fields{}
	assert.False(t, a.Equal(g))
}
