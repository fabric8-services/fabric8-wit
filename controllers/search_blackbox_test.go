package controllers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/search"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/goatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func getServiceAsUser() *goa.Service {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	service := testsupport.ServiceAsUser("TestSearch-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return service
}

func TestSearch(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()

	service := getServiceAsUser()
	wiRepo := workitem.NewWorkItemRepository(DB)

	_, err := wiRepo.Create(
		context.Background(),
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch",
			workitem.SystemDescription: nil,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		"")
	require.Nil(t, err)
	controller := NewSearchController(service, gormapplication.NewGormDB(DB))
	q := "specialwordforsearch"
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	require.NotEmpty(t, sr.Data)
	r := sr.Data[0]
	assert.Equal(t, "specialwordforsearch", r.Attributes[workitem.SystemTitle])
}

func TestSearchPagination(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()
	service := getServiceAsUser()

	wiRepo := workitem.NewWorkItemRepository(DB)

	_, err := wiRepo.Create(
		context.Background(),
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch2",
			workitem.SystemDescription: nil,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		"")
	require.Nil(t, err)

	controller := NewSearchController(service, gormapplication.NewGormDB(DB))
	q := "specialwordforsearch2"
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)

	// defaults in paging.go is 'pageSizeDefault = 20'
	assert.Equal(t, "http:///api/search?page[offset]=0&page[limit]=20&q=specialwordforsearch2", *sr.Links.First)
	assert.Equal(t, "http:///api/search?page[offset]=0&page[limit]=20&q=specialwordforsearch2", *sr.Links.Last)
	require.NotEmpty(t, sr.Data)
	r := sr.Data[0]
	assert.Equal(t, "specialwordforsearch2", r.Attributes[workitem.SystemTitle])
}

func TestSearchWithEmptyValue(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()
	service := getServiceAsUser()
	wiRepo := workitem.NewWorkItemRepository(DB)

	_, err := wiRepo.Create(
		context.Background(),
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch",
			workitem.SystemDescription: nil,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		"")
	require.Nil(t, err)

	controller := NewSearchController(service, gormapplication.NewGormDB(DB))
	q := ""
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	require.NotNil(t, sr.Data)
	assert.Empty(t, sr.Data)
}

func TestSearchWithDomainPortCombination(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()
	service := getServiceAsUser()
	wiRepo := workitem.NewWorkItemRepository(DB)

	description := "http://localhost:8080/detail/154687364529310 is related issue"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	_, err := wiRepo.Create(
		context.Background(),
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_new",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum", workitem.SystemState: workitem.SystemStateClosed,
		},
		"")
	require.Nil(t, err)

	controller := NewSearchController(service, gormapplication.NewGormDB(DB))
	q := `"http://localhost:8080/detail/154687364529310"`
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	require.NotEmpty(t, sr.Data)
	r := sr.Data[0]
	assert.Equal(t, description, r.Attributes[workitem.SystemDescription])
}

func TestSearchURLWithoutPort(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()
	service := getServiceAsUser()
	wiRepo := workitem.NewWorkItemRepository(DB)

	description := "This issue is related to http://localhost/detail/876394"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	_, err := wiRepo.Create(
		context.Background(),
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_without_port",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		"")
	require.Nil(t, err)

	controller := NewSearchController(service, gormapplication.NewGormDB(DB))
	q := `"http://localhost/detail/876394"`
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	require.NotEmpty(t, sr.Data)
	r := sr.Data[0]
	assert.Equal(t, description, r.Attributes[workitem.SystemDescription])
}

func TestUnregisteredURLWithPort(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()
	service := getServiceAsUser()
	wiRepo := workitem.NewWorkItemRepository(DB)

	description := "Related to http://some-other-domain:8080/different-path/154687364529310/ok issue"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	_, err := wiRepo.Create(
		context.Background(),
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_new",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		"")
	require.Nil(t, err)

	controller := NewSearchController(service, gormapplication.NewGormDB(DB))
	q := `http://some-other-domain:8080/different-path/`
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	require.NotEmpty(t, sr.Data)
	r := sr.Data[0]
	assert.Equal(t, description, r.Attributes[workitem.SystemDescription])
}

func TestUnwantedCharactersRelatedToSearchLogic(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()
	service := getServiceAsUser()
	wiRepo := workitem.NewWorkItemRepository(DB)

	expectedDescription := rendering.NewMarkupContentFromLegacy("Related to http://example-domain:8080/different-path/ok issue")

	_, err := wiRepo.Create(
		context.Background(),
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_new",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		"")
	require.Nil(t, err)

	controller := NewSearchController(service, gormapplication.NewGormDB(DB))
	// add url: in the query, that is not expected by the code hence need to make sure it gives expected result.
	q := `http://url:some-random-other-domain:8080/different-path/`
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	require.NotNil(t, sr.Data)
	assert.Empty(t, sr.Data)
}

func getWICreatePayload() *app.CreateWorkitemPayload {
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
func searchByURL(t *testing.T, customHost, queryString string) *app.SearchWorkItemList {
	service := getServiceAsUser()
	var resp interface{}
	var respSetter goatest.ResponseSetterFunc = func(r interface{}) { resp = r }
	newEncoder := func(io.Writer) goa.Encoder { return respSetter }
	service.Encoder = goa.NewHTTPEncoder()
	service.Encoder.Register(newEncoder, "*/*")
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
	ctx := service.Context
	goaCtx := goa.NewContext(goa.WithAction(ctx, "SearchTest"), rw, req, prms)
	showCtx, err := app.NewShowSearchContext(goaCtx, req, service)
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}
	ctrl := NewSearchController(service, gormapplication.NewGormDB(DB))
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
func verifySearchByKnownURLs(t *testing.T, wi *app.WorkItem2Single, host, searchQuery string) {
	result := searchByURL(t, host, searchQuery)
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
func TestAutoRegisterHostURL(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()
	service := getServiceAsUser()
	wiCtrl := NewWorkitemController(service, gormapplication.NewGormDB(DB))
	// create a WI, search by `list view URL` of newly created item
	newWI := getWICreatePayload()
	_, wi := test.CreateWorkitemCreated(t, service.Context, service, wiCtrl, newWI)
	require.NotNil(t, wi)
	customHost := "own.domain.one"
	queryString := fmt.Sprintf("http://%s/work-item/list/detail/%s", customHost, *wi.Data.ID)
	verifySearchByKnownURLs(t, wi, customHost, queryString)

	// Search by `board view URL` of newly created item
	customHost2 := "own.domain.two"
	queryString2 := fmt.Sprintf("http://%s/work-item/board/detail/%s", customHost2, *wi.Data.ID)
	verifySearchByKnownURLs(t, wi, customHost2, queryString2)
}
