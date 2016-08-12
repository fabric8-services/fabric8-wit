package criteria

import (
	"fmt"
	"testing"
)

func TestGetParent(t *testing.T) {
	l := Field("a")
	r := Literal(5)
	expr := Equals(l, r)
	if l.Parent() != expr {
		t.Errorf("parent should be %v, but is %v", expr, l.Parent())
	}
}

type Parent interface{}

type Parentable interface {
	Parent() Parent
	setParent(Parent)
}

type simple struct {
	parent Parent
}

func (exp *simple) Parent() Parent {
	return exp.parent
}

func (exp *simple) setParent(parent Parent) {
	exp.parent = parent
}

func TestSimple(t *testing.T) {

	p := simple{}
	c := simple{}
	c.setParent(p)
	if c.Parent() != p {
		t.Errorf("boingo")
	}
	fmt.Println("bla")
}
