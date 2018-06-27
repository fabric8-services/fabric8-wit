package test

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AssertError(t require.TestingT, actualError error, expectedType interface{}, expectedMsgAndArgs ...interface{}) {
	require.Error(t, actualError)
	assert.IsType(t, expectedType, errors.Cause(actualError))
	assert.Equal(t, messageFromMsgAndArgs(expectedMsgAndArgs...), actualError.Error())
}

func messageFromMsgAndArgs(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 || msgAndArgs == nil {
		return ""
	}
	if len(msgAndArgs) == 1 {
		return msgAndArgs[0].(string)
	}
	if len(msgAndArgs) > 1 {
		return fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}
	return ""
}
