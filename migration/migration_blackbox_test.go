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
	"github.com/almighty/almighty-core/resource"

	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	errs "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	dialect    gorm.Dialect
	gormDB     *gorm.DB
	sqlDB      *sql.DB
)

func setupTest() {
	var err error
	conf, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	configurationString := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s connect_timeout=%d",
		conf.GetPostgresHost(),
		conf.GetPostgresPort(),
		conf.GetPostgresUser(),
		conf.GetPostgresPassword(),
		conf.GetPostgresSSLMode(),
		conf.GetPostgresConnectionTimeout(),
	)

	db, err := sql.Open("postgres", configurationString)
	defer db.Close()
	if err != nil {
		panic(fmt.Errorf("Cannot connect to database: %s\n", err))
	}

	db.Exec("DROP DATABASE " + databaseName)

	_, err = db.Exec("CREATE DATABASE " + databaseName)
	if err != nil {
		panic(err)
	}

	migrations = migration.GetMigrations()
}

func TestMigrations(t *testing.T) {
	resource.Require(t, resource.Database)

	setupTest()

	configurationString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		conf.GetPostgresHost(),
		conf.GetPostgresPort(),
		conf.GetPostgresUser(),
		conf.GetPostgresPassword(),
		databaseName,
		conf.GetPostgresSSLMode(),
		conf.GetPostgresConnectionTimeout(),
	)
	var err error
	sqlDB, err = sql.Open("postgres", configurationString)
	defer sqlDB.Close()
	if err != nil {
		panic(fmt.Errorf("Cannot connect to DB: %s\n", err))
	}
	gormDB, err = gorm.Open("postgres", configurationString)
	defer gormDB.Close()
	if err != nil {
		panic(fmt.Errorf("Cannot connect to DB: %s\n", err))
	}
	dialect = gormDB.Dialect()
	dialect.SetDB(sqlDB)

	// We migrate the new database until initialMigratedVersion
	t.Run("TestMigration44", testMigration44)

	// Insert dummy test data to our database
	assert.Nil(t, runSQLscript(sqlDB, "044-insert-test-data.sql"))

	t.Run("TestMigration45", testMigration45)
	t.Run("TestMigration46", testMigration46)
	t.Run("TestMigration47", testMigration47)
	t.Run("TestMigration48", testMigration48)
	t.Run("TestMigration49", testMigration49)
	t.Run("TestMigration50", testMigration50)
	t.Run("TestMigration51", testMigration51)
	t.Run("TestMigration52", testMigration52)
	t.Run("testMigration53", testMigration53)
	t.Run("TestMigration54", testMigration54)
	t.Run("TestMigration55", testMigration55)
	t.Run("TestMigration56", testMigration56)
	t.Run("TestMigration57", testMigration57)
	t.Run("TestMigration60", testMigration60)
	t.Run("TestMigration61", testMigration61)
	t.Run("TestMigration62", testMigration62)

	// Perform the migration
	if err := migration.Migrate(sqlDB, databaseName); err != nil {
		t.Fatalf("Failed to execute the migration: %s\n", err)
	}
}

func testMigration44(t *testing.T) {
	var err error
	m := migrations[:initialMigratedVersion]
	for nextVersion := int64(0); nextVersion < int64(len(m)) && err == nil; nextVersion++ {
		var tx *sql.Tx
		tx, err = sqlDB.Begin()
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
}

func testMigration45(t *testing.T) {
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+1)], (initialMigratedVersion + 1))

	assert.True(t, gormDB.HasTable("work_items"))
	assert.True(t, dialect.HasColumn("work_items", "execution_order"))
	assert.True(t, dialect.HasIndex("work_items", "order_index"))

	assert.Nil(t, runSQLscript(sqlDB, "045-update-work-items.sql"))
}

func testMigration46(t *testing.T) {
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+2)], (initialMigratedVersion + 2))

	assert.True(t, gormDB.HasTable("oauth_state_references"))
	assert.True(t, dialect.HasColumn("oauth_state_references", "referrer"))
	assert.True(t, dialect.HasColumn("oauth_state_references", "id"))

	assert.Nil(t, runSQLscript(sqlDB, "046-insert-oauth-states.sql"))
}

func testMigration47(t *testing.T) {
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+3)], (initialMigratedVersion + 3))

	assert.True(t, gormDB.HasTable("codebases"))
	assert.True(t, dialect.HasColumn("codebases", "type"))
	assert.True(t, dialect.HasColumn("codebases", "url"))
	assert.True(t, dialect.HasColumn("codebases", "space_id"))
	assert.True(t, dialect.HasIndex("codebases", "ix_codebases_space_id"))

	assert.Nil(t, runSQLscript(sqlDB, "047-insert-codebases.sql"))
}

func testMigration48(t *testing.T) {
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+4)], (initialMigratedVersion + 4))

	assert.True(t, dialect.HasIndex("iterations", "ix_name"))

	// This script execution has to fail
	assert.NotNil(t, runSQLscript(sqlDB, "048-unique-idx-failed-insert.sql"))
}

func testMigration49(t *testing.T) {
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+5)], (initialMigratedVersion + 5))

	assert.True(t, dialect.HasIndex("areas", "ix_area_name"))

	// Tests that migration 49 set the system.area to the work_items and its value
	// is 71171e90-6d35-498f-a6a7-2083b5267c18
	rows, err := sqlDB.Query("SELECT count(*), fields->>'system.area' FROM work_items where fields != '{}' GROUP BY fields")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var fields string
		var count int
		err = rows.Scan(&count, &fields)
		assert.True(t, count == 2)
		assert.True(t, fields == "71171e90-6d35-498f-a6a7-2083b5267c18")
	}
}

func testMigration50(t *testing.T) {
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+6)], (initialMigratedVersion + 6))

	assert.True(t, dialect.HasColumn("users", "company"))

	assert.Nil(t, runSQLscript(sqlDB, "050-users-add-column-company.sql"))
}
func testMigration51(t *testing.T) {
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+7)], (initialMigratedVersion + 7))

	assert.True(t, dialect.HasIndex("work_item_link_types", "work_item_link_types_name_idx"))
}

func testMigration52(t *testing.T) {
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+8)], (initialMigratedVersion + 8))

	assert.True(t, dialect.HasIndex("spaces", "spaces_name_idx"))
}

func testMigration53(t *testing.T) {
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+9)], (initialMigratedVersion + 9))
	require.True(t, dialect.HasColumn("identities", "registration_completed"))

	// add new rows and check if the new column has the default value
	assert.Nil(t, runSQLscript(sqlDB, "053-edit-username.sql"))

	// check if ALL the existing rows & new rows have the default value
	rows, err := sqlDB.Query("SELECT registration_completed FROM identities")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var registration_completed bool
		err = rows.Scan(&registration_completed)
		assert.True(t, registration_completed == false)
	}
}

func testMigration54(t *testing.T) {
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+10)], (initialMigratedVersion + 10))

	assert.True(t, dialect.HasColumn("codebases", "stack_id"))

	assert.Nil(t, runSQLscript(sqlDB, "054-add-stackid-to-codebase.sql"))
}

func testMigration55(t *testing.T) {
	// migrate to previous version
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+10)], (initialMigratedVersion + 10))
	// fill DB with invalid data (ie, missing root area)
	assert.Nil(t, runSQLscript(sqlDB, "055-assign-root-area-if-missing.sql"))
	// then apply the fix
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+11)], (initialMigratedVersion + 11))
	// and verify that the root area is available
	rows, err := sqlDB.Query("select fields->>'system.area' from work_items where id = 12345")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var rootArea string
		err = rows.Scan(&rootArea)
		assert.NotEmpty(t, rootArea)
	}
}

func testMigration56(t *testing.T) {
	// migrate to previous version
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+11)], (initialMigratedVersion + 11))
	// fill DB with invalid data (ie, missing root area)
	assert.Nil(t, runSQLscript(sqlDB, "056-assign-root-iteration-if-missing.sql"))
	// then apply the fix
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+12)], (initialMigratedVersion + 12))
	// and verify that the root area is available
	rows, err := sqlDB.Query("select fields->>'system.iteration' from work_items where id = 12346")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var rootIteration string
		err = rows.Scan(&rootIteration)
		assert.NotEmpty(t, rootIteration)
	}
}

func testMigration57(t *testing.T) {
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+13)], (initialMigratedVersion + 13))

	assert.True(t, dialect.HasColumn("codebases", "last_used_workspace"))

	assert.Nil(t, runSQLscript(sqlDB, "057-add-last-used-workspace-to-codebase.sql"))
}

func testMigration60(t *testing.T) {
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+16)], (initialMigratedVersion + 16))

	assert.True(t, dialect.HasIndex("identities", "idx_identities_username"))
}

func testMigration61(t *testing.T) {
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+16)], (initialMigratedVersion + 16))

	// Add on purpose a duplicate to verify that we can successfully run this migration
	assert.Nil(t, runSQLscript(sqlDB, "061-add-duplicate-space-owner-name.sql"))

	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+17)], (initialMigratedVersion + 17))

	assert.True(t, dialect.HasIndex("spaces", "spaces_name_idx"))

	rows, err := sqlDB.Query("SELECT COUNT(*) FROM spaces WHERE name='test.space.one-renamed'")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var count int
		err = rows.Scan(&count)
		assert.True(t, count == 1)
	}

}

func testMigration62(t *testing.T) {
	migrateToVersion(sqlDB, migrations[:(initialMigratedVersion+18)], (initialMigratedVersion + 18))
	assert.Nil(t, runSQLscript(sqlDB, "062-workitem-related-changes.sql"))
	var createdAt time.Time
	var updatedAt time.Time
	var deletedAt time.Time
	var commentedAt time.Time
	var linkedAt time.Time

	// comments
	// work item 62001 was commented
	row := sqlDB.QueryRow("SELECT wi.commented_at, c.created_at FROM work_items wi left join comments c on c.parent_id::bigint = wi.id where wi.id = 62001")
	err := row.Scan(&commentedAt, &createdAt)
	require.Nil(t, err)
	assert.Equal(t, commentedAt, createdAt)
	// work item 62002 was commented, then the comment was updated
	row = sqlDB.QueryRow("SELECT wi.commented_at, c.updated_at FROM work_items wi left join comments c on c.parent_id::bigint = wi.id where wi.id = 62002")
	err = row.Scan(&commentedAt, &updatedAt)
	require.Nil(t, err)
	assert.Equal(t, commentedAt, updatedAt)
	// work item 62003 was commented, then the comment was (soft) deleted
	row = sqlDB.QueryRow("SELECT wi.commented_at, c.deleted_at FROM work_items wi left join comments c on c.parent_id::bigint = wi.id where wi.id = 62003")
	err = row.Scan(&commentedAt, &deletedAt)
	require.Nil(t, err)
	assert.Equal(t, commentedAt, deletedAt)

	// links
	// work items 62004 and 62005 were linked together
	row = sqlDB.QueryRow("SELECT wi.linked_at, wil.created_at FROM work_items wi left join work_item_links wil on wil.source_id = wi.id where wi.id = 62004")
	err = row.Scan(&linkedAt, &createdAt)
	require.Nil(t, err)
	assert.Equal(t, linkedAt, createdAt)
	row = sqlDB.QueryRow("SELECT wi.linked_at, wil.created_at FROM work_items wi left join work_item_links wil on wil.target_id = wi.id where wi.id = 62005")
	err = row.Scan(&linkedAt, &createdAt)
	require.Nil(t, err)
	assert.Equal(t, linkedAt, createdAt)
	// work items 62006 and 62007 were linked together, but then the link was deleted
	row = sqlDB.QueryRow("SELECT wi.linked_at, wil.deleted_at FROM work_items wi left join work_item_links wil on wil.source_id = wi.id where wi.id = 62006")
	err = row.Scan(&linkedAt, &deletedAt)
	require.Nil(t, err)
	assert.Equal(t, linkedAt, deletedAt)
	row = sqlDB.QueryRow("SELECT wi.linked_at, wil.deleted_at FROM work_items wi left join work_item_links wil on wil.target_id = wi.id where wi.id = 62007")
	err = row.Scan(&linkedAt, &deletedAt)
	require.Nil(t, err)
	assert.Equal(t, linkedAt, deletedAt)
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

// migrateToVersion runs the migration of all the scripts to a certain version
func migrateToVersion(db *sql.DB, m migration.Migrations, version int64) {
	var err error
	for nextVersion := int64(0); nextVersion < version && err == nil; nextVersion++ {
		var tx *sql.Tx
		tx, err = sqlDB.Begin()
		if err != nil {
			panic(fmt.Errorf("Failed to start transaction: %s\n", err))
		}

		if err = migration.MigrateToNextVersion(tx, &nextVersion, m, databaseName); err != nil {
			if err = tx.Rollback(); err != nil {
				panic(fmt.Errorf("error while rolling back transaction: ", err))
			}
			panic(fmt.Errorf("Failed to migrate to version after rolling back"))
		}

		if err = tx.Commit(); err != nil {
			panic(fmt.Errorf("Error during transaction commit: %s\n", err))
		}
	}
}
