package testcontext

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/area"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/comment"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

// A TestContext object is the result of a call to
//  NewContext()
// or
//  NewContextIsolated()
//
// Don't create one on your own!
type TestContext struct {
	info             map[kind]*createInfo
	db               *gorm.DB
	isolatedCreation bool
	ctx              context.Context

	// Use this test reference in customize-entity-callbacks in order let the
	// test context creation fail.
	T *testing.T

	Identities             []*account.Identity          // Itentities (if any) that were created for this test context.
	Iterations             []*iteration.Iteration       // Iterations (if any) that were created for this test context.
	Areas                  []*area.Area                 // Areas (if any) that were created for this test context.
	Spaces                 []*space.Space               // Spaces (if any) that were created for this test context.
	Codebases              []*codebase.Codebase         // Codebases (if any) that were created for this test context.
	WorkItems              []*workitem.WorkItem         // Work items (if any) that were created for this test context.
	Comments               []*comment.Comment           // Comments (if any) that were created for this test context.
	WorkItemTypes          []*workitem.WorkItemType     // Work item types (if any) that were created for this test context.
	WorkItemLinkTypes      []*link.WorkItemLinkType     // Work item link types (if any) that were created for this test context.
	WorkItemLinkCategories []*link.WorkItemLinkCategory // Work item link categories (if any) that were created for this test context.
	WorkItemLinks          []*link.WorkItemLink         // Work item links (if any) that were created for this test context.
}

// NewContext will create a test context by executing the recipies from the
// given recipe functions. If recipeFuncs is empty, nothing will happen. If
// there's an error during the setup, the given test t will fail.
//
// For example
//     NewContext(t, db, Comments(100))
// will create a work item (and everything required in order to create it) and
// author 100 comments for it. They will all be created by the same user if you
// don't tell the system to do it differently. For example, to create 100
// comments from 100 different users we can do the following:
//      NewContext(t, db, Identities(100), Comments(100, func(ctx *TestContext, idx int){
//          ctx.Comments[idx].Creator = ctx.Identities[idx].ID
//      }))
// That will create 100 identities and 100 comments and for each comment we're
// using the ID of one of the identities that have been created earlier. There's
// one important observation to make with this example: there's an order to how
// entities get created in the test context. That order is basically defined by
// the number of dependencies that each entity has. For example an identity has
// no dependency, so it will be created first and then can be accessed safely by
// any of the other entity creation functions. A comment for example depends on
// a work item which itself depends on a work item type and a space. The NewContext
// function does take of recursively resolving those dependcies first.
//
// If you just want to create 100 identities and 100 work items but don't care
// about resolving the dependencies automatically you can create the entities in
// isolation:
//      NewContextIsolated(t, db, Identities(100), Comments(100, func(ctx *TestContext, idx int){
//          ctx.Comments[idx].Creator = ctx.Identities[idx].ID
//          ctx.Comments[idx].ParentID = someExistingWorkItemID
//      }))
// Notice that I manually have to specify the ParentID of the work comment then
// because we cannot automatically resolve to which work item we will attach the
// comment.
func NewContext(t *testing.T, db *gorm.DB, recipeFuncs ...RecipeFunction) *TestContext {
	return newContext(t, db, false, recipeFuncs...)
}

// NewContextIsolated will create a test context by executing the recipies from
// the given recipe functions. If recipeFuncs is empty, nothing will happen. If
// there's an error during the setup, the given test t will fail.
//
// The difference to the normal NewContext function is that we will only create
// those object that where specified in the recipeFuncs. We will not create any
// object that is normally demanded by an object. For example, if you call
//     NewContext(t, db, WorkItems(1))
// you would (apart from other objects) get at least one work item AND a work
// item type because that is needed to create a work item. With
//     NewContextIsolated(t, db, Comments(2), WorkItems(1))
// on the other hand, we will only create a work item, two comments for it, and
// nothing more. And for sure your test will fail if you do that because you
// need to specify a space ID and a work item type ID for the created work item:
//     NewContextIsolated(t, db, Comments(2), WorkItems(1, func(ctx *TestContext, idx int){
//       ctx.WorkItems[idx].SpaceID = someExistingSpaceID
//       ctx.WorkItems[idx].WorkItemType = someExistingWorkItemTypeID
//     }))
func NewContextIsolated(t *testing.T, db *gorm.DB, setupFuncs ...RecipeFunction) *TestContext {
	return newContext(t, db, true, setupFuncs...)
}

type kind string

const (
	kindIdentities             kind = "identities"
	kindIterations             kind = "iterations"
	kindAreas                  kind = "areas"
	kindSpaces                 kind = "spaces"
	kindCodebases              kind = "codebases"
	kindWorkItems              kind = "work_items"
	kindComments               kind = "comments"
	kindWorkItemTypes          kind = "work_item_types"
	kindWorkItemLinkTypes      kind = "work_item_link_types"
	kindWorkItemLinkCategories kind = "work_item_link_categories"
	kindWorkItemLinks          kind = "work_item_links"
)

type createInfo struct {
	numInstances             int
	customizeEntityCallbacks []CustomizeEntityCallback
}

func (ctx *TestContext) runCustomizeEntityCallbacks(idx int, k kind) {
	if ctx.info[k] == nil {
		ctx.T.Fatalf("the creation info for kind %s is nil (this should not happen)", k)
	}
	for _, dfn := range ctx.info[k].customizeEntityCallbacks {
		dfn(ctx, idx)
	}
}

func (ctx *TestContext) setupInfo(n int, k kind, fns ...CustomizeEntityCallback) {
	require.True(ctx.T, n > 0, "the number of objects to create must always be greater than zero")
	if _, ok := ctx.info[k]; !ok {
		ctx.info[k] = &createInfo{}
	}
	maxN := n
	if maxN < ctx.info[k].numInstances {
		maxN = ctx.info[k].numInstances
	}
	ctx.info[k].numInstances = maxN
	ctx.info[k].customizeEntityCallbacks = append(ctx.info[k].customizeEntityCallbacks, fns...)
}

func newContext(t *testing.T, db *gorm.DB, isolatedCreation bool, recipeFuncs ...RecipeFunction) *TestContext {
	ctx := TestContext{
		T:                t,
		info:             map[kind]*createInfo{},
		db:               db,
		isolatedCreation: isolatedCreation,
		ctx:              context.Background(),
	}

	for _, fn := range recipeFuncs {
		fn(&ctx)
	}

	// actually make the objects that DON'T have any dependencies
	makeIdentities(&ctx)
	makeWorkItemLinkCategories(&ctx)

	// actually make the objects that DO have any dependencies
	makeSpaces(&ctx)
	makeWorkItemLinkTypes(&ctx)
	makeCodebases(&ctx)
	makeWorkItemTypes(&ctx)
	makeIterations(&ctx)
	makeAreas(&ctx)
	makeWorkItems(&ctx)
	makeComments(&ctx)
	makeWorkItemLinks(&ctx)

	return &ctx
}

func makeIdentities(ctx *TestContext) {
	if ctx.info[kindIdentities] == nil {
		return
	}
	ctx.Identities = make([]*account.Identity, ctx.info[kindIdentities].numInstances)
	for i := range ctx.Identities {
		ctx.Identities[i] = &account.Identity{
			Username:     testsupport.CreateRandomValidTestName("John Doe "),
			ProviderType: "test provider",
		}

		ctx.runCustomizeEntityCallbacks(i, kindIdentities)

		err := testsupport.CreateTestIdentityForAccountIdentity(ctx.db, ctx.Identities[i])
		require.Nil(ctx.T, err, "failed to create identity: %+v", ctx.Identities[i])
	}
}

func makeWorkItemLinkCategories(ctx *TestContext) {
	if ctx.info[kindWorkItemLinkCategories] == nil {
		return
	}
	ctx.WorkItemLinkCategories = make([]*link.WorkItemLinkCategory, ctx.info[kindWorkItemLinkCategories].numInstances)
	for i := range ctx.WorkItemLinkCategories {
		desc := "some description"
		ctx.WorkItemLinkCategories[i] = &link.WorkItemLinkCategory{
			Name:        testsupport.CreateRandomValidTestName("link category "),
			Description: &desc,
		}
		ctx.runCustomizeEntityCallbacks(i, kindWorkItemLinkCategories)
		_, err := link.NewWorkItemLinkCategoryRepository(ctx.db).Create(ctx.ctx, ctx.WorkItemLinkCategories[i])
		require.Nil(ctx.T, err, "failed to create work item link category: %+v", ctx.WorkItemLinkCategories[i])
	}
}

func makeSpaces(ctx *TestContext) {
	if ctx.info[kindSpaces] == nil {
		return
	}
	ctx.Spaces = make([]*space.Space, ctx.info[kindSpaces].numInstances)
	for i := range ctx.Spaces {
		ctx.Spaces[i] = &space.Space{
			Name:        testsupport.CreateRandomValidTestName("space "),
			Description: "Some description",
		}
		if !ctx.isolatedCreation {
			ctx.Spaces[i].OwnerId = ctx.Identities[0].ID
		}
		ctx.runCustomizeEntityCallbacks(i, kindSpaces)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.Spaces[i].OwnerId, "you must specify an owner ID for each space")
		}
		_, err := space.NewRepository(ctx.db).Create(ctx.ctx, ctx.Spaces[i])
		require.Nil(ctx.T, err, "failed to create space: %+v", ctx.Spaces[i])
	}
}

func makeWorkItemLinkTypes(ctx *TestContext) {
	if ctx.info[kindWorkItemLinkTypes] == nil {
		return
	}
	ctx.WorkItemLinkTypes = make([]*link.WorkItemLinkType, ctx.info[kindWorkItemLinkTypes].numInstances)
	for i := range ctx.WorkItemLinkTypes {
		desc := "some description"
		ctx.WorkItemLinkTypes[i] = &link.WorkItemLinkType{
			Name:        testsupport.CreateRandomValidTestName("work item link type "),
			Description: &desc,
			Topology:    link.TopologyTree,
			ForwardName: "forward name (e.g. blocks)",
			ReverseName: "reverse name (e.g. blocked by)",
		}
		if !ctx.isolatedCreation {
			ctx.WorkItemLinkTypes[i].SpaceID = ctx.Spaces[0].ID
			ctx.WorkItemLinkTypes[i].LinkCategoryID = ctx.WorkItemLinkCategories[0].ID
		}
		ctx.runCustomizeEntityCallbacks(i, kindWorkItemLinkTypes)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItemLinkTypes[i].SpaceID, "you must specify a space for each work item link type")
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItemLinkTypes[i].LinkCategoryID, "you must specify a link category for each work item link type")
		}
		_, err := link.NewWorkItemLinkTypeRepository(ctx.db).Create(ctx.ctx, ctx.WorkItemLinkTypes[i])
		require.Nil(ctx.T, err, "failed to create work item link type: %+v", ctx.WorkItemLinkTypes[i])
	}
}

func makeIterations(ctx *TestContext) {
	if ctx.info[kindIterations] == nil {
		return
	}
	ctx.Iterations = make([]*iteration.Iteration, ctx.info[kindIterations].numInstances)
	for i := range ctx.Iterations {
		desc := "Some description"
		ctx.Iterations[i] = &iteration.Iteration{
			Name:        testsupport.CreateRandomValidTestName("iteration "),
			Description: &desc,
		}
		if !ctx.isolatedCreation {
			ctx.Iterations[i].SpaceID = ctx.Spaces[0].ID
		}
		ctx.runCustomizeEntityCallbacks(i, kindIterations)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.Iterations[i].SpaceID, "you must specify a space ID for each iteration")
		}
		err := iteration.NewIterationRepository(ctx.db).Create(ctx.ctx, ctx.Iterations[i])
		require.Nil(ctx.T, err, "failed to create iteration: %+v", ctx.Iterations[i])
	}
}

func makeAreas(ctx *TestContext) {
	if ctx.info[kindAreas] == nil {
		return
	}
	ctx.Areas = make([]*area.Area, ctx.info[kindAreas].numInstances)
	for i := range ctx.Areas {
		//id := uuid.NewV4()
		ctx.Areas[i] = &area.Area{
			//ID:   id,
			Name: testsupport.CreateRandomValidTestName("area "), // + id.String(),
		}
		if !ctx.isolatedCreation {
			ctx.Areas[i].SpaceID = ctx.Spaces[0].ID
		}
		ctx.runCustomizeEntityCallbacks(i, kindAreas)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.Areas[i].SpaceID, "you must specify a space ID for each area")
		}
		err := area.NewAreaRepository(ctx.db).Create(ctx.ctx, ctx.Areas[i])
		require.Nil(ctx.T, err, "failed to create area: %+v", ctx.Areas[i])
	}
}

func makeCodebases(ctx *TestContext) {
	if ctx.info[kindCodebases] == nil {
		return
	}
	ctx.Codebases = make([]*codebase.Codebase, ctx.info[kindCodebases].numInstances)
	for i := range ctx.Codebases {
		stackID := "golang-default"
		ctx.Codebases[i] = &codebase.Codebase{
			Type:              "git",
			StackID:           &stackID,
			LastUsedWorkspace: "my-used-last-workspace",
			URL:               "git@github.com:fabric8-services/fabric8-wit.git",
		}
		if !ctx.isolatedCreation {
			ctx.Codebases[i].SpaceID = ctx.Spaces[0].ID
		}
		ctx.runCustomizeEntityCallbacks(i, kindCodebases)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.Codebases[i].SpaceID, "you must specify a space ID for each codebase")
		}
		err := codebase.NewCodebaseRepository(ctx.db).Create(ctx.ctx, ctx.Codebases[i])
		require.Nil(ctx.T, err, "failed to create codebase: %+v", ctx.Codebases[i])
	}
}

func makeWorkItemTypes(ctx *TestContext) {
	if ctx.info[kindWorkItemTypes] == nil {
		return
	}
	ctx.WorkItemTypes = make([]*workitem.WorkItemType, ctx.info[kindWorkItemTypes].numInstances)
	for i := range ctx.WorkItemTypes {
		desc := "this work item type was automatically generated"
		ctx.WorkItemTypes[i] = &workitem.WorkItemType{
			Name:        testsupport.CreateRandomValidTestName("work item type "),
			Description: &desc,
			Icon:        "fa-bug",
			Fields: map[string]workitem.FieldDefinition{
				workitem.SystemTitle:        {Type: workitem.SimpleType{Kind: "string"}, Required: true, Label: "Title", Description: "The title text of the work item"},
				workitem.SystemDescription:  {Type: workitem.SimpleType{Kind: "markup"}, Required: false, Label: "Description", Description: "A descriptive text of the work item"},
				workitem.SystemCreator:      {Type: workitem.SimpleType{Kind: "user"}, Required: true, Label: "Creator", Description: "The user that created the work item"},
				workitem.SystemRemoteItemID: {Type: workitem.SimpleType{Kind: "string"}, Required: false, Label: "Remote item", Description: "The ID of the remote work item"},
				workitem.SystemCreatedAt:    {Type: workitem.SimpleType{Kind: "instant"}, Required: false, Label: "Created at", Description: "The date and time when the work item was created"},
				workitem.SystemUpdatedAt:    {Type: workitem.SimpleType{Kind: "instant"}, Required: false, Label: "Updated at", Description: "The date and time when the work item was last updated"},
				workitem.SystemOrder:        {Type: workitem.SimpleType{Kind: "float"}, Required: false, Label: "Execution Order", Description: "Execution Order of the workitem."},
				workitem.SystemIteration:    {Type: workitem.SimpleType{Kind: "iteration"}, Required: false, Label: "Iteration", Description: "The iteration to which the work item belongs"},
				workitem.SystemArea:         {Type: workitem.SimpleType{Kind: "area"}, Required: false, Label: "Area", Description: "The area to which the work item belongs"},
				workitem.SystemCodebase:     {Type: workitem.SimpleType{Kind: "codebase"}, Required: false, Label: "Codebase", Description: "Contains codebase attributes to which this WI belongs to"},
				workitem.SystemAssignees: {
					Type: &workitem.ListType{
						SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
						ComponentType: workitem.SimpleType{Kind: workitem.KindUser}},
					Required:    false,
					Label:       "Assignees",
					Description: "The users that are assigned to the work item",
				},
				workitem.SystemState: {
					Type: &workitem.EnumType{
						SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
						BaseType:   workitem.SimpleType{Kind: workitem.KindString},
						Values: []interface{}{
							workitem.SystemStateNew,
							workitem.SystemStateOpen,
							workitem.SystemStateInProgress,
							workitem.SystemStateResolved,
							workitem.SystemStateClosed,
						},
					},

					Required:    true,
					Label:       "State",
					Description: "The state of the work item",
				},
			},
		}
		if !ctx.isolatedCreation {
			ctx.WorkItemTypes[i].SpaceID = ctx.Spaces[0].ID
		}
		ctx.runCustomizeEntityCallbacks(i, kindWorkItemTypes)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItemTypes[i].SpaceID, "you must specify a space ID for each work item type")
		}
		_, err := workitem.NewWorkItemTypeRepository(ctx.db).CreateFromModel(ctx.ctx, ctx.WorkItemTypes[i])
		require.Nil(ctx.T, err)
	}
}

func makeWorkItems(ctx *TestContext) {
	if ctx.info[kindWorkItems] == nil {
		return
	}
	ctx.WorkItems = make([]*workitem.WorkItem, ctx.info[kindWorkItems].numInstances)
	for i := range ctx.WorkItems {
		ctx.WorkItems[i] = &workitem.WorkItem{
			Fields: map[string]interface{}{
				workitem.SystemTitle: testsupport.CreateRandomValidTestName("work item "),
				workitem.SystemState: workitem.SystemStateNew,
			},
		}
		if !ctx.isolatedCreation {
			ctx.WorkItems[i].SpaceID = ctx.Spaces[0].ID
			ctx.WorkItems[i].Type = ctx.WorkItemTypes[0].ID
			ctx.WorkItems[i].Fields[workitem.SystemCreator] = ctx.Identities[0].ID.String()
		}
		ctx.runCustomizeEntityCallbacks(i, kindWorkItems)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItems[i].SpaceID, "you must specify a space ID for each work item")
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItems[i].Type, "you must specify a work item type ID for each work item")
			_, ok := ctx.WorkItems[i].Fields[workitem.SystemCreator]
			require.True(ctx.T, ok, "you must specify a work creator ID for the \"%s\" field in %+v", workitem.SystemCreator, ctx.WorkItems[i].Fields)
		}
		creatorIDStr, ok := ctx.WorkItems[i].Fields[workitem.SystemCreator].(string)
		require.True(ctx.T, ok, "failed to convert \"%s\" field to string", workitem.SystemCreator)
		creatorID, err := uuid.FromString(creatorIDStr)
		require.Nil(ctx.T, err, "failed to convert \"%s\" field to uuid.UUID", workitem.SystemCreator)

		wi, err := workitem.NewWorkItemRepository(ctx.db).Create(ctx.ctx, ctx.WorkItems[i].SpaceID, ctx.WorkItems[i].Type, ctx.WorkItems[i].Fields, creatorID)
		require.Nil(ctx.T, err, "failed to create work item: %+v", ctx.WorkItems[i])
		ctx.WorkItems[i] = wi
	}
}

func makeWorkItemLinks(ctx *TestContext) {
	if ctx.info[kindWorkItemLinks] == nil {
		return
	}
	ctx.WorkItemLinks = make([]*link.WorkItemLink, ctx.info[kindWorkItemLinks].numInstances)
	for i := range ctx.WorkItemLinks {
		ctx.WorkItemLinks[i] = &link.WorkItemLink{}
		if !ctx.isolatedCreation {
			ctx.WorkItemLinks[i].LinkTypeID = ctx.WorkItemLinkTypes[0].ID
			// this is the logic that ensures, each work item is only appearing
			// in one link
			ctx.WorkItemLinks[i].SourceID = ctx.WorkItems[2*i].ID
			ctx.WorkItemLinks[i].TargetID = ctx.WorkItems[2*i+1].ID
		}
		ctx.runCustomizeEntityCallbacks(i, kindWorkItemLinks)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItemLinks[i].LinkTypeID, "you must specify a work item link type for each work item link")
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItemLinks[i].SourceID, "you must specify a source work item for each work item link")
			require.NotEqual(ctx.T, uuid.Nil, ctx.WorkItemLinks[i].TargetID, "you must specify a target work item for each work item link")
		}
		// default choice for creatorID: take it from the creator of the source work item
		sourceWI, err := workitem.NewWorkItemRepository(ctx.db).LoadByID(ctx.ctx, ctx.WorkItemLinks[i].SourceID)
		require.Nil(ctx.T, err, "failed to load the source work item in order to fetch a creator ID for the link")
		creatorIDStr, ok := sourceWI.Fields[workitem.SystemCreator].(string)
		require.True(ctx.T, ok, "failed to fetch the %s field from the source work item %s", workitem.SystemCreator, ctx.WorkItemLinks[i].SourceID)
		creatorID, err := uuid.FromString(creatorIDStr)
		require.Nil(ctx.T, err, "failed to convert the string \"%s\" to a uuid.UUID object", creatorIDStr)

		wilt, err := link.NewWorkItemLinkRepository(ctx.db).Create(ctx.ctx, ctx.WorkItemLinks[i].SourceID, ctx.WorkItemLinks[i].TargetID, ctx.WorkItemLinks[i].LinkTypeID, creatorID)
		require.Nil(ctx.T, err, "failed to create work item link: %+v", ctx.WorkItemLinks[i])
		ctx.WorkItemLinks[i] = wilt
	}
}

func makeComments(ctx *TestContext) {
	if ctx.info[kindComments] == nil {
		return
	}
	ctx.Comments = make([]*comment.Comment, ctx.info[kindComments].numInstances)
	for i := range ctx.Comments {
		loremIpsum := `Lorem ipsum dolor sitamet, consectetur adipisicing elit, sed do eiusmod
tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam,
quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo
consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum
dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident,
sunt in culpa qui officia deserunt mollit anim id est laborum.`
		ctx.Comments[i] = &comment.Comment{
			Markup: rendering.SystemMarkupMarkdown,
			Body:   loremIpsum,
		}
		if !ctx.isolatedCreation {
			ctx.Comments[i].ParentID = ctx.WorkItems[0].ID
			ctx.Comments[i].Creator = ctx.Identities[0].ID
		}
		ctx.runCustomizeEntityCallbacks(i, kindComments)
		if ctx.isolatedCreation {
			require.NotEqual(ctx.T, uuid.Nil, ctx.Comments[i].ParentID, "you must specify a parent work item ID for each comment")
			require.NotEqual(ctx.T, uuid.Nil, ctx.Comments[i].Creator, "you must specify a creator ID for each comment")
		}
		err := comment.NewRepository(ctx.db).Create(ctx.ctx, ctx.Comments[i], ctx.Comments[i].Creator)
		require.Nil(ctx.T, err, "failed to create comment: %+v", ctx.Comments[i])
	}
}
