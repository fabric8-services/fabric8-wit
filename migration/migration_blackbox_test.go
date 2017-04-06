package migration_test

import (
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"testing"

	config "github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/migration"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	errs "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

// fn defines the type of function that can be part of a migration steps
type fn func(tx *sql.Tx) error

const (
	// Migration version where to start testing
	initialMigratedVersion = 45
	databaseName           = "test"
)

var (
	conf       *config.ConfigurationData
	migrations migration.Migrations
)

func init() {
	var err error
	conf, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	configurationString := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s",
		conf.GetPostgresHost(),
		conf.GetPostgresPort(),
		conf.GetPostgresUser(),
		conf.GetPostgresPassword(),
		conf.GetPostgresSSLMode(),
	)

	db, err := sql.Open("postgres", configurationString)
	defer db.Close()
	if err != nil {
		panic(fmt.Errorf("Cannot connect to DB: %s\n", err))
	}

	db.Exec("DROP DATABASE " + databaseName)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("CREATE DATABASE " + databaseName)
	if err != nil {
		panic(err)
	}

	migrations = migration.GetMigrations()
}

func TestMigrations(t *testing.T) {
	configurationString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		conf.GetPostgresHost(),
		conf.GetPostgresPort(),
		conf.GetPostgresUser(),
		conf.GetPostgresPassword(),
		databaseName,
		conf.GetPostgresSSLMode(),
	)

	db, err := sql.Open("postgres", configurationString)
	defer db.Close()
	if err != nil {
		panic(fmt.Errorf("Cannot connect to DB: %s\n", err))
	}
	gormDB, err := gorm.Open("postgres", configurationString)
	defer gormDB.Close()
	if err != nil {
		t.Fatalf("Cannot connect to DB: %s\n", err)
	}
	dialect := gormDB.Dialect()
	dialect.SetDB(db)

	m := migrations[:initialMigratedVersion]

	for nextVersion := int64(0); nextVersion < int64(len(m)) && err == nil; nextVersion++ {
		var tx *sql.Tx
		tx, err = db.Begin()
		if err != nil {
			t.Fatalf("Failed to start transaction: %s\n", err)
		}

		if err = migration.MigrateToNextVersion(tx, &nextVersion, m, databaseName); err != nil {
			t.Errorf("Failed to migrate to version %d: %s\n", nextVersion, err)

			if err = tx.Rollback(); err != nil {
				t.Fatalf("error while rolling back transaction: ", err)
			}
			t.Fatal("Failed to migrate to version after rolling back")
		}

		if err = tx.Commit(); err != nil {
			t.Fatalf("Error during transaction commit: %s\n", err)
		}
	}

	// Insert dummy test data
	assert.Nil(t, runSQLscript(db, "044-insert-test-data.sql"))

	// Migration 45
	migrationToVersion(db, migrations[:(initialMigratedVersion+1)], (initialMigratedVersion + 1))
	testMigration45(gormDB, dialect, t)
	assert.Nil(t, runSQLscript(db, "045-update-work-items.sql"))

	// Migration 46
	migrationToVersion(db, migrations[:(initialMigratedVersion+2)], (initialMigratedVersion + 2))
	testMigration46(gormDB, dialect, t)
	assert.Nil(t, runSQLscript(db, "046-insert-oauth-states.sql"))

	// Migration 47
	migrationToVersion(db, migrations[:(initialMigratedVersion+3)], (initialMigratedVersion + 3))
	testMigration47(gormDB, dialect, t)
	assert.Nil(t, runSQLscript(db, "047-insert-codebases.sql"))

	// Migration 48
	migrationToVersion(db, migrations[:(initialMigratedVersion+4)], (initialMigratedVersion + 4))
	testMigration48(gormDB, dialect, t)
	// This script execution has to fail
	assert.NotNil(t, runSQLscript(db, "048-unique-idx-failed-insert.sql"))

	// Perform the migration
	if err := migration.Migrate(db, databaseName); err != nil {
		t.Fatalf("Failed to execute the migration: %s\n", err)
	}
}

func testMigration45(db *gorm.DB, dialect gorm.Dialect, t *testing.T) {
	assert.True(t, db.HasTable("work_items"))
	assert.True(t, dialect.HasColumn("work_items", "execution_order"))
	assert.True(t, dialect.HasIndex("work_items", "order_index"))
}

func testMigration46(db *gorm.DB, dialect gorm.Dialect, t *testing.T) {
	assert.True(t, db.HasTable("oauth_state_references"))
	assert.True(t, dialect.HasColumn("oauth_state_references", "referrer"))
	assert.True(t, dialect.HasColumn("oauth_state_references", "id"))
}

func testMigration47(db *gorm.DB, dialect gorm.Dialect, t *testing.T) {
	assert.True(t, db.HasTable("codebases"))
	assert.True(t, dialect.HasColumn("codebases", "type"))
	assert.True(t, dialect.HasColumn("codebases", "url"))
	assert.True(t, dialect.HasColumn("codebases", "space_id"))
	assert.True(t, dialect.HasIndex("codebases", "ix_codebases_space_id"))
}

func testMigration48(db *gorm.DB, dialect gorm.Dialect, t *testing.T) {
	assert.True(t, dialect.HasIndex("iterations", "ix_name"))
}

// runSQLscript loads the given filename from the packaged SQL test files and
// executes it on the given database. Golang text/template module is used
// to handle all the optional arguments passed to the sql test files
func runSQLscript(db *sql.DB, sqlFilename string) error {
	var tx *sql.Tx
	tx, err := db.Begin()
	if err != nil {
		return errs.New(fmt.Sprintf("Failed to start transaction: %s\n", err))
	}
	if err := executeSQLTestFile(sqlFilename)(tx); err != nil {
		log.Warn(nil, nil, "Failed to execute data insertion of version: %s\n", err)
		if err = tx.Rollback(); err != nil {
			return errs.New(fmt.Sprintf("error while rolling back transaction: ", err))
		}
	}
	if err = tx.Commit(); err != nil {
		return errs.New(fmt.Sprintf("Error during transaction commit: %s\n", err))
	}

	return nil
}

// executeSQLTestFile loads the given filename from the packaged SQL files and
// executes it on the given database. Golang text/template module is used
// to handle all the optional arguments passed to the sql test files
func executeSQLTestFile(filename string, args ...string) fn {
	return func(db *sql.Tx) error {
		data, err := Asset(filename)
		if err != nil {
			return errs.WithStack(err)
		}

		if len(args) > 0 {
			tmpl, err := template.New("sql").Parse(string(data))
			if err != nil {
				return errs.WithStack(err)
			}
			var sqlScript bytes.Buffer
			writer := bufio.NewWriter(&sqlScript)
			err = tmpl.Execute(writer, args)
			if err != nil {
				return errs.WithStack(err)
			}
			// We need to flush the content of the writer
			writer.Flush()
			_, err = db.Exec(sqlScript.String())
		} else {
			_, err = db.Exec(string(data))
		}

		return errs.WithStack(err)
	}
}

func migrationToVersion(db *sql.DB, m migration.Migrations, version int64) {
	var (
		tx  *sql.Tx
		err error
	)
	tx, err = db.Begin()
	if err != nil {
		panic(fmt.Errorf("Failed to start transaction: %s\n", err))
	}

	if err = migration.MigrateToNextVersion(tx, &version, m, databaseName); err != nil {
		panic(fmt.Errorf("Failed to migrate to version %d: %s\n", version, err))
	}

	if err = tx.Commit(); err != nil {
		panic(fmt.Errorf("Error during transaction commit: %s\n", err))
	}
}
