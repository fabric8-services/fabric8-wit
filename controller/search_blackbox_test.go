package controller_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	config "github.com/almighty/almighty-core/configuration"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/search"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"

	"github.com/goadesign/goa"
	"github.com/goadesign/goa/goatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

func TestRunSearchTests(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &searchBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type searchBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	db                             *gormapplication.GormDB
	svc                            *goa.Service
	clean                          func()
	testIdentity                   account.Identity
	wiRepo                         *workitem.GormWorkItemRepository
	controller                     *SearchController
	spaceBlackBoxTestConfiguration *config.ConfigurationData
	ctx                            context.Context
}

func (s *searchBlackBoxTest) SetupSuite() {
	var err error
	s.DB, err = gorm.Open("postgres", wibConfiguration.GetPostgresConfigString())
	if err != nil {
		panic("Failed to connect database: " + err.Error())
	}
	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "test user", "test provider")
	require.Nil(s.T(), err)
	s.testIdentity = testIdentity
	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if wibConfiguration.GetPopulateCommonTypes() {
		if err := models.Transactional(s.DB, func(tx *gorm.DB) error {
			s.ctx = migration.NewMigrationContext(context.Background())
			return migration.PopulateCommonTypes(s.ctx, tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.wiRepo = workitem.NewWorkItemRepository(s.DB)
	spaceBlackBoxTestConfiguration, err := config.GetConfigurationData()
	require.Nil(s.T(), err)
	s.spaceBlackBoxTestConfiguration = spaceBlackBoxTestConfiguration
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	s.svc = testsupport.ServiceAsUser("WorkItemComment-Service", almtoken.NewManagerWithPrivateKey(priv), testIdentity)
	s.controller = NewSearchController(s.svc, gormapplication.NewGormDB(DB), spaceBlackBoxTestConfiguration)
}

func (s *searchBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *searchBlackBoxTest) TestSearchWorkItems() {
	// given
	_, err := s.wiRepo.Create(
		s.ctx,
		space.SystemSpace,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch",
			workitem.SystemDescription: nil,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	q := "specialwordforsearch"
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, q)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), "specialwordforsearch", r.Attributes[workitem.SystemTitle])
}

func (s *searchBlackBoxTest) TestSearchPagination() {
	// given
	_, err := s.wiRepo.Create(
		s.ctx,
		space.SystemSpace,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch2",
			workitem.SystemDescription: nil,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	q := "specialwordforsearch2"
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, q)
	// then
	// defaults in paging.go is 'pageSizeDefault = 20'
	assert.Equal(s.T(), "http:///api/search?page[offset]=0&page[limit]=20&q=specialwordforsearch2", *sr.Links.First)
	assert.Equal(s.T(), "http:///api/search?page[offset]=0&page[limit]=20&q=specialwordforsearch2", *sr.Links.Last)
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), "specialwordforsearch2", r.Attributes[workitem.SystemTitle])
}

func (s *searchBlackBoxTest) TestSearchWithEmptyValue() {
	_, err := s.wiRepo.Create(
		s.ctx,
		space.SystemSpace,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch",
			workitem.SystemDescription: nil,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	q := ""
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, q)
	// then
	require.NotNil(s.T(), sr.Data)
	assert.Empty(s.T(), sr.Data)
}

func (s *searchBlackBoxTest) TestSearchWithDomainPortCombination() {
	description := "http://localhost:8080/detail/154687364529310 is related issue"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	_, err := s.wiRepo.Create(
		s.ctx,
		space.SystemSpace,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_new",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum", workitem.SystemState: workitem.SystemStateClosed,
		},
		s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	q := `"http://localhost:8080/detail/154687364529310"`
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, q)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), description, r.Attributes[workitem.SystemDescription])
}

func (s *searchBlackBoxTest) TestSearchURLWithoutPort() {
	description := "This issue is related to http://localhost/detail/876394"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	_, err := s.wiRepo.Create(
		s.ctx,
		space.SystemSpace,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_without_port",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	q := `"http://localhost/detail/876394"`
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, q)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), description, r.Attributes[workitem.SystemDescription])
}

func (s *searchBlackBoxTest) TestUnregisteredURLWithPort() {
	description := "Related to http://some-other-domain:8080/different-path/154687364529310/ok issue"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	_, err := s.wiRepo.Create(
		s.ctx,
		space.SystemSpace,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_new",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	q := `http://some-other-domain:8080/different-path/`
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, q)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), description, r.Attributes[workitem.SystemDescription])
}

func (s *searchBlackBoxTest) TestUnwantedCharactersRelatedToSearchLogic() {
	expectedDescription := rendering.NewMarkupContentFromLegacy("Related to http://example-domain:8080/different-path/ok issue")

	_, err := s.wiRepo.Create(
		s.ctx,
		space.SystemSpace,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_new",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	// add url: in the query, that is not expected by the code hence need to make sure it gives expected result.
	q := `http://url:some-random-other-domain:8080/different-path/`
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, q)
	// then
	require.NotNil(s.T(), sr.Data)
	assert.Empty(s.T(), sr.Data)
}

func (s *searchBlackBoxTest) getWICreatePayload() *app.CreateWorkitemPayload {
	spaceSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(space.SystemSpace.String()))
	c := app.CreateWorkitemPayload{
		Data: &app.WorkItem2{
			Type:       APIStringTypeWorkItem,
			Attributes: map[string]interface{}{},
			Relationships: &app.WorkItemRelationships{
				BaseType: &app.RelationBaseType{
					Data: &app.BaseTypeData{
						Type: APIStringTypeWorkItemType,
						ID:   workitem.SystemUserStory,
					},
				},
				Space: space.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
			},
		},
	}
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	return &c
}

func getServiceAsUser(testIdentity account.Identity) *goa.Service {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	service := testsupport.ServiceAsUser("TestSearch-Service", almtoken.NewManagerWithPrivateKey(priv), testIdentity)
	return service
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
func (s *searchBlackBoxTest) verifySearchByKnownURLs(wi *app.WorkItem2Single, host, searchQuery string) {
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
	// service := getServiceAsUser(s.testIdentity)
	wiCtrl := NewWorkitemController(s.svc, gormapplication.NewGormDB(s.DB))
	// create a WI, search by `list view URL` of newly created item
	newWI := s.getWICreatePayload()
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, wiCtrl, newWI)
	require.NotNil(s.T(), wi)
	customHost := "own.domain.one"
	queryString := fmt.Sprintf("http://%s/work-item/list/detail/%s", customHost, *wi.Data.ID)
	s.verifySearchByKnownURLs(wi, customHost, queryString)

	// Search by `board view URL` of newly created item
	customHost2 := "own.domain.two"
	queryString2 := fmt.Sprintf("http://%s/work-item/board/detail/%s", customHost2, *wi.Data.ID)
	s.verifySearchByKnownURLs(wi, customHost2, queryString2)
}
