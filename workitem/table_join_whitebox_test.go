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
		TableName:        "iterations",
		TableAlias:       "iter",
		On:               JoinOnJSONField(SystemIteration, "iter.ID"),
		PrefixActivators: []string{"iteration."},
	}
	t.Run("has prefix", func(t *testing.T) {
		t.Parallel()
		require.True(t, j.HandlesFieldName(j.PrefixActivators[0]+"foobar"))
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

func Test_TableJoin_String(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// given
	j := TableJoin{
		TableName:        "iterations",
		TableAlias:       "iter",
		On:               JoinOnJSONField(SystemIteration, "iter.ID"),
		PrefixActivators: []string{"iteration."},
	}
	// when
	s := j.String()
	// then
	require.Equal(t, "LEFT JOIN "+j.TableName+" "+j.TableAlias+" ON "+j.On, s)
}

func Test_TableJoin_TranslateFieldName(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	t.Run("missing prefix", func(t *testing.T) {
		t.Parallel()
		// given
		j := TableJoin{TableName: "iterations", TableAlias: "iter", On: JoinOnJSONField(SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		// when
		col, err := j.TranslateFieldName("foo.bar")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("empty locator", func(t *testing.T) {
		t.Parallel()
		// given
		j := TableJoin{TableName: "iterations", TableAlias: "iter", On: JoinOnJSONField(SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0])
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("empty locator with whitespace", func(t *testing.T) {
		t.Parallel()
		// given
		j := TableJoin{TableName: "iterations", TableAlias: "iter", On: JoinOnJSONField(SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0] + "    ")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("not allowed ' in locator", func(t *testing.T) {
		t.Parallel()
		// given
		j := TableJoin{TableName: "iterations", TableAlias: "iter", On: JoinOnJSONField(SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0] + "foo'bar")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		// given
		j := TableJoin{TableName: "iterations", TableAlias: "iter", On: JoinOnJSONField(SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0] + "name")
		// then
		require.NoError(t, err)
		require.Equal(t, j.TableAlias+".name", col)
		require.Equal(t, []string{"name"}, j.HandledFields)
	})
	t.Run("explicitly allowed column", func(t *testing.T) {
		t.Parallel()
		// given
		j := TableJoin{TableName: "iterations", TableAlias: "iter", On: JoinOnJSONField(SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		// given
		j.AllowedColumns = []string{"name"}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0] + "name")
		// then
		require.NoError(t, err)
		require.Equal(t, j.TableAlias+".name", col)
		require.Equal(t, []string{"name"}, j.HandledFields)
	})
	t.Run("explicitly allowed column not matching", func(t *testing.T) {
		t.Parallel()
		// given
		j := TableJoin{TableName: "iterations", TableAlias: "iter", On: JoinOnJSONField(SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		j.AllowedColumns = []string{"name"}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0] + "foobar")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("explicitly disallowed column", func(t *testing.T) {
		t.Parallel()
		// given
		j := TableJoin{TableName: "iterations", TableAlias: "iter", On: JoinOnJSONField(SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		j.DisallowedColumns = []string{"name"}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0] + "name")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("explicitly disallowed column not matching", func(t *testing.T) {
		t.Parallel()
		// given
		j := TableJoin{TableName: "iterations", TableAlias: "iter", On: JoinOnJSONField(SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		j.DisallowedColumns = []string{"name"}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0] + "foobar")
		// then
		require.NoError(t, err)
		require.Equal(t, j.TableAlias+".foobar", col)
		require.Equal(t, []string{"foobar"}, j.HandledFields)
	})
	t.Run("combination of explicitly allowed and disallowed columns", func(t *testing.T) {
		t.Parallel()
		// given
		j := TableJoin{TableName: "iterations", TableAlias: "iter", On: JoinOnJSONField(SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		j.DisallowedColumns = []string{"name"}
		j.DisallowedColumns = []string{"foobar"}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0] + "random_field")
		// then
		require.NoError(t, err)
		require.Equal(t, j.TableAlias+".random_field", col)
		// when
		col, err = j.TranslateFieldName(j.PrefixActivators[0] + "name")
		// then
		require.NoError(t, err)
		require.Equal(t, j.TableAlias+".name", col)
		// when
		col, err = j.TranslateFieldName(j.PrefixActivators[0] + "foobar")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
}
