package criteria

import (
	"testing"
	"github.com/almighty/almighty-core/test"
)

func TestGetParent(t *testing.T) {
	test.SkiptTestIfNotUnitTest(t)
	l := Field("a")
	r := Literal(5)
	expr := Equals(l, r)
	if l.Parent() != expr {
		t.Errorf("parent should be %v, but is %v", expr, l.Parent())
	}
}
