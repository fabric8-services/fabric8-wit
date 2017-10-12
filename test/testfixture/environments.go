package testfixture

import (
	"github.com/fabric8-services/fabric8-wit/path"
	errs "github.com/pkg/errors"
)

// CreateWorkItemEnvironment returns a higher level recipe function that
// contains a little bit more business logic compared to the plain recipe
// functions that only set up dependencies according to database needs. In this
// particular case, we create a root iteration and root area and make additional
// areas be child of the root instances.
func CreateWorkItemEnvironment() RecipeFunction {
	return func(fxt *TestFixture) error {
		fxt.checkFuncs = append(fxt.checkFuncs, func() error {
			if len(fxt.Spaces) > 1 {
				return errs.Errorf("for ambiguity you should not create more than one space when using the CreateWorkItemEnvironment")
			}
			return nil
		})
		return fxt.deps(
			Identities(1),
			Spaces(1),
			WorkItemTypes(1),
			Iterations(1,
				func(fxt *TestFixture, idx int) error {
					if idx == 0 {
						fxt.Iterations[idx].Name = fxt.Spaces[0].Name
						fxt.Iterations[idx].Path = path.Path{}
					}
					return nil
				},
				PlaceIterationUnderRootIteration(),
			),
			Areas(1,
				func(fxt *TestFixture, idx int) error {
					if idx == 0 {
						fxt.Areas[idx].Name = fxt.Spaces[0].Name
						fxt.Areas[idx].Path = path.Path{}
					}
					return nil
				},
				PlaceAreaUnderRootArea(),
			),
		)
	}
}
