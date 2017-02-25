package path_test

import (
	"strings"
	"testing"

	"fmt"

	"github.com/almighty/almighty-core/path"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

func TestIsEmpty(t *testing.T) {
	lp := path.LtreePath{}
	require.True(t, lp.IsEmpty())
}

func TestThis(t *testing.T) {
	greatGrandParent := uuid.NewV4()
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.LtreePath{greatGrandParent, grandParent, immediateParent}
	require.Equal(t, immediateParent, lp.This())

	lp2 := path.LtreePath{}
	require.Equal(t, uuid.Nil, lp2.This())
}

func TestConvert(t *testing.T) {
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.LtreePath{grandParent, immediateParent}
	expected := fmt.Sprintf("%s.%s", grandParent, immediateParent)
	expected = strings.Replace(expected, "-", "_", -1)
	require.Equal(t, expected, lp.Convert())

	lp2 := path.LtreePath{}
	require.Empty(t, lp2.Convert())
}

func TestToString(t *testing.T) {
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.LtreePath{grandParent, immediateParent}
	expected := fmt.Sprintf("/%s/%s", grandParent, immediateParent)
	require.Equal(t, expected, lp.String())

	lp2 := path.LtreePath{}
	require.Equal(t, path.SepInService, lp2.String())
}

func TestRoot(t *testing.T) {
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.LtreePath{grandParent, immediateParent}
	require.Equal(t, path.LtreePath{grandParent}, lp.Root())

	lp2 := path.LtreePath{}
	require.Equal(t, path.LtreePath{uuid.Nil}, lp2.Root())
}

func TestParent(t *testing.T) {
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.LtreePath{grandParent, immediateParent}
	require.Equal(t, path.LtreePath{immediateParent}, lp.Parent())

	lp2 := path.LtreePath{}
	require.Equal(t, path.LtreePath{uuid.Nil}, lp2.Parent())
}
