package migration

import (
	"fmt"
	"log"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/transaction"
	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
	"golang.org/x/net/context"
)

const versionTableName string = "version"

// getCurrentVersion returns the highest version from the version
// table or -1 if that table does not exist.
//
// Returning -1 simplifies the logic of the migration process because
// the next version is always the current version + 1 which results
// in -1 + 1 = 0 which is exactly what we want as the first version.
func getCurrentVersion(db *gorm.DB) (int64, error) {
	if !db.HasTable(versionTableName) {
		return -1, nil
	}

	rows, err := db.Table(versionTableName).Select("max(version) as current").Rows()
	if err != nil {
		return -1, err
	}
	defer rows.Close()

	var current int64 = -1

	if rows.Next() {
		if err = rows.Scan(&current); err != nil {
			return -1, err
		}
	}

	return current, nil
}

// fn defines the type of function that can be part of a migration sequence
type fn func(tx *gorm.DB) error

// executeSQLFile loads the given filename from the packaged SQL files and
// executes it on the given database
func executeSQLFile(filename string) fn {
	return func(db *gorm.DB) error {
		data, err := Asset(filename)
		if err != nil {
			return err
		}
		return db.Exec(string(data)).Error
	}
}

// Migrate executes the required migration of the database on startup
func Migrate(db *gorm.DB) error {

	migrations := [][]fn{}

	// Version 0
	migrations = append(migrations, []fn{executeSQLFile("000-bootstrap.sql")})

	// Version 1
	migrations = append(migrations, []fn{executeSQLFile("001-common.sql")})

	// Version N
	//
	// In order to add an upgrade, simply append an array of MigrationFunc to the
	// the end of the "migrations" slice. The version numbers are determined by
	// the index in the array. The following code in comments show how you can
	// do a migration in 3 steps. If one of the steps fails, the others are not
	// executed.
	// If something goes wrong during the migration, all you need to do is return
	// an error that is not nil.

	/*
		migrations = append(migrations, []Func{
			func(db *gorm.DB) error {
				// Execute random go code
				return nil
			},
			executeSQLFile("YOUR_OWN_FILE.sql"),
			func(db *gorm.DB) error {
				// Execute random go code
				return nil
			},
		})
	*/

	var err error
	ts := models.NewGormTransactionSupport(db)

	// Ensure that every new transaction that gets created, will be serializable
	if err = ts.SetTransactionIsolationLevel(models.TXIsoLevelSerializable); err != nil {
		return err
	}

	for nextVersion := int64(0); nextVersion < int64(len(migrations)) && err == nil; nextVersion++ {

		err = transaction.Do(ts, func() error {
			// Try to get a lock on the version table.
			// But it has to exist in order to create a lock.
			if db.HasTable(versionTableName) {
				// Extract from the documentation:
				// "LOCK TABLE obtains a table-level lock, waiting if necessary for any conflicting locks to be released. [...]
				// Once obtained, the lock is held for the remainder of the current transaction.
				// (There is no UNLOCK TABLE command; locks are always released at transaction end.)""
				err = ts.TX().Exec(fmt.Sprintf("LOCK TABLE %s IN ACCESS EXCLUSIVE MODE", versionTableName)).Error
				if err != nil {
					return err
				}
			}

			// Determine current version and adjust the outmost loop
			// iterator variable "version"
			currentVersion, err := getCurrentVersion(ts.TX())
			if err != nil {
				return err
			}
			nextVersion = currentVersion + 1
			if nextVersion >= int64(len(migrations)) {
				// No further updates to apply (this is NOT an error)
				return nil
			}

			log.Printf("Attempt to update DB to version %d\n", nextVersion)

			// Apply all the updates of the next version
			for j := range migrations[nextVersion] {
				if err = migrations[nextVersion][j](ts.TX().Debug()); err != nil {
					return err
				}
			}

			// Finalize the update by inserting the new version into the version table
			err = ts.TX().Debug().Exec("insert into version (version) values(?)", nextVersion).Error
			if err != nil {
				log.Printf("Failed to update DB to version %d: %s\n", nextVersion, err)
			} else {
				log.Printf("Successfully updated DB to version %d\n", nextVersion)
			}
			return err
		})

		// If two concurrent transactions have been started and one results in a serialization_failure
		// we will ignore this error.
		// See https://www.postgresql.org/docs/9.3/static/sql-set-transaction.html
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code.Name() == "serialization_failure" {
				log.Printf("Ignoring %s: %s", pqErr.Code.Name(), err.Error())
				// Reset to nil to continue the loop
				err = nil
			}
		}

	}

	return err
}

// PopulateCommonTypes makes sure the database is populated with the correct types (e.g. system.bug etc.)
func PopulateCommonTypes(ctx context.Context, db *gorm.DB, witr models.WorkItemTypeRepository) error {
	// FIXME: Need to add this conditionally
	// q := `ALTER TABLE "tracker_queries" ADD CONSTRAINT "tracker_fk" FOREIGN KEY ("tracker") REFERENCES "trackers" ON DELETE CASCADE`
	// db.Exec(q)

	if err := createSystemUserstory(ctx, witr); err != nil {
		return err
	}
	if err := createSystemValueProposition(ctx, witr); err != nil {
		return err
	}
	if err := createSystemFundamental(ctx, witr); err != nil {
		return err
	}
	if err := createSystemExperience(ctx, witr); err != nil {
		return err
	}
	if err := createSystemFeature(ctx, witr); err != nil {
		return err
	}
	if err := createSystemBug(ctx, witr); err != nil {
		return err
	}
	return nil
}

func createSystemUserstory(ctx context.Context, witr models.WorkItemTypeRepository) error {
	return createCommon("system.userstory", ctx, witr)
}

func createSystemValueProposition(ctx context.Context, witr models.WorkItemTypeRepository) error {
	return createCommon("system.valueproposition", ctx, witr)
}

func createSystemFundamental(ctx context.Context, witr models.WorkItemTypeRepository) error {
	return createCommon("system.fundamental", ctx, witr)
}

func createSystemExperience(ctx context.Context, witr models.WorkItemTypeRepository) error {
	return createCommon("system.experience", ctx, witr)
}

func createSystemFeature(ctx context.Context, witr models.WorkItemTypeRepository) error {
	return createCommon("system.feature", ctx, witr)
}

func createSystemBug(ctx context.Context, witr models.WorkItemTypeRepository) error {
	return createCommon("system.bug", ctx, witr)
}

func createCommon(typeName string, ctx context.Context, witr models.WorkItemTypeRepository) error {
	_, err := witr.Load(ctx, typeName)
	switch err.(type) {
	case models.NotFoundError:
		stString := "string"
		_, err := witr.Create(ctx, nil, typeName, map[string]app.FieldDefinition{
			"system.title":       app.FieldDefinition{Type: &app.FieldType{Kind: "string"}, Required: true},
			"system.description": app.FieldDefinition{Type: &app.FieldType{Kind: "string"}, Required: false},
			"system.creator":     app.FieldDefinition{Type: &app.FieldType{Kind: "user"}, Required: true},
			"system.assignee":    app.FieldDefinition{Type: &app.FieldType{Kind: "user"}, Required: false},
			"system.state": app.FieldDefinition{
				Type: &app.FieldType{
					BaseType: &stString,
					Kind:     "enum",
					Values:   []interface{}{"new", "in progress", "resolved", "closed"},
				},
				Required: true,
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}
