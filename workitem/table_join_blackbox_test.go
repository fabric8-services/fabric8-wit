package workitem_test

import (
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type tableJoinTestSuite struct {
	gormtestsupport.DBTestSuite
}

func Test_TableJoinSuite(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &tableJoinTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *tableJoinTestSuite) TestIsValidate() {
	// given
	j := workitem.TableJoin{
		Active:           true,
		TableName:        "iterations",
		TableAlias:       "iter",
		On:               workitem.JoinOnJSONField(workitem.SystemIteration, "iter.ID"),
		PrefixActivators: []string{"iteration."},
	}
	s.T().Run("valid", func(t *testing.T) {
		//given
		j.HandledFields = []string{"name"}
		// when/then
		require.NoError(t, j.Validate(s.DB))
	})
	s.T().Run("not valid", func(t *testing.T) {
		//given
		j.HandledFields = []string{"some_field_that_definitively_does_not_exist_in_the_iterations_table"}
		// when/then
		require.Error(t, j.Validate(s.DB))
	})
}

func Test_TableJoin_HandlesFieldName(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// given
	j := workitem.TableJoin{
		TableName:        "iterations",
		TableAlias:       "iter",
		On:               workitem.JoinOnJSONField(workitem.SystemIteration, "iter.ID"),
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
	actual := workitem.JoinOnJSONField("system.iteration", "iter.ID")
	// then
	require.Equal(t, workitem.Column(workitem.WorkItemStorage{}.TableName(), "fields")+` @> concat('{"system.iteration": "', iter.ID, '"}')::jsonb`, actual)
}

func Test_TableJoin_String(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// given
	j := workitem.TableJoin{
		TableName:        "iterations",
		TableAlias:       "iter",
		On:               workitem.JoinOnJSONField(workitem.SystemIteration, "iter.ID"),
		PrefixActivators: []string{"iteration."},
	}
	// when
	s := j.GetJoinExpression()
	// then
	require.Equal(t, fmt.Sprintf(`LEFT JOIN "%s" "%s" ON %s`, j.TableName, j.TableAlias, j.On), s)
}

func Test_TableJoin_TranslateFieldName(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	t.Run("missing prefix", func(t *testing.T) {
		t.Parallel()
		// given
		j := workitem.TableJoin{TableName: "iterations", TableAlias: "iter", On: workitem.JoinOnJSONField(workitem.SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		// when
		col, err := j.TranslateFieldName("foo.bar")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("empty locator", func(t *testing.T) {
		t.Parallel()
		// given
		j := workitem.TableJoin{TableName: "iterations", TableAlias: "iter", On: workitem.JoinOnJSONField(workitem.SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0])
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("empty locator with whitespace", func(t *testing.T) {
		t.Parallel()
		// given
		j := workitem.TableJoin{TableName: "iterations", TableAlias: "iter", On: workitem.JoinOnJSONField(workitem.SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0] + "    ")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("not allowed ' in locator", func(t *testing.T) {
		t.Parallel()
		// given
		j := workitem.TableJoin{TableName: "iterations", TableAlias: "iter", On: workitem.JoinOnJSONField(workitem.SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0] + "foo'bar")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		// given
		j := workitem.TableJoin{TableName: "iterations", TableAlias: "iter", On: workitem.JoinOnJSONField(workitem.SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0] + "name")
		// then
		require.NoError(t, err)
		require.Equal(t, workitem.Column(j.TableAlias, "name"), col)
		require.Equal(t, []string{"name"}, j.HandledFields)
	})
	t.Run("explicitly allowed column", func(t *testing.T) {
		t.Parallel()
		// given
		j := workitem.TableJoin{TableName: "iterations", TableAlias: "iter", On: workitem.JoinOnJSONField(workitem.SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		// given
		j.AllowedColumns = []string{"name"}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0] + "name")
		// then
		require.NoError(t, err)
		require.Equal(t, workitem.Column(j.TableAlias, "name"), col)
		require.Equal(t, []string{"name"}, j.HandledFields)
	})
	t.Run("explicitly allowed column not matching", func(t *testing.T) {
		t.Parallel()
		// given
		j := workitem.TableJoin{TableName: "iterations", TableAlias: "iter", On: workitem.JoinOnJSONField(workitem.SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
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
		j := workitem.TableJoin{TableName: "iterations", TableAlias: "iter", On: workitem.JoinOnJSONField(workitem.SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
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
		j := workitem.TableJoin{TableName: "iterations", TableAlias: "iter", On: workitem.JoinOnJSONField(workitem.SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		j.DisallowedColumns = []string{"name"}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0] + "foobar")
		// then
		require.NoError(t, err)
		require.Equal(t, workitem.Column(j.TableAlias, "foobar"), col)
		require.Equal(t, []string{"foobar"}, j.HandledFields)
	})
	t.Run("combination of explicitly allowed and disallowed columns", func(t *testing.T) {
		t.Parallel()
		// given
		j := workitem.TableJoin{TableName: "iterations", TableAlias: "iter", On: workitem.JoinOnJSONField(workitem.SystemIteration, "iter.ID"), PrefixActivators: []string{"iteration."}}
		j.DisallowedColumns = []string{"name"}
		j.DisallowedColumns = []string{"foobar"}
		// when
		col, err := j.TranslateFieldName(j.PrefixActivators[0] + "random_field")
		// then
		require.NoError(t, err)
		require.Equal(t, workitem.Column(j.TableAlias, "random_field"), col)
		// when
		col, err = j.TranslateFieldName(j.PrefixActivators[0] + "name")
		// then
		require.NoError(t, err)
		require.Equal(t, workitem.Column(j.TableAlias, "name"), col)
		// when
		col, err = j.TranslateFieldName(j.PrefixActivators[0] + "foobar")
		// then
		require.Error(t, err)
		require.Empty(t, col)
	})
}

func Test_TableJoinMap_ActivateRequiredJoins(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	t.Run("nothing activated", func(t *testing.T) {
		t.Parallel()
		// given
		joins := workitem.DefaultTableJoins()
		// when
		err := joins.ActivateRequiredJoins()
		// then
		require.NoError(t, err)
		for k, j := range joins {
			require.False(t, j.Active, `joins "%s" should not be activated`, k)
		}
	})
	t.Run("recursively activated joins", func(t *testing.T) {
		t.Parallel()
		t.Run("check that nothing but space is joined", func(t *testing.T) {
			// given
			joins := workitem.DefaultTableJoins()
			// when
			f, err := joins["space"].TranslateFieldName("space.name")
			// then
			require.NoError(t, err)
			require.Equal(t, workitem.Column("space", "name"), f)
			// given
			toBeFound := map[string]struct{}{
				"space": {},
			}
			// then
			for k := range joins {
				_, ok := toBeFound[k]
				if !ok && joins[k].Active {
					t.Fatalf(`join "%s" was not supposed to be active`, k)
				}
				if ok && !joins[k].Active {
					t.Fatalf(`join "%s" was supposed to be active`, k)
				}
				delete(toBeFound, k)
			}
			require.Empty(t, toBeFound, "these joins where not activated: %+v", toBeFound)
		})
		t.Run("check that space, custom1 and custom2 are activated", func(t *testing.T) {
			// given
			joins := workitem.DefaultTableJoins()
			joins["custom1"] = &workitem.TableJoin{
				TableName:          "custom1",
				TableAlias:         "cust1",
				PrefixActivators:   []string{"custom1."},
				ActivateOtherJoins: []string{"space"},
			}
			joins["custom2"] = &workitem.TableJoin{
				TableName:          "custom2",
				TableAlias:         "cust2",
				PrefixActivators:   []string{"custom2."},
				ActivateOtherJoins: []string{"custom1"},
			}
			// when
			f, err := joins["custom2"].TranslateFieldName("custom2.foo")
			// then
			require.NoError(t, err)
			require.Equal(t, workitem.Column("cust2", "foo"), f)
			// when
			err = joins.ActivateRequiredJoins()
			require.NoError(t, err)
			toBeFound := map[string]struct{}{
				"custom2": {}, // should be active because we queried for "custom2"."foo"
				"custom1": {}, // should be active because it was activated by custom2
				"space":   {}, // should be active because it was activated by custom1
			}
			// then
			for k := range joins {
				_, ok := toBeFound[k]
				if !ok && joins[k].Active {
					t.Fatalf(`join "%s" was not supposed to be active`, k)
				}
				if ok && !joins[k].Active {
					t.Fatalf(`join "%s" was supposed to be active`, k)
				}
				delete(toBeFound, k)
			}
			require.Empty(t, toBeFound, "these joins where not activated: %+v", toBeFound)
		})
		t.Run("check that missing required joins are found", func(t *testing.T) {
			// given
			joins := workitem.TableJoinMap{
				"custom1": {
					TableName:          "custom1",
					TableAlias:         "cust1",
					PrefixActivators:   []string{"custom1."},
					ActivateOtherJoins: []string{"non_existing_join"},
				},
				"custom2": {
					TableName:          "custom2",
					TableAlias:         "cust2",
					PrefixActivators:   []string{"custom2."},
					ActivateOtherJoins: []string{"custom1"},
				},
			}
			// when
			f, err := joins["custom2"].TranslateFieldName("custom2.foo")
			// then
			require.NoError(t, err)
			require.Equal(t, workitem.Column("cust2", "foo"), f)
			// when
			err = joins.ActivateRequiredJoins()
			require.Error(t, err)
		})
	})
}

func Test_TableJoinMap_GetOrderdActivatedJoins(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	t.Run("nothing activated, so nothing in ordered activation list", func(t *testing.T) {
		t.Parallel()
		// given
		joins := workitem.DefaultTableJoins()
		// when
		list, err := joins.GetOrderdActivatedJoins()
		// then
		require.NoError(t, err)
		require.Empty(t, list)
	})
	t.Run("recursively activated joins", func(t *testing.T) {
		t.Run("check that space, custom1 and custom2 are activated", func(t *testing.T) {
			// given
			joins := workitem.DefaultTableJoins()
			joins["custom1"] = &workitem.TableJoin{
				TableName:          "custom1",
				TableAlias:         "cust1",
				PrefixActivators:   []string{"custom1."},
				ActivateOtherJoins: []string{"space"},
			}
			joins["custom2"] = &workitem.TableJoin{
				TableName:          "custom2",
				TableAlias:         "cust2",
				PrefixActivators:   []string{"custom2."},
				ActivateOtherJoins: []string{"custom1"},
			}
			// when
			f, err := joins["custom2"].TranslateFieldName("custom2.foo")
			// then
			require.NoError(t, err)
			require.Equal(t, workitem.Column("cust2", "foo"), f)
			// when
			list, err := joins.GetOrderdActivatedJoins()
			require.NoError(t, err)
			require.NotEmpty(t, list)
			// then
			require.Equal(t, []*workitem.TableJoin{joins["space"], joins["custom1"], joins["custom2"]}, list)
		})
	})
}

func Test_TableJoinMap_DelegateTo(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	t.Run("recursively activated joins", func(t *testing.T) {
		// given
		joins := workitem.TableJoinMap{
			"custom1": &workitem.TableJoin{
				TableName:  "custom1",
				TableAlias: "cust1",
			},
		}
		joins["custom2"] = &workitem.TableJoin{
			TableName:          "custom2",
			TableAlias:         "cust2",
			PrefixActivators:   []string{"custom2.", "custom1."},
			ActivateOtherJoins: []string{"custom1"},
			DelegateTo: map[string]*workitem.TableJoin{
				"custom1.": joins["custom1"],
			},
		}
		// when accessing a column through a delegation
		f, err := joins["custom2"].TranslateFieldName("custom1.foo")
		// then
		require.NoError(t, err)

		t.Run("check delegation", func(t *testing.T) {
			t.Run("field translates to custom1 field", func(t *testing.T) {
				require.Equal(t, workitem.Column("cust1", "foo"), f)
			})
			t.Run("custom1 is activated", func(t *testing.T) {
				require.True(t, joins["custom1"].Active)
			})
			t.Run("custom1 handles field \"foo\"", func(t *testing.T) {
				require.Len(t, joins["custom1"].HandledFields, 1)
				require.Equal(t, "foo", joins["custom1"].HandledFields[0])
			})
		})
		t.Run("order of activated joins", func(t *testing.T) {
			list, err := joins.GetOrderdActivatedJoins()
			require.NoError(t, err)
			require.NotEmpty(t, list)
			// then
			require.Equal(t, []*workitem.TableJoin{joins["custom1"], joins["custom2"]}, list)
		})
	})
}
