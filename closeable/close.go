package closeable

import (
	"context"
	"io"

	"github.com/fabric8-services/fabric8-wit/log"
)

// Close closes the given resource and logs the error if something wrong happened
func Close(ctx context.Context, c io.Closer) {
	if c != nil {
		err := c.Close()
		if err != nil {
			log.Error(ctx, map[string]interface{}{"error": err.Error()}, "error while closing the resource")
		}
	}

}
