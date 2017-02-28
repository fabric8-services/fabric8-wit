package workitem_test

import (
	"testing"
	"time"

	"github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	"github.com/almighty/almighty-core/workitem"
	"github.com/stretchr/testify/assert"

	uuid "github.com/satori/go.uuid"
)

func TestWorkItem_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := workitem.WorkItem{
		ID:      0,
		Type:    "foo",
		Version: 0,
		Fields: workitem.Fields{
			"foo": "bar",
		},
		SpaceID: space.SystemSpace,
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
	h.Fields = workitem.Fields{}
	assert.False(t, a.Equal(h))

	// Test Space
	i := a
	i.SpaceID = uuid.NewV4()
	assert.False(t, a.Equal(i))

	j := workitem.WorkItem{
		ID:      0,
		Type:    "foo",
		Version: 0,
		Fields: workitem.Fields{
			"foo": "bar",
		},
		SpaceID: space.SystemSpace,
	}
	assert.True(t, a.Equal(j))
	assert.True(t, j.Equal(a))
}
