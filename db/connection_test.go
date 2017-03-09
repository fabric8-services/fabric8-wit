//+build integration

package db

import (
	"testing"

	_ "github.com/lib/pq"
)

func TestGetDatabaseConnection(t *testing.T) {

	// more of a usage example as of now.
	connectionString := DetectConnectionString()
	dc := DatabaseConfiguration{connectionString}
	connection, err := dc.GetDatabaseConnection()

	if err != nil {
		t.Error("Failed to connect to database " + err.Error())
	} else {
		defer connection.Close()
	}
}
