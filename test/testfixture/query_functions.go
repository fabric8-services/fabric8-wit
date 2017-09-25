package testfixture

import (
	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/label"
	"github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"
)

// LabelByName returns the first label that has the given name (if any). If you
// have labels with the same name in different spaces you can also pass in one
// space ID to filter by space as well.
func (fxt *TestFixture) LabelByName(name string, spaceID ...uuid.UUID) *label.Label {
	for _, l := range fxt.Labels {
		if l.Name == name && len(spaceID) > 0 && l.SpaceID == spaceID[0] {
			return l
		} else if l.Name == name && len(spaceID) == 0 {
			return l
		}
	}
	return nil
}

// IterationByName returns the first iteration that has the given name (if any).
// If you have iterations with the same name in different spaces you can also
// pass in one space ID to filter by space as well.
func (fxt *TestFixture) IterationByName(name string, spaceID ...uuid.UUID) *iteration.Iteration {
	for _, i := range fxt.Iterations {
		if i.Name == name && len(spaceID) > 0 && i.SpaceID == spaceID[0] {
			return i
		} else if i.Name == name && len(spaceID) == 0 {
			return i
		}
	}
	return nil
}

// WorkItemTypeByName returns the first work item type that has the given name
// (if any). If you have work item type with the same name in different spaces
// you can also pass in one space ID to filter by space as well.
func (fxt *TestFixture) WorkItemTypeByName(name string, spaceID ...uuid.UUID) *workitem.WorkItemType {
	for _, wit := range fxt.WorkItemTypes {
		if wit.Name == name && len(spaceID) > 0 && wit.SpaceID == spaceID[0] {
			return wit
		} else if wit.Name == name && len(spaceID) == 0 {
			return wit
		}
	}
	return nil
}

// IdentityByUsername returns the first identity that has the given username (if
// any).
func (fxt *TestFixture) IdentityByUsername(username string) *account.Identity {
	for _, i := range fxt.Identities {
		if i.Username == username {
			return i
		}
	}
	return nil
}
