package testcontext

import "github.com/stretchr/testify/require"

// A RecipeFunction ... TODO(kwk): document me
type RecipeFunction func(ctx *TestContext)

const checkStr = "expected at least %d \"%s\" objects but found only %d"

// Identities tells the test context to create at least n identity objects.
//
// If called multiple times with differently n's, the biggest n wins. All
// customize-entitiy-callbacks fns from all calls will be respected when
// creating the test context.
//
// Here's an example how you can create 42 identites and give them a numbered
// user name like "John Doe 0", "John Doe 1", and so forth:
//    Identities(42, func(ctx *TestContext, idx int){
//        ctx.Identities[idx].Username = "Jane Doe " + strconv.FormatInt(idx, 10)
//    })
// Notice that the index idx goes from 0 to n-1 and that you have to manually
// lookup the object from the test context. The identity object referenced by
//    ctx.Identities[idx]
// is guaranteed to be ready to be used for creation. That means, you don't
// necessarily have to touch it to avoid unique key violation for example. This
// is totally optional.
func Identities(n int, fns ...CustomizeEntityCallback) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindIdentities, fns...)
	})
}

// CheckIdentities checks that the number of identities is at least as big as
// the given expected minumum length. If the context was not created with
// NewContextIsolated, this function also checks the minimum required number of
// depending objects. The context's tests fails if these checks fail. If
func (ctx *TestContext) CheckIdentities(expectedMinLen int) {
	l := len(ctx.Identities)
	require.True(ctx.T, l >= expectedMinLen, checkStr, expectedMinLen, kindIdentities, l)
}

// Spaces tells the test context to create at least n space objects. See also
// the Identities() function for more general information on n and fns.
//
// When called in NewContext() this function will call also call
//     Identities(1)
// but with NewContextIsolated(), no other objects will be created.
func Spaces(n int, fns ...CustomizeEntityCallback) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindSpaces, fns...)
		if !ctx.isolatedCreation {
			Identities(1)(ctx)
		}
	})
}

// CheckSpaces checks that the number of spaces is at least as big as the given
// expected minumum length. If the context was not created with
// NewContextIsolated, this function also checks the minimum required number of
// depending objects. The context's tests fails if these checks fail. If
func (ctx *TestContext) CheckSpaces(expectedMinLen int) {
	l := len(ctx.Spaces)
	require.True(ctx.T, l >= expectedMinLen, checkStr, expectedMinLen, kindSpaces, l)
	if !ctx.isolatedCreation {
		ctx.CheckIdentities(1)
	}
}

// Iterations tells the test context to create at least n iteration objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewContext() this function will call also call
//     Spaces(1)
// but with NewContextIsolated(), no other objects will be created.
func Iterations(n int, fns ...CustomizeEntityCallback) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindIterations, fns...)
		if !ctx.isolatedCreation {
			Spaces(1)(ctx)
		}
	})
}

// CheckIterations checks that the number of iterations is at least as big as
// the given expected minumum length. If the context was not created with
// NewContextIsolated, this function also checks the minimum required number of
// depending objects. The context's tests fails if these checks fail. If
func (ctx *TestContext) CheckIterations(expectedMinLen int) {
	l := len(ctx.Iterations)
	require.True(ctx.T, l >= expectedMinLen, checkStr, expectedMinLen, kindIterations, l)
	if !ctx.isolatedCreation {
		ctx.CheckSpaces(1)
	}
}

// Areas tells the test context to create at least n area objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewContext() this function will call also call
//     Spaces(1)
// but with NewContextIsolated(), no other objects will be created.
func Areas(n int, fns ...CustomizeEntityCallback) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindAreas, fns...)
		if !ctx.isolatedCreation {
			Spaces(1)(ctx)
		}
	})
}

// CheckAreas checks that the number of areas is at least as big as the given
// expected minumum length. If the context was not created with
// NewContextIsolated, this function also checks the minimum required number of
// depending objects. The context's tests fails if these checks fail. If
func (ctx *TestContext) CheckAreas(expectedMinLen int) {
	l := len(ctx.Areas)
	require.True(ctx.T, l >= expectedMinLen, checkStr, expectedMinLen, kindAreas, l)
	if !ctx.isolatedCreation {
		ctx.CheckSpaces(1)
	}
}

// Codebases tells the test context to create at least n codebase objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewContext() this function will call also call
//     Spaces(1)
// but with NewContextIsolated(), no other objects will be created.
func Codebases(n int, fns ...CustomizeEntityCallback) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindCodebases, fns...)
		if !ctx.isolatedCreation {
			Spaces(1)(ctx)
		}
	})
}

// CheckCodebases checks that the number of codebases is at least as big as the
// given expected minumum length. If the context was not created with
// NewContextIsolated, this function also checks the minimum required number of
// depending objects. The context's tests fails if these checks fail. If
func (ctx *TestContext) CheckCodebases(expectedMinLen int) {
	l := len(ctx.Codebases)
	require.True(ctx.T, l >= expectedMinLen, checkStr, expectedMinLen, kindCodebases, l)
	if !ctx.isolatedCreation {
		ctx.CheckSpaces(1)
	}
}

// WorkItems tells the test context to create at least n work item objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewContext() this function will call also call
//     Spaces(1)
//     WorkItemTypes(1)
//     Identities(1)
// but with NewContextIsolated(), no other objects will be created.
//
// Notice that the Number field of a work item is only set after the work item
// has been created, so any changes you make to
//     ctx.WorkItems[idx].Number
// will have no effect.
func WorkItems(n int, fns ...CustomizeEntityCallback) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindWorkItems, fns...)
		if !ctx.isolatedCreation {
			Spaces(1)(ctx) // for the space ID
			WorkItemTypes(1)(ctx)
			Identities(1)(ctx) // for the creator ID
		}
	})
}

// CheckWorkItems checks that the number of work items is at least as big as the
// given expected minumum length. If the context was not created with
// NewContextIsolated, this function also checks the minimum required number of
// depending objects. The context's tests fails if these checks fail. If
func (ctx *TestContext) CheckWorkItems(expectedMinLen int) {
	l := len(ctx.WorkItems)
	require.True(ctx.T, l >= expectedMinLen, checkStr, expectedMinLen, kindWorkItems, l)
	if !ctx.isolatedCreation {
		ctx.CheckSpaces(1)
		ctx.CheckWorkItemTypes(1)
		ctx.CheckIdentities(1)
	}
}

// Comments tells the test context to create at least n comment objects. See
// also the Identities() function for more general information on n and fns.
//
// When called in NewContext() this function will call also call
//     Identities(1)
//     WorkItems(1)
// but with NewContextIsolated(), no other objects will be created.
func Comments(n int, fns ...CustomizeEntityCallback) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindComments, fns...)
		if !ctx.isolatedCreation {
			Identities(1)(ctx) // for the creator
			WorkItems(1)(ctx)
		}
	})
}

// CheckComments checks that the number of comments is at least as big as the
// given expected minumum length. If the context was not created with
// NewContextIsolated, this function also checks the minimum required number of
// depending objects. The context's tests fails if these checks fail. If
func (ctx *TestContext) CheckComments(expectedMinLen int) {
	l := len(ctx.Comments)
	require.True(ctx.T, l >= expectedMinLen, checkStr, expectedMinLen, kindComments, l)
	if !ctx.isolatedCreation {
		ctx.CheckWorkItems(1)
		ctx.CheckIdentities(1)
	}
}

// WorkItemTypes tells the test context to create at least n work item type
// objects. See also the Identities() function for more general information on n
// and fns.
//
// When called in NewContext() this function will call also call
//     Spaces(1)
// but with NewContextIsolated(), no other objects will be created.
//
// The work item type that we create for each of the n instances is always the
// same and it tries to be compatible with the planner item work item type by
// specifying the same fields.
func WorkItemTypes(n int, fns ...CustomizeEntityCallback) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindWorkItemTypes, fns...)
		if !ctx.isolatedCreation {
			Spaces(1)(ctx)
		}
	})
}

// CheckWorkItemTypes checks that the number of work item types is at least as
// big as the given expected minumum length. If the context was not created with
// NewContextIsolated, this function also checks the minimum required number of
// depending objects. The context's tests fails if these checks fail. If
func (ctx *TestContext) CheckWorkItemTypes(expectedMinLen int) {
	l := len(ctx.WorkItemTypes)
	require.True(ctx.T, l >= expectedMinLen, checkStr, expectedMinLen, kindWorkItemTypes, l)
	if !ctx.isolatedCreation {
		ctx.CheckSpaces(1)
	}
}

// WorkItemLinkTypes tells the test context to create at least n work item link
// type objects. See also the Identities() function for more general information
// on n and fns.
//
// When called in NewContext() this function will call also call
//     Spaces(1)
//     WorkItemLinkCategories(1)
// but with NewContextIsolated(), no other objects will be created.
//
// We've created these helper functions that you should have a look at if you
// want to implement your own re-usable customize-entity-callbacks:
//     TopologyNetwork()
//     TopologyDirectedNetwork()
//     TopologyDependency()
//     TopologyTree()
//     Topology(topology string) // programmatically set the topology
// The topology functions above are neat because you don't have to write a full
// callback function yourself.
//
// By default a call to
//     WorkItemLinkTypes(1)
// equals
//     WorkItemLinkTypes(1, TopologyTree())
// because we automatically set the topology for each link type to be "tree".
func WorkItemLinkTypes(n int, fns ...CustomizeEntityCallback) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindWorkItemLinkTypes, fns...)
		if !ctx.isolatedCreation {
			Spaces(1)(ctx)
			WorkItemLinkCategories(1)(ctx)
		}
	})
}

// CheckWorkItemLinkTypes checks that the number of work item link types is at
// least as big as the given expected minumum length. If the context was not
// created with NewContextIsolated, this function also checks the minimum
// required number of depending objects. The context's tests fails if these
// checks fail. If
func (ctx *TestContext) CheckWorkItemLinkTypes(expectedMinLen int) {
	l := len(ctx.WorkItemLinkTypes)
	require.True(ctx.T, l >= expectedMinLen, checkStr, expectedMinLen, kindWorkItemLinkTypes, l)
	if !ctx.isolatedCreation {
		ctx.CheckSpaces(1)
		ctx.CheckWorkItemLinkCategories(1)
	}
}

// WorkItemLinkCategories tells the test context to create at least n work item
// link category objects. See also the Identities() function for more general
// information on n and fns.
//
// No other objects will be created.
func WorkItemLinkCategories(n int, fns ...CustomizeEntityCallback) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindWorkItemLinkCategories, fns...)
	})
}

// CheckWorkItemLinkCategories checks that the number of work item link
// categories is at least as big as the given expected minumum length. If the
// context was not created with NewContextIsolated, this function also checks
// the minimum required number of depending objects. The context's tests fails
// if these checks fail. If
func (ctx *TestContext) CheckWorkItemLinkCategories(expectedMinLen int) {
	l := len(ctx.WorkItemLinkCategories)
	require.True(ctx.T, l >= expectedMinLen, checkStr, expectedMinLen, kindWorkItemLinkCategories, l)
}

// WorkItemLinks tells the test context to create at least n work item link
// objects. See also the Identities() function for more general information
// on n and fns.
//
// When called in NewContext() this function will call also call
//     WorkItemLinkTypes(1)
//     WorkItems(2*n)
// but with NewContextIsolated(), no other objects will be created.
//
// Notice, that we will create two times the number of work items of your
// requested links. The way those links will be created can for sure be
// influenced using a customize-entity-callback; but by default we create each
// link between two distinct work items. That means, no link will include the
// same work item.
func WorkItemLinks(n int, fns ...CustomizeEntityCallback) RecipeFunction {
	return RecipeFunction(func(ctx *TestContext) {
		ctx.setupInfo(n, kindWorkItemLinks, fns...)
		if !ctx.isolatedCreation {
			WorkItemLinkTypes(1)(ctx)
			WorkItems(2 * n)(ctx)
		}
	})
}

// CheckWorkItemLinks checks that the number of work item links is at least as
// big as the given expected minumum length. If the context was not created with
// NewContextIsolated, this function also checks the minimum required number of
// depending objects. The context's tests fails if these checks fail. If
func (ctx *TestContext) CheckWorkItemLinks(expectedMinLen int) {
	l := len(ctx.WorkItemLinks)
	require.True(ctx.T, l >= expectedMinLen, checkStr, expectedMinLen, kindWorkItemLinks, l)
	if !ctx.isolatedCreation {
		ctx.CheckWorkItems(expectedMinLen * 2)
		ctx.CheckWorkItemLinkTypes(1)
	}
}
