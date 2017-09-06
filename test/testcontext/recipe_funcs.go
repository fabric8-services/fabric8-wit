package testcontext

// A RecipeFunction ... TODO(kwk): document me
type RecipeFunction func(ctx *TestContext)

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
