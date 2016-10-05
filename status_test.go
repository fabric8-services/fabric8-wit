package main

import (
	"testing"

	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"time"
)

func TestAuthorizeLoginOK(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	controller := LoginController{}
	_, res := test.AuthorizeLoginOK(t, nil, nil, &controller)

	if res.Token == "" {
		t.Error("Token not generated")
	}
}

func TestShowStatusOK(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.Database)
	controller := StatusController{db: DB}
	_, res := test.ShowStatusOK(t, nil, nil, &controller)

	if res.Commit != "0" {
		t.Error("Commit not found")
	}
	if res.StartTime != StartTime {
		t.Error("StartTime is not correct")
	}
	_, err := time.Parse("2006-01-02T15:04:05Z", res.StartTime)
	if err != nil {
		t.Error("Incorrect layout of StartTime: ", err.Error())
	}
}

func TestNewStatusController(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	svc := goa.New("TestNewStatusControllerService")
	assert.NotNil(t, NewStatusController(svc, nil))
}

var ValidJWTToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE0NjA4MzUyNzUsInNjb3BlcyI6WyJzeXN0ZW0iXX0.OHsz9bIN9nKemd8Rdm9lYapXOknh5nwvCN8ZD_YIVfCZ54MkoKiIjj_VsGclRMCykDtXD4Omg2mWuiaEDPoP4nHRjlWfup3Us29k78cpImBz6FwfK08J39pKr0Y7s-Qdpq_XGwdTEWx7Hk33nrgyZVdMfE4nRjCulkIWbhOxNDdjKqUSo3zknRQRWzZhVl8a1cMNG6EetFHe-pCEr3WpreeRZcoL948smll_16WYB8r3t2-jtW7CmrJwSx7ZMopD-AvOaAGsiExgNRUd5YcSX0zEl5mjwnSb-rqemQt4_BHs0zgufyDw5MtH0ZG8phNIbyWt3G1VaO3CqDt_Ixxh7Q"

var InValidJWTToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE0NjA4MzUyNzUsInNjb3BlcyI6WyJzeXN0ZW0iXX0.OHsz9bIN9nKemd8Rdm9lYapXOknh5nwvCN8ZD_YIVfCZ54MkoKiIjj_VsGclRMCykDtXD4Omg2mWuiaEDPoP4nHRjlWfup3Us29k78cpImBz6FwfK08J39pKr0Y7s-Qdpq_XGwdTEWx7Hk33nrgyZVdMfE4nRjCulkIWbhOxNDdjKqUSo3zknRQRWzZhVl8a1cMNG6EetFHe-pCEr3WpreeRZcoL948smll_16WYB8r3t2-jtW7CmrJwSx7ZMopD-AvOaAGsiExgNRUd5YcSX0zEl5mjwnSb-rqemQt4_BHs0zgufyDw5MtH0ZG8phNIbyWt3G1VaO3CqDt_Ixxh7"
