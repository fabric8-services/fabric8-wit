package main

import (
	"testing"

	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
)

func TestNewWorkitemController(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	assert.Panics(t, func() {
		NewWorkitemController(goa.New("Test service"), nil, nil)
	})
}

func TestParseInts(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	integers, err := parseInts(nil)
	assert.Equal(t, nil, err)
	assert.Equal(t, []int{}, integers)

	str := "1, 2, foo"
	_, err = parseInts(&str)
	assert.NotNil(t, err)
}

func TestParseLimit(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// Test parsing error in parseInts
	str := "1000, foo"
	integers, length, err := parseLimit(&str)
	assert.NotNil(t, err)
	assert.Equal(t, 0, length)
	assert.Nil(t, integers)

	// Test length = 1
	str = "1000"
	integers, length, err = parseLimit(&str)
	assert.Nil(t, err)
	assert.Equal(t, 1000, length)
	assert.Nil(t, integers)

	// Test empty string
	str = ""
	integers, length, err = parseLimit(&str)
	assert.Nil(t, err)
	assert.Equal(t, 100, length)
	assert.Nil(t, integers)
}
