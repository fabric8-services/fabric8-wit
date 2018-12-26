package remoteworkitem

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestLookupProvider(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	ts1 := TrackerSchedule{TrackerType: ProviderGithub}
	tp1 := lookupProvider(ts1)
	require.NotNil(t, tp1)

	ts2 := TrackerSchedule{TrackerType: ProviderJira}
	tp2 := lookupProvider(ts2)
	require.NotNil(t, tp2)

	ts3 := TrackerSchedule{TrackerType: "unknown"}
	tp3 := lookupProvider(ts3)
	require.Nil(t, tp3)
}
