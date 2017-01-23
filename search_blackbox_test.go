package main_test

import (
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func getServiceAsUser() *goa.Service {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	service := testsupport.ServiceAsUser("TestSearch-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
	return service
}

func TestSearch(t *testing.T) {
	resource.Require(t, resource.Database)
	defer gormsupport.DeleteCreatedEntities(DB)()

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
	defer gormsupport.DeleteCreatedEntities(DB)()
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
	assert.Equal(t, "http:///api/search?q=specialwordforsearch2&page[offset]=0&page[limit]=100", *sr.Links.First)
	assert.Equal(t, "http:///api/search?q=specialwordforsearch2&page[offset]=0&page[limit]=100", *sr.Links.Last)
	require.NotEmpty(t, sr.Data)
	r := sr.Data[0]
	assert.Equal(t, "specialwordforsearch2", r.Attributes[workitem.SystemTitle])
}

func TestSearchWithEmptyValue(t *testing.T) {
	resource.Require(t, resource.Database)
	defer gormsupport.DeleteCreatedEntities(DB)()
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
	defer gormsupport.DeleteCreatedEntities(DB)()
	service := getServiceAsUser()
	wiRepo := workitem.NewWorkItemRepository(DB)

	description := "http://localhost:8080/detail/154687364529310 is related issue"
	expectedDescription := workitem.MarkupContent{Content: description, Markup: workitem.SystemMarkupDefault}
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
	defer gormsupport.DeleteCreatedEntities(DB)()
	service := getServiceAsUser()
	wiRepo := workitem.NewWorkItemRepository(DB)

	description := "This issue is related to http://localhost/detail/876394"
	expectedDescription := workitem.MarkupContent{Content: description, Markup: workitem.SystemMarkupDefault}
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
	defer gormsupport.DeleteCreatedEntities(DB)()
	service := getServiceAsUser()
	wiRepo := workitem.NewWorkItemRepository(DB)

	description := "Related to http://some-other-domain:8080/different-path/154687364529310/ok issue"
	expectedDescription := workitem.MarkupContent{Content: description, Markup: workitem.SystemMarkupDefault}
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
	defer gormsupport.DeleteCreatedEntities(DB)()
	service := getServiceAsUser()
	wiRepo := workitem.NewWorkItemRepository(DB)

	expectedDescription := workitem.MarkupContent{Content: "Related to http://example-domain:8080/different-path/ok issue",
		Markup: workitem.SystemMarkupDefault}

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
