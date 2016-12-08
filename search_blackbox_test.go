package main_test

import (
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
)

func getServiceAsUser() *goa.Service {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	service := testsupport.ServiceAsUser("TestSearch-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
	return service
}

func TestSearch(t *testing.T) {
	resource.Require(t, resource.Database)
	service := getServiceAsUser()
	wiController := NewWorkitemController(service, gormapplication.NewGormDB(DB))

	wiPayload := app.CreateWorkItemPayload{
		Type: workitem.SystemBug,
		Fields: map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch",
			workitem.SystemDescription: "",
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       "closed"},
	}

	_, wiResult := test.CreateWorkitemCreated(t, service.Context, service, wiController, &wiPayload)

	controller := NewSearchController(service, gormapplication.NewGormDB(DB))
	q := "specialwordforsearch"
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	r := sr.Data[0]
	assert.Equal(t, "specialwordforsearch", r.Fields[workitem.SystemTitle])
	test.DeleteWorkitemOK(t, nil, nil, wiController, wiResult.ID)
}

func TestSearchPagination(t *testing.T) {
	resource.Require(t, resource.Database)
	service := getServiceAsUser()
	wiController := NewWorkitemController(service, gormapplication.NewGormDB(DB))

	wiPayload := app.CreateWorkItemPayload{
		Type: workitem.SystemBug,
		Fields: map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch2",
			workitem.SystemDescription: "",
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       "closed"},
	}

	_, wiResult := test.CreateWorkitemCreated(t, service.Context, service, wiController, &wiPayload)

	controller := NewSearchController(service, gormapplication.NewGormDB(DB))
	q := "specialwordforsearch2"
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	assert.Equal(t, "http:///api/search?q=specialwordforsearch2&page[offset]=0&page[limit]=100", *sr.Links.First)
	assert.Equal(t, "http:///api/search?q=specialwordforsearch2&page[offset]=0&page[limit]=100", *sr.Links.Last)
	r := sr.Data[0]
	assert.Equal(t, "specialwordforsearch2", r.Fields[workitem.SystemTitle])
	test.DeleteWorkitemOK(t, nil, nil, wiController, wiResult.ID)
}

func TestSearchWithEmptyValue(t *testing.T) {
	resource.Require(t, resource.Database)
	service := getServiceAsUser()
	wiController := NewWorkitemController(service, gormapplication.NewGormDB(DB))

	wiPayload := app.CreateWorkItemPayload{
		Type: workitem.SystemBug,
		Fields: map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch",
			workitem.SystemDescription: "",
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       "closed"},
	}

	_, wiResult := test.CreateWorkitemCreated(t, service.Context, service, wiController, &wiPayload)

	controller := NewSearchController(service, gormapplication.NewGormDB(DB))
	q := ""
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	assert.Equal(t, 0, len(sr.Data))
	test.DeleteWorkitemOK(t, nil, nil, wiController, wiResult.ID)
}

func TestSearchWithDomainPortCombination(t *testing.T) {
	resource.Require(t, resource.Database)
	service := getServiceAsUser()
	wiController := NewWorkitemController(service, gormapplication.NewGormDB(DB))

	expectedDescription := "http://localhost:8080/detail/154687364529310 is related issue"
	wiPayload := app.CreateWorkItemPayload{
		Type: workitem.SystemBug,
		Fields: map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_new",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       "closed"},
	}

	_, wiResult := test.CreateWorkitemCreated(t, service.Context, service, wiController, &wiPayload)

	controller := NewSearchController(service, gormapplication.NewGormDB(DB))
	q := `"http://localhost:8080/detail/154687364529310"`
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	assert.NotEqual(t, 0, len(sr.Data))
	r := sr.Data[0]
	assert.Equal(t, expectedDescription, r.Fields[workitem.SystemDescription])
	test.DeleteWorkitemOK(t, nil, nil, wiController, wiResult.ID)
}

func TestSearchURLWithoutPort(t *testing.T) {
	resource.Require(t, resource.Database)
	service := getServiceAsUser()
	wiController := NewWorkitemController(service, gormapplication.NewGormDB(DB))

	expectedDescription := "This issue is related to http://localhost/detail/876394"
	wiPayload := app.CreateWorkItemPayload{
		Type: workitem.SystemBug,
		Fields: map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_without_port",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       "closed"},
	}

	_, wiResult := test.CreateWorkitemCreated(t, service.Context, service, wiController, &wiPayload)

	controller := NewSearchController(service, gormapplication.NewGormDB(DB))
	q := `"http://localhost/detail/876394"`
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	assert.NotEqual(t, 0, len(sr.Data))
	r := sr.Data[0]
	assert.Equal(t, expectedDescription, r.Fields[workitem.SystemDescription])
	test.DeleteWorkitemOK(t, nil, nil, wiController, wiResult.ID)
}

func TestUnregisteredURLWithPort(t *testing.T) {
	resource.Require(t, resource.Database)
	service := getServiceAsUser()
	wiController := NewWorkitemController(service, gormapplication.NewGormDB(DB))
	expectedDescription := "Related to http://some-other-domain:8080/different-path/154687364529310/ok issue"
	wiPayload := app.CreateWorkItemPayload{
		Type: workitem.SystemBug,
		Fields: map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_new",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       "closed"},
	}

	_, wiResult := test.CreateWorkitemCreated(t, service.Context, service, wiController, &wiPayload)

	controller := NewSearchController(service, gormapplication.NewGormDB(DB))
	q := `http://some-other-domain:8080/different-path/`
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	assert.NotEqual(t, 0, len(sr.Data))
	r := sr.Data[0]
	assert.Equal(t, expectedDescription, r.Fields[workitem.SystemDescription])
	test.DeleteWorkitemOK(t, nil, nil, wiController, wiResult.ID)
}

func TestUnwantedCharactersRelatedToSearchLogic(t *testing.T) {
	resource.Require(t, resource.Database)
	service := getServiceAsUser()
	wiController := NewWorkitemController(service, gormapplication.NewGormDB(DB))
	expectedDescription := "Related to http://example-domain:8080/different-path/ok issue"
	wiPayload := app.CreateWorkItemPayload{
		Type: workitem.SystemBug,
		Fields: map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_new",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       "closed"},
	}

	_, wiResult := test.CreateWorkitemCreated(t, service.Context, service, wiController, &wiPayload)

	controller := NewSearchController(service, gormapplication.NewGormDB(DB))
	// add url: in the query, that is not expected by the code hence need to make sure it gives expected result.
	q := `http://url:some-random-other-domain:8080/different-path/`
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, q)
	assert.Equal(t, 0, len(sr.Data))
	test.DeleteWorkitemOK(t, nil, nil, wiController, wiResult.ID)
}
