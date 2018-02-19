package iteration_test

import (
	"strconv"
	"testing"

	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScan(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	type testData struct {
		Input          interface{}
		ExpectError    bool
		ExpectedOutput *string
	}
	data := []testData{
		{iteration.StateNew.String(), false, iteration.StateNew.StringPtr()},
		{iteration.StateStart.String(), false, iteration.StateStart.StringPtr()},
		{iteration.StateClose.String(), false, iteration.StateClose.StringPtr()},
		{iteration.StateNew, false, iteration.StateNew.StringPtr()},
		{iteration.StateStart, false, iteration.StateStart.StringPtr()},
		{iteration.StateClose, false, iteration.StateClose.StringPtr()},
		{nil, true, nil},
		{"", true, nil},
		{"foo", true, nil},
		{true, true, nil},
		{false, true, nil},
		{1234, true, nil},
	}
	for idx, td := range data {
		testName := "test " + strconv.Itoa(idx)
		if !td.ExpectError {
			switch state := td.Input.(type) {
			case string:
				testName += " " + state
			case iteration.State:
				testName += " " + state.String()
			default:
				t.Fatalf("failed to convert iteration state \"%+v\" to iteration.State or string", td.Input)
			}
		}
		t.Run(testName, func(t *testing.T) {
			iter := iteration.State("")
			err := iter.Scan(td.Input)
			if !td.ExpectError {
				require.NoError(t, err)
				assert.Equal(t, *td.ExpectedOutput, iter.String())
			} else {
				require.Error(t, err)
			}
		})
	}
}
