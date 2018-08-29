package gormsupport_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/require"
)

func TestVersioning_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := gormsupport.Versioning{
		Version: 42,
	}

	t.Run("equality", func(t *testing.T) {
		b := gormsupport.Versioning{
			Version: 42,
		}
		require.True(t, a.Equal(b))
	})
	t.Run("type difference", func(t *testing.T) {
		b := convert.DummyEqualer{}
		require.False(t, a.Equal(b))
	})
	t.Run("version difference", func(t *testing.T) {
		b := gormsupport.Versioning{
			Version: 123,
		}
		require.False(t, a.Equal(b))
	})
}
