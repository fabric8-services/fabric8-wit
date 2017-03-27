package migration

import (
	"bufio"
	"bytes"
	"database/sql"
	"net/http"
	"net/url"
	"sync"
	"text/template"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/space"
	"github.com/almighty/almighty-core/workitem"
	"github.com/almighty/almighty-core/workitem/link"

	"fmt"

	"github.com/goadesign/goa"
	"github.com/goadesign/goa/client"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
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

// mutex variable to lock/unlock the population of common types
var populateLocker = &sync.Mutex{}

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
				"next_version": nextVersion,
				"migrations":   m,
				"err":          err,
			}, "Rolling back transaction due to: %v", err)

			if err = tx.Rollback(); err != nil {
				log.Error(nil, map[string]interface{}{
					"next_version": nextVersion,
					"migrations":   m,
					"err":          err,
				}, "error while rolling back transaction: ", err)
				return errs.Errorf("Error while rolling back transaction: %s\n", err)
			}
			return oldErr
		}

		if err = tx.Commit(); err != nil {
			log.Error(nil, map[string]interface{}{
				"migrations": m,
				"err":        err,
			}, "error during transaction commit: %v", err)
			return errs.Errorf("Error during transaction commit: %s\n", err)
		}

	}

	if err != nil {
		log.Error(nil, map[string]interface{}{
			"migrations": m,
			"err":        err,
		}, "migration failed with error: %v", err)
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
	m = append(m, steps{executeSQLFile("028-identity-provider_url.sql")})

	// Version 29
	m = append(m, steps{executeSQLFile("029-identities-foreign-key.sql")})

	// Version 30
	m = append(m, steps{executeSQLFile("030-indentities-unique-index.sql")})

	// Version 31
	m = append(m, steps{executeSQLFile("031-iterations-parent-path-ltree.sql")})

	// Version 32
	m = append(m, steps{executeSQLFile("032-add-foreign-key-space-id.sql")})

	// Version 33
	m = append(m, steps{executeSQLFile("033-add-space-id-wilt.sql", space.SystemSpace.String(), "system.space", "Description of the space")})

	// Version 34
	m = append(m, steps{executeSQLFile("034-space-owner.sql")})

	// Version 35
	m = append(m, steps{executeSQLFile("035-wit-to-use-uuid.sql",
		workitem.SystemPlannerItem.String(),
		workitem.SystemUserStory.String(),
		workitem.SystemValueProposition.String(),
		workitem.SystemFundamental.String(),
		workitem.SystemExperience.String(),
		workitem.SystemFeature.String(),
		workitem.SystemScenario.String(),
		workitem.SystemBug.String())})

	// Version 36
	m = append(m, steps{executeSQLFile("036-add-icon-to-wit.sql")})

	// version 37
	m = append(m, steps{executeSQLFile("037-work-item-revisions.sql")})

	// Version 38
	m = append(m, steps{executeSQLFile("038-comment-revisions.sql")})

	// Version 39
	m = append(m, steps{executeSQLFile("039-comment-revisions-parentid.sql")})

	// Version 40
	m = append(m, steps{executeSQLFile("040-add-space-id-wi-wit-tq.sql", space.SystemSpace.String())})

	// version 41
	m = append(m, steps{executeSQLFile("041-unique-area-name-create-new-area.sql")})

	// Version 42
	m = append(m, steps{executeSQLFile("042-work-item-link-revisions.sql")})

	// Version 43
	m = append(m, steps{executeSQLFile("043-space-resources.sql")})

	// Version 44
	m = append(m, steps{executeSQLFile("044-add-contextinfo-column-users.sql")})

	// Version 45
	m = append(m, steps{executeSQLFile("045-adds-order-to-existing-wi.sql")})

	// Version 46
	m = append(m, steps{executeSQLFile("046-oauth-states.sql")})

	// Version 47
	m = append(m, steps{executeSQLFile("047-codebases.sql")})

	// Version 48
	m = append(m, steps{executeSQLFile("048-unique-iteration-name-create-new-iteration.sql")})
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
// executes it on the given database. Golang text/template module is used
// to handle all the optional arguments passed to the sql files
func executeSQLFile(filename string, args ...string) fn {
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
			"next_version":    *nextVersion,
			"current_version": currentVersion,
		}, "Current version %d. Nothing to update.", currentVersion)
		return nil
	}

	log.Info(nil, map[string]interface{}{
		"next_version":    *nextVersion,
		"current_version": currentVersion,
	}, "Attempt to update DB to version %v", *nextVersion)

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
		"next_version":    *nextVersion,
		"current_version": currentVersion,
	}, "Successfully updated DB to version %v", *nextVersion)

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

// NewMigrationContext aims to create a new goa context where to initialize the
// request and req_id context keys.
// NOTE: We need this function to initialize the goa.ContextRequest
func NewMigrationContext(ctx context.Context) context.Context {
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx = goa.NewContext(ctx, nil, req, params)
	// set a random request ID for the context
	var req_id string
	ctx, req_id = client.ContextWithRequestID(ctx)

	log.Debug(ctx, nil, "Initialized the migration context with Request ID: %v", req_id)

	return ctx
}

// BootstrapWorkItemLinking makes sure the database is populated with the correct work item link stuff (e.g. category and some basic types)
func BootstrapWorkItemLinking(ctx context.Context, linkCatRepo *link.GormWorkItemLinkCategoryRepository, spaceRepo *space.GormRepository, linkTypeRepo *link.GormWorkItemLinkTypeRepository) error {
	populateLocker.Lock()
	defer populateLocker.Unlock()
	if err := createOrUpdateSpace(ctx, spaceRepo, space.SystemSpace, "The system space is reserved for spaces that can to be manipulated by the user."); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdateWorkItemLinkCategory(ctx, linkCatRepo, link.SystemWorkItemLinkCategorySystem, "The system category is reserved for link types that are to be manipulated by the system only."); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdateWorkItemLinkCategory(ctx, linkCatRepo, link.SystemWorkItemLinkCategoryUser, "The user category is reserved for link types that can to be manipulated by the user."); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdateWorkItemLinkType(ctx, linkCatRepo, linkTypeRepo, spaceRepo, link.SystemWorkItemLinkTypeBugBlocker, "One bug blocks a planner item.", link.TopologyNetwork, "blocks", "blocked by", workitem.SystemBug, workitem.SystemPlannerItem, link.SystemWorkItemLinkCategorySystem, space.SystemSpace); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdateWorkItemLinkType(ctx, linkCatRepo, linkTypeRepo, spaceRepo, link.SystemWorkItemLinkPlannerItemRelated, "One planner item or a subtype of it relates to another one.", link.TopologyNetwork, "relates to", "is related to", workitem.SystemPlannerItem, workitem.SystemPlannerItem, link.SystemWorkItemLinkCategorySystem, space.SystemSpace); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdateWorkItemLinkType(ctx, linkCatRepo, linkTypeRepo, spaceRepo, link.SystemWorkItemLinkTypeParentChild, "One planner item or a subtype of it which is a parent of another one.", link.TopologyTree, "parent of", "child of", workitem.SystemPlannerItem, workitem.SystemPlannerItem, link.SystemWorkItemLinkCategorySystem, space.SystemSpace); err != nil {
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
			"category": name,
		}, "Work item link category %s exists, will update/overwrite the description", name)

		cat.Description = &description
		_, err = linkCatRepo.Save(ctx, *cat)
		return errs.WithStack(err)
	}
	return nil
}

func createOrUpdateSpace(ctx context.Context, spaceRepo *space.GormRepository, id uuid.UUID, description string) error {
	s, err := spaceRepo.Load(ctx, id)
	cause := errs.Cause(err)
	newSpace := &space.Space{
		Description: description,
		Name:        "system.space",
		ID:          id,
	}
	switch cause.(type) {
	case errors.NotFoundError:
		log.Info(ctx, map[string]interface{}{
			"pkg":      "migration",
			"space_id": id,
		}, "space %s will be created", id)
		_, err := spaceRepo.Create(ctx, newSpace)
		if err != nil {
			return errs.Wrapf(err, "failed to create space %s", id)
		}
	case nil:
		log.Info(ctx, map[string]interface{}{
			"pkg":      "migration",
			"space_id": id,
		}, "space %s exists, will update/overwrite the description", id)

		s.Description = description
		_, err = spaceRepo.Save(ctx, s)
		return errs.WithStack(err)
	}
	return nil
}

func createSpace(ctx context.Context, spaceRepo *space.GormRepository, id uuid.UUID, description string) error {
	_, err := spaceRepo.Load(ctx, id)
	cause := errs.Cause(err)
	newSpace := &space.Space{
		Description: description,
		Name:        "system.space",
		ID:          id,
	}
	switch cause.(type) {
	case errors.NotFoundError:
		log.Info(ctx, map[string]interface{}{
			"pkg":      "migration",
			"space_id": id,
		}, "space %s will be created", id)
		_, err := spaceRepo.Create(ctx, newSpace)
		if err != nil {
			return errs.Wrapf(err, "failed to create space %s", id)
		}
	}
	return nil
}

func createOrUpdateWorkItemLinkType(ctx context.Context, linkCatRepo *link.GormWorkItemLinkCategoryRepository, linkTypeRepo *link.GormWorkItemLinkTypeRepository, spaceRepo *space.GormRepository, name, description, topology, forwardName, reverseName string, sourceTypeID, targetTypeID uuid.UUID, linkCatName string, spaceId uuid.UUID) error {
	cat, err := linkCatRepo.LoadCategoryFromDB(ctx, linkCatName)
	if err != nil {
		return errs.WithStack(err)
	}

	space, err := spaceRepo.Load(ctx, spaceId)
	if err != nil {
		return errs.WithStack(err)
	}

	existingLinkType, err := linkTypeRepo.LoadTypeFromDBByNameAndCategory(ctx, name, cat.ID)
	linkType := link.WorkItemLinkType{
		Name:           name,
		Description:    &description,
		Topology:       topology,
		ForwardName:    forwardName,
		ReverseName:    reverseName,
		SourceTypeID:   sourceTypeID,
		TargetTypeID:   targetTypeID,
		LinkCategoryID: cat.ID,
		SpaceID:        space.ID,
	}

	cause := errs.Cause(err)
	switch cause.(type) {
	case errors.NotFoundError:
		_, err := linkTypeRepo.Create(ctx,
			linkType.Name,
			linkType.Description,
			linkType.SourceTypeID,
			linkType.TargetTypeID,
			linkType.ForwardName,
			linkType.ReverseName,
			linkType.Topology,
			linkType.LinkCategoryID,
			linkType.SpaceID)
		if err != nil {
			return errs.WithStack(err)
		}
	case nil:
		log.Info(ctx, map[string]interface{}{
			"wilt": name,
		}, "Work item link type %s exists, will update/overwrite all fields", name)
		linkType.ID = existingLinkType.ID
		linkType.Version = existingLinkType.Version
		_, err = linkTypeRepo.Save(ctx, linkType)
		return errs.WithStack(err)
	}
	return nil
}

// PopulateCommonTypes makes sure the database is populated with the correct types (e.g. bug etc.)
func PopulateCommonTypes(ctx context.Context, db *gorm.DB, witr *workitem.GormWorkItemTypeRepository) error {
	populateLocker.Lock()
	defer populateLocker.Unlock()
	if err := createSpace(ctx, space.NewRepository(db), space.SystemSpace, "The system space is reserved for spaces that can to be manipulated by the user."); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdateSystemPlannerItemType(ctx, witr, db, space.SystemSpace); err != nil {
		return errs.WithStack(err)
	}
	workitem.ClearGlobalWorkItemTypeCache() // Clear the WIT cache after updating existing WITs
	if err := createOrUpdatePlannerItemExtension(workitem.SystemUserStory, "User Story", "Desciption for User Story", "fa-map-marker", ctx, witr, db, space.SystemSpace); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdatePlannerItemExtension(workitem.SystemValueProposition, "Value Proposition", "Description for value proposition", "fa-gift", ctx, witr, db, space.SystemSpace); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdatePlannerItemExtension(workitem.SystemFundamental, "Fundamental", "Description for Fundamental", "fa-bank", ctx, witr, db, space.SystemSpace); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdatePlannerItemExtension(workitem.SystemExperience, "Experience", "Description for Experience", "fa-map", ctx, witr, db, space.SystemSpace); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdatePlannerItemExtension(workitem.SystemScenario, "Scenario", "Description for Scenario", "fa-adjust", ctx, witr, db, space.SystemSpace); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdatePlannerItemExtension(workitem.SystemFeature, "Feature", "Description for Feature", "fa-mouse-pointer", ctx, witr, db, space.SystemSpace); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdatePlannerItemExtension(workitem.SystemBug, "Bug", "Description for Bug", "fa-bug", ctx, witr, db, space.SystemSpace); err != nil {
		return errs.WithStack(err)
	}
	workitem.ClearGlobalWorkItemTypeCache() // Clear the WIT cache after updating existing WITs
	return nil
}

func createOrUpdateSystemPlannerItemType(ctx context.Context, witr *workitem.GormWorkItemTypeRepository, db *gorm.DB, spaceID uuid.UUID) error {
	fmt.Println("Creating or updating planner item type...")
	typeID := workitem.SystemPlannerItem
	typeName := "Planner Item"
	description := "Description for Planner Item"
	icon := "fa-bookmark"
	workItemTypeFields := map[string]workitem.FieldDefinition{
		workitem.SystemTitle:        {Type: workitem.SimpleType{Kind: "string"}, Required: true, Label: "Title", Description: "The title text of the work item"},
		workitem.SystemDescription:  {Type: workitem.SimpleType{Kind: "markup"}, Required: false, Label: "Description", Description: "A descriptive text of the work item"},
		workitem.SystemCreator:      {Type: workitem.SimpleType{Kind: "user"}, Required: true, Label: "Creator", Description: "The user that created the work item"},
		workitem.SystemRemoteItemID: {Type: workitem.SimpleType{Kind: "string"}, Required: false, Label: "Remote item", Description: "The ID of the remote work item"},
		workitem.SystemCreatedAt:    {Type: workitem.SimpleType{Kind: "instant"}, Required: false, Label: "Created at", Description: "The date and time when the work item was created"},
		workitem.SystemUpdatedAt:    {Type: workitem.SimpleType{Kind: "instant"}, Required: false, Label: "Updated at", Description: "The date and time when the work item was last updated"},
		workitem.SystemOrder:        {Type: workitem.SimpleType{Kind: "float"}, Required: false, Label: "Execution Order", Description: "Execution Order of the workitem."},
		workitem.SystemIteration:    {Type: workitem.SimpleType{Kind: "iteration"}, Required: false, Label: "Iteration", Description: "The iteration to which the work item belongs"},
		workitem.SystemArea:         {Type: workitem.SimpleType{Kind: "area"}, Required: false, Label: "Area", Description: "The area to which the work item belongs"},
		workitem.SystemCodebase:     {Type: workitem.SimpleType{Kind: "codebase"}, Required: false, Label: "Codebase", Description: "Contains codebase attributes to which this WI belongs to"},
		workitem.SystemAssignees: {
			Type: &workitem.ListType{
				SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
				ComponentType: workitem.SimpleType{Kind: workitem.KindUser}},
			Required:    false,
			Label:       "Assignees",
			Description: "The users that are assigned to the work item",
		},
		workitem.SystemState: {
			Type: &workitem.EnumType{
				SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
				BaseType:   workitem.SimpleType{Kind: workitem.KindString},
				Values: []interface{}{
					workitem.SystemStateNew,
					workitem.SystemStateOpen,
					workitem.SystemStateInProgress,
					workitem.SystemStateResolved,
					workitem.SystemStateClosed,
				},
			},

			Required:    true,
			Label:       "State",
			Description: "The state of the work item",
		},
	}
	return createOrUpdateType(typeID, spaceID, typeName, description, nil, workItemTypeFields, icon, ctx, witr, db)
}

func createOrUpdatePlannerItemExtension(typeID uuid.UUID, name string, description string, icon string, ctx context.Context, witr *workitem.GormWorkItemTypeRepository, db *gorm.DB, spaceID uuid.UUID) error {
	workItemTypeFields := map[string]workitem.FieldDefinition{}
	extTypeName := workitem.SystemPlannerItem
	return createOrUpdateType(typeID, spaceID, name, description, &extTypeName, workItemTypeFields, icon, ctx, witr, db)
}

func createOrUpdateType(typeID uuid.UUID, spaceID uuid.UUID, name string, description string, extendedTypeID *uuid.UUID, fields map[string]workitem.FieldDefinition, icon string, ctx context.Context, witr *workitem.GormWorkItemTypeRepository, db *gorm.DB) error {
	fmt.Println("Creating or updating planner item types...")
	wit, err := witr.LoadTypeFromDB(ctx, typeID)
	cause := errs.Cause(err)
	switch cause.(type) {
	case errors.NotFoundError:
		_, err := witr.Create(ctx, spaceID, &typeID, extendedTypeID, name, &description, icon, fields)
		if err != nil {
			return errs.WithStack(err)
		}
	case nil:
		log.Info(ctx, map[string]interface{}{
			"type_id": typeID,
		}, "Work item type %s exists, will update/overwrite the fields, name, icon, description and parentPath", typeID.String())

		path := workitem.LtreeSafeID(typeID)
		if extendedTypeID != nil {
			log.Info(ctx, map[string]interface{}{
				"type_id":          typeID,
				"extended_type_id": *extendedTypeID,
			}, "Work item type %v extends another type %v will copy fields from the extended type", typeID, *extendedTypeID)

			extendedWit, err := witr.LoadTypeFromDB(ctx, *extendedTypeID)
			if err != nil {
				return errs.WithStack(err)
			}
			path = extendedWit.Path + workitem.GetTypePathSeparator() + path

			//load fields from the extended type
			err = loadFields(ctx, extendedWit, fields)
			if err != nil {
				return errs.WithStack(err)
			}
		}

		if err != nil {
			return errs.WithStack(err)
		}
		wit.Name = name
		wit.Description = &description
		wit.Icon = icon
		wit.Fields = fields
		wit.Path = path
		db = db.Save(wit)
		return db.Error
	}
	fmt.Println("Creating or updating planner item type done.")

	return nil
}

func loadFields(ctx context.Context, wit *workitem.WorkItemType, into workitem.FieldDefinitions) error {
	// copy fields from wit
	for key, value := range wit.Fields {
		// do not overwrite already defined fields in the map
		if _, exist := into[key]; !exist {
			into[key] = value
		} else {
			// If field already exist, overwrite only the label and description
			into[key] = workitem.FieldDefinition{
				Label:       value.Label,
				Description: value.Description,
				Required:    into[key].Required,
				Type:        into[key].Type,
			}
		}
	}

	return nil
}
