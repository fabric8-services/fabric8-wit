package proxy_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest/proxy"
	"github.com/stretchr/testify/require"
)

func TestConvertHTTPErrorToTypeError(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	testCases := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedResult error
	}{
		{"http 500 test", 500, "I am an InternalServerError", errors.InternalError{}},
		{"http 400 test", 400, "I am an BadParameterError", errors.BadParameterError{}},
		{"http 401 test", 401, "I am an UnAuthorizedError", errors.UnauthorizedError{}},
		{"http 409 test", 409, "I am an StatusConflictError", errors.DataConflictError{}},
		{"http 403 test", 403, "I am an Forbidden", errors.ForbiddenError{}},
		{"http 404 test", 404, "I am an NotFoundError", errors.NotFoundError{}},
	}
	for _, tc := range testCases {
		// Note that we need to capture the range variable to ensure that tc
		// gets bound to the correct instance.
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			returnedErr := proxy.ConvertHTTPErrorCode(tc.statusCode, tc.responseBody)
			require.IsType(t, tc.expectedResult, returnedErr)
		})
	}
}
