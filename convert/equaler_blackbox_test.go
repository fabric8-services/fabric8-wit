package convert_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// foo implements the Equaler interface
type foo struct{}

// Ensure foo implements the Equaler interface
var _ convert.Equaler = foo{}
var _ convert.Equaler = (*foo)(nil)

func (f foo) Equal(u convert.Equaler) bool {
	_, ok := u.(foo)
	return ok
}

func (f foo) EqualValue(u convert.Equaler) bool {
	_, ok := u.(foo)
	return ok
}

func Test_EqualValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := convert.DummyEqualer{}
	b := convert.DummyEqualer{}

	// Test for type difference
	assert.False(t, a.EqualValue(foo{}))

	// Test for equality
	assert.True(t, a.EqualValue(b))
}

func Test_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := convert.DummyEqualer{}
	b := convert.DummyEqualer{}

	// Test for type difference
	assert.False(t, a.Equal(foo{}))

	// Test for equality
	assert.True(t, a.Equal(b))
}

func Test_EqualAndEqualValue(t *testing.T) {
	now := time.Now()
	nowPlus := now.Add(time.Hour * 10)

	p1 := person{Name: "Jimi Hendrix", CreatedAt: now}
	// Change time on top-level
	p2 := person{Name: "Jimi Hendrix", CreatedAt: nowPlus}
	m1 := musician{person: p1, Instrument: "Guitar", UpdatedAt: now}
	// Change time on top-level
	m2 := musician{person: p1, Instrument: "Guitar", UpdatedAt: nowPlus}
	// Change time on lower-level embedded struct
	m3 := musician{person: p2, Instrument: "Guitar", UpdatedAt: now}

	t.Run("test equality with itself", func(t *testing.T) {
		require.True(t, p1.Equal(p1))
		require.True(t, p1.EqualValue(p1))
		require.True(t, convert.CascadeEqual(p1, p1))

		require.True(t, m1.Equal(m1))
		require.True(t, m1.EqualValue(m1))
		require.True(t, convert.CascadeEqual(m1, m1))
	})

	t.Run("test equality when something on top-level is changed", func(t *testing.T) {
		require.False(t, p1.Equal(p2))
		require.True(t, p1.EqualValue(p2))
		require.False(t, convert.CascadeEqual(p1, p2))

		require.False(t, m1.Equal(m2))
		require.True(t, m1.EqualValue(m2))
		require.False(t, convert.CascadeEqual(m1, m2))
	})

	t.Run("test equality when something on lower-level is changed", func(t *testing.T) {
		require.False(t, m1.Equal(m3))
		require.True(t, m1.EqualValue(m3))
		require.False(t, convert.CascadeEqual(m1, m3))
	})
}

type person struct {
	Name      string
	CreatedAt time.Time // Will be ignored by EqualValue
}
type musician struct {
	person
	Instrument string
	UpdatedAt  time.Time // Will be ignored by EqualValue
}

func (f person) Equal(u convert.Equaler) bool {
	other, ok := u.(person)
	if !ok {
		return false
	}
	if f.Name != other.Name {
		return false
	}
	if f.CreatedAt != other.CreatedAt {
		return false
	}
	return true
}

func (f person) EqualValue(u convert.Equaler) bool {
	other, ok := u.(person)
	if !ok {
		return false
	}
	f.CreatedAt = other.CreatedAt
	return f.Equal(u)
}

func (f musician) Equal(u convert.Equaler) bool {
	other, ok := u.(musician)
	if !ok {
		return false
	}
	if f.Instrument != other.Instrument {
		return false
	}
	if f.UpdatedAt != other.UpdatedAt {
		return false
	}
	if !convert.CascadeEqual(f.person, other.person) {
		return false
	}
	return true
}

func (f musician) EqualValue(u convert.Equaler) bool {
	other, ok := u.(musician)
	if !ok {
		return false
	}
	f.UpdatedAt = other.UpdatedAt
	return f.Equal(u)
}
