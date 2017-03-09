package db

import (
	"fmt"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

const databaseName string = "postgres"

// DatabaseConfiguration describes what a configuration must look like.
type DatabaseConfiguration struct {
	connectionString string
}

// GetDatabaseConnection Returns a database connection handle.
func (databaseConfig DatabaseConfiguration) GetDatabaseConnection() (*gorm.DB, error) {

	db, err := gorm.Open(databaseName, fmt.Sprintf(databaseConfig.connectionString))
	return db, err
}

// DetectConnectionString helps us with the defaults
func DetectConnectionString() string {
	host := detectDatabaseHost()
	connectionString := fmt.Sprintf("host=%s user=postgres password=mysecretpassword sslmode=disable", host)
	return connectionString
}

// DetectDatabaseHost figures out the db host. The code needs to be modified based on our deployment strategy.
func detectDatabaseHost() string {
	dbHost := os.Getenv("DBHOST")
	if len(dbHost) == 0 {
		dbHost = "localhost"
	}
	return dbHost
}
