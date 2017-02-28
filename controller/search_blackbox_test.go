package controller_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	config "github.com/almighty/almighty-core/configuration"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/search"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"

	"github.com/goadesign/goa"
	"github.com/goadesign/goa/goatest"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type searchBlackBoxTest struct {
	gormsupport.DBTestSuite
	clean                          func()
	spaceBlackBoxTestConfiguration *config.ConfigurationData
	wiRepo                         *workitem.GormWorkItemRepository
	service                        *goa.Service
}

func TestRunSearchRepoBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &searchBlackBoxTest{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *searchBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if _, c := os.LookupEnv(resource.Database); c != false {
		if err := models.Transactional(s.DB, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(context.Background(), tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}

	var err error
	s.spaceBlackBoxTestConfiguration, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	s.service = getServiceAsUser()
	s.wiRepo = workitem.NewWorkItemRepository(s.DB)
}

func (s *searchBlackBoxTest) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

func (s *searchBlackBoxTest) TearDownTest() {
	s.clean()
}

func getServiceAsUser() *goa.Service {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	service := testsupport.ServiceAsUser("TestSearch-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return service
}

func (s *searchBlackBoxTest) TestSearch() {
	t := s.T()

	_, err := s.wiRepo.Create(
		context.Background(),
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch",
			workitem.SystemDescription: nil,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		uuid.NewV4())
	require.Nil(t, err)
	controller := NewSearchController(s.service, gormapplication.NewGormDB(s.DB), s.spaceBlackBoxTestConfiguration)
	q := "specialwordforsearch"
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	require.NotEmpty(t, sr.Data)
	r := sr.Data[0]
	assert.Equal(t, "specialwordforsearch", r.Attributes[workitem.SystemTitle])
}

func (s *searchBlackBoxTest) TestSearchPagination() {
	t := s.T()

	_, err := s.wiRepo.Create(
		context.Background(),
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch2",
			workitem.SystemDescription: nil,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		uuid.NewV4())
	require.Nil(t, err)

	controller := NewSearchController(s.service, gormapplication.NewGormDB(s.DB), s.spaceBlackBoxTestConfiguration)
	q := "specialwordforsearch2"
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)

	// defaults in paging.go is 'pageSizeDefault = 20'
	assert.Equal(t, "http:///api/search?page[offset]=0&page[limit]=20&q=specialwordforsearch2", *sr.Links.First)
	assert.Equal(t, "http:///api/search?page[offset]=0&page[limit]=20&q=specialwordforsearch2", *sr.Links.Last)
	require.NotEmpty(t, sr.Data)
	r := sr.Data[0]
	assert.Equal(t, "specialwordforsearch2", r.Attributes[workitem.SystemTitle])
}

func (s *searchBlackBoxTest) TestSearchWithEmptyValue() {
	t := s.T()

	_, err := s.wiRepo.Create(
		context.Background(),
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch",
			workitem.SystemDescription: nil,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		uuid.NewV4())
	require.Nil(t, err)

	controller := NewSearchController(s.service, gormapplication.NewGormDB(s.DB), s.spaceBlackBoxTestConfiguration)
	q := ""
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	require.NotNil(t, sr.Data)
	assert.Empty(t, sr.Data)
}

func (s *searchBlackBoxTest) TestSearchWithDomainPortCombination() {
	t := s.T()

	description := "http://localhost:8080/detail/154687364529310 is related issue"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	_, err := s.wiRepo.Create(
		context.Background(),
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_new",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum", workitem.SystemState: workitem.SystemStateClosed,
		},
		uuid.NewV4())
	require.Nil(t, err)

	controller := NewSearchController(s.service, gormapplication.NewGormDB(s.DB), s.spaceBlackBoxTestConfiguration)
	q := `"http://localhost:8080/detail/154687364529310"`
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	require.NotEmpty(t, sr.Data)
	r := sr.Data[0]
	assert.Equal(t, description, r.Attributes[workitem.SystemDescription])
}

func (s *searchBlackBoxTest) TestSearchURLWithoutPort() {
	t := s.T()

	description := "This issue is related to http://localhost/detail/876394"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	_, err := s.wiRepo.Create(
		context.Background(),
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_without_port",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		uuid.NewV4())
	require.Nil(t, err)

	controller := NewSearchController(s.service, gormapplication.NewGormDB(s.DB), s.spaceBlackBoxTestConfiguration)
	q := `"http://localhost/detail/876394"`
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	require.NotEmpty(t, sr.Data)
	r := sr.Data[0]
	assert.Equal(t, description, r.Attributes[workitem.SystemDescription])
}

func (s *searchBlackBoxTest) TestUnregisteredURLWithPort() {
	t := s.T()

	description := "Related to http://some-other-domain:8080/different-path/154687364529310/ok issue"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	_, err := s.wiRepo.Create(
		context.Background(),
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_new",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		uuid.NewV4())
	require.Nil(t, err)

	controller := NewSearchController(s.service, gormapplication.NewGormDB(s.DB), s.spaceBlackBoxTestConfiguration)
	q := `http://some-other-domain:8080/different-path/`
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	require.NotEmpty(t, sr.Data)
	r := sr.Data[0]
	assert.Equal(t, description, r.Attributes[workitem.SystemDescription])
}

func (s *searchBlackBoxTest) TestUnwantedCharactersRelatedToSearchLogic() {
	t := s.T()

	expectedDescription := rendering.NewMarkupContentFromLegacy("Related to http://example-domain:8080/different-path/ok issue")

	_, err := s.wiRepo.Create(
		context.Background(),
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_new",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		uuid.NewV4())
	require.Nil(t, err)

	controller := NewSearchController(s.service, gormapplication.NewGormDB(s.DB), s.spaceBlackBoxTestConfiguration)
	// add url: in the query, that is not expected by the code hence need to make sure it gives expected result.
	q := `http://url:some-random-other-domain:8080/different-path/`
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	require.NotNil(t, sr.Data)
	assert.Empty(t, sr.Data)
}

func (s *searchBlackBoxTest) getWICreatePayload() *app.CreateWorkitemPayload {
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
			},
		},
	}
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	return &c
}

// searchByURL copies much of the codebase from search_testing.go->ShowSearchOK
// and customises the values to add custom Host in the call.
func (s *searchBlackBoxTest) searchByURL(t *testing.T, customHost, queryString string) *app.SearchWorkItemList {
	var resp interface{}
	var respSetter goatest.ResponseSetterFunc = func(r interface{}) { resp = r }
	newEncoder := func(io.Writer) goa.Encoder { return respSetter }
	s.service.Encoder = goa.NewHTTPEncoder()
	s.service.Encoder.Register(newEncoder, "*/*")
	rw := httptest.NewRecorder()
	query := url.Values{}
	u := &url.URL{
		Path:     fmt.Sprintf("/api/search"),
		RawQuery: query.Encode(),
		Host:     customHost,
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{}
	prms["q"] = []string{queryString} // any value will do
	ctx := s.service.Context
	goaCtx := goa.NewContext(goa.WithAction(ctx, "SearchTest"), rw, req, prms)
	showCtx, err := app.NewShowSearchContext(goaCtx, req, s.service)
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}
	ctrl := NewSearchController(s.service, gormapplication.NewGormDB(s.DB), s.spaceBlackBoxTestConfiguration)
	// Perform action
	err = ctrl.Show(showCtx)

	// Validate response
	if err != nil {
		t.Fatalf("controller returned %s", err)
	}
	if rw.Code != 200 {
		t.Fatalf("invalid response status code: got %+v, expected 200", rw.Code)
	}
	mt, ok := resp.(*app.SearchWorkItemList)
	if !ok {
		t.Fatalf("invalid response media: got %+v, expected instance of app.SearchWorkItemList", resp)
	}
	return mt
}

// verifySearchByKnownURLs performs actual tests on search result and knwonURL map
func (s *searchBlackBoxTest) verifySearchByKnownURLs(t *testing.T, wi *app.WorkItem2Single, host, searchQuery string) {
	result := s.searchByURL(t, host, searchQuery)
	assert.NotEmpty(t, result.Data)
	assert.Equal(t, *wi.Data.ID, *result.Data[0].ID)

	known := search.GetAllRegisteredURLs()
	require.NotNil(t, known)
	assert.NotEmpty(t, known)
	assert.Contains(t, known[search.HostRegistrationKeyForListWI].URLRegex, host)
	assert.Contains(t, known[search.HostRegistrationKeyForBoardWI].URLRegex, host)
}

// TestAutoRegisterHostURL checks if client's host is neatly registered as a KnwonURL or not
// Uses helper functions verifySearchByKnownURLs, searchByURL, getWICreatePayload
func (s *searchBlackBoxTest) TestAutoRegisterHostURL() {
	t := s.T()
	service := getServiceAsUser()
	wiCtrl := NewWorkitemController(s.service, gormapplication.NewGormDB(s.DB))
	// create a WI, search by `list view URL` of newly created item
	newWI := s.getWICreatePayload()
	_, wi := test.CreateWorkitemCreated(t, s.service.Context, service, wiCtrl, newWI)
	require.NotNil(t, wi)
	customHost := "own.domain.one"
	queryString := fmt.Sprintf("http://%s/work-item/list/detail/%s", customHost, *wi.Data.ID)
	s.verifySearchByKnownURLs(t, wi, customHost, queryString)

	// Search by `board view URL` of newly created item
	customHost2 := "own.domain.two"
	queryString2 := fmt.Sprintf("http://%s/work-item/board/detail/%s", customHost2, *wi.Data.ID)
	s.verifySearchByKnownURLs(t, wi, customHost2, queryString2)
}
