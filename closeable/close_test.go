package closeable_test

import (
	"context"
	"database/sql"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fabric8-services/fabric8-wit/closeable"
)

func TestCloseable(t *testing.T) {

	t.Run("non nil", func(t *testing.T) {
		// given
		c := &Rows{}
		require.NotNil(t, c)
		// when
		closeable.Close(context.Background(), c)
		// then it should not fail
	})

	t.Run("nil", func(t *testing.T) {
		// given
		var c io.Closer
		require.Nil(t, c)
		// when
		closeable.Close(context.Background(), c)
		// then it should not fail
	})

	t.Run("non nil with nil value", func(t *testing.T) {
		// given
		c := newCloseable()
		// when
		closeable.Close(context.Background(), c)
		// then it should not fail
	})
}

func newCloseable() io.Closer {
	var c *Rows
	return c
}

type Rows struct {
	db *sql.DB
}

func (c *Rows) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}
