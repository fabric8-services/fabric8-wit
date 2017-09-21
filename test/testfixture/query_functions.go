package testfixture

import (
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/label"
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
