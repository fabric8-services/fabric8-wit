package controller_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	config "github.com/fabric8-services/fabric8-wit/configuration"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/search"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/goadesign/goa"
	"github.com/goadesign/goa/goatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestRunSearchTests(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &searchBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type searchBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	db                             *gormapplication.GormDB
	svc                            *goa.Service
	testIdentity                   account.Identity
	wiRepo                         *workitem.GormWorkItemRepository
	controller                     *SearchController
	spaceBlackBoxTestConfiguration *config.ConfigurationData
	testDir                        string
}

func (s *searchBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.testDir = filepath.Join("test-files", "search")
	s.db = gormapplication.NewGormDB(s.DB)
	var err error
	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "SearchBlackBoxTest user", "test provider")
	require.Nil(s.T(), err)
	s.testIdentity = *testIdentity

	s.wiRepo = workitem.NewWorkItemRepository(s.DB)
	spaceBlackBoxTestConfiguration, err := config.GetConfigurationData()
	require.Nil(s.T(), err)
	s.spaceBlackBoxTestConfiguration = spaceBlackBoxTestConfiguration
	s.svc = testsupport.ServiceAsUser("WorkItemComment-Service", s.testIdentity)
	s.controller = NewSearchController(s.svc, gormapplication.NewGormDB(s.DB), spaceBlackBoxTestConfiguration)
}

func (s *searchBlackBoxTest) TestSearchWorkItems() {
	// given
	q := "specialwordforsearch"
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
		wi := fxt.WorkItems[idx]
		wi.Fields[workitem.SystemTitle] = q
		wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
		return nil
	}))
	// when
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), q, r.Attributes[workitem.SystemTitle])
}

func (s *searchBlackBoxTest) TestSearchPagination() {
	// given
	q := "specialwordforsearch2"
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
		wi := fxt.WorkItems[idx]
		wi.Fields[workitem.SystemTitle] = q
		wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
		return nil
	}))
	// when
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &spaceIDStr)
	// then
	// defaults in paging.go is 'pageSizeDefault = 20'
	assert.Equal(s.T(), "http:///api/search?page[offset]=0&page[limit]=20&q=specialwordforsearch2", *sr.Links.First)
	assert.Equal(s.T(), "http:///api/search?page[offset]=0&page[limit]=20&q=specialwordforsearch2", *sr.Links.Last)
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), q, r.Attributes[workitem.SystemTitle])
}

func (s *searchBlackBoxTest) TestSearchWithEmptyValue() {
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
		wi := fxt.WorkItems[idx]
		wi.Fields[workitem.SystemTitle] = "specialwordforsearch"
		wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
		return nil
	}))
	// when
	q := ""
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, jerrs := test.ShowSearchBadRequest(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotNil(s.T(), jerrs)
	require.Len(s.T(), jerrs.Errors, 1)
	require.NotNil(s.T(), jerrs.Errors[0].ID)
}

func (s *searchBlackBoxTest) TestSearchWithDomainPortCombination() {
	description := "http://localhost:8080/detail/154687364529310 is related issue"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
		wi := fxt.WorkItems[idx]
		wi.Fields[workitem.SystemTitle] = "specialwordforsearch_new"
		wi.Fields[workitem.SystemDescription] = expectedDescription
		wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
		return nil
	}))
	// when
	q := `"http://localhost:8080/detail/154687364529310"`
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), description, r.Attributes[workitem.SystemDescription])
}

func (s *searchBlackBoxTest) TestSearchURLWithoutPort() {
	description := "This issue is related to http://localhost/detail/876394"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
		wi := fxt.WorkItems[idx]
		wi.Fields[workitem.SystemTitle] = "specialwordforsearch_without_port"
		wi.Fields[workitem.SystemDescription] = expectedDescription
		wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
		return nil
	}))
	// when
	q := `"http://localhost/detail/876394"`
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), description, r.Attributes[workitem.SystemDescription])
}

func (s *searchBlackBoxTest) TestUnregisteredURLWithPort() {
	description := "Related to http://some-other-domain:8080/different-path/154687364529310/ok issue"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
		wi := fxt.WorkItems[idx]
		wi.Fields[workitem.SystemTitle] = "specialwordforsearch_new"
		wi.Fields[workitem.SystemDescription] = expectedDescription
		wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
		return nil
	}))
	// when
	q := `http://some-other-domain:8080/different-path/`
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), description, r.Attributes[workitem.SystemDescription])
}

func (s *searchBlackBoxTest) TestUnwantedCharactersRelatedToSearchLogic() {
	expectedDescription := rendering.NewMarkupContentFromLegacy("Related to http://example-domain:8080/different-path/ok issue")
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
		wi := fxt.WorkItems[idx]
		wi.Fields[workitem.SystemTitle] = "specialwordforsearch_new"
		wi.Fields[workitem.SystemDescription] = expectedDescription
		wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
		return nil
	}))
	// when
	// add url: in the query, that is not expected by the code hence need to make sure it gives expected result.
	q := `http://url:some-random-other-domain:8080/different-path/`
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotNil(s.T(), sr.Data)
	assert.Empty(s.T(), sr.Data)
}

func (s *searchBlackBoxTest) getWICreatePayload() *app.CreateWorkitemsPayload {
	spaceID := space.SystemSpace
	spaceRelatedURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(spaceID.String()))
	witRelatedURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.WorkitemtypeHref(spaceID.String(), workitem.SystemTask.String()))
	c := app.CreateWorkitemsPayload{
		Data: &app.WorkItem{
			Type:       APIStringTypeWorkItem,
			Attributes: map[string]interface{}{},
			Relationships: &app.WorkItemRelationships{
				BaseType: &app.RelationBaseType{
					Data: &app.BaseTypeData{
						Type: APIStringTypeWorkItemType,
						ID:   workitem.SystemTask,
					},
					Links: &app.GenericLinks{
						Self:    &witRelatedURL,
						Related: &witRelatedURL,
					},
				},
				Space: app.NewSpaceRelation(spaceID, spaceRelatedURL),
			},
		},
	}
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	return &c
}

func getServiceAsUser(testIdentity account.Identity) *goa.Service {
	return testsupport.ServiceAsUser("TestSearch-Service", testIdentity)
}

// searchByURL copies much of the codebase from search_testing.go->ShowSearchOK
// and customises the values to add custom Host in the call.
func (s *searchBlackBoxTest) searchByURL(customHost, queryString string) *app.SearchWorkItemList {
	var resp interface{}
	var respSetter goatest.ResponseSetterFunc = func(r interface{}) { resp = r }
	newEncoder := func(io.Writer) goa.Encoder { return respSetter }
	s.svc.Encoder = goa.NewHTTPEncoder()
	s.svc.Encoder.Register(newEncoder, "*/*")
	rw := httptest.NewRecorder()
	query := url.Values{}
	u := &url.URL{
		Path:     fmt.Sprintf("/api/search"),
		RawQuery: query.Encode(),
		Host:     customHost,
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	require.Nil(s.T(), err)
	prms := url.Values{}
	prms["q"] = []string{queryString} // any value will do
	goaCtx := goa.NewContext(goa.WithAction(s.svc.Context, "SearchTest"), rw, req, prms)
	showCtx, err := app.NewShowSearchContext(goaCtx, req, s.svc)
	require.Nil(s.T(), err)
	// Perform action
	err = s.controller.Show(showCtx)
	// Validate response
	require.Nil(s.T(), err)
	require.Equal(s.T(), 200, rw.Code)
	mt, ok := resp.(*app.SearchWorkItemList)
	require.True(s.T(), ok)
	return mt
}

// verifySearchByKnownURLs performs actual tests on search result and knwonURL map
func (s *searchBlackBoxTest) verifySearchByKnownURLs(wi *app.WorkItemSingle, host, searchQuery string) {
	result := s.searchByURL(host, searchQuery)
	assert.NotEmpty(s.T(), result.Data)
	assert.Equal(s.T(), *wi.Data.ID, *result.Data[0].ID)

	known := search.GetAllRegisteredURLs()
	require.NotNil(s.T(), known)
	assert.NotEmpty(s.T(), known)
	assert.Contains(s.T(), known[search.HostRegistrationKeyForListWI].URLRegex, host)
	assert.Contains(s.T(), known[search.HostRegistrationKeyForBoardWI].URLRegex, host)
}

// TestAutoRegisterHostURL checks if client's host is neatly registered as a KnwonURL or not
// Uses helper functions verifySearchByKnownURLs, searchByURL, getWICreatePayload
func (s *searchBlackBoxTest) TestAutoRegisterHostURL() {
	wiCtrl := NewWorkitemsController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	// create a WI, search by `list view URL` of newly created item
	//fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))
	newWI := s.getWICreatePayload()
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, wiCtrl, space.SystemSpace, newWI)
	require.NotNil(s.T(), wi)
	customHost := "own.domain.one"
	queryString := fmt.Sprintf("http://%s/work-item/list/detail/%d", customHost, wi.Data.Attributes[workitem.SystemNumber])
	s.verifySearchByKnownURLs(wi, customHost, queryString)

	// Search by `board view URL` of newly created item
	customHost2 := "own.domain.two"
	queryString2 := fmt.Sprintf("http://%s/work-item/board/detail/%d", customHost2, wi.Data.Attributes[workitem.SystemNumber])
	s.verifySearchByKnownURLs(wi, customHost2, queryString2)
}

func (s *searchBlackBoxTest) TestSearchWorkItemsSpaceContext() {
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.Identities(1, tf.SetIdentityUsernames([]string{"pranav"})),
		tf.Spaces(2),
		tf.WorkItems(3+5, func(fxt *tf.TestFixture, idx int) error {
			wi := fxt.WorkItems[idx]
			wi.Fields[workitem.SystemCreator] = fxt.IdentityByUsername("pranav").ID.String()
			wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
			if idx < 3 {
				wi.SpaceID = fxt.Spaces[0].ID
				wi.Fields[workitem.SystemTitle] = testsupport.CreateRandomValidTestName("shutter_island common_word random - ")
			} else {
				wi.SpaceID = fxt.Spaces[1].ID
				wi.Fields[workitem.SystemTitle] = testsupport.CreateRandomValidTestName("inception common_word random - ")
			}
			return nil
		}),
	)

	// when
	q := "common_word"
	space1IDStr := fxt.Spaces[0].ID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &space1IDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	assert.Len(s.T(), sr.Data, 3)
	for _, item := range sr.Data {
		// make sure that retrived items are from space 1 only
		assert.Contains(s.T(), item.Attributes[workitem.SystemTitle], "shutter_island common_word")
	}
	space2IDStr := fxt.Spaces[1].ID.String()
	_, sr = test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, &space2IDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	assert.Len(s.T(), sr.Data, 5)
	for _, item := range sr.Data {
		// make sure that retrived items are from space 2 only
		assert.Contains(s.T(), item.Attributes[workitem.SystemTitle], "inception common_word")
	}

	// when searched without spaceID then it should get all related WI
	_, sr = test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, nil)
	// then
	require.NotEmpty(s.T(), sr.Data)
	assert.Len(s.T(), sr.Data, 8)
}

func (s *searchBlackBoxTest) TestSearchWorkItemsWithoutSpaceContext() {
	// given 2 spaces with 10 workitems in the first and 5 in the second space
	_ = tf.NewTestFixture(s.T(), s.DB,
		tf.Identities(1, tf.SetIdentityUsernames([]string{"pranav"})),
		tf.Spaces(2),
		tf.WorkItems(10+5, func(fxt *tf.TestFixture, idx int) error {
			wi := fxt.WorkItems[idx]
			wi.Fields[workitem.SystemCreator] = fxt.IdentityByUsername("pranav").ID.String()
			wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
			if idx < 10 {
				wi.SpaceID = fxt.Spaces[0].ID
				wi.Fields[workitem.SystemTitle] = testsupport.CreateRandomValidTestName("search_by_me common_word random - ")
			} else {
				wi.SpaceID = fxt.Spaces[1].ID
				wi.Fields[workitem.SystemTitle] = testsupport.CreateRandomValidTestName("search_by_me common_word random - ")
			}
			return nil
		}),
	)

	q := "search_by_me"
	// search without space context
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, nil, &q, nil)
	require.NotEmpty(s.T(), sr.Data)
	assert.Len(s.T(), sr.Data, 15)
}

func (s *searchBlackBoxTest) TestSearchFilter() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "specialwordforsearch"
			return nil
		}),
	)
	// when
	filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.WorkItems[0].SpaceID)
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), "specialwordforsearch", r.Attributes[workitem.SystemTitle])
}

// It creates 1 space
// creates and adds 2 collaborators in the space
// creates 2 iterations within it
// 8 work items with different states & iterations & assignees & types
// and tests multiple combinations of space, state, iteration, assignee, type
func (s *searchBlackBoxTest) TestSearchQueryScenarioDriven() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.Identities(3, tf.SetIdentityUsernames([]string{"spaceowner", "alice", "bob"})),
		tf.Iterations(2, tf.SetIterationNames([]string{"sprint1", "sprint2"})),
		tf.Labels(4, tf.SetLabelNames([]string{"important", "backend", "ui", "rest"})),
		tf.WorkItemTypes(2, tf.SetWorkItemTypeNames([]string{"bug", "feature"})),
		tf.WorkItems(3+5+1, func(fxt *tf.TestFixture, idx int) error {
			wi := fxt.WorkItems[idx]
			if idx < 3 {
				//wi.Fields[workitem.SystemTitle] = fmt.Sprintf("New issue #%d", idx)
				wi.Fields[workitem.SystemState] = workitem.SystemStateResolved
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("sprint1").ID.String()
				wi.Fields[workitem.SystemLabels] = []string{fxt.LabelByName("important").ID.String(), fxt.LabelByName("backend").ID.String()}
				wi.Fields[workitem.SystemAssignees] = []string{fxt.IdentityByUsername("alice").ID.String()}
				wi.Fields[workitem.SystemCreator] = fxt.IdentityByUsername("spaceowner").ID.String()
				wi.Type = fxt.WorkItemTypeByName("bug").ID
			} else if idx < 3+5 {
				//wi.Fields[workitem.SystemTitle] = fmt.Sprintf("Closed issue #%d", idx)
				wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("sprint2").ID.String()
				wi.Fields[workitem.SystemLabels] = []string{fxt.LabelByName("ui").ID.String()}
				wi.Fields[workitem.SystemAssignees] = []string{fxt.IdentityByUsername("bob").ID.String()}
				wi.Fields[workitem.SystemCreator] = fxt.IdentityByUsername("spaceowner").ID.String()
				wi.Type = fxt.WorkItemTypeByName("feature").ID
			} else {
				// wi.Fields[workitem.SystemTitle] = fmt.Sprintf("Unassigned issue")
				wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("sprint2").ID.String()
				wi.Fields[workitem.SystemCreator] = fxt.IdentityByUsername("spaceowner").ID.String()
				wi.Type = fxt.WorkItemTypeByName("feature").ID
			}
			return nil
		}),
	)
	spaceIDStr := fxt.WorkItems[0].SpaceID.String()

	s.T().Run("label IN IMPORTAND, UI", func(t *testing.T) {
		// following test does not include any "space" deliberately, hence if there
		// is any work item in the test-DB having state=resolved following count
		// will fail
		filter := fmt.Sprintf(`
				{"label": {"$IN": ["%s", "%s"]}}`,
			fxt.LabelByName("important").ID, fxt.LabelByName("ui").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotNil(t, result)
		fmt.Println(result.Data)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 8) // 3 important + 5 UI
	})

	s.T().Run("space=ID AND (label=Backend OR iteration=sprint2)", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$OR": [
						{"label": "%s"},
						{"iteration": "%s"}
					]}
				]}`,
			spaceIDStr, fxt.LabelByName("backend").ID, fxt.IterationByName("sprint2").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1) // 3 items with Backend label & 5+1 items with sprint2
	})

	s.T().Run("space=ID AND label=UI", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"label": "%s"}
				]}`,
			spaceIDStr, fxt.LabelByName("ui").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 5) // 5 items having UI label
	})

	s.T().Run("label=UI OR label=Backend", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$OR": [
					{"label":"%s"},
					{"label": "%s"}
				]}`,
			fxt.LabelByName("ui").ID, fxt.LabelByName("backend").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 8)
	})

	s.T().Run("space=ID AND label=REST : expect 0 itmes", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"label": "%s"}
				]}`,
			spaceIDStr, fxt.LabelByName("rest").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		assert.Len(t, result.Data, 0) // no items having REST label
	})

	s.T().Run("space=ID AND label != Backend", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"label": "%s", "negate": true}
				]}`,
			spaceIDStr, fxt.LabelByName("backend").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 5+1) // 6 items are not having Backend label
	})

	s.T().Run("state=resolved AND iteration=sprint1", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"state": "%s"},
					{"iteration": "%s"}
				]}`,
			workitem.SystemStateResolved, fxt.IterationByName("sprint1").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		require.Len(t, result.Data, 3) // resolved items having sprint1 are 3
	})

	s.T().Run("state=resolved AND iteration=sprint1 using EQ", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"state": {"$EQ": "%s"}},
					{"iteration": {"$EQ": "%s"}}
				]}`,
			workitem.SystemStateResolved, fxt.IterationByName("sprint1").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		require.Len(t, result.Data, 3) // resolved items having sprint1 are 3
	})

	s.T().Run("state=resolved AND iteration=sprint2", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"state": "%s"},
					{"iteration": "%s"}
				]}`,
			workitem.SystemStateResolved, fxt.IterationByName("sprint2").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.Len(t, result.Data, 0) // No items having state=resolved && sprint2
	})

	s.T().Run("state=resolved OR iteration=sprint2", func(t *testing.T) {
		// following test does not include any "space" deliberately, hence if there
		// is any work item in the test-DB having state=resolved following count
		// will fail
		filter := fmt.Sprintf(`
				{"$OR": [
					{"state": "%s"},
					{"iteration": "%s"}
				]}`,
			workitem.SystemStateResolved, fxt.IterationByName("sprint2").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1) // resolved items + items in sprint2
	})

	s.T().Run("state IN resolved, closed", func(t *testing.T) {
		// following test does not include any "space" deliberately, hence if there
		// is any work item in the test-DB having state=resolved following count
		// will fail
		filter := fmt.Sprintf(`
				{"state": {"$IN": ["%s", "%s"]}}`,
			workitem.SystemStateResolved, workitem.SystemStateClosed)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1) // state = resolved or state = closed
	})

	s.T().Run("space=ID AND (state=resolved OR iteration=sprint2)", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$OR": [
						{"state": "%s"},
						{"iteration": "%s"}
					]}
				]}`,
			spaceIDStr, workitem.SystemStateResolved, fxt.IterationByName("sprint2").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1)
	})

	s.T().Run("space=ID AND (state=resolved OR iteration=sprint2) using EQ", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space": {"$EQ": "%s"}},
					{"$OR": [
						{"state": {"$EQ": "%s"}},
						{"iteration": {"$EQ": "%s"}}
					]}
				]}`,
			spaceIDStr, workitem.SystemStateResolved, fxt.IterationByName("sprint2").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1)
	})

	s.T().Run("space=ID AND (state!=resolved AND iteration=sprint1)", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$AND": [
						{"state": "%s", "negate": true},
						{"iteration": "%s"}
					]}
				]}`,
			spaceIDStr, workitem.SystemStateResolved, fxt.IterationByName("sprint1").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		assert.Len(t, result.Data, 0)
	})

	s.T().Run("space=ID AND (state!=open AND iteration!=fake-iterationID)", func(t *testing.T) {
		fakeIterationID := uuid.NewV4()
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space": {"$EQ": "%s"}},
					{"$AND": [
						{"state": "%s", "negate": true},
						{"iteration": "%s", "negate": true}
					]}
				]}`,
			spaceIDStr, workitem.SystemStateOpen, fakeIterationID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 9) // all items are other than open state & in other thatn fake itr
	})

	s.T().Run("space!=ID AND (state!=open AND iteration!=fake-iterationID)", func(t *testing.T) {
		fakeIterationID := uuid.NewV4()
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space": {"$NE": "%s"}},
					{"$AND": [
						{"state": "%s", "negate": true},
						{"iteration": "%s", "negate": true}
					]}
				]}`,
			spaceIDStr, workitem.SystemStateOpen, fakeIterationID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		assert.Empty(t, result.Data) // all items are other than open state & in other thatn fake itr
	})

	s.T().Run("space=ID AND (state!=open AND iteration!=fake-iterationID) using NE", func(t *testing.T) {
		fakeIterationID := uuid.NewV4()
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$AND": [
						{"state": {"$NE": "%s"}},
						{"iteration": {"$NE": "%s"}}
					]}
				]}`,
			spaceIDStr, workitem.SystemStateOpen, fakeIterationID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 9) // all items are other than open state & in other thatn fake itr
	})

	s.T().Run("space=FakeID AND state=closed", func(t *testing.T) {
		fakeSpaceID1 := uuid.NewV4().String()
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"state": "%s"}
				]}`,
			fakeSpaceID1, workitem.SystemStateOpen)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &fakeSpaceID1)
		assert.Len(t, result.Data, 0) // we have 5 closed items but they are in different space
	})

	s.T().Run("space=spaceID AND state=closed AND assignee=bob", func(t *testing.T) {
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"assignee":"%s"},
					{"state": "%s"}
				]}`,
			spaceIDStr, fxt.IdentityByUsername("bob").ID, workitem.SystemStateClosed)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 5) // we have 5 closed items assigned to bob
	})

	s.T().Run("space=spaceID AND iteration=sprint1 AND assignee=alice", func(t *testing.T) {
		// Let's see what alice did in sprint1
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"assignee":"%s"},
					{"iteration": "%s"}
				]}`,
			spaceIDStr, fxt.IdentityByUsername("alice").ID, fxt.IterationByName("sprint1").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3) // alice worked on 3 issues in sprint1
	})

	s.T().Run("space=spaceID AND state!=closed AND iteration=sprint1 AND assignee=alice", func(t *testing.T) {
		// Let's see non-closed issues alice working on from sprint1
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"assignee":"%s"},
					{"state":"%s", "negate": true},
					{"iteration": "%s"}
				]}`,
			spaceIDStr, fxt.IdentityByUsername("alice").ID, workitem.SystemStateClosed, fxt.IterationByName("sprint1").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3)
	})

	s.T().Run("space=spaceID AND (state=closed or state=resolved)", func(t *testing.T) {
		// get me all closed and resolved work items from my space
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$OR": [
						{"state":"%s"},
						{"state":"%s"}
					]}
				]}`,
			spaceIDStr, workitem.SystemStateClosed, workitem.SystemStateResolved)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1) //resolved + closed
	})

	s.T().Run("space=spaceID AND (type=bug OR type=feature)", func(t *testing.T) {
		// get me all bugs or features in myspace
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$OR": [
						{"type":"%s"},
						{"type":"%s"}
					]}
				]}`,
			spaceIDStr, fxt.WorkItemTypeByName("bug").ID, fxt.WorkItemTypeByName("feature").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1) //bugs + features
	})

	s.T().Run("space=spaceID AND (workitemtype=bug OR workitemtype=feature)", func(t *testing.T) {
		// get me all bugs or features in myspace
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$OR": [
						{"workitemtype":"%s"},
						{"workitemtype":"%s"}
					]}
				]}`,
			spaceIDStr, fxt.WorkItemTypeByName("bug").ID, fxt.WorkItemTypeByName("feature").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3+5+1) //bugs + features
	})

	s.T().Run("space=spaceID AND (type=bug AND state=resolved AND (assignee=bob OR assignee=alice))", func(t *testing.T) {
		// get me all Resolved bugs assigned to bob or alice
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$AND": [
						{"$AND": [{"type":"%s"},{"state":"%s"}]},
						{"$OR": [{"assignee":"%s"},{"assignee":"%s"}]}
					]}
				]}`,
			spaceIDStr, fxt.WorkItemTypeByName("bug").ID, workitem.SystemStateResolved, fxt.IdentityByUsername("bob").ID, fxt.IdentityByUsername("alice").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3) //resolved bugs
	})

	s.T().Run("space=spaceID AND (workitemtype=bug AND state=resolved AND (assignee=bob OR assignee=alice))", func(t *testing.T) {
		// get me all Resolved bugs assigned to bob or alice
		filter := fmt.Sprintf(`
				{"$AND": [
					{"space":"%s"},
					{"$AND": [
						{"$AND": [{"workitemtype":"%s"},{"state":"%s"}]},
						{"$OR": [{"assignee":"%s"},{"assignee":"%s"}]}
					]}
				]}`,
			spaceIDStr, fxt.WorkItemTypeByName("bug").ID, workitem.SystemStateResolved, fxt.IdentityByUsername("bob").ID, fxt.IdentityByUsername("alice").ID)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 3) //resolved bugs
	})

	s.T().Run("bad expression missing curly brace", func(t *testing.T) {
		filter := fmt.Sprintf(`{"state": "0fe7b23e-c66e-43a9-ab1b-fbad9924fe7c"`)
		res, jerrs := test.ShowSearchBadRequest(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.NotNil(t, jerrs.Errors[0].ID)
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGolden(t, filepath.Join(s.testDir, "show", "bad_expression_missing_curly_brace.error.golden.json"), jerrs)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "bad_expression_missing_curly_brace.headers.golden.json"), res.Header())
	})

	s.T().Run("non existing key", func(t *testing.T) {
		filter := fmt.Sprintf(`{"nonexistingkey": "0fe7b23e-c66e-43a9-ab1b-fbad9924fe7c"}`)
		res, jerrs := test.ShowSearchBadRequest(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.NotNil(t, jerrs.Errors[0].ID)
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGolden(t, filepath.Join(s.testDir, "show", "non_existing_key.error.golden.json"), jerrs)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "non_existing_key.headers.golden.json"), res.Header())
	})

	s.T().Run("assignee=null before WI creation", func(t *testing.T) {
		filter := fmt.Sprintf(`
					{"$AND": [
						{"assignee":null}
					]}`,
		)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 1)
	})

	s.T().Run("assignee=null after WI creation (top-level)", func(t *testing.T) {
		filter := fmt.Sprintf(`
					{"assignee":null}`,
		)
		_, result := test.ShowSearchOK(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(t, result.Data)
		assert.Len(t, result.Data, 1)
	})

	s.T().Run("assignee=null with negate", func(t *testing.T) {
		filter := fmt.Sprintf(`{"$AND": [{"assignee":null, "negate": true}]}`)
		res, jerrs := test.ShowSearchBadRequest(t, nil, nil, s.controller, &filter, nil, nil, nil, nil, &spaceIDStr)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.NotNil(t, jerrs.Errors[0].ID)
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGolden(t, filepath.Join(s.testDir, "show", "assignee_null_negate.error.golden.json"), jerrs)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "assignee_null_negate.headers.golden.json"), res.Header())
	})
}
