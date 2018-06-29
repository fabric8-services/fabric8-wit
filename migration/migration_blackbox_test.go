package migration_test

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/davecgh/go-spew/spew"
	config "github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
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
	conf       *config.Registry
	migrations migration.Migrations
	dialect    gorm.Dialect
	gormDB     *gorm.DB
	sqlDB      *sql.DB
)

func setupTest(t *testing.T) {
	var err error
	conf, err = config.Get()
	require.NoError(t, err, "failed to setup the configuration")

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
	require.NoError(t, err, "cannot connect to database: %s", databaseName)

	_, err = db.Exec("DROP DATABASE " + databaseName)
	if err != nil && !gormsupport.IsInvalidCatalogName(err) {
		require.NoError(t, err, "failed to drop database %s", databaseName)
	}

	_, err = db.Exec("CREATE DATABASE " + databaseName)
	require.NoError(t, err, "failed to create database %s", databaseName)
	migrations = migration.GetMigrations()
}

func TestMigrations(t *testing.T) {
	resource.Require(t, resource.Database)

	setupTest(t)

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
	require.NoError(t, err, "cannot connect to DB %s", databaseName)

	gormDB, err = gorm.Open("postgres", configurationString)
	defer gormDB.Close()
	require.NoError(t, err, "cannot connect to DB %s", databaseName)
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
	t.Run("TestMigration63", testMigration63)
	t.Run("TestMigration65", testMigration65)
	t.Run("TestMigration66", testMigration66)
	t.Run("TestMigration67", testMigration67)
	t.Run("TestMigration71", testMigration71)
	t.Run("TestMigration72", testMigration72)
	t.Run("TestMigration73", testMigration73)
	t.Run("TestMigration74", testMigration74)
	t.Run("TestMigration75", testMigration75)
	t.Run("TestMigration76", testMigration76)
	t.Run("TestMigration79", testMigration79)
	t.Run("TestMigration80", testMigration80)
	t.Run("TestMigration81", testMigration81)
	t.Run("TestMigration82", testMigration82)
	t.Run("TestMigration84", testMigration84)
	t.Run("TestMigration85", testMigration85)
	t.Run("TestMigration86", testMigration86)
	t.Run("TestMigration87", testMigration87SpaceTemplates) // space templates
	t.Run("TestMigration88", testMigration88TypeGroups)     // type groups
	t.Run("TestMigration89", testMigration89FixupForSpaceTemplates)
	t.Run("TestMigration90", testMigration90QueriesVersion)
	t.Run("TestMigration91", testMigration91CommentsChildComments)
	t.Run("TestMigration92", testMigration92CommentRevisionsChildComments)
	t.Run("TestMigration93", testMigration93AddCVEScanToCodebases)
	t.Run("TestMigration94", testMigration94ChangesToAgileTemplate)
	t.Run("TestMigration95", testMigration95RemoveResolutionFieldFromImpediment)
	t.Run("TestMigration96", testMigration96ChangesToAgileTemplate)
	t.Run("TestMigration97", testMigration97RemoveResolutionFieldFromImpediment)

	// Perform the migration
	err = migration.Migrate(sqlDB, databaseName)
	require.NoError(t, err, "failed to execute database migration")
}

func testMigration44(t *testing.T) {
	var err error
	m := migrations[:initialMigratedVersion]
	for nextVersion := int64(0); nextVersion < int64(len(m)) && err == nil; nextVersion++ {
		var tx *sql.Tx
		tx, err = sqlDB.Begin()
		require.NoError(t, err, "failed to start transaction")

		if err = migration.MigrateToNextVersion(tx, &nextVersion, m, databaseName); err != nil {
			t.Errorf("failed to migrate to version %d: %s\n", nextVersion, err)
			errRollback := tx.Rollback()
			require.NoError(t, errRollback, "error while rolling back transaction")
			require.NoError(t, err, "failed to migrate to version after rolling back")
		}

		err = tx.Commit()
		require.NoError(t, err, "error during transaction commit")
	}
}

func testMigration45(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+1)], (initialMigratedVersion + 1))

	assert.True(t, gormDB.HasTable("work_items"))
	assert.True(t, dialect.HasColumn("work_items", "execution_order"))
	assert.True(t, dialect.HasIndex("work_items", "order_index"))

	assert.Nil(t, runSQLscript(sqlDB, "045-update-work-items.sql"))
}

func testMigration46(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+2)], (initialMigratedVersion + 2))

	assert.True(t, gormDB.HasTable("oauth_state_references"))
	assert.True(t, dialect.HasColumn("oauth_state_references", "referrer"))
	assert.True(t, dialect.HasColumn("oauth_state_references", "id"))

	assert.Nil(t, runSQLscript(sqlDB, "046-insert-oauth-states.sql"))
}

func testMigration47(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+3)], (initialMigratedVersion + 3))

	assert.True(t, gormDB.HasTable("codebases"))
	assert.True(t, dialect.HasColumn("codebases", "type"))
	assert.True(t, dialect.HasColumn("codebases", "url"))
	assert.True(t, dialect.HasColumn("codebases", "space_id"))
	assert.True(t, dialect.HasIndex("codebases", "ix_codebases_space_id"))

	assert.Nil(t, runSQLscript(sqlDB, "047-insert-codebases.sql"))
}

func testMigration48(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+4)], (initialMigratedVersion + 4))

	assert.True(t, dialect.HasIndex("iterations", "ix_name"))

	// This script execution has to fail
	assert.NotNil(t, runSQLscript(sqlDB, "048-unique-idx-failed-insert.sql"))
}

func testMigration49(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+5)], (initialMigratedVersion + 5))

	assert.True(t, dialect.HasIndex("areas", "ix_area_name"))

	// Tests that migration 49 set the system.area to the work_items and its value
	// is 71171e90-6d35-498f-a6a7-2083b5267c18
	rows, err := sqlDB.Query("SELECT count(*), fields->>'system.area' FROM work_items where fields != '{}' GROUP BY fields")
	require.NoError(t, err)
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
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+6)], (initialMigratedVersion + 6))

	assert.True(t, dialect.HasColumn("users", "company"))

	assert.Nil(t, runSQLscript(sqlDB, "050-users-add-column-company.sql"))
}
func testMigration51(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+7)], (initialMigratedVersion + 7))

	assert.True(t, dialect.HasIndex("work_item_link_types", "work_item_link_types_name_idx"))
}

func testMigration52(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+8)], (initialMigratedVersion + 8))

	assert.True(t, dialect.HasIndex("spaces", "spaces_name_idx"))
}

func testMigration53(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+9)], (initialMigratedVersion + 9))
	require.True(t, dialect.HasColumn("identities", "registration_completed"))

	// add new rows and check if the new column has the default value
	assert.Nil(t, runSQLscript(sqlDB, "053-edit-username.sql"))

	// check if ALL the existing rows & new rows have the default value
	rows, err := sqlDB.Query("SELECT registration_completed FROM identities")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var registration_completed bool
		err = rows.Scan(&registration_completed)
		assert.False(t, registration_completed)
	}
}

func testMigration54(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+10)], (initialMigratedVersion + 10))

	assert.True(t, dialect.HasColumn("codebases", "stack_id"))

	assert.Nil(t, runSQLscript(sqlDB, "054-add-stackid-to-codebase.sql"))
}

func testMigration55(t *testing.T) {
	// migrate to previous version
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+10)], (initialMigratedVersion + 10))
	// fill DB with invalid data (ie, missing root area)
	assert.Nil(t, runSQLscript(sqlDB, "055-assign-root-area-if-missing.sql"))
	// then apply the fix
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+11)], (initialMigratedVersion + 11))
	// and verify that the root area is available
	rows, err := sqlDB.Query("select fields->>'system.area' from work_items where id = 12345")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var rootArea string
		err = rows.Scan(&rootArea)
		assert.NotEmpty(t, rootArea)
	}
}

func testMigration56(t *testing.T) {
	// migrate to previous version
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+11)], (initialMigratedVersion + 11))
	// fill DB with invalid data (ie, missing root area)
	assert.Nil(t, runSQLscript(sqlDB, "056-assign-root-iteration-if-missing.sql"))
	// then apply the fix
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+12)], (initialMigratedVersion + 12))
	// and verify that the root area is available
	rows, err := sqlDB.Query("select fields->>'system.iteration' from work_items where id = 12346")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var rootIteration string
		err = rows.Scan(&rootIteration)
		assert.NotEmpty(t, rootIteration)
	}
}

func testMigration57(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+13)], (initialMigratedVersion + 13))

	assert.True(t, dialect.HasColumn("codebases", "last_used_workspace"))

	assert.Nil(t, runSQLscript(sqlDB, "057-add-last-used-workspace-to-codebase.sql"))
}

func testMigration60(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+16)], (initialMigratedVersion + 16))

	assert.True(t, dialect.HasIndex("identities", "idx_identities_username"))
}

func testMigration61(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+16)], (initialMigratedVersion + 16))

	// Add on purpose a duplicate to verify that we can successfully run this migration
	assert.Nil(t, runSQLscript(sqlDB, "061-add-duplicate-space-owner-name.sql"))

	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+17)], (initialMigratedVersion + 17))

	assert.True(t, dialect.HasIndex("spaces", "spaces_name_idx"))

	rows, err := sqlDB.Query("SELECT COUNT(*) FROM spaces WHERE name='test.space.one-renamed'")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var count int
		err = rows.Scan(&count)
		assert.True(t, count == 1)
	}

}

func testMigration63(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+19)], (initialMigratedVersion + 19))
	assert.Nil(t, runSQLscript(sqlDB, "063-workitem-related-changes.sql"))
	var createdAt time.Time
	var deletedAt time.Time
	var relationshipsChangeddAt time.Time

	// comments
	// work item 62001 was commented
	row := sqlDB.QueryRow("SELECT wi.relationships_changed_at, c.created_at FROM work_items wi left join comments c on c.parent_id::bigint = wi.id where wi.id = 62001")
	err := row.Scan(&relationshipsChangeddAt, &createdAt)
	require.NoError(t, err)
	assert.Equal(t, relationshipsChangeddAt, createdAt)
	// work item 62003 was commented, then the comment was (soft) deleted
	row = sqlDB.QueryRow("SELECT wi.relationships_changed_at, c.deleted_at FROM work_items wi left join comments c on c.parent_id::bigint = wi.id where wi.id = 62003")
	err = row.Scan(&relationshipsChangeddAt, &deletedAt)
	require.NoError(t, err)
	assert.Equal(t, relationshipsChangeddAt, deletedAt)

	// links
	// work items 62004 and 62005 were linked together
	row = sqlDB.QueryRow("SELECT wi.relationships_changed_at, wil.created_at FROM work_items wi left join work_item_links wil on wil.source_id = wi.id where wi.id = 62004")
	err = row.Scan(&relationshipsChangeddAt, &createdAt)
	require.NoError(t, err)
	assert.Equal(t, relationshipsChangeddAt, createdAt)
	row = sqlDB.QueryRow("SELECT wi.relationships_changed_at, wil.created_at FROM work_items wi left join work_item_links wil on wil.target_id = wi.id where wi.id = 62005")
	err = row.Scan(&relationshipsChangeddAt, &createdAt)
	require.NoError(t, err)
	assert.Equal(t, relationshipsChangeddAt, createdAt)
	// work items 62008 and 62009 were linked together, but then the link was deleted
	row = sqlDB.QueryRow("SELECT wi.relationships_changed_at, wil.deleted_at FROM work_items wi left join work_item_links wil on wil.source_id = wi.id where wi.id = 62008")
	err = row.Scan(&relationshipsChangeddAt, &deletedAt)
	require.NoError(t, err)
	assert.Equal(t, relationshipsChangeddAt, deletedAt)
	row = sqlDB.QueryRow("SELECT wi.relationships_changed_at, wil.deleted_at FROM work_items wi left join work_item_links wil on wil.target_id = wi.id where wi.id = 62009")
	err = row.Scan(&relationshipsChangeddAt, &deletedAt)
	require.NoError(t, err)
	assert.Equal(t, relationshipsChangeddAt, deletedAt)
}

func testMigration65(t *testing.T) {
	// migrate to previous version
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+20)], (initialMigratedVersion + 20))
	// fill DB with data (ie, work items, links, comments, etc on different spaces)
	assert.Nil(t, runSQLscript(sqlDB, "065-workitem-id-unique-per-space.sql"))
	// then apply the change
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+21)], (initialMigratedVersion + 21))
	// and verify that the work item id sequence table is filled as expected
	type WorkItemSequence struct {
		SpaceID    uuid.UUID `sql:"type:UUID"`
		CurrentVal int
	}
	space1, err := uuid.FromString("11111111-0000-0000-0000-000000000000")
	require.NoError(t, err)
	space2, err := uuid.FromString("22222222-0000-0000-0000-000000000000")
	require.NoError(t, err)
	expectations := make(map[uuid.UUID]int)
	expectations[space1] = 12348
	expectations[space2] = 12350
	for spaceID, expectedCurrentVal := range expectations {
		var currentVal int
		stmt, err := sqlDB.Prepare("select current_val from work_item_number_sequences where space_id = $1")
		require.NoError(t, err)
		err = stmt.QueryRow(spaceID.String()).Scan(&currentVal)
		require.NoError(t, err)
		require.NotNil(t, currentVal)
		assert.Equal(t, expectedCurrentVal, currentVal)
	}
}
func testMigration66(t *testing.T) {
	// migrate to previous version
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+21)], (initialMigratedVersion + 21))
	// fill DB with data (ie, work items, links, comments, etc on different spaces)
	assert.Nil(t, runSQLscript(sqlDB, "066-work_item_links_data_integrity.sql"))
	// then apply the change
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+22)], (initialMigratedVersion + 22))
	var workitemLinkId string
	// verify that the first record was not removed
	err := sqlDB.QueryRow("select id from work_item_links where id = '00000066-0000-0000-0000-000000000001'").Scan(&workitemLinkId)
	require.NoError(t, err)
	// verify that the 3 other records where deleted (because of invalid/null data)
	stmt, err := sqlDB.Prepare("select id from work_item_links where id = $1")
	require.NoError(t, err)
	for _, id := range []string{"00000066-0000-0000-0000-000000000002", "00000066-0000-0000-0000-000000000003", "00000066-0000-0000-0000-000000000004"} {
		err = stmt.QueryRow(id).Scan(&workitemLinkId)
		assert.Equal(t, sql.ErrNoRows, err, "link with id='%v' was not removed", id)
	}

}

func testMigration67(t *testing.T) {
	// migrate to previous version
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+22)], (initialMigratedVersion + 22))
	// fill DB with data
	assert.Nil(t, runSQLscript(sqlDB, "067-comment-parentid-uuid.sql"))
	// then apply the change
	migrateToVersion(t, sqlDB, migrations[:(initialMigratedVersion+23)], (initialMigratedVersion + 23))
	// verify the data
	var parentID uuid.UUID
	stmt, err := sqlDB.Prepare("select parent_id from comments where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("00000067-0000-0000-0000-000000000000").Scan(&parentID)
	require.NoError(t, err)
	assert.NotNil(t, parentID)
	stmt, err = sqlDB.Prepare("select comment_parent_id from comment_revisions where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("00000067-0000-0000-0000-000000000000").Scan(&parentID)
	require.NoError(t, err)
	assert.NotNil(t, parentID)
}

func testMigration71(t *testing.T) {
	// migrate to version
	migrateToVersion(t, sqlDB, migrations[:72], 72)
	// fill DB with data
	assert.Nil(t, runSQLscript(sqlDB, "071-iteration-related-changes.sql"))
	// verify the data
	var updatedAt *time.Time
	var deletedAt *time.Time
	var relationshipsChangedAt *time.Time

	// verify work item 1 linked to iteration 1
	stmt, err := sqlDB.Prepare("select updated_at from work_items where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("11111111-7171-0000-0000-000000000000").Scan(&updatedAt)
	require.NoError(t, err)
	require.NotNil(t, updatedAt)
	stmt, err = sqlDB.Prepare("select relationships_changed_at from iterations where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("11111111-7171-0000-0000-000000000000").Scan(&relationshipsChangedAt)
	require.NoError(t, err)
	require.NotNil(t, relationshipsChangedAt)
	assert.Equal(t, *updatedAt, *relationshipsChangedAt)
	// verify work item 2 linked to iteration 2 then iteration 3
	stmt, err = sqlDB.Prepare("select updated_at from work_items where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("22222222-7171-0000-0000-000000000000").Scan(&updatedAt)
	require.NoError(t, err)
	require.NotNil(t, updatedAt)
	stmt, err = sqlDB.Prepare("select relationships_changed_at from iterations where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("22222222-7171-0000-0000-000000000000").Scan(&relationshipsChangedAt)
	require.NoError(t, err)
	require.NotNil(t, relationshipsChangedAt)
	assert.Equal(t, *updatedAt, *relationshipsChangedAt)
	stmt, err = sqlDB.Prepare("select relationships_changed_at from iterations where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("33333333-7171-0000-0000-000000000000").Scan(&relationshipsChangedAt)
	require.NoError(t, err)
	require.NotNil(t, relationshipsChangedAt)
	assert.Equal(t, *updatedAt, *relationshipsChangedAt)
	// verify work item 3 linked to iteration 4
	stmt, err = sqlDB.Prepare("select deleted_at from work_items where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("33333333-7171-0000-0000-000000000000").Scan(&deletedAt)
	require.NoError(t, err)
	require.NotNil(t, deletedAt)
	stmt, err = sqlDB.Prepare("select relationships_changed_at from iterations where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("44444444-7171-0000-0000-000000000000").Scan(&relationshipsChangedAt)
	require.NoError(t, err)
	require.NotNil(t, relationshipsChangedAt)
	assert.Equal(t, *deletedAt, *relationshipsChangedAt)
	// verify work item 4 linked to iteration 5
	// we will expect 1hr earlier for the last relationship change
	stmt, err = sqlDB.Prepare("select (updated_at - interval '1 hour') from work_items where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("44444444-7171-0000-0000-000000000000").Scan(&updatedAt)
	require.NoError(t, err)
	require.NotNil(t, updatedAt)
	stmt, err = sqlDB.Prepare("select relationships_changed_at from iterations where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("55555555-7171-0000-0000-000000000000").Scan(&relationshipsChangedAt)
	require.NoError(t, err)
	require.NotNil(t, relationshipsChangedAt)
	assert.Equal(t, updatedAt.String(), relationshipsChangedAt.String())
}

func testMigration72(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:73], 73)
	assert.True(t, dialect.HasColumn("iterations", "user_active"))
}

func testMigration73(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:74], 74)
	assert.True(t, dialect.HasTable("labels"))
	assert.True(t, dialect.HasIndex("labels", "label_name_idx"))

	// These script execution has to fail
	assert.NotNil(t, runSQLscript(sqlDB, "073-label-empty-name.sql"))
	assert.NotNil(t, runSQLscript(sqlDB, "073-label-same-name.sql"))
	assert.NotNil(t, runSQLscript(sqlDB, "073-label-color-code.sql"))
	assert.NotNil(t, runSQLscript(sqlDB, "073-label-color-code2.sql"))
}

func testMigration74(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:75], 75)
	assert.True(t, dialect.HasColumn("labels", "border_color"))
}

func testMigration75(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:76], 76)
	assert.True(t, dialect.HasIndex("labels", "labels_name_space_id_unique_idx"))
}

func testMigration76(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:77], 77)
	assert.False(t, dialect.HasTable("space_resources"))
	assert.False(t, dialect.HasTable("oauth_state_references"))
}

func testMigration79(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:80], 80)
	count := -1
	gormDB.Table("work_items").Where(`Fields->>'system.labels'='[]'`).Count(&count)
	assert.Equal(t, 0, count)

	gormDB.Table("work_items").Where(`Fields->>'system.assignees'='[]'`).Count(&count)
	assert.Equal(t, 0, count)
}

func testMigration80(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:80], 80)
	require.Nil(t, runSQLscript(sqlDB, "080-old-link-type-relics.sql",
		space.SystemSpace.String(),
		link.SystemWorkItemLinkTypeBugBlockerID.String(),
		link.SystemWorkItemLinkPlannerItemRelatedID.String(),
		link.SystemWorkItemLinkTypeParentChildID.String(),
		link.SystemWorkItemLinkCategorySystemID.String(),
		link.SystemWorkItemLinkCategoryUserID.String(),
	))

	// When we migrate the DB to version 80 all but the known link types and
	// categories should be gone, which is what we test below.
	migrateToVersion(t, sqlDB, migrations[:81], 81)

	t.Run("only known link types exist", func(t *testing.T) {
		// Make sure no other link type other than the known ones are present
		linkTypesToBeFound := map[uuid.UUID]struct{}{
			link.SystemWorkItemLinkTypeBugBlockerID:     {},
			link.SystemWorkItemLinkPlannerItemRelatedID: {},
			link.SystemWorkItemLinkTypeParentChildID:    {},
		}
		rows, err := sqlDB.Query("SELECT id FROM work_item_link_types")
		require.NoError(t, err)
		for rows.Next() {
			var id uuid.UUID
			err = rows.Scan(&id)
			require.NoError(t, err, "failed to scan link type: %+v", err)
			_, ok := linkTypesToBeFound[id]
			require.True(t, ok, "link type should not exist: %s", id)
			delete(linkTypesToBeFound, id)
		}
		require.Empty(t, linkTypesToBeFound, "not all link types have been found: %+v", spew.Sdump(linkTypesToBeFound))
	})

	t.Run("only known link categories exist", func(t *testing.T) {
		// Make sure no other link categories other than the known ones are present
		linkCategoriesToBeFound := map[uuid.UUID]struct{}{
			link.SystemWorkItemLinkCategorySystemID: {},
			link.SystemWorkItemLinkCategoryUserID:   {},
		}
		rows, err := sqlDB.Query("SELECT id FROM work_item_link_categories")
		require.NoError(t, err)
		for rows.Next() {
			var id uuid.UUID
			err = rows.Scan(&id)
			require.NoError(t, err, "failed to scan link category: %+v", err)
			_, ok := linkCategoriesToBeFound[id]
			require.True(t, ok, "link category should not exist: %s", id)
			delete(linkCategoriesToBeFound, id)
		}
		require.Empty(t, linkCategoriesToBeFound, "not all link categories have been found: %+v", spew.Sdump(linkCategoriesToBeFound))
	})
}

func testMigration81(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:82], 82)
	assert.True(t, dialect.HasTable("queries"))
	assert.True(t, dialect.HasIndex("queries", "query_creator_idx"))

	// These script execution has to fail
	assert.NotNil(t, runSQLscript(sqlDB, "081-query-conflict.sql"))
	assert.NotNil(t, runSQLscript(sqlDB, "081-query-null-title.sql"))
	assert.NotNil(t, runSQLscript(sqlDB, "081-query-empty-title.sql"))
	assert.NotNil(t, runSQLscript(sqlDB, "081-query-no-creator.sql"))
}

func testMigration82(t *testing.T) {
	// migrate to version
	migrateToVersion(t, sqlDB, migrations[:83], 83)
	// fill DB with data
	assert.Nil(t, runSQLscript(sqlDB, "082-iteration-related-changes.sql"))
	// verify the data
	var updatedAt *time.Time
	var deletedAt *time.Time
	var relationshipsChangedAt *time.Time

	// verify work item 1 linked to iteration 1
	stmt, err := sqlDB.Prepare("select updated_at from work_items where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("11111111-8282-0000-0000-000000000000").Scan(&updatedAt)
	require.NoError(t, err)
	require.NotNil(t, updatedAt)
	stmt, err = sqlDB.Prepare("select relationships_changed_at from iterations where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("11111111-8282-0000-0000-000000000000").Scan(&relationshipsChangedAt)
	require.NoError(t, err)
	require.NotNil(t, relationshipsChangedAt)
	assert.Equal(t, *updatedAt, *relationshipsChangedAt)
	// verify work item 2 linked to iteration 2 then iteration 3
	stmt, err = sqlDB.Prepare("select updated_at from work_items where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("22222222-8282-0000-0000-000000000000").Scan(&updatedAt)
	require.NoError(t, err)
	require.NotNil(t, updatedAt)
	stmt, err = sqlDB.Prepare("select relationships_changed_at from iterations where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("22222222-8282-0000-0000-000000000000").Scan(&relationshipsChangedAt)
	require.NoError(t, err)
	require.NotNil(t, relationshipsChangedAt)
	assert.Equal(t, *updatedAt, *relationshipsChangedAt)
	stmt, err = sqlDB.Prepare("select relationships_changed_at from iterations where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("33333333-8282-0000-0000-000000000000").Scan(&relationshipsChangedAt)
	require.NoError(t, err)
	require.NotNil(t, relationshipsChangedAt)
	assert.Equal(t, *updatedAt, *relationshipsChangedAt)
	// verify work item 3 linked to iteration 4
	stmt, err = sqlDB.Prepare("select deleted_at from work_items where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("33333333-8282-0000-0000-000000000000").Scan(&deletedAt)
	require.NoError(t, err)
	require.NotNil(t, deletedAt)
	stmt, err = sqlDB.Prepare("select relationships_changed_at from iterations where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("44444444-8282-0000-0000-000000000000").Scan(&relationshipsChangedAt)
	require.NoError(t, err)
	require.NotNil(t, relationshipsChangedAt)
	assert.Equal(t, *deletedAt, *relationshipsChangedAt)
	// verify work item 4 linked to iteration 5
	// we will expect 1hr earlier for the last relationship change
	stmt, err = sqlDB.Prepare("select (updated_at - interval '1 hour') from work_items where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("44444444-8282-0000-0000-000000000000").Scan(&updatedAt)
	require.NoError(t, err)
	require.NotNil(t, updatedAt)
	stmt, err = sqlDB.Prepare("select relationships_changed_at from iterations where id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow("55555555-8282-0000-0000-000000000000").Scan(&relationshipsChangedAt)
	require.NoError(t, err)
	require.NotNil(t, relationshipsChangedAt)
	assert.Equal(t, updatedAt.String(), relationshipsChangedAt.String())
}

func testMigration84(t *testing.T) {
	// migrate to version so that we create duplicate data
	migrateToVersion(t, sqlDB, migrations[:84], 84)

	// create dummy space and add entry in codebases that are duplicate
	assert.Nil(t, runSQLscript(sqlDB, "084-codebases-spaceid-url-idx-setup.sql"))

	// migrate to current version, which applies unique index
	// and removes duplicate
	migrateToVersion(t, sqlDB, migrations[:85], 85)

	// try to add duplicate entry, which should fail
	assert.NotNil(t, runSQLscript(sqlDB, "084-codebases-spaceid-url-idx-violate.sql"))

	// see that the existing space is not the deleted one but the one that is
	// available in the valid one
	assert.Nil(t, runSQLscript(sqlDB, "084-codebases-spaceid-url-idx-test.sql"))

	// cleanup
	assert.Nil(t, runSQLscript(sqlDB, "084-codebases-spaceid-url-idx-cleanup.sql"))
}

func testMigration85(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:85], 85)

	expectWorkItemFieldsToBe := func(t *testing.T, witID uuid.UUID, expectedFields string) {
		row := sqlDB.QueryRow("SELECT fields FROM work_items WHERE id = $1", witID.String())
		require.NotNil(t, row)
		var actualFields string
		err := row.Scan(&actualFields)
		require.NoError(t, err)
		require.Equal(t, expectedFields, actualFields)
	}

	// create two work items, one with the 'system.number' field and one without
	// and check that they've been created as expected.
	assert.Nil(t, runSQLscript(sqlDB, "085-delete-system.number-json-field.sql"))
	expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("27adc1a2-1ded-43b8-a125-12777139496c"), `{"system.title": "Work item 1", "system.number": 1234}`)
	expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("c106c056-2fec-4e56-83f0-cac31bb7ac1f"), `{"system.title": "Work item 2"}`)

	// migrate to current version, which removes the 'system.number' field from
	// work items and check that no work item has it.
	migrateToVersion(t, sqlDB, migrations[:86], 86)
	expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("27adc1a2-1ded-43b8-a125-12777139496c"), `{"system.title": "Work item 1"}`)
	expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("c106c056-2fec-4e56-83f0-cac31bb7ac1f"), `{"system.title": "Work item 2"}`)
}

func testMigration86(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:87], 87)
	require.True(t, dialect.HasColumn("work_item_types", "can_construct"))
}

func testMigration87SpaceTemplates(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:88], 88)
	assert.True(t, dialect.HasTable("space_templates"))
	assert.True(t, dialect.HasColumn("spaces", "space_template_id"))
	assert.True(t, dialect.HasColumn("work_item_types", "space_template_id"))
	assert.True(t, dialect.HasColumn("work_item_link_types", "space_template_id"))
}

func testMigration88TypeGroups(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:89], 89)
	assert.True(t, dialect.HasTable("work_item_type_groups"))
	assert.True(t, dialect.HasTable("work_item_type_group_members"))
	assert.True(t, dialect.HasTable("work_item_child_types"))
}

func testMigration89FixupForSpaceTemplates(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:89], 89)

	// Before this change, the planner item type was assigned to the legacy template
	stmt, err := sqlDB.Prepare("SELECT space_template_id FROM work_item_types WHERE id = $1")
	require.NoError(t, err)
	var stID string
	err = stmt.QueryRow(workitem.SystemPlannerItem).Scan(&stID)
	require.NoError(t, err)
	require.Equal(t, spacetemplate.SystemLegacyTemplateID.String(), stID)

	// Before this change, all link types where assigned to the legacy template
	stmt, err = sqlDB.Prepare("SELECT count(*) FROM work_item_link_types WHERE space_template_id = $1")
	require.NoError(t, err)
	var cnt int
	err = stmt.QueryRow(spacetemplate.SystemLegacyTemplateID).Scan(&cnt)
	require.NoError(t, err)
	require.Equal(t, 3, cnt)

	migrateToVersion(t, sqlDB, migrations[:90], 90)

	// After this change, the planner item type is assigned to the base template
	stmt, err = sqlDB.Prepare("SELECT space_template_id FROM work_item_types WHERE id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow(workitem.SystemPlannerItem).Scan(&stID)
	require.NoError(t, err)
	require.Equal(t, spacetemplate.SystemBaseTemplateID.String(), stID)

	// After this change, all link types are assigned to the base template
	stmt, err = sqlDB.Prepare("SELECT count(*) FROM work_item_link_types WHERE space_template_id = $1")
	require.NoError(t, err)
	err = stmt.QueryRow(spacetemplate.SystemBaseTemplateID).Scan(&cnt)
	require.NoError(t, err)
	require.Equal(t, 3, cnt)
}

func testMigration90QueriesVersion(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:91], 91)
	assert.True(t, dialect.HasColumn("queries", "version"))
}

func testMigration91CommentsChildComments(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:92], 92)
	assert.True(t, dialect.HasColumn("comments", "parent_comment_id"))
}

func testMigration92CommentRevisionsChildComments(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:93], 93)
	assert.True(t, dialect.HasColumn("comment_revisions", "comment_parent_comment_id"))
}

func testMigration93AddCVEScanToCodebases(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:93], 93)

	// setup the space needed for running all the tests
	require.Nil(t, runSQLscript(sqlDB, "093-codebases-add-cve-scan-setup.sql"))

	// an entry of codebase without an cve_scan field
	require.Nil(t, runSQLscript(sqlDB, "093-codebases-add-cve-scan-without.sql"))

	// an entry of codebase with an cve_scan field
	require.NotNil(t, runSQLscript(sqlDB, "093-codebases-add-cve-scan-with.sql"))

	// migrate to the current version
	migrateToVersion(t, sqlDB, migrations[:94], 94)

	// an entry of codebase with an cve_scan field
	require.Nil(t, runSQLscript(sqlDB, "093-codebases-add-cve-scan-with2.sql"))

	rows, err := sqlDB.Query("SELECT * FROM codebases WHERE id='1975cf7a-e3fb-4ca6-b368-bc8ee66fccee' AND cve_scan='t';")
	require.NoError(t, err)
	require.True(t, rows.Next(), "no row found with cve_scan=true")

	rows, err = sqlDB.Query("SELECT * FROM codebases WHERE id='cdc691c8-534d-48b3-9f72-7836bc5b9188' AND cve_scan='f';")
	require.NoError(t, err)
	require.True(t, rows.Next(), "no row found with cve_scan=false")

	// do the cleanup
	require.Nil(t, runSQLscript(sqlDB, "093-codebases-add-cve-scan-cleanup.sql"))
}

func testMigration94ChangesToAgileTemplate(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:94], 94)

	expectWorkItemTypeFieldsToBe := func(t *testing.T, witID uuid.UUID, expectedFields string) {
		row := sqlDB.QueryRow("SELECT fields FROM work_item_types WHERE id = $1", witID.String())
		require.NotNil(t, row)
		var actualFields string
		err := row.Scan(&actualFields)
		require.NoError(t, err)

		actualMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(actualFields), &actualMap)
		require.NoError(t, err)
		expectedMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(expectedFields), &expectedMap)
		require.NoError(t, err)

		require.Equal(t, actualMap, expectedMap)
	}

	expectWorkItemFieldsToBe := func(t *testing.T, wiID uuid.UUID, expectedFields string) {
		row := sqlDB.QueryRow("SELECT fields FROM work_items WHERE id = $1", wiID.String())
		require.NotNil(t, row)
		var actualFields string
		err := row.Scan(&actualFields)
		require.NoError(t, err)

		actualMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(actualFields), &actualMap)
		require.NoError(t, err)
		expectedMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(expectedFields), &expectedMap)
		require.NoError(t, err)

		require.Equal(t, actualMap, expectedMap)
	}

	// create two work items, one with the fields and one without
	// and check that they've been created as expected.
	require.Nil(t, runSQLscript(sqlDB, "094-changes-to-agile-template-test.sql"))

	t.Run("before", func(t *testing.T) {
		t.Run("theme", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("5182fc8c-b1d6-4c3d-83ca-6a3c781fa18a"), `{"system.area": {"Type": {"Kind": "area"}, "Label": "Area", "Required": false, "Description": "The area to which the work item belongs"}, "system.order": {"Type": {"Kind": "float"}, "Label": "Execution Order", "Required": false, "Description": "Execution Order of the workitem."}, "system.state": {"Type": {"Kind": "enum", "Values": ["new", "open", "in progress", "resolved", "closed"], "BaseType": {"Kind": "string"}}, "Label": "State", "Required": true, "Description": "The state of the work item"}, "system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}, "system.labels": {"Type": {"Kind": "list", "ComponentType": {"Kind": "label"}}, "Label": "Labels", "Required": false, "Description": "List of labels attached to the work item"}, "business_value": {"Type": {"Kind": "integer"}, "Label": "Business Value", "Required": false, "Description": "The business value of this work item."}, "effort": {"Type": {"Kind": "float"}, "Label": "Effort", "Required": false, "Description": "The effort that was given to this workitem within its space."}, "time_criticality": {"Type": {"Kind": "float"}, "Label": "Time Criticality", "Required": false, "Description": "The time criticality that was given to this workitem within its space."}, "system.creator": {"Type": {"Kind": "user"}, "Label": "Creator", "Required": true, "Description": "The user that created the work item"}, "system.codebase": {"Type": {"Kind": "codebase"}, "Label": "Codebase", "Required": false, "Description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"Type": {"Kind": "list", "ComponentType": {"Kind": "user"}}, "Label": "Assignees", "Required": false, "Description": "The users that are assigned to the work item"}, "system.iteration": {"Type": {"Kind": "iteration"}, "Label": "Iteration", "Required": false, "Description": "The iteration to which the work item belongs"}, "system.created_at": {"Type": {"Kind": "instant"}, "Label": "Created at", "Required": false, "Description": "The date and time when the work item was created"}, "system.updated_at": {"Type": {"Kind": "instant"}, "Label": "Updated at", "Required": false, "Description": "The date and time when the work item was last updated"}, "system.description": {"Type": {"Kind": "markup"}, "Label": "Description", "Required": false, "Description": "A descriptive text of the work item"}, "system.remote_item_id": {"Type": {"Kind": "string"}, "Label": "Remote item", "Required": false, "Description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("cf84c888-ac28-493d-a0cd-978b78568040"), `{"system.title": "Work item 1", "effort": 12.34, "business_value": 1234, "time_criticality": 56.78}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("8bbb542c-4f5c-44bb-9272-e1a8f24e6eb2"), `{"system.title": "Work item 2"}`)
		})
		t.Run("epic", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("2c169431-a55d-49eb-af74-cc19e895356f"), `{"system.area": {"Type": {"Kind": "area"}, "Label": "Area", "Required": false, "Description": "The area to which the work item belongs"}, "system.order": {"Type": {"Kind": "float"}, "Label": "Execution Order", "Required": false, "Description": "Execution Order of the workitem."}, "system.state": {"Type": {"Kind": "enum", "Values": ["new", "open", "in progress", "resolved", "closed"], "BaseType": {"Kind": "string"}}, "Label": "State", "Required": true, "Description": "The state of the work item"}, "system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}, "system.labels": {"Type": {"Kind": "list", "ComponentType": {"Kind": "label"}}, "Label": "Labels", "Required": false, "Description": "List of labels attached to the work item"}, "component": {"Type": {"Kind": "string"}, "Label": "Component", "Required": false, "Description": "The component value of this work item."}, "business_value": {"Type": {"Kind": "integer"}, "Label": "Business Value", "Required": false, "Description": "The business value of this work item."}, "effort": {"Type": {"Kind": "float"}, "Label": "Effort", "Required": false, "Description": "The effort that was given to this workitem within its space."}, "time_criticality": {"Type": {"Kind": "float"}, "Label": "Time Criticality", "Required": false, "Description": "The time criticality that was given to this workitem within its space."}, "system.creator": {"Type": {"Kind": "user"}, "Label": "Creator", "Required": true, "Description": "The user that created the work item"}, "system.codebase": {"Type": {"Kind": "codebase"}, "Label": "Codebase", "Required": false, "Description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"Type": {"Kind": "list", "ComponentType": {"Kind": "user"}}, "Label": "Assignees", "Required": false, "Description": "The users that are assigned to the work item"}, "system.iteration": {"Type": {"Kind": "iteration"}, "Label": "Iteration", "Required": false, "Description": "The iteration to which the work item belongs"}, "system.created_at": {"Type": {"Kind": "instant"}, "Label": "Created at", "Required": false, "Description": "The date and time when the work item was created"}, "system.updated_at": {"Type": {"Kind": "instant"}, "Label": "Updated at", "Required": false, "Description": "The date and time when the work item was last updated"}, "system.description": {"Type": {"Kind": "markup"}, "Label": "Description", "Required": false, "Description": "A descriptive text of the work item"}, "system.remote_item_id": {"Type": {"Kind": "string"}, "Label": "Remote item", "Required": false, "Description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("4aebb314-a8c1-4e9c-96b6-074769d16934"), `{"system.title": "Work item 3", "effort": 12.34, "business_value": 1234, "time_criticality": 56.78, "component": "Component 1"}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("9c53fb2b-c6af-48a1-bef1-6fa547ea72fa"), `{"system.title": "Work item 4"}`)
		})
		t.Run("story", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("6ff83406-caa7-47a9-9200-4ca796be11bb"), `{"system.area": {"Type": {"Kind": "area"}, "Label": "Area", "Required": false, "Description": "The area to which the work item belongs"}, "system.order": {"Type": {"Kind": "float"}, "Label": "Execution Order", "Required": false, "Description": "Execution Order of the workitem."}, "system.state": {"Type": {"Kind": "enum", "Values": ["new", "open", "in progress", "resolved", "closed"], "BaseType": {"Kind": "string"}}, "Label": "State", "Required": true, "Description": "The state of the work item"}, "system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}, "system.labels": {"Type": {"Kind": "list", "ComponentType": {"Kind": "label"}}, "Label": "Labels", "Required": false, "Description": "List of labels attached to the work item"}, "effort": {"Type": {"Kind": "float"}, "Label": "Effort", "Required": false, "Description": "The effort that was given to this workitem within its space."}, "system.creator": {"Type": {"Kind": "user"}, "Label": "Creator", "Required": true, "Description": "The user that created the work item"}, "system.codebase": {"Type": {"Kind": "codebase"}, "Label": "Codebase", "Required": false, "Description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"Type": {"Kind": "list", "ComponentType": {"Kind": "user"}}, "Label": "Assignees", "Required": false, "Description": "The users that are assigned to the work item"}, "system.iteration": {"Type": {"Kind": "iteration"}, "Label": "Iteration", "Required": false, "Description": "The iteration to which the work item belongs"}, "system.created_at": {"Type": {"Kind": "instant"}, "Label": "Created at", "Required": false, "Description": "The date and time when the work item was created"}, "system.updated_at": {"Type": {"Kind": "instant"}, "Label": "Updated at", "Required": false, "Description": "The date and time when the work item was last updated"}, "system.description": {"Type": {"Kind": "markup"}, "Label": "Description", "Required": false, "Description": "A descriptive text of the work item"}, "system.remote_item_id": {"Type": {"Kind": "string"}, "Label": "Remote item", "Required": false, "Description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("68f83154-8d76-49c1-8be0-063ce90f803d"), `{"system.title": "Work item 5", "effort": 12.34}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("17e2081f-812d-4f4e-9c51-c537406bd1d8"), `{"system.title": "Work item 6"}`)
		})
	})

	// Migrate to the current version. That removes fields from each work item
	// and its work item type. We then check that those fields have actually
	// been removed from the work items and their types.
	migrateToVersion(t, sqlDB, migrations[:95], 95)

	t.Run("after", func(t *testing.T) {
		t.Run("theme", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("5182fc8c-b1d6-4c3d-83ca-6a3c781fa18a"), `{"system.area": {"Type": {"Kind": "area"}, "Label": "Area", "Required": false, "Description": "The area to which the work item belongs"}, "system.order": {"Type": {"Kind": "float"}, "Label": "Execution Order", "Required": false, "Description": "Execution Order of the workitem."}, "system.state": {"Type": {"Kind": "enum", "Values": ["new", "open", "in progress", "resolved", "closed"], "BaseType": {"Kind": "string"}}, "Label": "State", "Required": true, "Description": "The state of the work item"}, "system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}, "system.labels": {"Type": {"Kind": "list", "ComponentType": {"Kind": "label"}}, "Label": "Labels", "Required": false, "Description": "List of labels attached to the work item"}, "system.creator": {"Type": {"Kind": "user"}, "Label": "Creator", "Required": true, "Description": "The user that created the work item"}, "system.codebase": {"Type": {"Kind": "codebase"}, "Label": "Codebase", "Required": false, "Description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"Type": {"Kind": "list", "ComponentType": {"Kind": "user"}}, "Label": "Assignees", "Required": false, "Description": "The users that are assigned to the work item"}, "system.iteration": {"Type": {"Kind": "iteration"}, "Label": "Iteration", "Required": false, "Description": "The iteration to which the work item belongs"}, "system.created_at": {"Type": {"Kind": "instant"}, "Label": "Created at", "Required": false, "Description": "The date and time when the work item was created"}, "system.updated_at": {"Type": {"Kind": "instant"}, "Label": "Updated at", "Required": false, "Description": "The date and time when the work item was last updated"}, "system.description": {"Type": {"Kind": "markup"}, "Label": "Description", "Required": false, "Description": "A descriptive text of the work item"}, "system.remote_item_id": {"Type": {"Kind": "string"}, "Label": "Remote item", "Required": false, "Description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("27adc1a2-1ded-43b8-a125-12777139496c"), `{"system.title": "Work item 1"}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("c106c056-2fec-4e56-83f0-cac31bb7ac1f"), `{"system.title": "Work item 2"}`)
		})

		t.Run("epic", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("2c169431-a55d-49eb-af74-cc19e895356f"), `{"system.area": {"Type": {"Kind": "area"}, "Label": "Area", "Required": false, "Description": "The area to which the work item belongs"}, "system.order": {"Type": {"Kind": "float"}, "Label": "Execution Order", "Required": false, "Description": "Execution Order of the workitem."}, "system.state": {"Type": {"Kind": "enum", "Values": ["new", "open", "in progress", "resolved", "closed"], "BaseType": {"Kind": "string"}}, "Label": "State", "Required": true, "Description": "The state of the work item"}, "system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}, "system.labels": {"Type": {"Kind": "list", "ComponentType": {"Kind": "label"}}, "Label": "Labels", "Required": false, "Description": "List of labels attached to the work item"}, "system.creator": {"Type": {"Kind": "user"}, "Label": "Creator", "Required": true, "Description": "The user that created the work item"}, "system.codebase": {"Type": {"Kind": "codebase"}, "Label": "Codebase", "Required": false, "Description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"Type": {"Kind": "list", "ComponentType": {"Kind": "user"}}, "Label": "Assignees", "Required": false, "Description": "The users that are assigned to the work item"}, "system.iteration": {"Type": {"Kind": "iteration"}, "Label": "Iteration", "Required": false, "Description": "The iteration to which the work item belongs"}, "system.created_at": {"Type": {"Kind": "instant"}, "Label": "Created at", "Required": false, "Description": "The date and time when the work item was created"}, "system.updated_at": {"Type": {"Kind": "instant"}, "Label": "Updated at", "Required": false, "Description": "The date and time when the work item was last updated"}, "system.description": {"Type": {"Kind": "markup"}, "Label": "Description", "Required": false, "Description": "A descriptive text of the work item"}, "system.remote_item_id": {"Type": {"Kind": "string"}, "Label": "Remote item", "Required": false, "Description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("4aebb314-a8c1-4e9c-96b6-074769d16934"), `{"system.title": "Work item 3"}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("9c53fb2b-c6af-48a1-bef1-6fa547ea72fa"), `{"system.title": "Work item 4"}`)
		})

		t.Run("story", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("6ff83406-caa7-47a9-9200-4ca796be11bb"), `{"system.area": {"Type": {"Kind": "area"}, "Label": "Area", "Required": false, "Description": "The area to which the work item belongs"}, "system.order": {"Type": {"Kind": "float"}, "Label": "Execution Order", "Required": false, "Description": "Execution Order of the workitem."}, "system.state": {"Type": {"Kind": "enum", "Values": ["new", "open", "in progress", "resolved", "closed"], "BaseType": {"Kind": "string"}}, "Label": "State", "Required": true, "Description": "The state of the work item"}, "system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}, "system.labels": {"Type": {"Kind": "list", "ComponentType": {"Kind": "label"}}, "Label": "Labels", "Required": false, "Description": "List of labels attached to the work item"}, "system.creator": {"Type": {"Kind": "user"}, "Label": "Creator", "Required": true, "Description": "The user that created the work item"}, "system.codebase": {"Type": {"Kind": "codebase"}, "Label": "Codebase", "Required": false, "Description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"Type": {"Kind": "list", "ComponentType": {"Kind": "user"}}, "Label": "Assignees", "Required": false, "Description": "The users that are assigned to the work item"}, "system.iteration": {"Type": {"Kind": "iteration"}, "Label": "Iteration", "Required": false, "Description": "The iteration to which the work item belongs"}, "system.created_at": {"Type": {"Kind": "instant"}, "Label": "Created at", "Required": false, "Description": "The date and time when the work item was created"}, "system.updated_at": {"Type": {"Kind": "instant"}, "Label": "Updated at", "Required": false, "Description": "The date and time when the work item was last updated"}, "system.description": {"Type": {"Kind": "markup"}, "Label": "Description", "Required": false, "Description": "A descriptive text of the work item"}, "system.remote_item_id": {"Type": {"Kind": "string"}, "Label": "Remote item", "Required": false, "Description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("68f83154-8d76-49c1-8be0-063ce90f803d"), `{"system.title": "Work item 5"}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("17e2081f-812d-4f4e-9c51-c537406bd1d8"), `{"system.title": "Work item 6"}`)
		})
	})
}

func testMigration95RemoveResolutionFieldFromImpediment(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:95], 95)

	expectWorkItemTypeFieldsToBe := func(t *testing.T, witID uuid.UUID, expectedFields string) {
		row := sqlDB.QueryRow("SELECT fields FROM work_item_types WHERE id = $1", witID.String())
		require.NotNil(t, row)
		var actualFields string
		err := row.Scan(&actualFields)
		require.NoError(t, err)

		actualMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(actualFields), &actualMap)
		require.NoError(t, err)
		expectedMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(expectedFields), &expectedMap)
		require.NoError(t, err)

		require.Equal(t, actualMap, expectedMap)
	}

	expectWorkItemFieldsToBe := func(t *testing.T, wiID uuid.UUID, expectedFields string) {
		row := sqlDB.QueryRow("SELECT fields FROM work_items WHERE id = $1", wiID.String())
		require.NotNil(t, row)
		var actualFields string
		err := row.Scan(&actualFields)
		require.NoError(t, err)

		actualMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(actualFields), &actualMap)
		require.NoError(t, err)
		expectedMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(expectedFields), &expectedMap)
		require.NoError(t, err)

		require.Equal(t, actualMap, expectedMap)
	}

	// Create two "impediment" work items, one with and one without the
	// "resolution" field the fields and one without and check that they've been
	// created as expected.
	require.Nil(t, runSQLscript(sqlDB, "095-remove-resolution-field-from-impediment.sql"))

	t.Run("before", func(t *testing.T) {
		t.Run("impediment", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("03b9bb64-4f65-4fa7-b165-494cd4f01401"), `{"resolution": {"type": {"values": ["Done", "Rejected", "Duplicate", "Incomplete Description", "Can not Reproduce", "Partially Completed", "Deferred", "Wont Fix", "Out of Date", "Explained", "Verified"], "base_type": {"kind": "string"}, "simple_type": {"kind": "enum"}, "rewritable_values": false}, "label": "Resolution", "required": false, "read_only": false, "description": "The reason why this work items state was last changed.\n"}, "system.area": {"type": {"kind": "area"}, "label": "Area", "required": false, "read_only": false, "description": "The area to which the work item belongs"}, "system.order": {"type": {"kind": "float"}, "label": "Execution Order", "required": false, "read_only": true, "description": "Execution Order of the workitem"}, "system.state": {"type": {"values": ["New", "Open", "In Progress", "Resolved", "Closed"], "base_type": {"kind": "string"}, "simple_type": {"kind": "enum"}, "rewritable_values": false}, "label": "State", "required": true, "read_only": false, "description": "The state of the impediment."}, "system.title": {"type": {"kind": "string"}, "label": "Title", "required": true, "read_only": false, "description": "The title text of the work item"}, "system.labels": {"type": {"simple_type": {"kind": "list"}, "component_type": {"kind": "label"}}, "label": "Labels", "required": false, "read_only": false, "description": "List of labels attached to the work item"}, "system.number": {"type": {"kind": "integer"}, "label": "Number", "required": false, "read_only": true, "description": "The unique number that was given to this workitem within its space."}, "system.creator": {"type": {"kind": "user"}, "label": "Creator", "required": true, "read_only": false, "description": "The user that created the work item"}, "system.codebase": {"type": {"kind": "codebase"}, "label": "Codebase", "required": false, "read_only": false, "description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"type": {"simple_type": {"kind": "list"}, "component_type": {"kind": "user"}}, "label": "Assignees", "required": false, "read_only": false, "description": "The users that are assigned to the work item"}, "system.iteration": {"type": {"kind": "iteration"}, "label": "Iteration", "required": false, "read_only": false, "description": "The iteration to which the work item belongs"}, "system.created_at": {"type": {"kind": "instant"}, "label": "Created at", "required": false, "read_only": true, "description": "The date and time when the work item was created"}, "system.updated_at": {"type": {"kind": "instant"}, "label": "Updated at", "required": false, "read_only": true, "description": "The date and time when the work item was last updated"}, "system.description": {"type": {"kind": "markup"}, "label": "Description", "required": false, "read_only": false, "description": "A descriptive text of the work item"}, "system.remote_item_id": {"type": {"kind": "string"}, "label": "Remote item", "required": false, "read_only": false, "description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("24ed462d-0430-4ffe-ba4f-7b5725b6a48c"), `{"system.title": "Work item 1", "resolution": "Rejected"}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("6a870ee3-e57c-4f98-9c7a-3cdf2ef5c2ef"), `{"system.title": "Work item 2"}`)
		})
	})

	// Migrate to the current version. That removes the "resolution" field from
	// the "impediment" work items and their work item type. We then check that
	// the "resolution" field has actually been removed from the work items and
	// their type.
	migrateToVersion(t, sqlDB, migrations[:96], 96)

	t.Run("after", func(t *testing.T) {
		t.Run("impediment", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("03b9bb64-4f65-4fa7-b165-494cd4f01401"), `{"system.area": {"type": {"kind": "area"}, "label": "Area", "required": false, "read_only": false, "description": "The area to which the work item belongs"}, "system.order": {"type": {"kind": "float"}, "label": "Execution Order", "required": false, "read_only": true, "description": "Execution Order of the workitem"}, "system.state": {"type": {"values": ["New", "Open", "In Progress", "Resolved", "Closed"], "base_type": {"kind": "string"}, "simple_type": {"kind": "enum"}, "rewritable_values": false}, "label": "State", "required": true, "read_only": false, "description": "The state of the impediment."}, "system.title": {"type": {"kind": "string"}, "label": "Title", "required": true, "read_only": false, "description": "The title text of the work item"}, "system.labels": {"type": {"simple_type": {"kind": "list"}, "component_type": {"kind": "label"}}, "label": "Labels", "required": false, "read_only": false, "description": "List of labels attached to the work item"}, "system.number": {"type": {"kind": "integer"}, "label": "Number", "required": false, "read_only": true, "description": "The unique number that was given to this workitem within its space."}, "system.creator": {"type": {"kind": "user"}, "label": "Creator", "required": true, "read_only": false, "description": "The user that created the work item"}, "system.codebase": {"type": {"kind": "codebase"}, "label": "Codebase", "required": false, "read_only": false, "description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"type": {"simple_type": {"kind": "list"}, "component_type": {"kind": "user"}}, "label": "Assignees", "required": false, "read_only": false, "description": "The users that are assigned to the work item"}, "system.iteration": {"type": {"kind": "iteration"}, "label": "Iteration", "required": false, "read_only": false, "description": "The iteration to which the work item belongs"}, "system.created_at": {"type": {"kind": "instant"}, "label": "Created at", "required": false, "read_only": true, "description": "The date and time when the work item was created"}, "system.updated_at": {"type": {"kind": "instant"}, "label": "Updated at", "required": false, "read_only": true, "description": "The date and time when the work item was last updated"}, "system.description": {"type": {"kind": "markup"}, "label": "Description", "required": false, "read_only": false, "description": "A descriptive text of the work item"}, "system.remote_item_id": {"type": {"kind": "string"}, "label": "Remote item", "required": false, "read_only": false, "description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("24ed462d-0430-4ffe-ba4f-7b5725b6a48c"), `{"system.title": "Work item 1"}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("6a870ee3-e57c-4f98-9c7a-3cdf2ef5c2ef"), `{"system.title": "Work item 2"}`)
		})
	})
}

func testMigration96ChangesToAgileTemplate(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:96], 96)

	expectWorkItemTypeFieldsToBe := func(t *testing.T, witID uuid.UUID, expectedFields string) {
		row := sqlDB.QueryRow("SELECT fields FROM work_item_types WHERE id = $1", witID.String())
		require.NotNil(t, row)
		var actualFields string
		err := row.Scan(&actualFields)
		require.NoError(t, err)

		actualMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(actualFields), &actualMap)
		require.NoError(t, err)
		expectedMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(expectedFields), &expectedMap)
		require.NoError(t, err)

		require.Equal(t, actualMap, expectedMap)
	}

	expectWorkItemFieldsToBe := func(t *testing.T, wiID uuid.UUID, expectedFields string) {
		row := sqlDB.QueryRow("SELECT fields FROM work_items WHERE id = $1", wiID.String())
		require.NotNil(t, row)
		var actualFields string
		err := row.Scan(&actualFields)
		require.NoError(t, err)

		actualMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(actualFields), &actualMap)
		require.NoError(t, err)
		expectedMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(expectedFields), &expectedMap)
		require.NoError(t, err)

		require.Equal(t, actualMap, expectedMap)
	}

	// create two work items, one with the fields and one without
	// and check that they've been created as expected.
	require.Nil(t, runSQLscript(sqlDB, "096-changes-to-agile-template-test.sql"))

	t.Run("before", func(t *testing.T) {
		t.Run("theme", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("5182fc8c-b1d6-4c3d-83ca-6a3c781fa18a"), `{"system.area": {"Type": {"Kind": "area"}, "Label": "Area", "Required": false, "Description": "The area to which the work item belongs"}, "system.order": {"Type": {"Kind": "float"}, "Label": "Execution Order", "Required": false, "Description": "Execution Order of the workitem."}, "system.state": {"Type": {"Kind": "enum", "Values": ["new", "open", "in progress", "resolved", "closed"], "BaseType": {"Kind": "string"}}, "Label": "State", "Required": true, "Description": "The state of the work item"}, "system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}, "system.labels": {"Type": {"Kind": "list", "ComponentType": {"Kind": "label"}}, "Label": "Labels", "Required": false, "Description": "List of labels attached to the work item"}, "business_value": {"Type": {"Kind": "integer"}, "Label": "Business Value", "Required": false, "Description": "The business value of this work item."}, "effort": {"Type": {"Kind": "float"}, "Label": "Effort", "Required": false, "Description": "The effort that was given to this workitem within its space."}, "time_criticality": {"Type": {"Kind": "float"}, "Label": "Time Criticality", "Required": false, "Description": "The time criticality that was given to this workitem within its space."}, "system.creator": {"Type": {"Kind": "user"}, "Label": "Creator", "Required": true, "Description": "The user that created the work item"}, "system.codebase": {"Type": {"Kind": "codebase"}, "Label": "Codebase", "Required": false, "Description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"Type": {"Kind": "list", "ComponentType": {"Kind": "user"}}, "Label": "Assignees", "Required": false, "Description": "The users that are assigned to the work item"}, "system.iteration": {"Type": {"Kind": "iteration"}, "Label": "Iteration", "Required": false, "Description": "The iteration to which the work item belongs"}, "system.created_at": {"Type": {"Kind": "instant"}, "Label": "Created at", "Required": false, "Description": "The date and time when the work item was created"}, "system.updated_at": {"Type": {"Kind": "instant"}, "Label": "Updated at", "Required": false, "Description": "The date and time when the work item was last updated"}, "system.description": {"Type": {"Kind": "markup"}, "Label": "Description", "Required": false, "Description": "A descriptive text of the work item"}, "system.remote_item_id": {"Type": {"Kind": "string"}, "Label": "Remote item", "Required": false, "Description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("cf84c888-ac28-493d-a0cd-978b78568011"), `{"system.title": "Work item 1", "effort": 12.34, "business_value": 1234, "time_criticality": 56.78}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("8bbb542c-4f5c-44bb-9272-e1a8f24e6e22"), `{"system.title": "Work item 2"}`)
		})
		t.Run("epic", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("2c169431-a55d-49eb-af74-cc19e895356f"), `{"system.area": {"Type": {"Kind": "area"}, "Label": "Area", "Required": false, "Description": "The area to which the work item belongs"}, "system.order": {"Type": {"Kind": "float"}, "Label": "Execution Order", "Required": false, "Description": "Execution Order of the workitem."}, "system.state": {"Type": {"Kind": "enum", "Values": ["new", "open", "in progress", "resolved", "closed"], "BaseType": {"Kind": "string"}}, "Label": "State", "Required": true, "Description": "The state of the work item"}, "system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}, "system.labels": {"Type": {"Kind": "list", "ComponentType": {"Kind": "label"}}, "Label": "Labels", "Required": false, "Description": "List of labels attached to the work item"}, "component": {"Type": {"Kind": "string"}, "Label": "Component", "Required": false, "Description": "The component value of this work item."}, "business_value": {"Type": {"Kind": "integer"}, "Label": "Business Value", "Required": false, "Description": "The business value of this work item."}, "effort": {"Type": {"Kind": "float"}, "Label": "Effort", "Required": false, "Description": "The effort that was given to this workitem within its space."}, "time_criticality": {"Type": {"Kind": "float"}, "Label": "Time Criticality", "Required": false, "Description": "The time criticality that was given to this workitem within its space."}, "system.creator": {"Type": {"Kind": "user"}, "Label": "Creator", "Required": true, "Description": "The user that created the work item"}, "system.codebase": {"Type": {"Kind": "codebase"}, "Label": "Codebase", "Required": false, "Description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"Type": {"Kind": "list", "ComponentType": {"Kind": "user"}}, "Label": "Assignees", "Required": false, "Description": "The users that are assigned to the work item"}, "system.iteration": {"Type": {"Kind": "iteration"}, "Label": "Iteration", "Required": false, "Description": "The iteration to which the work item belongs"}, "system.created_at": {"Type": {"Kind": "instant"}, "Label": "Created at", "Required": false, "Description": "The date and time when the work item was created"}, "system.updated_at": {"Type": {"Kind": "instant"}, "Label": "Updated at", "Required": false, "Description": "The date and time when the work item was last updated"}, "system.description": {"Type": {"Kind": "markup"}, "Label": "Description", "Required": false, "Description": "A descriptive text of the work item"}, "system.remote_item_id": {"Type": {"Kind": "string"}, "Label": "Remote item", "Required": false, "Description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("4aebb314-a8c1-4e9c-96b6-074769d16933"), `{"system.title": "Work item 3", "effort": 12.34, "business_value": 1234, "time_criticality": 56.78, "component": "Component 1"}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("9c53fb2b-c6af-48a1-bef1-6fa547ea7244"), `{"system.title": "Work item 4"}`)
		})
		t.Run("story", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("6ff83406-caa7-47a9-9200-4ca796be11bb"), `{"system.area": {"Type": {"Kind": "area"}, "Label": "Area", "Required": false, "Description": "The area to which the work item belongs"}, "system.order": {"Type": {"Kind": "float"}, "Label": "Execution Order", "Required": false, "Description": "Execution Order of the workitem."}, "system.state": {"Type": {"Kind": "enum", "Values": ["new", "open", "in progress", "resolved", "closed"], "BaseType": {"Kind": "string"}}, "Label": "State", "Required": true, "Description": "The state of the work item"}, "system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}, "system.labels": {"Type": {"Kind": "list", "ComponentType": {"Kind": "label"}}, "Label": "Labels", "Required": false, "Description": "List of labels attached to the work item"}, "effort": {"Type": {"Kind": "float"}, "Label": "Effort", "Required": false, "Description": "The effort that was given to this workitem within its space."}, "system.creator": {"Type": {"Kind": "user"}, "Label": "Creator", "Required": true, "Description": "The user that created the work item"}, "system.codebase": {"Type": {"Kind": "codebase"}, "Label": "Codebase", "Required": false, "Description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"Type": {"Kind": "list", "ComponentType": {"Kind": "user"}}, "Label": "Assignees", "Required": false, "Description": "The users that are assigned to the work item"}, "system.iteration": {"Type": {"Kind": "iteration"}, "Label": "Iteration", "Required": false, "Description": "The iteration to which the work item belongs"}, "system.created_at": {"Type": {"Kind": "instant"}, "Label": "Created at", "Required": false, "Description": "The date and time when the work item was created"}, "system.updated_at": {"Type": {"Kind": "instant"}, "Label": "Updated at", "Required": false, "Description": "The date and time when the work item was last updated"}, "system.description": {"Type": {"Kind": "markup"}, "Label": "Description", "Required": false, "Description": "A descriptive text of the work item"}, "system.remote_item_id": {"Type": {"Kind": "string"}, "Label": "Remote item", "Required": false, "Description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("68f83154-8d76-49c1-8be0-063ce90f8055"), `{"system.title": "Work item 5", "effort": 12.34}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("17e2081f-812d-4f4e-9c51-c537406bd166"), `{"system.title": "Work item 6"}`)
		})
	})

	// Migrate to the current version. That removes fields from each work item
	// and its work item type. We then check that those fields have actually
	// been removed from the work items and their types.
	migrateToVersion(t, sqlDB, migrations[:97], 97)

	t.Run("after", func(t *testing.T) {
		t.Run("theme", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("5182fc8c-b1d6-4c3d-83ca-6a3c781fa18a"), `{"system.area": {"Type": {"Kind": "area"}, "Label": "Area", "Required": false, "Description": "The area to which the work item belongs"}, "system.order": {"Type": {"Kind": "float"}, "Label": "Execution Order", "Required": false, "Description": "Execution Order of the workitem."}, "system.state": {"Type": {"Kind": "enum", "Values": ["new", "open", "in progress", "resolved", "closed"], "BaseType": {"Kind": "string"}}, "Label": "State", "Required": true, "Description": "The state of the work item"}, "system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}, "system.labels": {"Type": {"Kind": "list", "ComponentType": {"Kind": "label"}}, "Label": "Labels", "Required": false, "Description": "List of labels attached to the work item"}, "system.creator": {"Type": {"Kind": "user"}, "Label": "Creator", "Required": true, "Description": "The user that created the work item"}, "system.codebase": {"Type": {"Kind": "codebase"}, "Label": "Codebase", "Required": false, "Description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"Type": {"Kind": "list", "ComponentType": {"Kind": "user"}}, "Label": "Assignees", "Required": false, "Description": "The users that are assigned to the work item"}, "system.iteration": {"Type": {"Kind": "iteration"}, "Label": "Iteration", "Required": false, "Description": "The iteration to which the work item belongs"}, "system.created_at": {"Type": {"Kind": "instant"}, "Label": "Created at", "Required": false, "Description": "The date and time when the work item was created"}, "system.updated_at": {"Type": {"Kind": "instant"}, "Label": "Updated at", "Required": false, "Description": "The date and time when the work item was last updated"}, "system.description": {"Type": {"Kind": "markup"}, "Label": "Description", "Required": false, "Description": "A descriptive text of the work item"}, "system.remote_item_id": {"Type": {"Kind": "string"}, "Label": "Remote item", "Required": false, "Description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("cf84c888-ac28-493d-a0cd-978b78568011"), `{"system.title": "Work item 1"}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("8bbb542c-4f5c-44bb-9272-e1a8f24e6e22"), `{"system.title": "Work item 2"}`)
		})

		t.Run("epic", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("2c169431-a55d-49eb-af74-cc19e895356f"), `{"system.area": {"Type": {"Kind": "area"}, "Label": "Area", "Required": false, "Description": "The area to which the work item belongs"}, "system.order": {"Type": {"Kind": "float"}, "Label": "Execution Order", "Required": false, "Description": "Execution Order of the workitem."}, "system.state": {"Type": {"Kind": "enum", "Values": ["new", "open", "in progress", "resolved", "closed"], "BaseType": {"Kind": "string"}}, "Label": "State", "Required": true, "Description": "The state of the work item"}, "system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}, "system.labels": {"Type": {"Kind": "list", "ComponentType": {"Kind": "label"}}, "Label": "Labels", "Required": false, "Description": "List of labels attached to the work item"}, "system.creator": {"Type": {"Kind": "user"}, "Label": "Creator", "Required": true, "Description": "The user that created the work item"}, "system.codebase": {"Type": {"Kind": "codebase"}, "Label": "Codebase", "Required": false, "Description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"Type": {"Kind": "list", "ComponentType": {"Kind": "user"}}, "Label": "Assignees", "Required": false, "Description": "The users that are assigned to the work item"}, "system.iteration": {"Type": {"Kind": "iteration"}, "Label": "Iteration", "Required": false, "Description": "The iteration to which the work item belongs"}, "system.created_at": {"Type": {"Kind": "instant"}, "Label": "Created at", "Required": false, "Description": "The date and time when the work item was created"}, "system.updated_at": {"Type": {"Kind": "instant"}, "Label": "Updated at", "Required": false, "Description": "The date and time when the work item was last updated"}, "system.description": {"Type": {"Kind": "markup"}, "Label": "Description", "Required": false, "Description": "A descriptive text of the work item"}, "system.remote_item_id": {"Type": {"Kind": "string"}, "Label": "Remote item", "Required": false, "Description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("4aebb314-a8c1-4e9c-96b6-074769d16933"), `{"system.title": "Work item 3"}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("9c53fb2b-c6af-48a1-bef1-6fa547ea7244"), `{"system.title": "Work item 4"}`)
		})

		t.Run("story", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("6ff83406-caa7-47a9-9200-4ca796be11bb"), `{"system.area": {"Type": {"Kind": "area"}, "Label": "Area", "Required": false, "Description": "The area to which the work item belongs"}, "system.order": {"Type": {"Kind": "float"}, "Label": "Execution Order", "Required": false, "Description": "Execution Order of the workitem."}, "system.state": {"Type": {"Kind": "enum", "Values": ["new", "open", "in progress", "resolved", "closed"], "BaseType": {"Kind": "string"}}, "Label": "State", "Required": true, "Description": "The state of the work item"}, "system.title": {"Type": {"Kind": "string"}, "Label": "Title", "Required": true, "Description": "The title text of the work item"}, "system.labels": {"Type": {"Kind": "list", "ComponentType": {"Kind": "label"}}, "Label": "Labels", "Required": false, "Description": "List of labels attached to the work item"}, "system.creator": {"Type": {"Kind": "user"}, "Label": "Creator", "Required": true, "Description": "The user that created the work item"}, "system.codebase": {"Type": {"Kind": "codebase"}, "Label": "Codebase", "Required": false, "Description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"Type": {"Kind": "list", "ComponentType": {"Kind": "user"}}, "Label": "Assignees", "Required": false, "Description": "The users that are assigned to the work item"}, "system.iteration": {"Type": {"Kind": "iteration"}, "Label": "Iteration", "Required": false, "Description": "The iteration to which the work item belongs"}, "system.created_at": {"Type": {"Kind": "instant"}, "Label": "Created at", "Required": false, "Description": "The date and time when the work item was created"}, "system.updated_at": {"Type": {"Kind": "instant"}, "Label": "Updated at", "Required": false, "Description": "The date and time when the work item was last updated"}, "system.description": {"Type": {"Kind": "markup"}, "Label": "Description", "Required": false, "Description": "A descriptive text of the work item"}, "system.remote_item_id": {"Type": {"Kind": "string"}, "Label": "Remote item", "Required": false, "Description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("68f83154-8d76-49c1-8be0-063ce90f8055"), `{"system.title": "Work item 5"}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("17e2081f-812d-4f4e-9c51-c537406bd166"), `{"system.title": "Work item 6"}`)
		})
	})
}

func testMigration97RemoveResolutionFieldFromImpediment(t *testing.T) {
	migrateToVersion(t, sqlDB, migrations[:97], 97)

	expectWorkItemTypeFieldsToBe := func(t *testing.T, witID uuid.UUID, expectedFields string) {
		row := sqlDB.QueryRow("SELECT fields FROM work_item_types WHERE id = $1", witID.String())
		require.NotNil(t, row)
		var actualFields string
		err := row.Scan(&actualFields)
		require.NoError(t, err)

		actualMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(actualFields), &actualMap)
		require.NoError(t, err)
		expectedMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(expectedFields), &expectedMap)
		require.NoError(t, err)

		require.Equal(t, actualMap, expectedMap)
	}

	expectWorkItemFieldsToBe := func(t *testing.T, wiID uuid.UUID, expectedFields string) {
		row := sqlDB.QueryRow("SELECT fields FROM work_items WHERE id = $1", wiID.String())
		require.NotNil(t, row)
		var actualFields string
		err := row.Scan(&actualFields)
		require.NoError(t, err)

		actualMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(actualFields), &actualMap)
		require.NoError(t, err)
		expectedMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(expectedFields), &expectedMap)
		require.NoError(t, err)

		require.Equal(t, actualMap, expectedMap)
	}

	// Create two "impediment" work items, one with and one without the
	// "resolution" field the fields and one without and check that they've been
	// created as expected.
	require.Nil(t, runSQLscript(sqlDB, "097-remove-resolution-field-from-impediment.sql"))

	t.Run("before", func(t *testing.T) {
		t.Run("impediment", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("03b9bb64-4f65-4fa7-b165-494cd4f01401"), `{"resolution": {"type": {"values": ["Done", "Rejected", "Duplicate", "Incomplete Description", "Can not Reproduce", "Partially Completed", "Deferred", "Wont Fix", "Out of Date", "Explained", "Verified"], "base_type": {"kind": "string"}, "simple_type": {"kind": "enum"}, "rewritable_values": false}, "label": "Resolution", "required": false, "read_only": false, "description": "The reason why this work items state was last changed.\n"}, "system.area": {"type": {"kind": "area"}, "label": "Area", "required": false, "read_only": false, "description": "The area to which the work item belongs"}, "system.order": {"type": {"kind": "float"}, "label": "Execution Order", "required": false, "read_only": true, "description": "Execution Order of the workitem"}, "system.state": {"type": {"values": ["New", "Open", "In Progress", "Resolved", "Closed"], "base_type": {"kind": "string"}, "simple_type": {"kind": "enum"}, "rewritable_values": false}, "label": "State", "required": true, "read_only": false, "description": "The state of the impediment."}, "system.title": {"type": {"kind": "string"}, "label": "Title", "required": true, "read_only": false, "description": "The title text of the work item"}, "system.labels": {"type": {"simple_type": {"kind": "list"}, "component_type": {"kind": "label"}}, "label": "Labels", "required": false, "read_only": false, "description": "List of labels attached to the work item"}, "system.number": {"type": {"kind": "integer"}, "label": "Number", "required": false, "read_only": true, "description": "The unique number that was given to this workitem within its space."}, "system.creator": {"type": {"kind": "user"}, "label": "Creator", "required": true, "read_only": false, "description": "The user that created the work item"}, "system.codebase": {"type": {"kind": "codebase"}, "label": "Codebase", "required": false, "read_only": false, "description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"type": {"simple_type": {"kind": "list"}, "component_type": {"kind": "user"}}, "label": "Assignees", "required": false, "read_only": false, "description": "The users that are assigned to the work item"}, "system.iteration": {"type": {"kind": "iteration"}, "label": "Iteration", "required": false, "read_only": false, "description": "The iteration to which the work item belongs"}, "system.created_at": {"type": {"kind": "instant"}, "label": "Created at", "required": false, "read_only": true, "description": "The date and time when the work item was created"}, "system.updated_at": {"type": {"kind": "instant"}, "label": "Updated at", "required": false, "read_only": true, "description": "The date and time when the work item was last updated"}, "system.description": {"type": {"kind": "markup"}, "label": "Description", "required": false, "read_only": false, "description": "A descriptive text of the work item"}, "system.remote_item_id": {"type": {"kind": "string"}, "label": "Remote item", "required": false, "read_only": false, "description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("24ed462d-0430-4ffe-ba4f-7b5725b6a411"), `{"system.title": "Work item 1", "resolution": "Rejected"}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("6a870ee3-e57c-4f98-9c7a-3cdf2ef5c222"), `{"system.title": "Work item 2"}`)
		})
	})

	// Migrate to the current version. That removes the "resolution" field from
	// the "impediment" work items and their work item type. We then check that
	// the "resolution" field has actually been removed from the work items and
	// their type.
	migrateToVersion(t, sqlDB, migrations[:98], 98)

	t.Run("after", func(t *testing.T) {
		t.Run("impediment", func(t *testing.T) {
			expectWorkItemTypeFieldsToBe(t, uuid.FromStringOrNil("03b9bb64-4f65-4fa7-b165-494cd4f01401"), `{"system.area": {"type": {"kind": "area"}, "label": "Area", "required": false, "read_only": false, "description": "The area to which the work item belongs"}, "system.order": {"type": {"kind": "float"}, "label": "Execution Order", "required": false, "read_only": true, "description": "Execution Order of the workitem"}, "system.state": {"type": {"values": ["New", "Open", "In Progress", "Resolved", "Closed"], "base_type": {"kind": "string"}, "simple_type": {"kind": "enum"}, "rewritable_values": false}, "label": "State", "required": true, "read_only": false, "description": "The state of the impediment."}, "system.title": {"type": {"kind": "string"}, "label": "Title", "required": true, "read_only": false, "description": "The title text of the work item"}, "system.labels": {"type": {"simple_type": {"kind": "list"}, "component_type": {"kind": "label"}}, "label": "Labels", "required": false, "read_only": false, "description": "List of labels attached to the work item"}, "system.number": {"type": {"kind": "integer"}, "label": "Number", "required": false, "read_only": true, "description": "The unique number that was given to this workitem within its space."}, "system.creator": {"type": {"kind": "user"}, "label": "Creator", "required": true, "read_only": false, "description": "The user that created the work item"}, "system.codebase": {"type": {"kind": "codebase"}, "label": "Codebase", "required": false, "read_only": false, "description": "Contains codebase attributes to which this WI belongs to"}, "system.assignees": {"type": {"simple_type": {"kind": "list"}, "component_type": {"kind": "user"}}, "label": "Assignees", "required": false, "read_only": false, "description": "The users that are assigned to the work item"}, "system.iteration": {"type": {"kind": "iteration"}, "label": "Iteration", "required": false, "read_only": false, "description": "The iteration to which the work item belongs"}, "system.created_at": {"type": {"kind": "instant"}, "label": "Created at", "required": false, "read_only": true, "description": "The date and time when the work item was created"}, "system.updated_at": {"type": {"kind": "instant"}, "label": "Updated at", "required": false, "read_only": true, "description": "The date and time when the work item was last updated"}, "system.description": {"type": {"kind": "markup"}, "label": "Description", "required": false, "read_only": false, "description": "A descriptive text of the work item"}, "system.remote_item_id": {"type": {"kind": "string"}, "label": "Remote item", "required": false, "read_only": false, "description": "The ID of the remote work item"}}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("24ed462d-0430-4ffe-ba4f-7b5725b6a411"), `{"system.title": "Work item 1"}`)
			expectWorkItemFieldsToBe(t, uuid.FromStringOrNil("6a870ee3-e57c-4f98-9c7a-3cdf2ef5c222"), `{"system.title": "Work item 2"}`)
		})
	})
}

// runSQLscript loads the given filename from the packaged SQL test files and
// executes it on the given database. Golang text/template module is used
// to handle all the optional arguments passed to the sql test files
func runSQLscript(db *sql.DB, sqlFilename string, args ...string) error {
	var tx *sql.Tx
	tx, err := db.Begin()
	if err != nil {
		return errs.Wrapf(err, "failed to start transaction with file %s", sqlFilename)
	}
	if err := executeSQLTestFile(sqlFilename, args...)(tx); err != nil {
		log.Warn(nil, nil, "failed to execute data insertion using '%s': %s\n", sqlFilename, err)
		errRollback := tx.Rollback()
		if errRollback != nil {
			return errs.Wrapf(err, "error while rolling back transaction for file %s", sqlFilename)
		}
		return errs.Wrapf(err, "failed to execute data insertion using file %s", sqlFilename)
	}
	err = tx.Commit()
	return errs.Wrapf(err, "error during transaction commit for file %s", sqlFilename)
}

// executeSQLTestFile loads the given filename from the packaged SQL files and
// executes it on the given database. Golang text/template module is used
// to handle all the optional arguments passed to the sql test files
func executeSQLTestFile(filename string, args ...string) fn {
	return func(db *sql.Tx) error {
		log.Info(nil, nil, "executing SQL test script '%s'", filename)
		data, err := Asset(filename)
		if err != nil {
			return errs.Wrapf(err, "failed to load SQL template %s", filename)
		}

		if len(args) > 0 {
			tmpl, templErr := template.New("sql").Parse(string(data))
			if templErr != nil {
				return errs.Wrapf(templErr, "failed to create new SQL template from file %s", filename)
			}
			var sqlScript bytes.Buffer
			writer := bufio.NewWriter(&sqlScript)
			tmplExecErr := tmpl.Execute(writer, args)
			if tmplExecErr != nil {
				return errs.Wrapf(tmplExecErr, "failed to execute SQL template from file %s", filename)
			}
			// We need to flush the content of the writer
			writer.Flush()
			_, err = db.Exec(sqlScript.String())
		} else {
			_, err = db.Exec(string(data))
		}

		return errs.Wrapf(err, "failed to execute SQL query from file %s", filename)
	}
}

// migrateToVersion runs the migration of all the scripts to a certain version
func migrateToVersion(t *testing.T, db *sql.DB, m migration.Migrations, version int64) {
	var err error
	for nextVersion := int64(0); nextVersion < version && err == nil; nextVersion++ {
		var tx *sql.Tx
		tx, err = sqlDB.Begin()
		require.NoError(t, err, "failed to start tansaction for version %d", version)
		if err = migration.MigrateToNextVersion(tx, &nextVersion, m, databaseName); err != nil {
			errRollback := tx.Rollback()
			require.NoError(t, errRollback, "failed to roll back transaction for version %d", version)
			require.NoError(t, err, "failed to migrate to version %d", version)
		}

		err = tx.Commit()
		require.NoError(t, err, "error during transaction commit")
	}
}
