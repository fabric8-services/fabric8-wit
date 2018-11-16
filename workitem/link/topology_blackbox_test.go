package link_test

import (
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/stretchr/testify/require"
)

func TestTopology_String(t *testing.T) {
	t.Parallel()
	inOut := map[link.Topology]string{
		link.TopologyDependency:      "dependency",
		link.TopologyDirectedNetwork: "directed_network",
		link.TopologyNetwork:         "network",
		link.TopologyTree:            "tree",
	}
	for in, out := range inOut {
		in := in
		out := out
		t.Run(in.String(), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, out, in.String())
		})
	}
}

func TestTopology_Scan(t *testing.T) {
	t.Parallel()

	dependency := link.TopologyDependency
	directedNetwork := link.TopologyDirectedNetwork
	network := link.TopologyNetwork
	tree := link.TopologyTree

	type testData struct {
		in          interface{}
		expectError bool
		result      *link.Topology
	}

	testDataArr := []testData{
		// with type: link.Topology
		{link.TopologyDependency, false, &dependency},
		{link.TopologyDirectedNetwork, false, &directedNetwork},
		{link.TopologyNetwork, false, &network},
		{link.TopologyTree, false, &tree},
		// with type: string
		{link.TopologyDependency.String(), false, &dependency},
		{link.TopologyDirectedNetwork.String(), false, &directedNetwork},
		{link.TopologyNetwork.String(), false, &network},
		{link.TopologyTree.String(), false, &tree},
		// with type: []byte
		{[]byte(link.TopologyDependency.String()), false, &dependency},
		{[]byte(link.TopologyDirectedNetwork.String()), false, &directedNetwork},
		{[]byte(link.TopologyNetwork.String()), false, &network},
		{[]byte(link.TopologyTree.String()), false, &tree},
		// with mixed and invalid types/values
		{nil, true, nil},
		{"foo", true, nil},
		{false, true, nil},
		{true, true, nil},
	}

	for i, td := range testDataArr {
		i := i
		td := td
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			var l link.Topology
			err := l.Scan(td.in)
			if td.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, *td.result, l)
			}
		})
	}
}

func TestTopology_CheckValid(t *testing.T) {
	t.Parallel()
	expectErrorArr := map[link.Topology]bool{
		link.TopologyDependency:      false,
		link.TopologyDirectedNetwork: false,
		link.TopologyNetwork:         false,
		link.TopologyTree:            false,
		link.Topology(""):            true,
		link.Topology("foo"):         true,
	}
	for topo, expectError := range expectErrorArr {
		topo := topo
		expectError := expectError
		t.Run(topo.String(), func(t *testing.T) {
			t.Parallel()
			err := topo.CheckValid()
			if !expectError {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
