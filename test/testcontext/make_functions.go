package testcontext

import (
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
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

func makeIdentities(ctx *TestContext) error {
	if ctx.info[kindIdentities] == nil {
		return nil
	}
	ctx.Identities = make([]*account.Identity, ctx.info[kindIdentities].numInstances)
	for i := range ctx.Identities {
		ctx.Identities[i] = &account.Identity{
			Username:     testsupport.CreateRandomValidTestName("John Doe "),
			ProviderType: "test provider",
		}
		if err := ctx.runCustomizeEntityCallbacks(i, kindIdentities); err != nil {
			return err
		}
		err := testsupport.CreateTestIdentityForAccountIdentity(ctx.db, ctx.Identities[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create identity: %+v", ctx.Identities[i])
		}
	}
	return nil
}

func makeWorkItemLinkCategories(ctx *TestContext) error {
	if ctx.info[kindWorkItemLinkCategories] == nil {
		return nil
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
		if err != nil {
			return errs.Wrapf(err, "failed to create work item link category: %+v", ctx.WorkItemLinkCategories[i])
		}
	}
	return nil
}

func makeSpaces(ctx *TestContext) error {
	if ctx.info[kindSpaces] == nil {
		return nil
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
			if ctx.Spaces[i].OwnerId == uuid.Nil {
				return errs.New("you must specify an owner ID for each space")
			}
		}
		_, err := space.NewRepository(ctx.db).Create(ctx.ctx, ctx.Spaces[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create space: %+v", ctx.Spaces[i])
		}
	}
	return nil
}

func makeWorkItemLinkTypes(ctx *TestContext) error {
	if ctx.info[kindWorkItemLinkTypes] == nil {
		return nil
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
			if ctx.WorkItemLinkTypes[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space for each work item link type")
			}
			if ctx.WorkItemLinkTypes[i].LinkCategoryID == uuid.Nil {
				return errs.New("you must specify a link category for each work item link type")
			}
		}
		_, err := link.NewWorkItemLinkTypeRepository(ctx.db).Create(ctx.ctx, ctx.WorkItemLinkTypes[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create work item link type: %+v", ctx.WorkItemLinkTypes[i])
		}
	}
	return nil
}

func makeIterations(ctx *TestContext) error {
	if ctx.info[kindIterations] == nil {
		return nil
	}
	ctx.Iterations = make([]*iteration.Iteration, ctx.info[kindIterations].numInstances)
	for i := range ctx.Iterations {
		desc := "Some description"
		f := false
		ctx.Iterations[i] = &iteration.Iteration{
			Name:        testsupport.CreateRandomValidTestName("iteration "),
			Description: &desc,
			UserActive:  &f,
		}
		if !ctx.isolatedCreation {
			ctx.Iterations[i].SpaceID = ctx.Spaces[0].ID
		}
		ctx.runCustomizeEntityCallbacks(i, kindIterations)
		if ctx.isolatedCreation {
			if ctx.Iterations[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each iteration")
			}
		}
		err := iteration.NewIterationRepository(ctx.db).Create(ctx.ctx, ctx.Iterations[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create iteration: %+v", ctx.Iterations[i])
		}
	}
	return nil
}

func makeAreas(ctx *TestContext) error {
	if ctx.info[kindAreas] == nil {
		return nil
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
			if ctx.Areas[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each area")
			}
		}
		err := area.NewAreaRepository(ctx.db).Create(ctx.ctx, ctx.Areas[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create area: %+v", ctx.Areas[i])
		}
	}
	return nil
}

func makeCodebases(ctx *TestContext) error {
	if ctx.info[kindCodebases] == nil {
		return nil
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
			if ctx.Codebases[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each codebase")
			}
		}
		err := codebase.NewCodebaseRepository(ctx.db).Create(ctx.ctx, ctx.Codebases[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create codebase: %+v", ctx.Codebases[i])
		}
	}
	return nil
}

func makeWorkItemTypes(ctx *TestContext) error {
	if ctx.info[kindWorkItemTypes] == nil {
		return nil
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
			if ctx.WorkItemTypes[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each work item type")
			}
		}
		_, err := workitem.NewWorkItemTypeRepository(ctx.db).CreateFromModel(ctx.ctx, ctx.WorkItemTypes[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create work item type %+v", ctx.WorkItemTypes[i])
		}
	}
	return nil
}

func makeWorkItems(ctx *TestContext) error {
	if ctx.info[kindWorkItems] == nil {
		return nil
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
			if ctx.WorkItems[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each work item")
			}
			if ctx.WorkItems[i].Type == uuid.Nil {
				return errs.New("you must specify a work item type ID for each work item")
			}
			_, ok := ctx.WorkItems[i].Fields[workitem.SystemCreator]
			if !ok {
				return errs.Errorf("you must specify a work creator ID for the \"%s\" field in %+v", workitem.SystemCreator, ctx.WorkItems[i].Fields)
			}
		}
		creatorIDStr, ok := ctx.WorkItems[i].Fields[workitem.SystemCreator].(string)
		if !ok {
			return errs.Errorf("failed to convert \"%s\" field to string", workitem.SystemCreator)
		}
		creatorID, err := uuid.FromString(creatorIDStr)
		if err != nil {
			return errs.Wrapf(err, "failed to convert \"%s\" field to uuid.UUID", workitem.SystemCreator)
		}

		wi, err := workitem.NewWorkItemRepository(ctx.db).Create(ctx.ctx, ctx.WorkItems[i].SpaceID, ctx.WorkItems[i].Type, ctx.WorkItems[i].Fields, creatorID)
		if err != nil {
			return errs.Wrapf(err, "failed to create work item: %+v", ctx.WorkItems[i])
		}
		ctx.WorkItems[i] = wi
	}
	return nil
}

func makeWorkItemLinks(ctx *TestContext) error {
	if ctx.info[kindWorkItemLinks] == nil {
		return nil
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
			if ctx.WorkItemLinks[i].LinkTypeID == uuid.Nil {
				return errs.New("you must specify a work item link type for each work item link")
			}
			if ctx.WorkItemLinks[i].SourceID == uuid.Nil {
				return errs.New("you must specify a source work item for each work item link")
			}
			if ctx.WorkItemLinks[i].TargetID == uuid.Nil {
				return errs.New("you must specify a target work item for each work item link")
			}
		}
		// default choice for creatorID: take it from the creator of the source work item
		sourceWI, err := workitem.NewWorkItemRepository(ctx.db).LoadByID(ctx.ctx, ctx.WorkItemLinks[i].SourceID)
		if err != nil {
			return errs.Wrapf(err, "failed to load the source work item in order to fetch a creator ID for the link")
		}
		creatorIDStr, ok := sourceWI.Fields[workitem.SystemCreator].(string)
		if !ok {
			return errs.Errorf("failed to fetch the %s field from the source work item %s", workitem.SystemCreator, ctx.WorkItemLinks[i].SourceID)
		}
		creatorID, err := uuid.FromString(creatorIDStr)
		if err != nil {
			return errs.Wrapf(err, "failed to convert the string \"%s\" to a uuid.UUID object", creatorIDStr)
		}

		wilt, err := link.NewWorkItemLinkRepository(ctx.db).Create(ctx.ctx, ctx.WorkItemLinks[i].SourceID, ctx.WorkItemLinks[i].TargetID, ctx.WorkItemLinks[i].LinkTypeID, creatorID)
		if err != nil {
			return errs.Wrapf(err, "failed to create work item link: %+v", ctx.WorkItemLinks[i])
		}
		ctx.WorkItemLinks[i] = wilt
	}
	return nil
}

func makeComments(ctx *TestContext) error {
	if ctx.info[kindComments] == nil {
		return nil
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
			if ctx.Comments[i].ParentID == uuid.Nil {
				return errs.New("you must specify a parent work item ID for each comment")
			}
			if ctx.Comments[i].Creator == uuid.Nil {
				return errs.New("you must specify a creator ID for each comment")
			}
		}
		err := comment.NewRepository(ctx.db).Create(ctx.ctx, ctx.Comments[i], ctx.Comments[i].Creator)
		if err != nil {
			return errs.Wrapf(err, "failed to create comment: %+v", ctx.Comments[i])
		}
	}
	return nil
}
