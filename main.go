package main

import (
	"flag"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"context"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/controller"
	witmiddleware "github.com/fabric8-services/fabric8-wit/goamiddleware"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/models"
	"github.com/fabric8-services/fabric8-wit/notification"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/space/authz"
	"github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"

	"github.com/goadesign/goa"
	"github.com/goadesign/goa/logging/logrus"
	"github.com/goadesign/goa/middleware"
	"github.com/goadesign/goa/middleware/gzip"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
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
		if envConfigPath, ok := os.LookupEnv("F8_CONFIG_FILE_PATH"); ok {
			configFilePath = envConfigPath
		}
	}

	config, err := configuration.New(configFilePath)
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"config_file_path": configFilePath,
			"err":              err,
		}, "failed to setup the configuration")
	}

	if printConfig {
		os.Exit(0)
	}

	// Initialized developer mode flag and log level for the logger
	log.InitializeLogger(config.IsLogJSON(), config.GetLogLevel())

	printUserInfo()

	var db *gorm.DB
	for {
		db, err = gorm.Open("postgres", config.GetPostgresConfigString())
		if err != nil {
			db.Close()
			log.Logger().Errorf("ERROR: Unable to open connection to database %v", err)
			log.Logger().Infof("Retrying to connect in %v...", config.GetPostgresConnectionRetrySleep())
			time.Sleep(config.GetPostgresConnectionRetrySleep())
		} else {
			defer db.Close()
			break
		}
	}

	if config.IsPostgresDeveloperModeEnabled() && log.IsDebug() {
		db = db.Debug()
	}

	if config.GetPostgresConnectionMaxIdle() > 0 {
		log.Logger().Infof("Configured connection pool max idle %v", config.GetPostgresConnectionMaxIdle())
		db.DB().SetMaxIdleConns(config.GetPostgresConnectionMaxIdle())
	}
	if config.GetPostgresConnectionMaxOpen() > 0 {
		log.Logger().Infof("Configured connection pool max open %v", config.GetPostgresConnectionMaxOpen())
		db.DB().SetMaxOpenConns(config.GetPostgresConnectionMaxOpen())
	}

	// Set the database transaction timeout
	application.SetDatabaseTransactionTimeout(config.GetPostgresTransactionTimeout())

	// Migrate the schema
	err = migration.Migrate(db.DB(), config.GetPostgresDatabase())
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
	if config.GetPopulateCommonTypes() {
		ctx := migration.NewMigrationContext(context.Background())

		if err := models.Transactional(db, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(ctx, tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			log.Panic(ctx, map[string]interface{}{
				"err": err,
			}, "failed to populate common types")
		}
		if err := models.Transactional(db, func(tx *gorm.DB) error {
			return migration.BootstrapWorkItemLinking(ctx, link.NewWorkItemLinkCategoryRepository(tx), space.NewRepository(tx), link.NewWorkItemLinkTypeRepository(tx))
		}); err != nil {
			log.Panic(ctx, map[string]interface{}{
				"err": err,
			}, "failed to bootstap work item linking")
		}
	}

	// Create service
	service := goa.New("wit")

	// Mount middleware
	service.Use(middleware.RequestID())
	// Use our own log request to inject identity id and modify other properties
	service.Use(gzip.Middleware(9))
	service.Use(jsonapi.ErrorHandler(service, true))
	service.Use(middleware.Recover())

	service.WithLogger(goalogrus.New(log.Logger()))

	// Setup Account/Login/Security
	identityRepository := account.NewIdentityRepository(db)
	userRepository := account.NewUserRepository(db)

	var notificationChannel notification.Channel = &notification.DevNullChannel{}
	if config.GetNotificationServiceURL() != "" {
		log.Logger().Infof("Enabling Notification service %v", config.GetNotificationServiceURL())
		channel, err := notification.NewServiceChannel(config)
		if err != nil {
			log.Panic(nil, map[string]interface{}{
				"err": err,
				"url": config.GetNotificationServiceURL(),
			}, "failed to parse notification service url")
		}
		notificationChannel = channel
	}

	appDB := gormapplication.NewGormDB(db)

	tokenManager, err := token.NewManager(config)
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to create token manager")
	}
	// Middleware that extracts and stores the token in the context
	jwtMiddlewareTokenContext := witmiddleware.TokenContext(tokenManager.PublicKeys(), nil, app.NewJWTSecurity())
	service.Use(jwtMiddlewareTokenContext)

	service.Use(login.InjectTokenManager(tokenManager))
	service.Use(log.LogRequest(config.IsPostgresDeveloperModeEnabled()))
	app.UseJWTMiddleware(service, goajwt.New(tokenManager.PublicKeys(), nil, app.NewJWTSecurity()))

	spaceAuthzService := authz.NewAuthzService(config)
	service.Use(authz.InjectAuthzService(spaceAuthzService))

	loginService := login.NewKeycloakOAuthProvider(identityRepository, userRepository, tokenManager, appDB)
	loginCtrl := controller.NewLoginController(service, loginService, config, identityRepository)
	app.MountLoginController(service, loginCtrl)

	logoutCtrl := controller.NewLogoutController(service, config)
	app.MountLogoutController(service, logoutCtrl)

	// Mount "status" controller
	statusCtrl := controller.NewStatusController(service, db)
	app.MountStatusController(service, statusCtrl)

	// Mount "workitem" controller
	//workitemCtrl := controller.NewWorkitemController(service, appDB, config)
	workitemCtrl := controller.NewNotifyingWorkitemController(service, appDB, notificationChannel, config)
	app.MountWorkitemController(service, workitemCtrl)

	// Mount "named workitem" controller
	namedWorkitemsCtrl := controller.NewNamedWorkItemsController(service, appDB)
	app.MountNamedWorkItemsController(service, namedWorkitemsCtrl)

	// Mount "workitems" controller
	//workitemsCtrl := controller.NewWorkitemsController(service, appDB, config)
	workitemsCtrl := controller.NewNotifyingWorkitemsController(service, appDB, notificationChannel, config)
	app.MountWorkitemsController(service, workitemsCtrl)

	// Mount "workitemtype" controller
	workitemtypeCtrl := controller.NewWorkitemtypeController(service, appDB, config)
	app.MountWorkitemtypeController(service, workitemtypeCtrl)

	// Mount "work item link category" controller
	workItemLinkCategoryCtrl := controller.NewWorkItemLinkCategoryController(service, appDB)
	app.MountWorkItemLinkCategoryController(service, workItemLinkCategoryCtrl)

	// Mount "work item link type" controller
	workItemLinkTypeCtrl := controller.NewWorkItemLinkTypeController(service, appDB, config)
	app.MountWorkItemLinkTypeController(service, workItemLinkTypeCtrl)

	// Mount "work item link" controller
	workItemLinkCtrl := controller.NewWorkItemLinkController(service, appDB, config)
	app.MountWorkItemLinkController(service, workItemLinkCtrl)

	// Mount "work item comments" controller
	//workItemCommentsCtrl := controller.NewWorkItemCommentsController(service, appDB, config)
	workItemCommentsCtrl := controller.NewNotifyingWorkItemCommentsController(service, appDB, notificationChannel, config)
	app.MountWorkItemCommentsController(service, workItemCommentsCtrl)

	// Mount "work item relationships links" controller
	workItemRelationshipsLinksCtrl := controller.NewWorkItemRelationshipsLinksController(service, appDB, config)
	app.MountWorkItemRelationshipsLinksController(service, workItemRelationshipsLinksCtrl)

	// Mount "comments" controller
	//commentsCtrl := controller.NewCommentsController(service, appDB, config)
	commentsCtrl := controller.NewNotifyingCommentsController(service, appDB, notificationChannel, config)
	app.MountCommentsController(service, commentsCtrl)

	// Mount "work item labels relationships" controller
	workItemLabelCtrl := controller.NewWorkItemLabelsController(service, appDB, config)
	app.MountWorkItemLabelsController(service, workItemLabelCtrl)

	if config.GetFeatureWorkitemRemote() {
		// Scheduler to fetch and import remote tracker items
		scheduler = remoteworkitem.NewScheduler(db)
		defer scheduler.Stop()

		accessTokens := controller.GetAccessTokens(config)
		scheduler.ScheduleAllQueries(service.Context, accessTokens)

		// Mount "tracker" controller
		c5 := controller.NewTrackerController(service, appDB, scheduler, config)
		app.MountTrackerController(service, c5)

		// Mount "trackerquery" controller
		c6 := controller.NewTrackerqueryController(service, appDB, scheduler, config)
		app.MountTrackerqueryController(service, c6)
	}

	// Mount "space" controller
	spaceCtrl := controller.NewSpaceController(service, appDB, config, auth.NewAuthzResourceManager(config))
	app.MountSpaceController(service, spaceCtrl)

	// Mount "user" controller
	userCtrl := controller.NewUserController(service, config)
	if config.GetTenantServiceURL() != "" {
		log.Logger().Infof("Enabling Init Tenant service %v", config.GetTenantServiceURL())
		userCtrl.InitTenant = account.NewInitTenant(config)
	}
	app.MountUserController(service, userCtrl)

	userServiceCtrl := controller.NewUserServiceController(service)
	userServiceCtrl.UpdateTenant = account.NewUpdateTenant(config)
	userServiceCtrl.CleanTenant = account.NewCleanTenant(config)
	userServiceCtrl.ShowTenant = account.NewShowTenant(config)
	app.MountUserServiceController(service, userServiceCtrl)

	// Mount "search" controller
	searchCtrl := controller.NewSearchController(service, appDB, config)
	app.MountSearchController(service, searchCtrl)

	// Mount "users" controller
	usersCtrl := controller.NewUsersController(service, appDB, config)
	app.MountUsersController(service, usersCtrl)

	// Mount "labels" controller
	labelCtrl := controller.NewLabelController(service, appDB, config)
	app.MountLabelController(service, labelCtrl)

	// Mount "iterations" controller
	iterationCtrl := controller.NewIterationController(service, appDB, config)
	app.MountIterationController(service, iterationCtrl)

	// Mount "spaceiterations" controller
	spaceIterationCtrl := controller.NewSpaceIterationsController(service, appDB, config)
	app.MountSpaceIterationsController(service, spaceIterationCtrl)

	// Mount "userspace" controller
	userspaceCtrl := controller.NewUserspaceController(service, db)
	app.MountUserspaceController(service, userspaceCtrl)

	// Mount "render" controller
	renderCtrl := controller.NewRenderController(service)
	app.MountRenderController(service, renderCtrl)

	// Mount "areas" controller
	areaCtrl := controller.NewAreaController(service, appDB, config)
	app.MountAreaController(service, areaCtrl)

	spaceAreaCtrl := controller.NewSpaceAreasController(service, appDB, config)
	app.MountSpaceAreasController(service, spaceAreaCtrl)

	filterCtrl := controller.NewFilterController(service, config)
	app.MountFilterController(service, filterCtrl)

	// Mount "namedspaces" controller
	namedSpacesCtrl := controller.NewNamedspacesController(service, appDB)
	app.MountNamedspacesController(service, namedSpacesCtrl)

	// Mount "plannerBacklog" controller
	plannerBacklogCtrl := controller.NewPlannerBacklogController(service, appDB, config)
	app.MountPlannerBacklogController(service, plannerBacklogCtrl)

	// Mount "codebase" controller
	codebaseCtrl := controller.NewCodebaseController(service, appDB, config)
	codebaseCtrl.ShowTenant = account.NewShowTenant(config)
	codebaseCtrl.NewCheClient = controller.NewDefaultCheClient(config)

	app.MountCodebaseController(service, codebaseCtrl)

	// Mount "spacecodebases" controller
	spaceCodebaseCtrl := controller.NewSpaceCodebasesController(service, appDB)
	app.MountSpaceCodebasesController(service, spaceCodebaseCtrl)

	// Mount "collaborators" controller
	collaboratorsCtrl := controller.NewCollaboratorsController(service, config)
	app.MountCollaboratorsController(service, collaboratorsCtrl)

	// Mount "space template" controller
	spaceTemplateCtrl := controller.NewSpaceTemplateController(service, appDB)
	app.MountSpaceTemplateController(service, spaceTemplateCtrl)

	// Mount "type group" controller with "show" action
	workItemTypeGroupCtrl := controller.NewWorkItemTypeGroupController(service, appDB)
	app.MountWorkItemTypeGroupController(service, workItemTypeGroupCtrl)

	// Mount "type groups" controller with "list" action
	workItemTypeGroupsCtrl := controller.NewWorkItemTypeGroupsController(service, appDB)
	app.MountWorkItemTypeGroupsController(service, workItemTypeGroupsCtrl)

	log.Logger().Infoln("Git Commit SHA: ", controller.Commit)
	log.Logger().Infoln("UTC Build Time: ", controller.BuildTime)
	log.Logger().Infoln("UTC Start Time: ", controller.StartTime)
	log.Logger().Infoln("Dev mode:       ", config.IsPostgresDeveloperModeEnabled())
	log.Logger().Infoln("GOMAXPROCS:     ", runtime.GOMAXPROCS(-1))
	log.Logger().Infoln("NumCPU:         ", runtime.NumCPU())

	http.Handle("/api/", service.Mux)
	http.Handle("/", http.FileServer(assetFS()))
	http.Handle("/favicon.ico", http.NotFoundHandler())

	// Start/mount metrics http
	if config.GetHTTPAddress() == config.GetMetricsHTTPAddress() {
		http.Handle("/metrics", prometheus.Handler())
	} else {
		go func(metricAddress string) {
			mx := http.NewServeMux()
			mx.Handle("/metrics", prometheus.Handler())
			if err := http.ListenAndServe(metricAddress, mx); err != nil {
				log.Error(nil, map[string]interface{}{
					"addr": metricAddress,
					"err":  err,
				}, "unable to connect to metrics server")
				service.LogError("startup", "err", err)
			}
		}(config.GetMetricsHTTPAddress())
	}

	// Start http
	if err := http.ListenAndServe(config.GetHTTPAddress(), nil); err != nil {
		log.Error(nil, map[string]interface{}{
			"addr": config.GetHTTPAddress(),
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
