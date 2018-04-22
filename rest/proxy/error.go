package proxy

import (
	"github.com/fabric8-services/fabric8-wit/errors"
	"net/http"
)

// ConvertHTTPErrorCode converts an http status code to an instance of type error
func ConvertHTTPErrorCode(statusCode int, responseBody string) error {

	switch statusCode {

	case http.StatusNotFound:
		return errors.NewNotFoundErrorFromString(responseBody)

	case http.StatusBadRequest:
		return errors.NewBadParameterErrorFromString(responseBody)

	case http.StatusConflict:
		return errors.NewDataConflictError(responseBody)

	case http.StatusUnauthorized:
		return errors.NewUnauthorizedError(responseBody)

	case http.StatusForbidden:
		return errors.NewForbiddenError(responseBody)

	default:
		return errors.NewInternalErrorFromString(responseBody)
	}
}
