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
var plannerTestIssues2 = uuid.FromStringOrNil("5cccf59d-fed6-4c4a-a420-eebe83fe09e1")

const plannerTestIssuesName = "planner.testIssues"

func TestCategory_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := category.Category{
		ID:   plannerTestIssues1,
		Name: "planner.testIssues",
	}

	// Test type difference
	b := convert.DummyEqualer{}
	assert.False(t, a.Equal(b))

	// Test lifecycle difference
	c := a
	c.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
	assert.False(t, a.Equal(c))

	// Test ID difference
	g := a
	g.ID = plannerTestIssues2
	assert.False(t, a.Equal(g))

	// Test Name difference
	h := a
	h.Name = plannerTestIssuesName
	assert.False(t, a.Equal(h))

	j := category.Category{
		ID:   plannerTestIssues1,
		Name: "planner.testIssues",
	}
	assert.True(t, a.Equal(j))
	assert.True(t, j.Equal(a))
}
