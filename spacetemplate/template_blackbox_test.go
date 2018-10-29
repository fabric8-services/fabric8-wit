package spacetemplate_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SpaceTemplate_EqualAndEqualValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	expectedID := uuid.FromStringOrNil("83a6b035-8cb2-4952-b0a2-f4831099726c")
	expected := spacetemplate.SpaceTemplate{
		ID:           expectedID,
		Name:         "empty template " + expectedID.String(),
		Description:  ptr.String("description for empty template " + expectedID.String()),
		CanConstruct: true,
	}

	t.Run("different types", func(t *testing.T) {
		t.Parallel()
		assert.False(t, expected.Equal(convert.DummyEqualer{}))
		assert.False(t, expected.EqualValue(convert.DummyEqualer{}))
	})

	t.Run("id", func(t *testing.T) {
		t.Parallel()
		actual := expected
		actual.ID = uuid.NewV4()
		assert.False(t, expected.Equal(actual))
		assert.False(t, expected.EqualValue(actual))
	})

	t.Run("version", func(t *testing.T) {
		t.Parallel()
		actual := expected
		actual.Version = 10
		assert.False(t, expected.Equal(actual))
		assert.True(t, expected.EqualValue(actual))
	})

	t.Run("lifecycle", func(t *testing.T) {
		t.Parallel()
		actual := expected
		actual.CreatedAt = time.Now()
		assert.False(t, expected.Equal(actual))
		assert.True(t, expected.EqualValue(actual))
	})

	t.Run("name", func(t *testing.T) {
		t.Parallel()
		actual := expected
		actual.Name = "something else"
		assert.False(t, expected.Equal(actual))
		assert.False(t, expected.EqualValue(actual))
	})

	t.Run("can construct", func(t *testing.T) {
		t.Parallel()
		actual := expected
		actual.CanConstruct = false
		assert.False(t, expected.Equal(actual))
		assert.False(t, expected.EqualValue(actual))
	})

	t.Run("description nil", func(t *testing.T) {
		t.Parallel()
		actual := expected
		actual.Description = nil
		assert.False(t, expected.Equal(actual))
		assert.False(t, expected.EqualValue(actual))
	})

	t.Run("description not nil", func(t *testing.T) {
		t.Parallel()
		actual := expected
		actualDescription := "some other description"
		actual.Description = &actualDescription
		assert.False(t, expected.Equal(actual))
		assert.False(t, expected.EqualValue(actual))
	})

	t.Run("equalness", func(t *testing.T) {
		t.Parallel()
		actual := expected
		assert.True(t, expected.Equal(actual))
		assert.True(t, expected.EqualValue(actual))
	})
}

func Test_SpaceTemplate_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		// given
		s := spacetemplate.SpaceTemplate{
			ID:          uuid.NewV4(),
			Name:        "foobar",
			Description: ptr.String("some description"),
		}
		// when/then
		require.NoError(t, s.Validate())
	})
	t.Run("invalid", func(t *testing.T) {
		t.Run("no name", func(t *testing.T) {
			// given
			s := spacetemplate.SpaceTemplate{
				ID:          uuid.NewV4(),
				Description: ptr.String("some description"),
			}
			// when/then
			require.Error(t, s.Validate())
		})
		t.Run("zero ID", func(t *testing.T) {
			// given
			s := spacetemplate.SpaceTemplate{
				ID:          uuid.UUID{},
				Name:        "foobar",
				Description: ptr.String("some description"),
			}
			// when/then
			require.Error(t, s.Validate())
		})
	})
}
