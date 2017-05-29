package controller_test

import (
	"net/http"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/workitem/link"
	"github.com/stretchr/testify/require"

	"github.com/goadesign/goa"
)

func createWorkItemTypeCombination(t *testing.T, appl application.Application, wiltcCtrl *WorkItemLinkTypeCombinationController, witModel link.WorkItemLinkTypeCombination) (http.ResponseWriter, *app.WorkItemLinkTypeCombinationSingle) {
	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	wit, err := ConvertWorkItemLinkTypeCombinationFromModel(appl, reqLong, witModel)
	require.Nil(t, err)
	payload := app.CreateWorkItemLinkTypeCombinationPayload{
		Data: wit,
	}
	responseWriter, wi := test.CreateWorkItemLinkTypeCombinationCreated(t, nil, nil, wiltcCtrl, witModel.SpaceID, &payload)
	return responseWriter, wi
}
