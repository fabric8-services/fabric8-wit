package path_test

import (
	"strings"
	"testing"

	"fmt"

	"github.com/almighty/almighty-core/path"
	"github.com/almighty/almighty-core/resource"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsEmpty(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	lp := path.LtreePath{}
	require.True(t, lp.IsEmpty())
}

func TestThis(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	greatGrandParent := uuid.NewV4()
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.LtreePath{greatGrandParent, grandParent, immediateParent}
	assert.Equal(t, immediateParent, lp.This())

	lp2 := path.LtreePath{}
	require.Equal(t, uuid.Nil, lp2.This())
}

func TestConvert(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.LtreePath{grandParent, immediateParent}
	expected := fmt.Sprintf("%s.%s", grandParent, immediateParent)
	expected = strings.Replace(expected, "-", "_", -1)
	assert.Equal(t, expected, lp.Convert())

	lp2 := path.LtreePath{}
	require.Empty(t, lp2.Convert())
}

func TestToString(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.LtreePath{grandParent, immediateParent}
	expected := fmt.Sprintf("/%s/%s", grandParent, immediateParent)
	require.Equal(t, expected, lp.String())

	lp2 := path.LtreePath{}
	require.Equal(t, path.SepInService, lp2.String())
}

func TestRoot(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.LtreePath{grandParent, immediateParent}
	assert.Equal(t, path.LtreePath{grandParent}, lp.Root())

	lp2 := path.LtreePath{}
	assert.Equal(t, path.LtreePath{uuid.Nil}, lp2.Root())
}

func TestParent(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.LtreePath{grandParent, immediateParent}
	require.Equal(t, path.LtreePath{immediateParent}, lp.Parent())

	lp2 := path.LtreePath{}
	require.Equal(t, path.LtreePath{uuid.Nil}, lp2.Parent())
}

func TestValuerImplementation(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.LtreePath{grandParent, immediateParent}
	expected := fmt.Sprintf("%s.%s", grandParent, immediateParent)
	expected = strings.Replace(expected, "-", "_", -1)
	v, err := lp.Value()
	require.Nil(t, err)
	assert.Equal(t, expected, v)
}

func TestScannerImplementation(t *testing.T) {
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.LtreePath{grandParent, immediateParent}
	v, err := lp.Value()
	require.Nil(t, err)

	lp2 := path.LtreePath{}
	err2 := lp2.Scan([]byte(v.(string)))
	require.Nil(t, err2)
	require.Len(t, lp2, 2)
	assert.Equal(t, lp2, lp)
	assert.Equal(t, lp[0], lp2[0])

	lp3 := path.LtreePath{}
	err3 := lp2.Scan(nil)
	require.Nil(t, err3)
	assert.Len(t, lp3, 0)
}
