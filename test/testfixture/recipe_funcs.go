package testfixture

import (
	errs "github.com/pkg/errors"
)

// A RecipeFunction tells the test fixture to create n objects of a given kind.
// You can pass in customize-entity-functions in order to manipulate the objects
// before they get created.
type RecipeFunction func(fxt *TestFixture) error

const checkStr = "expected at least %d \"%s\" objects but found only %d"

func (fxt *TestFixture) deps(fns ...RecipeFunction) error {
	if !fxt.isolatedCreation {
		for _, fn := range fns {
			if err := fn(fxt); err != nil {
				return errs.Wrap(err, "failed to setup dependency")
			}
		}
	}
	return nil
}

// CustomizeIdentityFunc is directly compatible with CustomizeEntityFunc
// but it can only be used for the Identites() recipe-function.
type CustomizeIdentityFunc CustomizeEntityFunc

// Identities tells the test fixture to create at least n identity objects.
//
// If called multiple times with differently n's, the biggest n wins. All
// customize-entitiy-functions fns from all calls will be respected when
// creating the test fixture.
//
// Here's an example how you can create 42 identites and give them a numbered
// user name like "John Doe 0", "John Doe 1", and so forth:
//    Identities(42, func(fxt *TestFixture, idx int) error{
//        fxt.Identities[idx].Username = "Jane Doe " + strconv.FormatInt(idx, 10)
//        return nil
//    })
// Notice that the index idx goes from 0 to n-1 and that you have to manually
// lookup the object from the test fixture. The identity object referenced by
//    fxt.Identities[idx]
// is guaranteed to be ready to be used for creation. That means, you don't
// necessarily have to touch it to avoid unique key violation for example. This
// is totally optional.
func Identities(n int, fns ...CustomizeIdentityFunc) RecipeFunction {
	return func(fxt *TestFixture) error {
		fxt.checkFuncs = append(fxt.checkFuncs, func() error {
			l := len(fxt.Identities)
			if l < n {
				return errs.Errorf(checkStr, n, kindIdentities, l)
			}
			return nil
		})
		// Convert fns to []CustomizeEntityFunc
		customFuncs := make([]CustomizeEntityFunc, len(fns))
		for idx := range fns {
			customFuncs[idx] = CustomizeEntityFunc(fns[idx])
		}
		return fxt.setupInfo(n, kindIdentities, customFuncs...)
	}
}

// CustomizeSpaceFunc is directly compatible with CustomizeEntityFunc
// but it can only be used for the Spaces() recipe-function.
type CustomizeSpaceFunc CustomizeEntityFunc

// Spaces tells the test fixture to create at least n space objects. See also
// the Identities() function for more general information on n and fns.
//
// When called in NewFixture() this function will call also call
//     Identities(1)
// but with NewFixtureIsolated(), no other objects will be created.
func Spaces(n int, fns ...CustomizeSpaceFunc) RecipeFunction {
	return func(fxt *TestFixture) error {
		fxt.checkFuncs = append(fxt.checkFuncs, func() error {
			l := len(fxt.Spaces)
			if l < n {
				return errs.Errorf(checkStr, n, kindSpaces, l)
			}
			return nil
		})
		// Convert fns to []CustomizeEntityFunc
		customFuncs := make([]CustomizeEntityFunc, len(fns))
		for idx := range fns {
			customFuncs[idx] = CustomizeEntityFunc(fns[idx])
		}
		if err := fxt.setupInfo(n, kindSpaces, customFuncs...); err != nil {
			return err
		}
		return fxt.deps(Identities(1))
	}
}

// CustomizeIterationFunc is directly compatible with
// CustomizeEntityFunc but it can only be used for the Iterations()
// recipe-function.
type CustomizeIterationFunc CustomizeEntityFunc

// Iterations tells the test fixture to create at least n iteration objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewFixture() this function will call also call
//     Spaces(1)
// but with NewFixtureIsolated(), no other objects will be created.
func Iterations(n int, fns ...CustomizeIterationFunc) RecipeFunction {
	return func(fxt *TestFixture) error {
		fxt.checkFuncs = append(fxt.checkFuncs, func() error {
			l := len(fxt.Iterations)
			if l < n {
				return errs.Errorf(checkStr, n, kindIterations, l)
			}
			return nil
		})
		// Convert fns to []CustomizeEntityFunc
		customFuncs := make([]CustomizeEntityFunc, len(fns))
		for idx := range fns {
			customFuncs[idx] = CustomizeEntityFunc(fns[idx])
		}
		if err := fxt.setupInfo(n, kindIterations, customFuncs...); err != nil {
			return err
		}
		return fxt.deps(Spaces(1))
	}
}

// CustomizeAreaFunc is directly compatible with CustomizeEntityFunc but
// it can only be used for the Areas() recipe-function.
type CustomizeAreaFunc CustomizeEntityFunc

// Areas tells the test fixture to create at least n area objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewFixture() this function will call also call
//     Spaces(1)
// but with NewFixtureIsolated(), no other objects will be created.
func Areas(n int, fns ...CustomizeAreaFunc) RecipeFunction {
	return func(fxt *TestFixture) error {
		fxt.checkFuncs = append(fxt.checkFuncs, func() error {
			l := len(fxt.Areas)
			if l < n {
				return errs.Errorf(checkStr, n, kindAreas, l)
			}
			return nil
		})
		// Convert fns to []CustomizeEntityFunc
		customFuncs := make([]CustomizeEntityFunc, len(fns))
		for idx := range fns {
			customFuncs[idx] = CustomizeEntityFunc(fns[idx])
		}
		if err := fxt.setupInfo(n, kindAreas, customFuncs...); err != nil {
			return err
		}
		return fxt.deps(Spaces(1))
	}
}

// CustomizeCodebaseFunc is directly compatible with CustomizeEntityFunc
// but it can only be used for the Codebases() recipe-function.
type CustomizeCodebaseFunc CustomizeEntityFunc

// Codebases tells the test fixture to create at least n codebase objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewFixture() this function will call also call
//     Spaces(1)
// but with NewFixtureIsolated(), no other objects will be created.
func Codebases(n int, fns ...CustomizeCodebaseFunc) RecipeFunction {
	return func(fxt *TestFixture) error {
		fxt.checkFuncs = append(fxt.checkFuncs, func() error {
			l := len(fxt.Codebases)
			if l < n {
				return errs.Errorf(checkStr, n, kindCodebases, l)
			}
			return nil
		})
		// Convert fns to []CustomizeEntityFunc
		customFuncs := make([]CustomizeEntityFunc, len(fns))
		for idx := range fns {
			customFuncs[idx] = CustomizeEntityFunc(fns[idx])
		}
		if err := fxt.setupInfo(n, kindCodebases, customFuncs...); err != nil {
			return err
		}
		return fxt.deps(Spaces(1))
	}
}

// CustomizeWorkItemFunc is directly compatible with CustomizeEntityFunc
// but it can only be used for the WorkItems() recipe-function.
type CustomizeWorkItemFunc CustomizeEntityFunc

// WorkItems tells the test fixture to create at least n work item objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewFixture() this function will call also call
//     Spaces(1)
//     WorkItemTypes(1)
//     Identities(1)
// but with NewFixtureIsolated(), no other objects will be created.
//
// Notice that the Number field of a work item is only set after the work item
// has been created, so any changes you make to
//     fxt.WorkItems[idx].Number
// will have no effect.
func WorkItems(n int, fns ...CustomizeWorkItemFunc) RecipeFunction {
	return func(fxt *TestFixture) error {
		fxt.checkFuncs = append(fxt.checkFuncs, func() error {
			l := len(fxt.WorkItems)
			if l < n {
				return errs.Errorf(checkStr, n, kindWorkItems, l)
			}
			return nil
		})
		// Convert fns to []CustomizeEntityFunc
		customFuncs := make([]CustomizeEntityFunc, len(fns))
		for idx := range fns {
			customFuncs[idx] = CustomizeEntityFunc(fns[idx])
		}
		if err := fxt.setupInfo(n, kindWorkItems, customFuncs...); err != nil {
			return err
		}
		return fxt.deps(Spaces(1), WorkItemTypes(1), Identities(1))
	}
}

// CustomizeCommentFunc is directly compatible with CustomizeEntityFunc
// but it can only be used for the Comments() recipe-function.
type CustomizeCommentFunc CustomizeEntityFunc

// Comments tells the test fixture to create at least n comment objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewFixture() this function will call also call
//     Identities(1)
//     WorkItems(1)
// but with NewFixtureIsolated(), no other objects will be created.
func Comments(n int, fns ...CustomizeWorkItemFunc) RecipeFunction {
	return func(fxt *TestFixture) error {
		fxt.checkFuncs = append(fxt.checkFuncs, func() error {
			l := len(fxt.Comments)
			if l < n {
				return errs.Errorf(checkStr, n, kindComments, l)
			}
			return nil
		})
		// Convert fns to []CustomizeEntityFunc
		customFuncs := make([]CustomizeEntityFunc, len(fns))
		for idx := range fns {
			customFuncs[idx] = CustomizeEntityFunc(fns[idx])
		}
		if err := fxt.setupInfo(n, kindComments, customFuncs...); err != nil {
			return err
		}
		return fxt.deps(WorkItems(1), Identities(1))
	}
}

// CustomizeWorkItemTypeFunc is directly compatible with
// CustomizeEntityFunc but it can only be used for the WorkItemTypes()
// recipe-function.
type CustomizeWorkItemTypeFunc CustomizeEntityFunc

// WorkItemTypes tells the test fixture to create at least n work item type
// objects. See also the Identities() function for more general information on n
// and fns.
//
// When called in NewFixture() this function will call also call
//     Spaces(1)
// but with NewFixtureIsolated(), no other objects will be created.
//
// The work item type that we create for each of the n instances is always the
// same and it tries to be compatible with the planner item work item type by
// specifying the same fields.
func WorkItemTypes(n int, fns ...CustomizeWorkItemTypeFunc) RecipeFunction {
	return func(fxt *TestFixture) error {
		fxt.checkFuncs = append(fxt.checkFuncs, func() error {
			l := len(fxt.WorkItemTypes)
			if l < n {
				return errs.Errorf(checkStr, n, kindWorkItemTypes, l)
			}
			return nil
		})
		// Convert fns to []CustomizeEntityFunc
		customFuncs := make([]CustomizeEntityFunc, len(fns))
		for idx := range fns {
			customFuncs[idx] = CustomizeEntityFunc(fns[idx])
		}
		if err := fxt.setupInfo(n, kindWorkItemTypes, customFuncs...); err != nil {
			return err
		}
		return fxt.deps(Spaces(1))
	}
}

// CustomizeWorkItemLinkTypeFunc is directly compatible with
// CustomizeEntityFunc but it can only be used for the WorkItemLinkTypes()
// recipe-function.
type CustomizeWorkItemLinkTypeFunc CustomizeEntityFunc

// WorkItemLinkTypes tells the test fixture to create at least n work item link
// type objects. See also the Identities() function for more general information
// on n and fns.
//
// When called in NewFixture() this function will call also call
//     Spaces(1)
//     WorkItemLinkCategories(1)
// but with NewFixtureIsolated(), no other objects will be created.
//
// We've created these helper functions that you should have a look at if you
// want to implement your own re-usable customize-entity-functions:
//     SetTopologies(topology ...link.Topology)
// The topology functions above are neat because you don't have to write a full
// function function yourself.
//
// By default a call to
//     WorkItemLinkTypes(1)
// equals
//     WorkItemLinkTypes(1, SetTopologies(link.TopologyTree))
// because we automatically set the topology for each link type to be "tree".
func WorkItemLinkTypes(n int, fns ...CustomizeWorkItemLinkTypeFunc) RecipeFunction {
	return func(fxt *TestFixture) error {
		fxt.checkFuncs = append(fxt.checkFuncs, func() error {
			l := len(fxt.WorkItemLinkTypes)
			if l < n {
				return errs.Errorf(checkStr, n, kindWorkItemLinkTypes, l)
			}
			return nil
		})
		// Convert fns to []CustomizeEntityFunc
		customFuncs := make([]CustomizeEntityFunc, len(fns))
		for idx := range fns {
			customFuncs[idx] = CustomizeEntityFunc(fns[idx])
		}
		if err := fxt.setupInfo(n, kindWorkItemLinkTypes, customFuncs...); err != nil {
			return err
		}
		return fxt.deps(Spaces(1), WorkItemLinkCategories(1))
	}
}

// CustomizeWorkItemLinkCategoryFunc is directly compatible with
// CustomizeEntityFunc but it can only be used for the
// WorkItemLinkCategories() recipe-function.
type CustomizeWorkItemLinkCategoryFunc CustomizeEntityFunc

// WorkItemLinkCategories tells the test fixture to create at least n work item
// link category objects. See also the Identities() function for more general
// information on n and fns.
//
// No other objects will be created.
func WorkItemLinkCategories(n int, fns ...CustomizeWorkItemLinkCategoryFunc) RecipeFunction {
	return func(fxt *TestFixture) error {
		fxt.checkFuncs = append(fxt.checkFuncs, func() error {
			l := len(fxt.WorkItemLinkCategories)
			if l < n {
				return errs.Errorf(checkStr, n, kindWorkItemLinkCategories, l)
			}
			return nil
		})
		// Convert fns to []CustomizeEntityFunc
		customFuncs := make([]CustomizeEntityFunc, len(fns))
		for idx := range fns {
			customFuncs[idx] = CustomizeEntityFunc(fns[idx])
		}
		return fxt.setupInfo(n, kindWorkItemLinkCategories, customFuncs...)
	}
}

// CustomizeWorkItemLinkFunc is directly compatible with
// CustomizeEntityFunc but it can only be used for the WorkItemLinks()
// recipe-function.
type CustomizeWorkItemLinkFunc CustomizeEntityFunc

// WorkItemLinks tells the test fixture to create at least n work item link
// objects. See also the Identities() function for more general information
// on n and fns.
//
// When called in NewFixture() this function will call also call
//     WorkItemLinkTypes(1)
//     WorkItems(2*n)
// but with NewFixtureIsolated(), no other objects will be created.
//
// Notice, that we will create two times the number of work items of your
// requested links. The way those links will be created can for sure be
// influenced using a customize-entity-function; but by default we create each
// link between two distinct work items. That means, no link will include the
// same work item.
func WorkItemLinks(n int, fns ...CustomizeWorkItemLinkFunc) RecipeFunction {
	return func(fxt *TestFixture) error {
		fxt.normalLinkCreation = true
		if fxt.customLinkCreation {
			return errs.New("When you have WorkItemLinksCustom() in your recipe you may no longer use WorkItemLinks().")
		}
		fxt.checkFuncs = append(fxt.checkFuncs, func() error {
			l := len(fxt.WorkItemLinks)
			if l < n {
				return errs.Errorf(checkStr, n, kindWorkItemLinks, l)
			}
			return nil
		})
		// Convert fns to []CustomizeEntityFunc
		customFuncs := make([]CustomizeEntityFunc, len(fns))
		for idx := range fns {
			customFuncs[idx] = CustomizeEntityFunc(fns[idx])
		}
		if err := fxt.setupInfo(n, kindWorkItemLinks, customFuncs...); err != nil {
			return err
		}
		return fxt.deps(WorkItemLinkTypes(1), WorkItems(2*n))
	}
}

// WorkItemLinksCustom behaves the same ways as WorkItemLinks but it makes no
// assumption on the number of work items that are going to be created. That
// means you can request and arbitrary number of links but you are also
// responsible of creating the the work items that you want to use in your
// links.
//
// NOTE: Normal WorkItemLinks creates twice the amount of work items as links.
func WorkItemLinksCustom(n int, fns ...CustomizeWorkItemLinkFunc) RecipeFunction {
	return func(fxt *TestFixture) error {
		fxt.customLinkCreation = true
		if fxt.normalLinkCreation {
			return errs.New("When you have WorkItemLinks() in your recipe you may no longer use WorkItemLinksCustom().")
		}
		fxt.checkFuncs = append(fxt.checkFuncs, func() error {
			l := len(fxt.WorkItemLinks)
			if l < n {
				return errs.Errorf(checkStr, n, kindWorkItemLinks, l)
			}
			return nil
		})
		// Convert fns to []CustomizeEntityFunc
		customFuncs := make([]CustomizeEntityFunc, len(fns))
		for idx := range fns {
			customFuncs[idx] = CustomizeEntityFunc(fns[idx])
		}
		if err := fxt.setupInfo(n, kindWorkItemLinks, customFuncs...); err != nil {
			return err
		}
		return fxt.deps(WorkItemLinkTypes(1) /*, WorkItems(2*n)*/)
	}
}

// CustomizeLabelFunc is directly compatible with CustomizeEntityFunc
// but it can only be used for the Labels() recipe-function.
type CustomizeLabelFunc CustomizeEntityFunc

// Labels tells the test fixture to create at least n label objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewFixture() this function will call also call
//     Spaces(1)
// but with NewFixtureIsolated(), no other objects will be created.
func Labels(n int, fns ...CustomizeLabelFunc) RecipeFunction {
	return func(fxt *TestFixture) error {
		fxt.checkFuncs = append(fxt.checkFuncs, func() error {
			l := len(fxt.Labels)
			if l < n {
				return errs.Errorf(checkStr, n, kindLabels, l)
			}
			return nil
		})
		// Convert fns to []CustomizeEntityFunc
		customFuncs := make([]CustomizeEntityFunc, len(fns))
		for idx := range fns {
			customFuncs[idx] = CustomizeEntityFunc(fns[idx])
		}
		if err := fxt.setupInfo(n, kindLabels, customFuncs...); err != nil {
			return err
		}
		return fxt.deps(Spaces(1))
	}
}

// CustomizeTrackerFunc is directly compatible with CustomizeEntityFunc
// but it can only be used for the Trackers() recipe-function.
type CustomizeTrackerFunc CustomizeEntityFunc

// Trackers tells the test fixture to create at least n tracker objects. See
// also the Identities() function for more general information on n and fns.
func Trackers(n int, fns ...CustomizeTrackerFunc) RecipeFunction {
	return func(fxt *TestFixture) error {
		fxt.checkFuncs = append(fxt.checkFuncs, func() error {
			l := len(fxt.Trackers)
			if l < n {
				return errs.Errorf(checkStr, n, kindTrackers, l)
			}
			return nil
		})
		// Convert fns to []CustomizeEntityFunc
		customFuncs := make([]CustomizeEntityFunc, len(fns))
		for idx := range fns {
			customFuncs[idx] = CustomizeEntityFunc(fns[idx])
		}
		if err := fxt.setupInfo(n, kindTrackers, customFuncs...); err != nil {
			return err
		}
		return fxt.deps()
	}
}
