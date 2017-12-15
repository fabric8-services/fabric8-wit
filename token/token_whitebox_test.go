package token

import (
	"testing"

	"fmt"

	config "github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/require"
)

func TestRemoteTokensLoaded(t *testing.T) {
	t.Skipf("We're skipping this test on purpose until we can properly run remote tests on CI")
	resource.Require(t, resource.Remote)
	c, err := config.Get()
	if err != nil {
		panic(fmt.Errorf("failed to setup the configuration: %s", err.Error()))
	}
	m, err := NewManager(c)
	require.NoError(t, err)
	require.NotNil(t, m)
	tm, ok := m.(*tokenManager)
	require.True(t, ok)

	require.NotEmpty(t, tm.PublicKeys())
	require.Equal(t, len(tm.publicKeys), len(m.PublicKeys()))
	require.Equal(t, len(tm.publicKeys), len(tm.publicKeysMap))
	for i, k := range tm.publicKeys {
		require.NotEqual(t, "", k.KeyID)
		require.NotNil(t, m.PublicKey(k.KeyID))
		require.Equal(t, m.PublicKeys()[i], k.Key)
	}
}
