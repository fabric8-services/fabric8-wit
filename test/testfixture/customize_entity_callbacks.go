package testfixture

import "github.com/fabric8-services/fabric8-wit/workitem/link"

// A CustomizeEntityCallback acts as a generic callback to the various
// recipe-functions (e.g. Identities(), Spaces(), etc.). The current test
// fixture is given with the fxt argument and the position of the object that
// will be created next is indicated by the index idx. That index can be used to
// look up e.g. a space with
//     s := fxt.Spaces[idx]
// That space s will be a ready-to-create space object on that you can modify to
// your liking.
//
// Notice that when you lookup objects in the test fixture, you can only safely
// access those object on which the entity depends, because those are guaranteed
// to be already created. For example when you try to access a work item type
// from a customize-entity-callback, it will not be very useful:
//     NewFixture(db, WorkItemTypes(1), Spaces(1, func(fxt *TestFixture, idx int) error{
//         fmt.Println(fxt.WorkItemType[0].ID) // WARNING: not yet set
//         return nil
//     }))
// On the other hand, you can safely lookup the space ID when you're in the
// customize-entity-callback for a work item:
//     NewFixture(db, WorkItem(1, func(fxt *TestFixture, idx int) error{
//         fmt.Println(fxt.Space[0].ID) // safe to access
//         return nil
//     }))
//
// Notice that you can do all kinds of distribution related functions in a
// customize-entitiy-callback. For example, you can control which identity owns
// a space or define what work item type each work item shall have. If not
// otherwise specified (e.g. as for WorkItemLinks()) we use a straight forward
// approach. So for example if you write
//     NewFixture(t, db, Identities(10), Spaces(100))
// then we will create 10 identites and 100 spaces and the owner of all spaces
// will be identified with the ID of the first identity:
//     fxt.Identities[0].ID
// If you want a different distribution, you can create your own customize-
// entitiy-callback (see Identities() for an example).
//
// If you for some error reason you want your test fixture creation to fail you
// can use the fxt.T test instance:
//      NewFixture(db, Identities(100, func(fxt *TestFixture, idx int) error{
//          return errors.New("some test failure reason")
//      }))
type CustomizeEntityCallback func(fxt *TestFixture, idx int) error

// Topology ensures that all created link types will have the given topology
// type.
func Topology(topology string) CustomizeEntityCallback {
	return CustomizeEntityCallback(func(fxt *TestFixture, idx int) error {
		fxt.WorkItemLinkTypes[idx].Topology = topology
		return nil
	})
}

// TopologyNetwork ensures that all created link types will have the "network"
// topology type.
func TopologyNetwork() CustomizeEntityCallback {
	return Topology(link.TopologyNetwork)
}

// TopologyDirectedNetwork ensures that all created link types will have the
// "directed network" topology type.
func TopologyDirectedNetwork() CustomizeEntityCallback {
	return Topology(link.TopologyDirectedNetwork)
}

// TopologyDependency ensures that all created link types will have the
// "dependency" topology type.
func TopologyDependency() CustomizeEntityCallback {
	return Topology(link.TopologyDependency)
}

// TopologyTree ensures that all created link types will have the "tree"
// topology type.
func TopologyTree() CustomizeEntityCallback {
	return Topology(link.TopologyTree)
}
