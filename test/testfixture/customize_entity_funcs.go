package testfixture

import (
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	errs "github.com/pkg/errors"
)

// A CustomizeEntityFunc acts as a generic function to the various
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
// from a customize-entity-function, it will not be very useful:
//     NewFixture(db, WorkItemTypes(1), Spaces(1, func(fxt *TestFixture, idx int) error{
//         fmt.Println(fxt.WorkItemType[0].ID) // WARNING: not yet set
//         return nil
//     }))
// On the other hand, you can safely lookup the space ID when you're in the
// customize-entity-function for a work item:
//     NewFixture(db, WorkItem(1, func(fxt *TestFixture, idx int) error{
//         fmt.Println(fxt.Space[0].ID) // safe to access
//         return nil
//     }))
//
// Notice that you can do all kinds of distribution related functions in a
// customize-entitiy-function. For example, you can control which identity owns
// a space or define what work item type each work item shall have. If not
// otherwise specified (e.g. as for WorkItemLinks()) we use a straight forward
// approach. So for example if you write
//     NewFixture(t, db, Identities(10), Spaces(100))
// then we will create 10 identites and 100 spaces and the owner of all spaces
// will be identified with the ID of the first identity:
//     fxt.Identities[0].ID
// If you want a different distribution, you can create your own customize-
// entitiy-function (see Identities() for an example).
//
// If you for some error reason you want your test fixture creation to fail you
// can use the fxt.T test instance:
//      NewFixture(db, Identities(100, func(fxt *TestFixture, idx int) error{
//          return errors.New("some test failure reason")
//      }))
type CustomizeEntityFunc func(fxt *TestFixture, idx int) error

// SetTopologies takes the given topologies and uses them during creation of
// work item link types. The length of requested work item link types and the number of topologies must
// match or the NewFixture call will return an error.
func SetTopologies(topologies ...link.Topology) CustomizeWorkItemLinkTypeFunc {
	return func(fxt *TestFixture, idx int) error {
		if len(fxt.WorkItemLinkTypes) != len(topologies) {
			return errs.Errorf("number of topologies (%d) must match number of work item link types to create (%d)", len(topologies), len(fxt.WorkItemLinkTypes))
		}
		fxt.WorkItemLinkTypes[idx].Topology = topologies[idx]
		return nil
	}
}

// UserActive ensures that all created iterations have the given user activation
// state
func UserActive(active bool) CustomizeIterationFunc {
	return func(fxt *TestFixture, idx int) error {
		fxt.Iterations[idx].UserActive = active
		return nil
	}
}

// SetLabelNames takes the given names and uses them during creation of labels.
// The length of requested labels and the number of names must match or the
// NewFixture call will return an error.
func SetLabelNames(names ...string) CustomizeLabelFunc {
	return func(fxt *TestFixture, idx int) error {
		if len(fxt.Labels) != len(names) {
			return errs.Errorf("number of names (%d) must match number of labels to create (%d)", len(names), len(fxt.Labels))
		}
		fxt.Labels[idx].Name = names[idx]
		return nil
	}
}

// SetIterationNames takes the given names and uses them during creation of
// iterations. The length of requested iterations and the number of names must
// match or the NewFixture call will return an error.
func SetIterationNames(names ...string) CustomizeIterationFunc {
	return func(fxt *TestFixture, idx int) error {
		if len(fxt.Iterations) != len(names) {
			return errs.Errorf("number of names (%d) must match number of iterations to create (%d)", len(names), len(fxt.Iterations))
		}
		fxt.Iterations[idx].Name = names[idx]
		return nil
	}
}

// PlaceIterationUnderRootIteration when asking for more than one iteration, all
// but the first one will be placed under the first iteration (aka root
// iteration).
func PlaceIterationUnderRootIteration() CustomizeIterationFunc {
	return func(fxt *TestFixture, idx int) error {
		if idx > 0 {
			fxt.Iterations[idx].MakeChildOf(*fxt.Iterations[0])
		}
		return nil
	}
}

// PlaceAreaUnderRootArea when asking for more than one area, all but the first
// one will be placed under the first area (aka root area).
func PlaceAreaUnderRootArea() CustomizeAreaFunc {
	return func(fxt *TestFixture, idx int) error {
		if idx > 0 {
			fxt.Areas[idx].MakeChildOf(*fxt.Areas[0])
		}
		return nil
	}
}

// SetWorkItemField takes the given values and uses them during creation of work
// items to set field values. The length of requested work items and the number
// of values must match or the NewFixture call will return an error.
func SetWorkItemField(fieldName string, values ...interface{}) CustomizeWorkItemFunc {
	return func(fxt *TestFixture, idx int) error {
		if len(fxt.WorkItems) < len(values) {
			return errs.Errorf("number of \"%s\" fields (%d) must be smaller or equal to number of work items to create (%d)", fieldName, len(values), len(fxt.WorkItems))
		}
		// Gracefully return when only a fraction of work items needs to set a
		// field value.
		if idx >= len(values) {
			return nil
		}
		witID := fxt.WorkItems[idx].Type
		wit := fxt.WorkItemTypeByID(witID)
		if wit == nil {
			return errs.Errorf("failed to find work item type with ID %s in test fixture", witID)
		}
		field, ok := wit.Fields[fieldName]
		if !ok {
			return errs.Errorf("failed to find field \"%s\" in work item type %s", fieldName, witID)
		}
		v, err := field.Type.ConvertToModel(values[idx])
		if err != nil {
			return errs.Wrapf(err, "failed to set field \"%s\" in work item type %s to: %+v", fieldName, wit.Name, values[idx])
		}
		fxt.WorkItems[idx].Fields[fieldName] = v
		return nil
	}
}

// SetWorkItemTypeNames takes the given names and uses them during creation of
// work item types. The length of requested work item types and the number of
// names must match or the NewFixture call will return an error.
func SetWorkItemTypeNames(names ...string) CustomizeWorkItemTypeFunc {
	return func(fxt *TestFixture, idx int) error {
		if len(fxt.WorkItemTypes) != len(names) {
			return errs.Errorf("number of names (%d) must match number of work item types to create (%d)", len(names), len(fxt.WorkItemTypes))
		}
		fxt.WorkItemTypes[idx].Name = names[idx]
		return nil
	}
}

// SetIdentityUsernames takes the given usernames and uses them during creation
// of identities. The length of requested work item types and the number of
// usernames must match or the NewFixture call will return an error.
func SetIdentityUsernames(usernames ...string) CustomizeIdentityFunc {
	return func(fxt *TestFixture, idx int) error {
		if len(fxt.Identities) != len(usernames) {
			return errs.Errorf("number of usernames (%d) must match number of identites to create (%d)", len(usernames), len(fxt.Identities))
		}
		fxt.Identities[idx].Username = usernames[idx]
		return nil
	}
}

// SetWorkItemTitles takes the given titles and uses them during creation of
// work items. The length of requested work items and the number of titles must
// match or the NewFixture call will return an error.
func SetWorkItemTitles(titleStrings ...interface{}) CustomizeWorkItemFunc {
	return SetWorkItemField(workitem.SystemTitle, titleStrings...)
}

// SetWorkItemLinkTypeNames takes the given names and uses them during creation
// of work item link types. The length of requested work item link types and the
// number of names must match or the NewFixture call will return an error.
func SetWorkItemLinkTypeNames(names ...string) CustomizeWorkItemLinkTypeFunc {
	return func(fxt *TestFixture, idx int) error {
		if len(fxt.WorkItemLinkTypes) != len(names) {
			return errs.Errorf("number of names (%d) must match number of work item link types to create (%d)", len(names), len(fxt.WorkItemLinkTypes))
		}
		fxt.WorkItemLinkTypes[idx].Name = names[idx]
		return nil
	}
}
