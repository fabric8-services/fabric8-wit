package migration

import (
	"database/sql"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/workitem"
	"github.com/almighty/almighty-core/workitem/link"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	"golang.org/x/net/context"
)

// AdvisoryLockID is a random number that should be used within the application
// by anybody who wants to modify the "version" table.
const AdvisoryLockID = 42

// fn defines the type of function that can be part of a migration steps
type fn func(tx *sql.Tx) error

// steps defines a collection of all the functions that make up a version
type steps []fn

// migrations defines all a collection of all the steps
type migrations []steps

// Migrate executes the required migration of the database on startup.
// For each successful migration, an entry will be written into the "version"
// table, that states when a certain version was reached.
func Migrate(db *sql.DB) error {
	var err error

	if db == nil {
		return errs.Errorf("Database handle is nil\n")
	}

	m := getMigrations()

	var tx *sql.Tx
	for nextVersion := int64(0); nextVersion < int64(len(m)) && err == nil; nextVersion++ {

		tx, err = db.Begin()
		if err != nil {
			return errs.Errorf("Failed to start transaction: %s\n", err)
		}

		err = migrateToNextVersion(tx, &nextVersion, m)

		if err != nil {
			oldErr := err
			log.Info(nil, map[string]interface{}{
				"pkg":         "migration",
				"nextVersion": nextVersion,
				"migrations":  m,
				"err":         err,
			}, "Rolling back transaction due to: ", err)

			if err = tx.Rollback(); err != nil {
				log.Error(nil, map[string]interface{}{
					"nextVersion": nextVersion,
					"migrations":  m,
					"err":         err,
				}, "error while rolling back transaction: ", err)
				return errs.Errorf("Error while rolling back transaction: %s\n", err)
			}
			return oldErr
		}

		if err = tx.Commit(); err != nil {
			log.Error(nil, map[string]interface{}{
				"migrations": m,
				"err":        err,
			}, "error during transaction commit: ", err)
			return errs.Errorf("Error during transaction commit: %s\n", err)
		}

	}

	if err != nil {
		log.Error(nil, map[string]interface{}{
			"migrations": m,
			"err":        err,
		}, "migration failed with error: ", err)
		return errs.Errorf("Migration failed with error: %s\n", err)
	}

	return nil
}

// getMigrations returns the migrations all the migrations we have.
// Add your own migration to the end of this function.
// IMPORTANT: ALWAYS APPEND AT THE END AND DON'T CHANGE THE ORDER OF MIGRATIONS!
func getMigrations() migrations {
	m := migrations{}

	// Version 0
	m = append(m, steps{executeSQLFile("000-bootstrap.sql")})

	// Version 1
	m = append(m, steps{executeSQLFile("001-common.sql")})

	// Version 2
	m = append(m, steps{executeSQLFile("002-tracker-items.sql")})

	// Version 3
	m = append(m, steps{executeSQLFile("003-login.sql")})

	// Version 4
	m = append(m, steps{executeSQLFile("004-drop-tracker-query-id.sql")})

	// Version 5
	m = append(m, steps{executeSQLFile("005-add-search-index.sql")})

	// Version 6
	m = append(m, steps{executeSQLFile("006-rename-parent-path.sql")})

	// Version 7
	m = append(m, steps{executeSQLFile("007-work-item-links.sql")})

	// Version 8
	m = append(m, steps{executeSQLFile("008-soft-delete-or-resurrect.sql")})

	// Version 9
	m = append(m, steps{executeSQLFile("009-drop-wit-trigger.sql")})

	// Version 10
	m = append(m, steps{executeSQLFile("010-comments.sql")})

	// Version 11
	m = append(m, steps{executeSQLFile("011-projects.sql")})

	// Version 12
	m = append(m, steps{executeSQLFile("012-unique-work-item-links.sql")})

	// version 13
	m = append(m, steps{executeSQLFile("013-iterations.sql")})

	// Version 14
	m = append(m, steps{executeSQLFile("014-wi-fields-index.sql")})

	// Version 15
	m = append(m, steps{executeSQLFile("015-rename-projects-to-spaces.sql")})

	// Version 16
	m = append(m, steps{executeSQLFile("016-drop-wi-links-trigger.sql")})

	// Version 17
	m = append(m, steps{executeSQLFile("017-alter-iterations.sql")})

	// Version 18
	m = append(m, steps{executeSQLFile("018-rewrite-wits.sql")})

	// Version 19
	m = append(m, steps{executeSQLFile("019-add-state-iterations.sql")})

	// Version 20
	m = append(m, steps{executeSQLFile("020-work-item-description-update-search-index.sql")})

	// Version 21
	m = append(m, steps{executeSQLFile("021-add-space-description.sql")})

	// Version 22
	m = append(m, steps{executeSQLFile("022-work-item-description-update.sql")})

	// Version 23
	m = append(m, steps{executeSQLFile("023-comment-markup.sql")})

	// Version 24
	m = append(m, steps{executeSQLFile("024-comment-markup-default.sql")})

	// Version 25
	m = append(m, steps{executeSQLFile("025-refactor-identities-users.sql")})

	// version 26
	m = append(m, steps{executeSQLFile("026-areas.sql")})

	// version 27
	m = append(m, steps{executeSQLFile("027-areas-index.sql")})

	// Version 28
	m = append(m, steps{executeSQLFile("028-identity_provider_url.sql")})

	// Version 29
	m = append(m, steps{executeSQLFile("029-iterations-parent-path-ltree.sql")})

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
		m = append(m, steps{
			func(db *sql.Tx) error {
				// Execute random go code
				return nil
			},
			executeSQLFile("YOUR_OWN_FILE.sql"),
			func(db *sql.Tx) error {
				// Execute random go code
				return nil
			},
		})
	*/

	return m
}

// executeSQLFile loads the given filename from the packaged SQL files and
// executes it on the given database
func executeSQLFile(filename string) fn {
	return func(db *sql.Tx) error {
		data, err := Asset(filename)
		if err != nil {
			return errs.WithStack(err)
		}
		_, err = db.Exec(string(data))
		return errs.WithStack(err)
	}
}

// migrateToNextVersion migrates the database to the nextVersion.
// If the database is already at nextVersion or higher, the nextVersion
// will be set to the actual next version.
func migrateToNextVersion(tx *sql.Tx, nextVersion *int64, m migrations) error {
	// Obtain exclusive transaction level advisory that doesn't depend on any table.
	// Once obtained, the lock is held for the remainder of the current transaction.
	// (There is no UNLOCK TABLE command; locks are always released at transaction end.)
	if _, err := tx.Exec("SELECT pg_advisory_xact_lock($1)", AdvisoryLockID); err != nil {
		return errs.Errorf("Failed to acquire lock: %s\n", err)
	}

	// Determine current version and adjust the outmost loop
	// iterator variable "version"
	currentVersion, err := getCurrentVersion(tx)
	if err != nil {
		return errs.WithStack(err)
	}
	*nextVersion = currentVersion + 1
	if *nextVersion >= int64(len(m)) {
		// No further updates to apply (this is NOT an error)
		log.Info(nil, map[string]interface{}{
			"pkg":            "migration",
			"nextVersion":    *nextVersion,
			"currentVersion": currentVersion,
		}, "Current version %d. Nothing to update.", currentVersion)
		return nil
	}

	log.Info(nil, map[string]interface{}{
		"pkg":            "migration",
		"nextVersion":    *nextVersion,
		"currentVersion": currentVersion,
	}, "Attempt to update DB to version ", *nextVersion)

	// Apply all the updates of the next version
	for j := range m[*nextVersion] {
		if err := m[*nextVersion][j](tx); err != nil {
			return errs.Errorf("Failed to execute migration of step %d of version %d: %s\n", j, *nextVersion, err)
		}
	}

	if _, err := tx.Exec("INSERT INTO version(version) VALUES($1)", *nextVersion); err != nil {
		return errs.Errorf("Failed to update DB to version %d: %s\n", *nextVersion, err)
	}

	log.Info(nil, map[string]interface{}{
		"pkg":            "migration",
		"nextVersion":    *nextVersion,
		"currentVersion": currentVersion,
	}, "Successfully updated DB to version ", *nextVersion)

	return nil
}

// getCurrentVersion returns the highest version from the version
// table or -1 if that table does not exist.
//
// Returning -1 simplifies the logic of the migration process because
// the next version is always the current version + 1 which results
// in -1 + 1 = 0 which is exactly what we want as the first version.
func getCurrentVersion(db *sql.Tx) (int64, error) {
	row := db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_catalog='postgres' AND table_name='version')")

	var exists bool
	if err := row.Scan(&exists); err != nil {
		return -1, errs.Errorf("Failed to scan if table \"version\" exists: %s\n", err)
	}

	if !exists {
		// table doesn't exist
		return -1, nil
	}

	row = db.QueryRow("SELECT max(version) as current FROM version")

	var current int64 = -1
	if err := row.Scan(&current); err != nil {
		return -1, errs.Errorf("Failed to scan max version in table \"version\": %s\n", err)
	}

	return current, nil
}

// BootstrapWorkItemLinking makes sure the database is populated with the correct work item link stuff (e.g. category and some basic types)
func BootstrapWorkItemLinking(ctx context.Context, linkCatRepo *link.GormWorkItemLinkCategoryRepository, linkTypeRepo *link.GormWorkItemLinkTypeRepository) error {
	if err := createOrUpdateWorkItemLinkCategory(ctx, linkCatRepo, link.SystemWorkItemLinkCategorySystem, "The system category is reserved for link types that are to be manipulated by the system only."); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdateWorkItemLinkCategory(ctx, linkCatRepo, link.SystemWorkItemLinkCategoryUser, "The user category is reserved for link types that can to be manipulated by the user."); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdateWorkItemLinkType(ctx, linkCatRepo, linkTypeRepo, link.SystemWorkItemLinkTypeBugBlocker, "One bug blocks a planner item.", link.TopologyNetwork, "blocks", "blocked by", workitem.SystemBug, workitem.SystemPlannerItem, link.SystemWorkItemLinkCategorySystem); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdateWorkItemLinkType(ctx, linkCatRepo, linkTypeRepo, link.SystemWorkItemLinkPlannerItemRelated, "One planner item or a subtype of it relates to another one.", link.TopologyNetwork, "relates to", "is related to", workitem.SystemPlannerItem, workitem.SystemPlannerItem, link.SystemWorkItemLinkCategorySystem); err != nil {
		return errs.WithStack(err)
	}
	return nil
}

func createOrUpdateWorkItemLinkCategory(ctx context.Context, linkCatRepo *link.GormWorkItemLinkCategoryRepository, name string, description string) error {
	cat, err := linkCatRepo.LoadCategoryFromDB(ctx, name)
	cause := errs.Cause(err)
	switch cause.(type) {
	case errors.NotFoundError:
		_, err := linkCatRepo.Create(ctx, &name, &description)
		if err != nil {
			return errs.WithStack(err)
		}
	case nil:
		log.Info(ctx, map[string]interface{}{
			"pkg":      "migration",
			"category": name,
		}, "Work item link category %s exists, will update/overwrite the description", name)

		cat.Description = &description
		linkCat := link.ConvertLinkCategoryFromModel(*cat)
		_, err = linkCatRepo.Save(ctx, linkCat)
		return errs.WithStack(err)
	}
	return nil
}

func createOrUpdateWorkItemLinkType(ctx context.Context, linkCatRepo *link.GormWorkItemLinkCategoryRepository, linkTypeRepo *link.GormWorkItemLinkTypeRepository, name, description, topology, forwardName, reverseName, sourceTypeName, targetTypeName, linkCatName string) error {
	cat, err := linkCatRepo.LoadCategoryFromDB(ctx, linkCatName)
	if err != nil {
		return errs.WithStack(err)
	}

	linkType, err := linkTypeRepo.LoadTypeFromDBByNameAndCategory(ctx, name, cat.ID)
	lt := link.WorkItemLinkType{
		Name:           name,
		Description:    &description,
		Topology:       topology,
		ForwardName:    forwardName,
		ReverseName:    reverseName,
		SourceTypeName: sourceTypeName,
		TargetTypeName: targetTypeName,
		LinkCategoryID: cat.ID,
	}

	cause := errs.Cause(err)
	switch cause.(type) {
	case errors.NotFoundError:
		_, err := linkTypeRepo.Create(ctx, lt.Name, lt.Description, lt.SourceTypeName, lt.TargetTypeName, lt.ForwardName, lt.ReverseName, lt.Topology, lt.LinkCategoryID)
		if err != nil {
			return errs.WithStack(err)
		}
	case nil:
		log.Info(ctx, map[string]interface{}{
			"pkg":  "migration",
			"wilt": name,
		}, "Work item link type %s exists, will update/overwrite all fields", name)

		lt.ID = linkType.ID
		lt.Version = linkType.Version
		_, err = linkTypeRepo.Save(ctx, link.ConvertLinkTypeFromModel(lt))
		return errs.WithStack(err)
	}
	return nil
}

// PopulateCommonTypes makes sure the database is populated with the correct types (e.g. bug etc.)
func PopulateCommonTypes(ctx context.Context, db *gorm.DB, witr *workitem.GormWorkItemTypeRepository) error {

	if err := createOrUpdateSystemPlannerItemType(ctx, witr, db); err != nil {
		return errs.WithStack(err)
	}
	workitem.ClearGlobalWorkItemTypeCache() // Clear the WIT cache after updating existing WITs
	if err := createOrUpdatePlannerItemExtension(workitem.SystemUserStory, ctx, witr, db); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdatePlannerItemExtension(workitem.SystemValueProposition, ctx, witr, db); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdatePlannerItemExtension(workitem.SystemFundamental, ctx, witr, db); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdatePlannerItemExtension(workitem.SystemExperience, ctx, witr, db); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdatePlannerItemExtension(workitem.SystemScenario, ctx, witr, db); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdatePlannerItemExtension(workitem.SystemFeature, ctx, witr, db); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdatePlannerItemExtension(workitem.SystemBug, ctx, witr, db); err != nil {
		return errs.WithStack(err)
	}
	workitem.ClearGlobalWorkItemTypeCache() // Clear the WIT cache after updating existing WITs
	return nil
}

func createOrUpdateSystemPlannerItemType(ctx context.Context, witr *workitem.GormWorkItemTypeRepository, db *gorm.DB) error {
	typeName := workitem.SystemPlannerItem
	stString := "string"
	stUser := "user"
	workItemTypeFields := map[string]app.FieldDefinition{
		workitem.SystemTitle:        {Type: &app.FieldType{Kind: "string"}, Required: true},
		workitem.SystemDescription:  {Type: &app.FieldType{Kind: "markup"}, Required: false},
		workitem.SystemCreator:      {Type: &app.FieldType{Kind: "user"}, Required: true},
		workitem.SystemRemoteItemID: {Type: &app.FieldType{Kind: "string"}, Required: false},
		workitem.SystemCreatedAt:    {Type: &app.FieldType{Kind: "instant"}, Required: false},
		workitem.SystemIteration:    {Type: &app.FieldType{Kind: "iteration"}, Required: false},
		workitem.SystemArea:         {Type: &app.FieldType{Kind: "area"}, Required: false},

		workitem.SystemAssignees: {
			Type: &app.FieldType{
				ComponentType: &stUser,
				Kind:          "list",
			},
			Required: false,
		},
		workitem.SystemState: {
			Type: &app.FieldType{
				BaseType: &stString,
				Kind:     "enum",
				Values: []interface{}{
					workitem.SystemStateNew,
					workitem.SystemStateOpen,
					workitem.SystemStateInProgress,
					workitem.SystemStateResolved,
					workitem.SystemStateClosed,
				},
			},
			Required: true,
		},
	}

	return createOrUpdateType(typeName, nil, workItemTypeFields, ctx, witr, db)
}

func createOrUpdatePlannerItemExtension(typeName string, ctx context.Context, witr *workitem.GormWorkItemTypeRepository, db *gorm.DB) error {
	workItemTypeFields := map[string]app.FieldDefinition{}
	extTypeName := workitem.SystemPlannerItem
	return createOrUpdateType(typeName, &extTypeName, workItemTypeFields, ctx, witr, db)
}

func createOrUpdateType(typeName string, extendedTypeName *string, fields map[string]app.FieldDefinition, ctx context.Context, witr *workitem.GormWorkItemTypeRepository, db *gorm.DB) error {
	wit, err := witr.LoadTypeFromDB(ctx, typeName)
	cause := errs.Cause(err)
	switch cause.(type) {
	case errors.NotFoundError:
		_, err := witr.Create(ctx, extendedTypeName, typeName, fields)
		if err != nil {
			return errs.WithStack(err)
		}
	case nil:
		log.Info(ctx, map[string]interface{}{
			"pkg":      "migration",
			"typeName": typeName,
		}, "Work item type %s exists, will update/overwrite the fields only and parentPath", typeName)

		path := typeName
		convertedFields, err := workitem.TEMPConvertFieldTypesToModel(fields)
		if extendedTypeName != nil {
			log.Info(ctx, map[string]interface{}{
				"pkg":              "migration",
				"typeName":         typeName,
				"extendedTypeName": *extendedTypeName,
			}, "Work item type %s extends another type %v will copy fields from the extended type", typeName, *extendedTypeName)

			extendedWit, err := witr.LoadTypeFromDB(ctx, *extendedTypeName)
			if err != nil {
				return errs.WithStack(err)
			}
			path = extendedWit.Path + workitem.GetTypePathSeparator() + path

			//load fields from the extended type
			err = loadFields(ctx, extendedWit, convertedFields)
			if err != nil {
				return errs.WithStack(err)
			}
		}

		if err != nil {
			return errs.WithStack(err)
		}
		wit.Fields = convertedFields
		wit.Path = path
		db = db.Save(wit)
		return db.Error
	}
	return nil
}

func loadFields(ctx context.Context, wit *workitem.WorkItemType, into workitem.FieldDefinitions) error {
	// copy fields from wit
	for key, value := range wit.Fields {
		// do not overwrite already defined fields in the map
		if _, exist := into[key]; !exist {
			into[key] = value
		}
	}

	return nil
}
