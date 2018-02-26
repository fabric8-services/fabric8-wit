package workitem

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/require"
)

func Test_TableJoin_HandlesFieldName(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// given
	j := TableJoin{
		tableName:        "iterations",
		tableAlias:       "iter",
		on:               JoinOnJSONField(SystemIteration, "iter.ID"),
		prefixActivators: []string{"iteration."},
	}
	t.Run("has prefix", func(t *testing.T) {
		t.Parallel()
		require.True(t, j.HandlesFieldName(j.prefixActivators[0]+"foobar"))
	})
	t.Run("missing prefix", func(t *testing.T) {
		t.Parallel()
		require.False(t, j.HandlesFieldName("foo.bar"))
	})
}

func Test_JoinOnJSONField(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// when
	actual := JoinOnJSONField("system.iteration", "iter.ID")
	// then
	require.Equal(t, `fields@> concat('{"system.iteration": "', iter.ID, '"}')::jsonb`, actual)
}

func Test_TableJoin_Activate(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// given
	j := TableJoin{active: false}
	require.False(t, j.IsActive())
	// when
	j.Activate()
	// then
	require.True(t, j.active)
	require.True(t, j.IsActive())
}

func Test_TableJoin_String(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// given
	j := TableJoin{
		tableName:        "iterations",
		tableAlias:       "iter",
		on:               JoinOnJSONField(SystemIteration, "iter.ID"),
		prefixActivators: []string{"iteration."},
	}
	// when
	s := j.String()
	// then
	require.Equal(t, "LEFT JOIN "+j.tableName+" "+j.tableAlias+" ON "+j.on, s)
}

func Test_TableJoin_TranslateFieldName(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// given
	j := TableJoin{
		tableName:        "iterations",
		tableAlias:       "iter",
		on:               JoinOnJSONField(SystemIteration, "iter.ID"),
		prefixActivators: []string{"iteration."},
	}
	t.Run("missing prefix", func(t *testing.T) {
		t.Parallel()
		// when
		col, err := j.TranslateFieldName("foo.bar")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("empty locator", func(t *testing.T) {
		t.Parallel()
		// when
		col, err := j.TranslateFieldName(j.prefixActivators[0])
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("empty locator with whitespace", func(t *testing.T) {
		t.Parallel()
		// when
		col, err := j.TranslateFieldName(j.prefixActivators[0] + "    ")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("not allowed ' in locator", func(t *testing.T) {
		t.Parallel()
		// when
		col, err := j.TranslateFieldName(j.prefixActivators[0] + "foo'bar")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		// when
		col, err := j.TranslateFieldName(j.prefixActivators[0] + "name")
		// then
		require.NoError(t, err)
		require.Equal(t, j.tableAlias+".name", col)
	})
	t.Run("explicitly allowed column", func(t *testing.T) {
		t.Parallel()
		// given
		a := j
		a.allowedColumns = []string{"name"}
		// when
		col, err := a.TranslateFieldName(a.prefixActivators[0] + "name")
		// then
		require.NoError(t, err)
		require.Equal(t, j.tableAlias+".name", col)
	})
	t.Run("explicitly allowed column not matching", func(t *testing.T) {
		t.Parallel()
		// given
		a := j
		a.allowedColumns = []string{"name"}
		// when
		col, err := a.TranslateFieldName(a.prefixActivators[0] + "foobar")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("explicitly disallowed column", func(t *testing.T) {
		t.Parallel()
		// given
		a := j
		a.disallowedColumns = []string{"name"}
		// when
		col, err := a.TranslateFieldName(a.prefixActivators[0] + "name")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("explicitly disallowed column not matching", func(t *testing.T) {
		t.Parallel()
		// given
		a := j
		a.disallowedColumns = []string{"name"}
		// when
		col, err := a.TranslateFieldName(a.prefixActivators[0] + "foobar")
		// then
		require.NoError(t, err)
		require.Equal(t, j.tableAlias+".foobar", col)
	})
	t.Run("combination of explicitly allowed and disallowed columns", func(t *testing.T) {
		t.Parallel()
		// given
		a := j
		a.disallowedColumns = []string{"name"}
		a.disallowedColumns = []string{"foobar"}
		// when
		col, err := a.TranslateFieldName(a.prefixActivators[0] + "random_field")
		// then
		require.NoError(t, err)
		require.Equal(t, j.tableAlias+".random_field", col)
		// when
		col, err = a.TranslateFieldName(a.prefixActivators[0] + "name")
		// then
		require.NoError(t, err)
		require.Equal(t, j.tableAlias+".name", col)
		// when
		col, err = a.TranslateFieldName(a.prefixActivators[0] + "foobar")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
}
