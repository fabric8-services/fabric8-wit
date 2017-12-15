package testfixture

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/area"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/comment"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/label"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// A TestFixture object is the result of a call to
//  NewFixture()
// or
//  NewFixtureIsolated()
//
// Don't create one on your own!
type TestFixture struct {
	info               map[kind]*createInfo
	db                 *gorm.DB
	isolatedCreation   bool
	ctx                context.Context
	checkFuncs         []func() error
	customLinkCreation bool // on when you've used WorkItemLinksCustom in your recipe
	normalLinkCreation bool // on when you've used WorkItemLinks in your recipe

	Identities             []*account.Identity          // Itentities (if any) that were created for this test fixture.
	Iterations             []*iteration.Iteration       // Iterations (if any) that were created for this test fixture.
	Areas                  []*area.Area                 // Areas (if any) that were created for this test fixture.
	Spaces                 []*space.Space               // Spaces (if any) that were created for this test fixture.
	Codebases              []*codebase.Codebase         // Codebases (if any) that were created for this test fixture.
	WorkItems              []*workitem.WorkItem         // Work items (if any) that were created for this test fixture.
	Comments               []*comment.Comment           // Comments (if any) that were created for this test fixture.
	WorkItemTypes          []*workitem.WorkItemType     // Work item types (if any) that were created for this test fixture.
	WorkItemLinkTypes      []*link.WorkItemLinkType     // Work item link types (if any) that were created for this test fixture.
	WorkItemLinkCategories []*link.WorkItemLinkCategory // Work item link categories (if any) that were created for this test fixture.
	WorkItemLinks          []*link.WorkItemLink         // Work item links (if any) that were created for this test fixture.
	Labels                 []*label.Label
	Trackers               []*remoteworkitem.Tracker // Remote work item tracker (if any) that were created for this test fixture.
}

// NewFixture will create a test fixture by executing the recipies from the
// given recipe functions. If recipeFuncs is empty, nothing will happen.
//
// For example
//     NewFixture(db, Comments(100))
// will create a work item (and everything required in order to create it) and
// author 100 comments for it. They will all be created by the same user if you
// don't tell the system to do it differently. For example, to create 100
// comments from 100 different users we can do the following:
//      NewFixture(db, Identities(100), Comments(100, func(fxt *TestFixture, idx int) error{
//          fxt.Comments[idx].Creator = fxt.Identities[idx].ID
//          return nil
//      }))
// That will create 100 identities and 100 comments and for each comment we're
// using the ID of one of the identities that have been created earlier. There's
// one important observation to make with this example: there's an order to how
// entities get created in the test fixture. That order is basically defined by
// the number of dependencies that each entity has. For example an identity has
// no dependency, so it will be created first and then can be accessed safely by
// any of the other entity creation functions. A comment for example depends on
// a work item which itself depends on a work item type and a space. The NewFixture
// function does take care of recursively resolving those dependcies first.
//
// If you just want to create 100 identities and 100 work items but don't care
// about resolving the dependencies automatically you can create the entities in
// isolation:
//      NewFixtureIsolated(db, Identities(100), Comments(100, func(fxt *TestFixture, idx int) error{
//          fxt.Comments[idx].Creator = fxt.Identities[idx].ID
//          fxt.Comments[idx].ParentID = someExistingWorkItemID
//          return nil
//      }))
// Notice that I manually have to specify the ParentID of the work comment then
// because we cannot automatically resolve to which work item we will attach the
// comment.
func NewFixture(db *gorm.DB, recipeFuncs ...RecipeFunction) (*TestFixture, error) {
	return newFixture(db, false, recipeFuncs...)
}

// NewTestFixture does the same as NewFixture except that it automatically
// fails the given test if the fixture could not be created correctly.
func NewTestFixture(t testing.TB, db *gorm.DB, recipeFuncs ...RecipeFunction) *TestFixture {
	resource.Require(t, resource.Database)

	tc, err := NewFixture(db, recipeFuncs...)
	require.NoError(t, err, "%+v", err)
	require.NotNil(t, tc)
	return tc
}

// NewFixtureIsolated will create a test fixture by executing the recipies from
// the given recipe functions. If recipeFuncs is empty, nothing will happen.
//
// The difference to the normal NewFixture function is that we will only create
// those object that where specified in the recipeFuncs. We will not create any
// object that is normally demanded by an object. For example, if you call
//     NewFixture(t, db, WorkItems(1))
// you would (apart from other objects) get at least one work item AND a work
// item type because that is needed to create a work item. With
//     NewFixtureIsolated(t, db, Comments(2), WorkItems(1))
// on the other hand, we will only create a work item, two comments for it, and
// nothing more. And for sure your test will fail if you do that because you
// need to specify a space ID and a work item type ID for the created work item:
//     NewFixtureIsolated(db, Comments(2), WorkItems(1, func(fxt *TestFixture, idx int) error{
//       fxt.WorkItems[idx].SpaceID = someExistingSpaceID
//       fxt.WorkItems[idx].WorkItemType = someExistingWorkItemTypeID
//       return nil
//     }))
func NewFixtureIsolated(db *gorm.DB, setupFuncs ...RecipeFunction) (*TestFixture, error) {
	return newFixture(db, true, setupFuncs...)
}

// Check runs all check functions that each recipe-function has registered to
// check that the amount of objects has been created that were demanded in the
// recipe function.
//
// In this example
//     fxt, _:= NewFixture(db, WorkItems(2))
//     err = fxt.Check()
// err will only be nil if at least two work items have been created and all of
// the dependencies that a work item requires. Look into the documentation of
// each recipe-function to find out what dependencies each entity has.
//
// Notice, that check is called at the end of NewFixture() and its derivatives,
// so if you don't mess with the fixture after it was created, there's no need
// to call Check() again.
func (fxt *TestFixture) Check() error {
	for _, fn := range fxt.checkFuncs {
		if err := fn(); err != nil {
			return errs.Wrap(err, "check function failed")
		}
	}
	return nil
}

type kind string

const (
	kindIdentities             kind = "identity"
	kindIterations             kind = "iteration"
	kindAreas                  kind = "area"
	kindSpaces                 kind = "space"
	kindCodebases              kind = "codebase"
	kindWorkItems              kind = "work_item"
	kindComments               kind = "comment"
	kindWorkItemTypes          kind = "work_item_type"
	kindWorkItemLinkTypes      kind = "work_item_link_type"
	kindWorkItemLinkCategories kind = "work_item_link_category"
	kindWorkItemLinks          kind = "work_item_link"
	kindLabels                 kind = "label"
	kindTrackers               kind = "tracker"
)

type createInfo struct {
	numInstances         int
	customizeEntityFuncs []CustomizeEntityFunc
}

func (fxt *TestFixture) runCustomizeEntityFuncs(idx int, k kind) error {
	if fxt.info[k] == nil {
		return errs.Errorf("the creation info for kind %s is nil (this should not happen)", k)
	}
	for _, dfn := range fxt.info[k].customizeEntityFuncs {
		if err := dfn(fxt, idx); err != nil {
			return errs.Wrapf(err, "failed to run customize-entity-callbacks for kind %s", k)
		}
	}
	return nil
}

func (fxt *TestFixture) setupInfo(n int, k kind, fns ...CustomizeEntityFunc) error {
	if n <= 0 {
		return errs.Errorf("the number of objects to create must always be greater than zero: %d", n)
	}
	if _, ok := fxt.info[k]; !ok {
		fxt.info[k] = &createInfo{}
	}
	maxN := n
	if maxN < fxt.info[k].numInstances {
		maxN = fxt.info[k].numInstances
	}
	fxt.info[k].numInstances = maxN
	fxt.info[k].customizeEntityFuncs = append(fxt.info[k].customizeEntityFuncs, fns...)
	return nil
}

func newFixture(db *gorm.DB, isolatedCreation bool, recipeFuncs ...RecipeFunction) (*TestFixture, error) {
	fxt := TestFixture{
		checkFuncs:       []func() error{},
		info:             map[kind]*createInfo{},
		db:               db,
		isolatedCreation: isolatedCreation,
		ctx:              context.Background(),
	}
	for _, fn := range recipeFuncs {
		if err := fn(&fxt); err != nil {
			return nil, errs.Wrap(err, "failed to execute recipe function")
		}
	}
	makeFuncs := []func(fxt *TestFixture) error{
		// make the objects that DON'T have any dependency
		makeIdentities,
		makeTrackers,
		makeWorkItemLinkCategories,
		// actually make the objects that DO have dependencies
		makeSpaces,
		makeLabels,
		makeWorkItemLinkTypes,
		makeCodebases,
		makeWorkItemTypes,
		makeIterations,
		makeAreas,
		makeWorkItems,
		makeComments,
		makeWorkItemLinks,
	}
	for _, fn := range makeFuncs {
		if err := fn(&fxt); err != nil {
			return nil, errs.Wrap(err, "failed to make objects")
		}
	}
	if err := fxt.Check(); err != nil {
		return nil, errs.Wrap(err, "test fixture did not pass checks")
	}
	return &fxt, nil
}
