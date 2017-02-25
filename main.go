package main

import (
	"flag"
	"net/http"
	"os"
	"os/user"
	"time"

	"golang.org/x/net/context"

	"golang.org/x/oauth2"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"

	logrus "github.com/Sirupsen/logrus"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"
	"github.com/almighty/almighty-core/workitem/link"

	"github.com/goadesign/goa"
	"github.com/goadesign/goa/client"
	goalogrus "github.com/goadesign/goa/logging/logrus"
	"github.com/goadesign/goa/middleware"
	"github.com/goadesign/goa/middleware/gzip"
	"github.com/goadesign/goa/middleware/security/jwt"
)

var (
	// Commit current build commit set by build script
	Commit = "0"
	// BuildTime set by build script in ISO 8601 (UTC) format: YYYY-MM-DDThh:mm:ssTZD (see https://www.w3.org/TR/NOTE-datetime for details)
	BuildTime = "0"
	// StartTime in ISO 8601 (UTC) format
	StartTime = time.Now().UTC().Format("2006-01-02T15:04:05Z")
)

func main() {
	// --------------------------------------------------------------------
	// Parse flags
	// --------------------------------------------------------------------
	var configFilePath string
	var printConfig bool
	var migrateDB bool
	var scheduler *remoteworkitem.Scheduler
	flag.StringVar(&configFilePath, "config", "", "Path to the config file to read")
	flag.BoolVar(&printConfig, "printConfig", false, "Prints the config (including merged environment variables) and exits")
	flag.BoolVar(&migrateDB, "migrateDatabase", false, "Migrates the database to the newest version and exits.")
	flag.Parse()

	// Override default -config switch with environment variable only if -config switch was
	// not explicitly given via the command line.
	configSwitchIsSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "config" {
			configSwitchIsSet = true
		}
	})
	if !configSwitchIsSet {
		if envConfigPath, ok := os.LookupEnv("ALMIGHTY_CONFIG_FILE_PATH"); ok {
			configFilePath = envConfigPath
		}
	}

	var err error
	if err = configuration.Setup(configFilePath); err != nil {
		logrus.Panic(nil, map[string]interface{}{
			"configFilePath": configFilePath,
			"err":            err,
		}, "failed to setup the configuration")
	}

	if printConfig {
		os.Exit(0)
	}

	// Initialized developer mode flag for the logger
	log.InitializeLogger(configuration.IsPostgresDeveloperModeEnabled())

	printUserInfo()

	var db *gorm.DB
	for {
		db, err = gorm.Open("postgres", configuration.GetPostgresConfigString())
		if err != nil {
			db.Close()
			log.Logger().Errorf("ERROR: Unable to open connection to database %v", err)
			log.Logger().Infof("Retrying to connect in %v...", configuration.GetPostgresConnectionRetrySleep())
			time.Sleep(configuration.GetPostgresConnectionRetrySleep())
		} else {
			defer db.Close()
			break
		}
	}

	if configuration.IsPostgresDeveloperModeEnabled() {
		db = db.Debug()
	}

	if configuration.GetPostgresConnectionMaxIdle() > 0 {
		log.Logger().Infof("Configured connection pool max idle %v", configuration.GetPostgresConnectionMaxIdle())
		db.DB().SetMaxIdleConns(configuration.GetPostgresConnectionMaxIdle())
	}
	if configuration.GetPostgresConnectionMaxOpen() > 0 {
		log.Logger().Infof("Configured connection pool max open %v", configuration.GetPostgresConnectionMaxOpen())
		db.DB().SetMaxOpenConns(configuration.GetPostgresConnectionMaxOpen())
	}

	// Migrate the schema
	err = migration.Migrate(db.DB())
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed migration")
	}

	// Nothing to here except exit, since the migration is already performed.
	if migrateDB {
		os.Exit(0)
	}

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if configuration.GetPopulateCommonTypes() {
		// set a random request ID for the context
		ctx, req_id := client.ContextWithRequestID(context.Background())
		log.Debug(ctx, nil, "Initializing the population of the database... Request ID: %v", req_id)

		if err := models.Transactional(db, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(ctx, tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			log.Panic(ctx, map[string]interface{}{
				"err": err,
			}, "failed to populate common types")
		}
		if err := models.Transactional(db, func(tx *gorm.DB) error {
			return migration.BootstrapWorkItemLinking(ctx, link.NewWorkItemLinkCategoryRepository(tx), link.NewWorkItemLinkTypeRepository(tx))
		}); err != nil {
			log.Panic(ctx, map[string]interface{}{
				"err": err,
			}, "failed to bootstap work item linking")
		}
	}

	// Scheduler to fetch and import remote tracker items
	scheduler = remoteworkitem.NewScheduler(db)
	defer scheduler.Stop()
	scheduler.ScheduleAllQueries()

	// Create service
	service := goa.New("alm")

	// Mount middleware
	service.Use(middleware.RequestID())
	service.Use(middleware.LogRequest(configuration.IsPostgresDeveloperModeEnabled()))
	service.Use(gzip.Middleware(9))
	service.Use(jsonapi.ErrorHandler(service, true))
	service.Use(middleware.Recover())

	service.WithLogger(goalogrus.New(log.Logger()))

	publicKey, err := token.ParsePublicKey(configuration.GetTokenPublicKey())
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to parse public token")
	}

	// Setup Account/Login/Security
	identityRepository := account.NewIdentityRepository(db)
	userRepository := account.NewUserRepository(db)

	tokenManager := token.NewManager(publicKey)
	app.UseJWTMiddleware(service, jwt.New(publicKey, nil, app.NewJWTSecurity()))
	service.Use(login.InjectTokenManager(tokenManager))

	// Mount "login" controller
	oauth := &oauth2.Config{
		ClientID:     configuration.GetKeycloakClientID(),
		ClientSecret: configuration.GetKeycloakSecret(),
		Scopes:       []string{"user:email"},
		Endpoint:     oauth2.Endpoint{},
	}

	appDB := gormapplication.NewGormDB(db)

	loginService := login.NewKeycloakOAuthProvider(oauth, identityRepository, userRepository, tokenManager, appDB)
	loginCtrl := NewLoginController(service, loginService, tokenManager)
	app.MountLoginController(service, loginCtrl)

	// Mount "status" controller
	statusCtrl := NewStatusController(service, db)
	app.MountStatusController(service, statusCtrl)

	// Mount "workitem" controller
	workitemCtrl := NewWorkitemController(service, appDB)
	app.MountWorkitemController(service, workitemCtrl)

	// Mount "workitemtype" controller
	workitemtypeCtrl := NewWorkitemtypeController(service, appDB)
	app.MountWorkitemtypeController(service, workitemtypeCtrl)

	// Mount "work item link category" controller
	workItemLinkCategoryCtrl := NewWorkItemLinkCategoryController(service, appDB)
	app.MountWorkItemLinkCategoryController(service, workItemLinkCategoryCtrl)

	// Mount "work item link type" controller
	workItemLinkTypeCtrl := NewWorkItemLinkTypeController(service, appDB)
	app.MountWorkItemLinkTypeController(service, workItemLinkTypeCtrl)

	// Mount "work item link" controller
	workItemLinkCtrl := NewWorkItemLinkController(service, appDB)
	app.MountWorkItemLinkController(service, workItemLinkCtrl)

	// Mount "work item comments" controller
	workItemCommentsCtrl := NewWorkItemCommentsController(service, appDB)
	app.MountWorkItemCommentsController(service, workItemCommentsCtrl)

	// Mount "work item relationships links" controller
	workItemRelationshipsLinksCtrl := NewWorkItemRelationshipsLinksController(service, appDB)
	app.MountWorkItemRelationshipsLinksController(service, workItemRelationshipsLinksCtrl)

	// Mount "comments" controller
	commentsCtrl := NewCommentsController(service, appDB)
	app.MountCommentsController(service, commentsCtrl)

	// Mount "tracker" controller
	c5 := NewTrackerController(service, appDB, scheduler)
	app.MountTrackerController(service, c5)

	// Mount "trackerquery" controller
	c6 := NewTrackerqueryController(service, appDB, scheduler)
	app.MountTrackerqueryController(service, c6)

	// Mount "space" controller
	spaceCtrl := NewSpaceController(service, appDB)
	app.MountSpaceController(service, spaceCtrl)

	// Mount "user" controller
	userCtrl := NewUserController(service, appDB, tokenManager)
	app.MountUserController(service, userCtrl)

	// Mount "search" controller
	searchCtrl := NewSearchController(service, appDB)
	app.MountSearchController(service, searchCtrl)

	// Mount "indentity" controller
	identityCtrl := NewIdentityController(service, appDB)
	app.MountIdentityController(service, identityCtrl)

	// Mount "users" controller
	usersCtrl := NewUsersController(service, appDB)
	app.MountUsersController(service, usersCtrl)

	// Mount "iterations" controller
	iterationCtrl := NewIterationController(service, appDB)
	app.MountIterationController(service, iterationCtrl)

	// Mount "spaceiterations" controller
	spaceIterationCtrl := NewSpaceIterationsController(service, appDB)
	app.MountSpaceIterationsController(service, spaceIterationCtrl)

	// Mount "userspace" controller
	userspaceCtrl := NewUserspaceController(service, db)
	app.MountUserspaceController(service, userspaceCtrl)

	// Mount "render" controller
	renderCtrl := NewRenderController(service)
	app.MountRenderController(service, renderCtrl)

	// Mount "areas" controller
	areaCtrl := NewAreaController(service, appDB)
	app.MountAreaController(service, areaCtrl)

	spaceAreaCtrl := NewSpaceAreasController(service, appDB)
	app.MountSpaceAreasController(service, spaceAreaCtrl)

	log.Logger().Infoln("Git Commit SHA: ", Commit)
	log.Logger().Infoln("UTC Build Time: ", BuildTime)
	log.Logger().Infoln("UTC Start Time: ", StartTime)
	log.Logger().Infoln("Dev mode:       ", configuration.IsPostgresDeveloperModeEnabled())

	http.Handle("/api/", service.Mux)
	http.Handle("/", http.FileServer(assetFS()))
	http.Handle("/favicon.ico", http.NotFoundHandler())

	// Start http
	if err := http.ListenAndServe(configuration.GetHTTPAddress(), nil); err != nil {
		log.Error(nil, map[string]interface{}{
			"addr": configuration.GetHTTPAddress(),
			"err":  err,
		}, "unable to connect to server")
		service.LogError("startup", "err", err)
	}

}

func printUserInfo() {
	u, err := user.Current()
	if err != nil {
		log.Warn(nil, map[string]interface{}{
			"err": err,
		}, "failed to get current user")
	} else {
		log.Info(nil, map[string]interface{}{
			"username": u.Username,
			"uuid":     u.Uid,
		}, "Running as user name '%s' with UID %s.", u.Username, u.Uid)
		g, err := user.LookupGroupId(u.Gid)
		if err != nil {
			log.Warn(nil, map[string]interface{}{
				"err": err,
			}, "failed to lookup group")
		} else {
			log.Info(nil, map[string]interface{}{
				"groupname": g.Name,
				"gid":       g.Gid,
			}, "Running as as group '%s' with GID %s.", g.Name, g.Gid)
		}
	}

}
