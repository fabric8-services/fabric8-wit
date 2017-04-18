package category_test

import (
	"testing"
	"time"

	"github.com/almighty/almighty-core/category"
	"github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/resource"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

var plannerTestIssues1 = uuid.FromStringOrNil("5cccf59d-fed6-4c4a-a420-eebe83fe09e1")
var plannerTestIssues2 = uuid.FromStringOrNil("c1729f3d-f7d3-408f-8e36-5bb27933b0b8")

const plannerTestIssuesName = "planner.testIssues"

func TestCategory_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := category.Category{
		ID:   plannerTestIssues1,
		Name: "planner.testIssues1",
	}

	// Test type difference
	b := convert.DummyEqualer{}
	assert.False(t, a.Equal(b))

	// Test lifecycle difference
	c := a
	c.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
	assert.False(t, a.Equal(c))

	// Test ID difference
	d := a
	d.ID = plannerTestIssues2
	assert.False(t, a.Equal(d))

	// Test Name difference
	e := a
	e.Name = plannerTestIssuesName
	assert.False(t, a.Equal(e))

	f := category.Category{
		ID:   a.ID,
		Name: a.Name,
	}
	assert.True(t, a.Equal(f))
	assert.True(t, f.Equal(a))
}
