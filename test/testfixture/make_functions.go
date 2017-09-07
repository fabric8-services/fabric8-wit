package testfixture

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

func makeIdentities(fxt *TestFixture) error {
	if fxt.info[kindIdentities] == nil {
		return nil
	}
	fxt.Identities = make([]*account.Identity, fxt.info[kindIdentities].numInstances)
	for i := range fxt.Identities {
		fxt.Identities[i] = &account.Identity{
			Username:     testsupport.CreateRandomValidTestName("John Doe "),
			ProviderType: "test provider",
		}
		if err := fxt.runCustomizeEntityCallbacks(i, kindIdentities); err != nil {
			return err
		}
		err := testsupport.CreateTestIdentityForAccountIdentity(fxt.db, fxt.Identities[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create identity: %+v", fxt.Identities[i])
		}
	}
	return nil
}

func makeWorkItemLinkCategories(fxt *TestFixture) error {
	if fxt.info[kindWorkItemLinkCategories] == nil {
		return nil
	}
	fxt.WorkItemLinkCategories = make([]*link.WorkItemLinkCategory, fxt.info[kindWorkItemLinkCategories].numInstances)
	wilcRepo := link.NewWorkItemLinkCategoryRepository(fxt.db)
	for i := range fxt.WorkItemLinkCategories {
		desc := "some description"
		fxt.WorkItemLinkCategories[i] = &link.WorkItemLinkCategory{
			Name:        testsupport.CreateRandomValidTestName("link category "),
			Description: &desc,
		}
		fxt.runCustomizeEntityCallbacks(i, kindWorkItemLinkCategories)
		_, err := wilcRepo.Create(fxt.ctx, fxt.WorkItemLinkCategories[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create work item link category: %+v", fxt.WorkItemLinkCategories[i])
		}
	}
	return nil
}

func makeSpaces(fxt *TestFixture) error {
	if fxt.info[kindSpaces] == nil {
		return nil
	}
	fxt.Spaces = make([]*space.Space, fxt.info[kindSpaces].numInstances)
	spaceRepo := space.NewRepository(fxt.db)
	for i := range fxt.Spaces {
		fxt.Spaces[i] = &space.Space{
			Name:        testsupport.CreateRandomValidTestName("space "),
			Description: "Some description",
		}
		if !fxt.isolatedCreation {
			fxt.Spaces[i].OwnerId = fxt.Identities[0].ID
		}
		fxt.runCustomizeEntityCallbacks(i, kindSpaces)
		if fxt.isolatedCreation {
			if fxt.Spaces[i].OwnerId == uuid.Nil {
				return errs.New("you must specify an owner ID for each space")
			}
		}
		_, err := spaceRepo.Create(fxt.ctx, fxt.Spaces[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create space: %+v", fxt.Spaces[i])
		}
	}
	return nil
}

func makeWorkItemLinkTypes(fxt *TestFixture) error {
	if fxt.info[kindWorkItemLinkTypes] == nil {
		return nil
	}
	fxt.WorkItemLinkTypes = make([]*link.WorkItemLinkType, fxt.info[kindWorkItemLinkTypes].numInstances)
	wiltRepo := link.NewWorkItemLinkTypeRepository(fxt.db)
	for i := range fxt.WorkItemLinkTypes {
		desc := "some description"
		fxt.WorkItemLinkTypes[i] = &link.WorkItemLinkType{
			Name:        testsupport.CreateRandomValidTestName("work item link type "),
			Description: &desc,
			Topology:    link.TopologyTree,
			ForwardName: "forward name (e.g. blocks)",
			ReverseName: "reverse name (e.g. blocked by)",
		}
		if !fxt.isolatedCreation {
			fxt.WorkItemLinkTypes[i].SpaceID = fxt.Spaces[0].ID
			fxt.WorkItemLinkTypes[i].LinkCategoryID = fxt.WorkItemLinkCategories[0].ID
		}
		fxt.runCustomizeEntityCallbacks(i, kindWorkItemLinkTypes)
		if fxt.isolatedCreation {
			if fxt.WorkItemLinkTypes[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space for each work item link type")
			}
			if fxt.WorkItemLinkTypes[i].LinkCategoryID == uuid.Nil {
				return errs.New("you must specify a link category for each work item link type")
			}
		}
		_, err := wiltRepo.Create(fxt.ctx, fxt.WorkItemLinkTypes[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create work item link type: %+v", fxt.WorkItemLinkTypes[i])
		}
	}
	return nil
}

func makeIterations(fxt *TestFixture) error {
	if fxt.info[kindIterations] == nil {
		return nil
	}
	fxt.Iterations = make([]*iteration.Iteration, fxt.info[kindIterations].numInstances)
	iterationRepo := iteration.NewIterationRepository(fxt.db)
	for i := range fxt.Iterations {
		desc := "Some description"
		f := false
		fxt.Iterations[i] = &iteration.Iteration{
			Name:        testsupport.CreateRandomValidTestName("iteration "),
			Description: &desc,
			UserActive:  &f,
		}
		if !fxt.isolatedCreation {
			fxt.Iterations[i].SpaceID = fxt.Spaces[0].ID
		}
		fxt.runCustomizeEntityCallbacks(i, kindIterations)
		if fxt.isolatedCreation {
			if fxt.Iterations[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each iteration")
			}
		}
		err := iterationRepo.Create(fxt.ctx, fxt.Iterations[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create iteration: %+v", fxt.Iterations[i])
		}
	}
	return nil
}

func makeAreas(fxt *TestFixture) error {
	if fxt.info[kindAreas] == nil {
		return nil
	}
	fxt.Areas = make([]*area.Area, fxt.info[kindAreas].numInstances)
	areaRepo := area.NewAreaRepository(fxt.db)
	for i := range fxt.Areas {
		//id := uuid.NewV4()
		fxt.Areas[i] = &area.Area{
			//ID:   id,
			Name: testsupport.CreateRandomValidTestName("area "), // + id.String(),
		}
		if !fxt.isolatedCreation {
			fxt.Areas[i].SpaceID = fxt.Spaces[0].ID
		}
		fxt.runCustomizeEntityCallbacks(i, kindAreas)
		if fxt.isolatedCreation {
			if fxt.Areas[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each area")
			}
		}
		err := areaRepo.Create(fxt.ctx, fxt.Areas[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create area: %+v", fxt.Areas[i])
		}
	}
	return nil
}

func makeCodebases(fxt *TestFixture) error {
	if fxt.info[kindCodebases] == nil {
		return nil
	}
	fxt.Codebases = make([]*codebase.Codebase, fxt.info[kindCodebases].numInstances)
	codebaseRepo := codebase.NewCodebaseRepository(fxt.db)
	for i := range fxt.Codebases {
		stackID := "golang-default"
		fxt.Codebases[i] = &codebase.Codebase{
			Type:              "git",
			StackID:           &stackID,
			LastUsedWorkspace: "my-used-last-workspace",
			URL:               "git@github.com:fabric8-services/fabric8-wit.git",
		}
		if !fxt.isolatedCreation {
			fxt.Codebases[i].SpaceID = fxt.Spaces[0].ID
		}
		fxt.runCustomizeEntityCallbacks(i, kindCodebases)
		if fxt.isolatedCreation {
			if fxt.Codebases[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each codebase")
			}
		}
		err := codebaseRepo.Create(fxt.ctx, fxt.Codebases[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create codebase: %+v", fxt.Codebases[i])
		}
	}
	return nil
}

func makeWorkItemTypes(fxt *TestFixture) error {
	if fxt.info[kindWorkItemTypes] == nil {
		return nil
	}
	fxt.WorkItemTypes = make([]*workitem.WorkItemType, fxt.info[kindWorkItemTypes].numInstances)
	witRepo := workitem.NewWorkItemTypeRepository(fxt.db)
	for i := range fxt.WorkItemTypes {
		desc := "this work item type was automatically generated"
		fxt.WorkItemTypes[i] = &workitem.WorkItemType{
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
		if !fxt.isolatedCreation {
			fxt.WorkItemTypes[i].SpaceID = fxt.Spaces[0].ID
		}
		fxt.runCustomizeEntityCallbacks(i, kindWorkItemTypes)
		if fxt.isolatedCreation {
			if fxt.WorkItemTypes[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each work item type")
			}
		}
		_, err := witRepo.CreateFromModel(fxt.ctx, fxt.WorkItemTypes[i])
		if err != nil {
			return errs.Wrapf(err, "failed to create work item type %+v", fxt.WorkItemTypes[i])
		}
	}
	return nil
}

func makeWorkItems(fxt *TestFixture) error {
	if fxt.info[kindWorkItems] == nil {
		return nil
	}
	fxt.WorkItems = make([]*workitem.WorkItem, fxt.info[kindWorkItems].numInstances)
	wiRepo := workitem.NewWorkItemRepository(fxt.db)
	for i := range fxt.WorkItems {
		fxt.WorkItems[i] = &workitem.WorkItem{
			Fields: map[string]interface{}{
				workitem.SystemTitle: testsupport.CreateRandomValidTestName("work item "),
				workitem.SystemState: workitem.SystemStateNew,
			},
		}
		if !fxt.isolatedCreation {
			fxt.WorkItems[i].SpaceID = fxt.Spaces[0].ID
			fxt.WorkItems[i].Type = fxt.WorkItemTypes[0].ID
			fxt.WorkItems[i].Fields[workitem.SystemCreator] = fxt.Identities[0].ID.String()
		}
		fxt.runCustomizeEntityCallbacks(i, kindWorkItems)
		if fxt.isolatedCreation {
			if fxt.WorkItems[i].SpaceID == uuid.Nil {
				return errs.New("you must specify a space ID for each work item")
			}
			if fxt.WorkItems[i].Type == uuid.Nil {
				return errs.New("you must specify a work item type ID for each work item")
			}
			_, ok := fxt.WorkItems[i].Fields[workitem.SystemCreator]
			if !ok {
				return errs.Errorf("you must specify a work creator ID for the \"%s\" field in %+v", workitem.SystemCreator, fxt.WorkItems[i].Fields)
			}
		}
		creatorIDStr, ok := fxt.WorkItems[i].Fields[workitem.SystemCreator].(string)
		if !ok {
			return errs.Errorf("failed to convert \"%s\" field to string in %+v", workitem.SystemCreator, fxt.WorkItems[i].Fields)
		}
		creatorID, err := uuid.FromString(creatorIDStr)
		if err != nil {
			return errs.Wrapf(err, "failed to convert \"%s\" field to uuid.UUID", workitem.SystemCreator)
		}

		wi, err := wiRepo.Create(fxt.ctx, fxt.WorkItems[i].SpaceID, fxt.WorkItems[i].Type, fxt.WorkItems[i].Fields, creatorID)
		if err != nil {
			return errs.Wrapf(err, "failed to create work item: %+v", fxt.WorkItems[i])
		}
		fxt.WorkItems[i] = wi
	}
	return nil
}

func makeWorkItemLinks(fxt *TestFixture) error {
	if fxt.info[kindWorkItemLinks] == nil {
		return nil
	}
	fxt.WorkItemLinks = make([]*link.WorkItemLink, fxt.info[kindWorkItemLinks].numInstances)
	wilRepo := link.NewWorkItemLinkRepository(fxt.db)
	for i := range fxt.WorkItemLinks {
		fxt.WorkItemLinks[i] = &link.WorkItemLink{}
		if !fxt.isolatedCreation {
			fxt.WorkItemLinks[i].LinkTypeID = fxt.WorkItemLinkTypes[0].ID
			// this is the logic that ensures, each work item is only appearing
			// in one link
			fxt.WorkItemLinks[i].SourceID = fxt.WorkItems[2*i].ID
			fxt.WorkItemLinks[i].TargetID = fxt.WorkItems[2*i+1].ID
		}
		fxt.runCustomizeEntityCallbacks(i, kindWorkItemLinks)
		if fxt.isolatedCreation {
			if fxt.WorkItemLinks[i].LinkTypeID == uuid.Nil {
				return errs.New("you must specify a work item link type for each work item link")
			}
			if fxt.WorkItemLinks[i].SourceID == uuid.Nil {
				return errs.New("you must specify a source work item for each work item link")
			}
			if fxt.WorkItemLinks[i].TargetID == uuid.Nil {
				return errs.New("you must specify a target work item for each work item link")
			}
		}
		// default choice for creatorID: take it from the creator of the source work item
		sourceWI, err := workitem.NewWorkItemRepository(fxt.db).LoadByID(fxt.ctx, fxt.WorkItemLinks[i].SourceID)
		if err != nil {
			return errs.Wrapf(err, "failed to load the source work item in order to fetch a creator ID for the link")
		}
		creatorIDStr, ok := sourceWI.Fields[workitem.SystemCreator].(string)
		if !ok {
			return errs.Errorf("failed to fetch the %s field from the source work item %s", workitem.SystemCreator, fxt.WorkItemLinks[i].SourceID)
		}
		creatorID, err := uuid.FromString(creatorIDStr)
		if err != nil {
			return errs.Wrapf(err, "failed to convert the string \"%s\" to a uuid.UUID object", creatorIDStr)
		}

		wilt, err := wilRepo.Create(fxt.ctx, fxt.WorkItemLinks[i].SourceID, fxt.WorkItemLinks[i].TargetID, fxt.WorkItemLinks[i].LinkTypeID, creatorID)
		if err != nil {
			return errs.Wrapf(err, "failed to create work item link: %+v", fxt.WorkItemLinks[i])
		}
		fxt.WorkItemLinks[i] = wilt
	}
	return nil
}

func makeComments(fxt *TestFixture) error {
	if fxt.info[kindComments] == nil {
		return nil
	}
	fxt.Comments = make([]*comment.Comment, fxt.info[kindComments].numInstances)
	commentRepo := comment.NewRepository(fxt.db)
	for i := range fxt.Comments {
		loremIpsum := `Lorem ipsum dolor sitamet, consectetur adipisicing elit, sed do eiusmod
	tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam,
	quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo
	consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum
	dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident,
	sunt in culpa qui officia deserunt mollit anim id est laborum.`
		fxt.Comments[i] = &comment.Comment{
			Markup: rendering.SystemMarkupMarkdown,
			Body:   loremIpsum,
		}
		if !fxt.isolatedCreation {
			fxt.Comments[i].ParentID = fxt.WorkItems[0].ID
			fxt.Comments[i].Creator = fxt.Identities[0].ID
		}
		fxt.runCustomizeEntityCallbacks(i, kindComments)
		if fxt.isolatedCreation {
			if fxt.Comments[i].ParentID == uuid.Nil {
				return errs.New("you must specify a parent work item ID for each comment")
			}
			if fxt.Comments[i].Creator == uuid.Nil {
				return errs.New("you must specify a creator ID for each comment")
			}
		}
		err := commentRepo.Create(fxt.ctx, fxt.Comments[i], fxt.Comments[i].Creator)
		if err != nil {
			return errs.Wrapf(err, "failed to create comment: %+v", fxt.Comments[i])
		}
	}
	return nil
}
