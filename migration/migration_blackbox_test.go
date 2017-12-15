package migration_test

import (
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"testing"
	"time"

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
