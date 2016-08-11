package utils

import (
	"fmt"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

// DatabaseConfiguration describes what a configuration must look like.
type DatabaseConfiguration struct {
	databaseName     string // postgres , mysql , oracle
	connectionString string
}

// GetDatabaseConnection Returns a database connection handle.
func (databaseConfig DatabaseConfiguration) GetDatabaseConnection() (*gorm.DB, error) {

	db, err := gorm.Open(databaseConfig.databaseName, fmt.Sprintf(databaseConfig.connectionString))

	defer db.Close()
	return db, err
}

// DetectConnectionString helps us with the defaults
func DetectConnectionString() string {
	host := detectDatabaseHost()
	connectionString := fmt.Sprintf("host=%s user=postgres password=mysecretpassword sslmode=disable", host)
}

// DetectDatabaseName fetches the database name from env variables.
func DetectDatabaseName() string {
	return "postgres" // Shouldn't change in a while.
}

// DetectDatabaseHost figures out the db host. The code needs to be modified based on our deployment strategy.
func detectDatabaseHost() string {
	dbHost := os.Getenv("DBHOST")
	if len(dbHost) == 0 {
		dbHost = "localhost"
	}
	return dbHost
}
