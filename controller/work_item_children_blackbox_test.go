package controller_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestSuiteWorkItemChildren(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemChildSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

// The workItemChildSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type workItemChildSuite struct {
	gormtestsupport.DBTestSuite
	workitemLinkCtrl *WorkItemLinkController
	workItemCtrl     *WorkitemController
	workItemsCtrl    *WorkitemsController
	svc              *goa.Service
	typeCtrl         *WorkitemtypeController
	fxt              *tf.TestFixture
	testDir          string
}

const (
	hasChildren   bool = true
	hasNoChildren bool = false
)

// The SetupTest method will be run before every test in the suite.
// It will make sure that some resources that we rely on do exists.
func (s *workItemChildSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.testDir = filepath.Join("test-files", "work_item")

	s.fxt = tf.NewTestFixture(s.T(), s.DB,
		tf.Spaces(1),
		tf.WorkItems(3, tf.SetWorkItemTitles("bug1", "bug2", "bug3")),
	)

	svc := testsupport.ServiceAsUser("WorkItemLink-Service", *s.fxt.Identities[0])
	require.NotNil(s.T(), svc)
	s.workitemLinkCtrl = NewWorkItemLinkController(svc, s.GormDB, s.Configuration)
	require.NotNil(s.T(), s.workitemLinkCtrl)

	svc = testsupport.ServiceAsUser("TestWorkItem-Service", *s.fxt.Identities[0])
	require.NotNil(s.T(), svc)
	s.svc = svc
	s.workItemCtrl = NewWorkitemController(svc, s.GormDB, s.Configuration)
	require.NotNil(s.T(), s.workItemCtrl)

	svc = testsupport.ServiceAsUser("TestWorkItems-Service", *s.fxt.Identities[0])
	require.NotNil(s.T(), svc)
	s.svc = svc
	s.workItemsCtrl = NewWorkitemsController(svc, s.GormDB, s.Configuration)
	require.NotNil(s.T(), s.workItemCtrl)

	// Create a test user identity
	s.svc = testsupport.ServiceAsUser("TestWorkItem-Service", *s.fxt.Identities[0])
	require.NotNil(s.T(), s.svc)
}

func (s *workItemChildSuite) linkWorkItems(t *testing.T, sourceTitle, targetTitle string) link.WorkItemLink {
	src := s.fxt.WorkItemByTitle(sourceTitle)
	require.NotNil(t, src)
	tgt := s.fxt.WorkItemByTitle(targetTitle)
	require.NotNil(t, tgt)

	fxt := tf.NewTestFixture(t, s.DB,
		tf.WorkItemLinksCustom(1, func(fxt *tf.TestFixture, idx int) error {
			l := fxt.WorkItemLinks[idx]
			l.LinkTypeID = link.SystemWorkItemLinkTypeParentChildID
			l.SourceID = s.fxt.WorkItemByTitle(sourceTitle).ID
			l.TargetID = s.fxt.WorkItemByTitle(targetTitle).ID
			return nil
		}),
	)
	return *fxt.WorkItemLinks[0]
}

// checkChildrenRelationship runs a variety of checks on a given work item
// regarding the children relationships
func checkChildrenRelationship(t *testing.T, wi *app.WorkItem, expectedHasChildren ...bool) {
	t.Logf("Checking relationships for work item with id=%s", *wi.ID)
	require.NotNil(t, wi.Relationships.Children, "no 'children' relationship found in work item %s", *wi.ID)
	require.NotNil(t, wi.Relationships.Children.Links, "no 'links' found in 'children' relationship in work item %s", *wi.ID)
	require.NotNil(t, wi.Relationships.Children.Meta, "no 'meta' found in 'children' relationship in work item %s", *wi.ID)
	hasChildren, hasChildrenFound := wi.Relationships.Children.Meta["hasChildren"]
	require.True(t, hasChildrenFound, "no 'hasChildren' found in 'meta' object of 'children' relationship in work item %s", *wi.ID)
	if expectedHasChildren != nil && len(expectedHasChildren) > 0 {
		assert.Equal(t, expectedHasChildren[0], hasChildren, "work item %s is supposed to have children? %v", *wi.ID, expectedHasChildren[0])
	}
}

//-----------------------------------------------------------------------------
// Actual tests
//-----------------------------------------------------------------------------

func (s *workItemChildSuite) TestChildren() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB,
			tf.WorkItems(3, tf.SetWorkItemTitles("parent", "child1", "child2")),
			tf.WorkItemLinks(2, func(fxt *tf.TestFixture, idx int) error {
				l := fxt.WorkItemLinks[idx]
				l.LinkTypeID = link.SystemWorkItemLinkTypeParentChildID
				l.SourceID = fxt.WorkItems[0].ID
				l.TargetID = fxt.WorkItems[idx+1].ID
				return nil
			}),
		)
		t.Run("show", func(t *testing.T) {
			t.Run("has children", func(t *testing.T) {
				// when
				_, workItem := test.ShowWorkitemOK(t, s.svc.Context, s.svc, s.workItemCtrl, fxt.WorkItemByTitle("parent").ID, nil, nil)
				// then
				compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.has_children.res.payload.golden.json"), workItem)
				checkChildrenRelationship(t, workItem.Data, hasChildren)
			})
			t.Run("no children", func(t *testing.T) {
				// when
				_, workItem := test.ShowWorkitemOK(t, s.svc.Context, s.svc, s.workItemCtrl, fxt.WorkItemByTitle("child1").ID, nil, nil)
				// then
				compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.has_no_children.res.payload.golden.json"), workItem)
				checkChildrenRelationship(t, workItem.Data, hasNoChildren)
			})
		})
		t.Run("list", func(t *testing.T) {
			// when
			res, workItemList := test.ListChildrenWorkitemOK(t, s.svc.Context, s.svc, s.workItemCtrl, fxt.WorkItemByTitle("parent").ID, nil, nil, nil, nil)
			// then
			compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list_children", "ok.res.payload.golden.json"), workItemList)
			toBeFound := id.Slice{fxt.WorkItemByTitle("child1").ID, fxt.WorkItemByTitle("child2").ID}.ToMap()
			for _, wi := range workItemList.Data {
				_, ok := toBeFound[*wi.ID]
				require.True(t, ok, "found unexpected work item: %+v", *wi.ID)
				delete(toBeFound, *wi.ID)
			}
			require.Empty(t, toBeFound, "failed to find work items: %+v", toBeFound)
			assertResponseHeaders(t, res)
		})
	})
	s.T().Run("timing concerned tests", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB,
			tf.WorkItems(3, tf.SetWorkItemTitles("parent", "child1", "child2")),
			tf.WorkItemLinks(2, func(fxt *tf.TestFixture, idx int) error {
				l := fxt.WorkItemLinks[idx]
				l.LinkTypeID = link.SystemWorkItemLinkTypeParentChildID
				l.SourceID = fxt.WorkItems[0].ID
				l.TargetID = fxt.WorkItems[idx+1].ID
				return nil
			}),
		)

		t.Run("using expired if modified since header", func(t *testing.T) {
			// when
			updatedAt, ok := fxt.WorkItemByTitle("parent").Fields[workitem.SystemUpdatedAt].(time.Time)
			require.True(t, ok)
			ifModifiedSince := app.ToHTTPTime(updatedAt.Add(-1 * time.Hour))
			res, workItemList := test.ListChildrenWorkitemOK(t, s.svc.Context, s.svc, s.workItemCtrl, fxt.WorkItemByTitle("parent").ID, nil, nil, &ifModifiedSince, nil)
			// then
			toBeFound := id.Slice{fxt.WorkItemByTitle("child1").ID, fxt.WorkItemByTitle("child2").ID}.ToMap()
			for _, wi := range workItemList.Data {
				_, ok := toBeFound[*wi.ID]
				require.True(t, ok, "found unexpected work item: %+v", *wi.ID)
				delete(toBeFound, *wi.ID)
			}
			require.Empty(t, toBeFound, "failed to find work items: %+v", toBeFound)
			assertResponseHeaders(t, res)
		})
		t.Run("using expired if none match header", func(t *testing.T) {
			// when
			ifNoneMatch := "foo"
			res, workItemList := test.ListChildrenWorkitemOK(t, s.svc.Context, s.svc, s.workItemCtrl, fxt.WorkItemByTitle("parent").ID, nil, nil, nil, &ifNoneMatch)
			// then
			toBeFound := id.Slice{fxt.WorkItemByTitle("child1").ID, fxt.WorkItemByTitle("child2").ID}.ToMap()
			for _, wi := range workItemList.Data {
				_, ok := toBeFound[*wi.ID]
				require.True(t, ok, "found unexpected work item: %+v", *wi.ID)
				delete(toBeFound, *wi.ID)
			}
			require.Empty(t, toBeFound, "failed to find work items: %+v", toBeFound)
			assertResponseHeaders(t, res)
		})
		t.Run("not modified using if modified since header", func(t *testing.T) {
			// given
			res, _ := test.ListChildrenWorkitemOK(t, s.svc.Context, s.svc, s.workItemCtrl, fxt.WorkItemByTitle("parent").ID, nil, nil, nil, nil)
			ifModifiedSince := res.Header()[app.LastModified][0]
			// when
			res = test.ListChildrenWorkitemNotModified(t, s.svc.Context, s.svc, s.workItemCtrl, fxt.WorkItemByTitle("parent").ID, nil, nil, &ifModifiedSince, nil)
			// then
			assertResponseHeaders(t, res)
		})
		t.Run("not modified using if none match header", func(t *testing.T) {
			res, _ := test.ListChildrenWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, fxt.WorkItemByTitle("parent").ID, nil, nil, nil, nil)
			// when
			ifNoneMatch := res.Header()[app.ETag][0]
			res = test.ListChildrenWorkitemNotModified(t, s.svc.Context, s.svc, s.workItemCtrl, fxt.WorkItemByTitle("parent").ID, nil, nil, nil, &ifNoneMatch)
			// then
			assertResponseHeaders(t, res)
		})
	})
}

func (s *workItemChildSuite) TestWorkItemListFilterByNoParents() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.WorkItems(3, tf.SetWorkItemTitles("parent", "child1", "child2")),
		tf.WorkItemLinksCustom(2, func(fxt *tf.TestFixture, idx int) error {
			l := fxt.WorkItemLinks[idx]
			l.LinkTypeID = link.SystemWorkItemLinkTypeParentChildID
			l.SourceID = fxt.WorkItems[0].ID
			l.TargetID = fxt.WorkItems[idx+1].ID
			return nil
		}),
	)

	s.T().Run("without parentexists filter", func(t *testing.T) {
		// given
		var pe *bool
		// when
		_, result := test.ListWorkitemsOK(t, nil, nil, s.workItemsCtrl, fxt.Spaces[0].ID, nil, nil, nil, nil, nil, pe, nil, nil, nil, nil, nil, nil, nil)
		// then
		toBeFound := id.Slice{fxt.WorkItemByTitle("parent").ID, fxt.WorkItemByTitle("child1").ID, fxt.WorkItemByTitle("child2").ID}.ToMap()
		for _, wi := range result.Data {
			_, ok := toBeFound[*wi.ID]
			require.True(t, ok, "found unexpected work item: %s", wi.Attributes[workitem.SystemTitle].(string))
			delete(toBeFound, *wi.ID)
		}
		require.Empty(t, toBeFound, "failed to find work items: %+v", toBeFound)
	})

	s.T().Run("with parentexists value set to false", func(t *testing.T) {
		// given
		pe := false
		// when
		_, result := test.ListWorkitemsOK(t, nil, nil, s.workItemsCtrl, fxt.Spaces[0].ID, nil, nil, nil, nil, nil, &pe, nil, nil, nil, nil, nil, nil, nil)
		// then
		toBeFound := id.Slice{fxt.WorkItemByTitle("parent").ID}.ToMap()
		for _, wi := range result.Data {
			_, ok := toBeFound[*wi.ID]
			require.True(t, ok, "found unexpected work item: %+v", *wi.ID)
			delete(toBeFound, *wi.ID)
		}
		require.Empty(t, toBeFound, "failed to find work items: %+v", toBeFound)
		assert.Len(t, result.Data, 1)
	})

	s.T().Run("with parentexists value set to true", func(t *testing.T) {
		// given
		pe := true
		// when
		_, result := test.ListWorkitemsOK(t, nil, nil, s.workItemsCtrl, fxt.Spaces[0].ID, nil, nil, nil, nil, nil, &pe, nil, nil, nil, nil, nil, nil, nil)
		// then
		toBeFound := id.Slice{fxt.WorkItemByTitle("parent").ID, fxt.WorkItemByTitle("child1").ID, fxt.WorkItemByTitle("child2").ID}.ToMap()
		for _, wi := range result.Data {
			_, ok := toBeFound[*wi.ID]
			require.True(t, ok, "found unexpected work item: %+v", fxt.WorkItemByID(*wi.ID).Fields[workitem.SystemTitle].(string))
			delete(toBeFound, *wi.ID)
		}
		require.Empty(t, toBeFound, "failed to find work items: %+v", toBeFound)
		assert.Len(t, result.Data, 3)
	})

}

// ------------------------------------------------------------------------
// Testing that the 'show' and 'list' operations return an updated list of
// work items when one of them has been linked to another one, or a link
// was updated or (soft) delete
// ------------------------------------------------------------------------

func (s *workItemChildSuite) TestCreateLinkToChildrenThenShowOK() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	s.linkWorkItems(s.T(), "bug1", "bug2")
	// when
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenShowOKUsingExpiredIfModifiedSinceHeader() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	s.linkWorkItems(s.T(), "bug1", "bug2")
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt.Add(-1 * time.Hour))
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenShowOKUsingIfModifiedSinceHeader() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.T(), "bug1", "bug2")
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt)
	log.Warn(nil, map[string]interface{}{"wi_id": s.fxt.WorkItemByTitle("bug1").ID}, "Using ifModifiedSince=%v", ifModifiedSince)
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenShowOKUsingExpiredIfNoneMatchHeader() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	s.linkWorkItems(s.T(), "bug1", "bug2")
	// when
	ifNoneMatch := "foo"
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenShowOKUsingIfNoneMatchHeader() {
	// given
	res, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	s.linkWorkItems(s.T(), "bug1", "bug2")
	// when
	ifNoneMatch := res.Header()[app.ETag][0]
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenShowOK() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, workitemLink12.ID)
	// when
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenShowOKUsingExpiredIfModifiedSinceHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, workitemLink12.ID)
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt.Add(-1 * time.Hour))
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenShowOKUsingIfModifiedSinceHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	time.Sleep(1 * time.Second)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, workitemLink12.ID)
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt)
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenShowOKUsingExpiredIfNoneMatchHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, workitemLink12.ID)
	// when
	ifNoneMatch := "foo"
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenShowOKUsingIfNoneMatchHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.T(), "bug1", "bug2")
	res, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, workitemLink12.ID)
	// when
	ifNoneMatch := res.Header()[app.ETag][0]
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenShowOK() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	s.linkWorkItems(s.T(), "bug1", "bug3")
	// when
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenShowOKUsingExpiredIfModifiedSinceHeader() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	// add another link
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.T(), "bug1", "bug3")
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt.Add(-1 * time.Hour))
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenShowOKUsingIfModifiedSinceHeader() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	// add another link
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.T(), "bug1", "bug3")
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt)
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenShowOKUsingExpiredIfNoneMatchHeader() {
	// given
	// create a link
	s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	// add another link
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.T(), "bug1", "bug3")
	// when
	ifNoneMatch := "foo"
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenShowOKUsingIfNoneMatchHeader() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.T(), "bug1", "bug2")
	res, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	// add another link
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.T(), "bug1", "bug3")
	// when
	ifNoneMatch := res.Header()[app.ETag][0]
	res, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)

}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenListOK() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	s.linkWorkItems(s.T(), "bug1", "bug2")
	// when
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, s.fxt.WorkItemByTitle("bug1").ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenListOKUsingExpiredIfModifiedSinceHeader() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	s.linkWorkItems(s.T(), "bug1", "bug2")
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt.Add(-1 * time.Hour))
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, s.fxt.WorkItemByTitle("bug1").ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenOKThenListUsingIfModifiedSinceHeader() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.T(), "bug1", "bug2")
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt)
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, s.fxt.WorkItemByTitle("bug1").ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenListOKUsingExpiredIfNoneMatchHeader() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	s.linkWorkItems(s.T(), "bug1", "bug2")
	// when
	ifNoneMatch := "foo"
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, s.fxt.WorkItemByTitle("bug1").ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenListOKUsingIfNoneMatchHeader() {
	// given
	res, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.T(), "bug1", "bug2")
	// when
	ifNoneMatch := res.Header()[app.ETag][0]
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, s.fxt.WorkItemByTitle("bug1").ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkThenListToChildrenOK() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, workitemLink12.ID)
	// when
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, s.fxt.WorkItemByTitle("bug1").ID), hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenListOKUsingExpiredIfModifiedSinceHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, workitemLink12.ID)
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt.Add(-1 * time.Hour))
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, s.fxt.WorkItemByTitle("bug1").ID), hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenListOKUsingIfModifiedSinceHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	time.Sleep(1 * time.Second)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, workitemLink12.ID)
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt)
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, s.fxt.WorkItemByTitle("bug1").ID), hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenListOKUsingExpiredIfNoneMatchHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, workitemLink12.ID)
	// when
	ifNoneMatch := "foo"
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, s.fxt.WorkItemByTitle("bug1").ID), hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenListOKUsingIfNoneMatchHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.T(), "bug1", "bug2")
	res, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	time.Sleep(1 * time.Second)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, workitemLink12.ID)
	// when
	ifNoneMatch := res.Header()[app.ETag][0]
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, s.fxt.WorkItemByTitle("bug1").ID), hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenListOK() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	s.linkWorkItems(s.T(), "bug1", "bug3")
	// when
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, s.fxt.WorkItemByTitle("bug1").ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenListOKUsingExpiredIfModifiedSinceHeader() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	s.linkWorkItems(s.T(), "bug1", "bug3")
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt.Add(-1 * time.Hour))
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, s.fxt.WorkItemByTitle("bug1").ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenListOKUsingIfModifiedSinceHeader() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.T(), "bug1", "bug3")
	// when/then
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt)
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, s.fxt.WorkItemByTitle("bug1").ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenListOKUsingExpiredIfNoneMatchHeader() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.T(), "bug1", "bug2")
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	s.linkWorkItems(s.T(), "bug1", "bug3")
	// when
	ifNoneMatch := "foo"
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, s.fxt.WorkItemByTitle("bug1").ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenListOKUsingIfNoneMatchHeader() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.T(), "bug1", "bug2")
	res, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.T(), "bug1", "bug3")
	// when
	ifNoneMatch := res.Header()[app.ETag][0]
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, s.fxt.WorkItemByTitle("bug1").ID), hasChildren)
}

func lookupWorkitem(t *testing.T, wiList app.WorkItemList, wiID uuid.UUID) *app.WorkItem {
	for _, wiData := range wiList.Data {
		if *wiData.ID == wiID {
			return wiData
		}
	}
	t.Errorf("Failed to look-up work item with id='%s'", wiID)
	return nil
}

func lookupWorkitemFromSearchList(t *testing.T, wiList app.SearchWorkItemList, wiID uuid.UUID) *app.WorkItem {
	for _, wiData := range wiList.Data {
		if *wiData.ID == wiID {
			return wiData
		}
	}
	t.Fatalf("Failed to look-up work item with id='%s'", wiID)
	return nil
}

type searchParentExistsSuite struct {
	workItemChildSuite
	searchCtrl *SearchController
}

func (s *searchParentExistsSuite) SetupSuite() {
	s.workItemChildSuite.SetupSuite()
}

func (s *searchParentExistsSuite) SetupTest() {
	s.workItemChildSuite.SetupTest()

	s.svc = testsupport.ServiceAsUser("Search-Service", *s.fxt.Identities[0])
	s.searchCtrl = NewSearchController(s.svc, s.GormDB, s.Configuration)
}

func TestSearchParentExists(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &searchParentExistsSuite{workItemChildSuite: workItemChildSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()}})
}

func (s *searchParentExistsSuite) TestSearchWorkItemListFilterUsingParentExists() {
	s.linkWorkItems(s.T(), "bug1", "bug2")
	s.linkWorkItems(s.T(), "bug1", "bug3")

	s.T().Run("without parentexists filter", func(t *testing.T) {
		// given
		var pe *bool
		// when
		sid := space.SystemSpace.String()
		test.ShowSearchBadRequest(t, nil, nil, s.searchCtrl, nil, pe, nil, nil, nil, &sid)
	})
	s.T().Run("with parentexists value set to false", func(t *testing.T) {
		// given
		pe := false
		// when
		filter := fmt.Sprintf(`
			{"$AND": [
				{"space":"%[1]s"},
				{"type":"%[2]s"}
			]}`,
			s.fxt.Spaces[0].ID.String(),
			s.fxt.WorkItemByTitle("bug1").Type)

		_, result := test.ShowSearchOK(t, nil, nil, s.searchCtrl, &filter, &pe, nil, nil, nil, nil)
		// then
		assert.Len(t, result.Data, 1)
		checkChildrenRelationship(t, lookupWorkitemFromSearchList(t, *result, s.fxt.WorkItemByTitle("bug1").ID), hasChildren)
	})

	s.T().Run("with parentexists value set to true", func(t *testing.T) {
		// given
		pe := true
		// when
		sid := space.SystemSpace.String()
		filter := fmt.Sprintf(`
			{"$AND": [
				{"space":"%[1]s"},
				{"type":"%[2]s"}
			]}`,
			s.fxt.Spaces[0].ID.String(),
			s.fxt.WorkItemByTitle("bug1").Type)

		_, result := test.ShowSearchOK(t, nil, nil, s.searchCtrl, &filter, &pe, nil, nil, nil, &sid)
		// then
		assert.Len(t, result.Data, 3)
		checkChildrenRelationship(t, lookupWorkitemFromSearchList(t, *result, s.fxt.WorkItemByTitle("bug1").ID), hasChildren)
		checkChildrenRelationship(t, lookupWorkitemFromSearchList(t, *result, s.fxt.WorkItemByTitle("bug2").ID), hasNoChildren)
		checkChildrenRelationship(t, lookupWorkitemFromSearchList(t, *result, s.fxt.WorkItemByTitle("bug3").ID), hasNoChildren)
	})

}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenListChildren() {
	// create workitem links then remove it
	workitemLink1 := s.linkWorkItems(s.T(), "bug1", "bug2")
	workitemLink2 := s.linkWorkItems(s.T(), "bug1", "bug3")

	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)

	// check number of children
	_, childrenList := test.ListChildrenWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil, nil, nil)
	require.Equal(s.T(), 2, childrenList.Meta.TotalCount)

	// delete link
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, workitemLink1.ID)

	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)

	// check number of children
	_, childrenList = test.ListChildrenWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil, nil, nil)
	require.Equal(s.T(), 1, childrenList.Meta.TotalCount)

	// delete link
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, workitemLink2.ID)

	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil)
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)

	// check number of children
	_, childrenList = test.ListChildrenWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, s.fxt.WorkItemByTitle("bug1").ID, nil, nil, nil, nil)
	require.Equal(s.T(), 0, childrenList.Meta.TotalCount)
}
