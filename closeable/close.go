package closeable

import (
	"context"
	"io"
	"reflect"

	"github.com/fabric8-services/fabric8-wit/log"
)

// Close closes the given resource and logs the error if something wrong happened
func Close(ctx context.Context, c io.Closer) {
	// need to verify that the value of the `c` interface if not nil, too
	if c != nil && !reflect.ValueOf(c).IsNil() {
		err := c.Close()
		if err != nil {
			log.Error(ctx, map[string]interface{}{"error": err.Error()}, "error while closing the resource")
		}
	}

}
