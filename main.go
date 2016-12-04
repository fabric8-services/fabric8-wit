package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"time"

	"golang.org/x/net/context"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
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
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	if printConfig {
		fmt.Printf("%s\n", configuration.String())
		os.Exit(0)
	}

	printUserInfo()

	var db *gorm.DB
	for i := 1; i <= configuration.GetPostgresConnectionMaxRetries(); i++ {
		log.Printf("Opening DB connection attempt %d of %d\n", i, configuration.GetPostgresConnectionMaxRetries())
		db, err = gorm.Open("postgres", configuration.GetPostgresConfigString())
		if err != nil {
			db.Close()
			time.Sleep(configuration.GetPostgresConnectionRetrySleep())
		} else {
			defer db.Close()
			break
		}
	}
	if err != nil {
		panic("Could not open connection to database")
	}

	if configuration.IsPostgresDeveloperModeEnabled() {
		db = db.Debug()
	}

	// Migrate the schema
	err = migration.Migrate(db.DB())
	if err != nil {
		panic(err.Error())
	}

	// Nothing to here except exit, since the migration is already performed.
	if migrateDB {
		os.Exit(0)
	}

	// Make sure the database is populated with the correct types (e.g. system.bug etc.)
	if configuration.GetPopulateCommonTypes() {
		if err := models.Transactional(db, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(context.Background(), tx, models.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
		if err := models.Transactional(db, func(tx *gorm.DB) error {
			return migration.BootstrapWorkItemLinking(context.Background(), models.NewWorkItemLinkCategoryRepository(tx), models.NewWorkItemLinkTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
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
	service.Use(middleware.LogRequest(true))
	service.Use(gzip.Middleware(9))
	service.Use(jsonapi.ErrorHandler(service, true))
	service.Use(middleware.Recover())

	privateKey, err := token.ParsePrivateKey(configuration.GetTokenPrivateKey())
	if err != nil {
		panic(err)
	}
	publicKey, err := token.ParsePublicKey(configuration.GetTokenPublicKey())
	if err != nil {
		panic(err)
	}

	// Setup Account/Login/Security
	identityRepository := account.NewIdentityRepository(db)
	userRepository := account.NewUserRepository(db)

	tokenManager := token.NewManager(publicKey, privateKey)
	app.UseJWTMiddleware(service, jwt.New(publicKey, nil, app.NewJWTSecurity()))
	service.Use(login.InjectTokenManager(tokenManager))

	// Mount "login" controller
	oauth := &oauth2.Config{
		ClientID:     configuration.GetGithubClientID(),
		ClientSecret: configuration.GetGithubSecret(),
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}

	loginService := login.NewGitHubOAuth(oauth, identityRepository, userRepository, tokenManager)
	loginCtrl := NewLoginController(service, loginService, tokenManager)
	app.MountLoginController(service, loginCtrl)

	// Mount "status" controller
	statusCtrl := NewStatusController(service, db)
	app.MountStatusController(service, statusCtrl)

	appDB := gormapplication.NewGormDB(db)

	// Mount "workitem" controller
	workitemCtrl := NewWorkitemController(service, appDB)
	app.MountWorkitemController(service, workitemCtrl)

	workitem2Ctrl := NewWorkitem2Controller(service, appDB)
	app.MountWorkitem2Controller(service, workitem2Ctrl)

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

	// Mount "tracker" controller
	c5 := NewTrackerController(service, appDB, scheduler)
	app.MountTrackerController(service, c5)

	// Mount "trackerquery" controller
	c6 := NewTrackerqueryController(service, appDB, scheduler)
	app.MountTrackerqueryController(service, c6)

	// Mount "user" controller
	userCtrl := NewUserController(service, identityRepository, tokenManager)
	app.MountUserController(service, userCtrl)

	// Mount "search" controller
	searchCtrl := NewSearchController(service, appDB)
	app.MountSearchController(service, searchCtrl)

	// Mount "indentity" controller
	identityCtrl := NewIdentityController(service, appDB)
	app.MountIdentityController(service, identityCtrl)

	fmt.Println("Git Commit SHA: ", Commit)
	fmt.Println("UTC Build Time: ", BuildTime)
	fmt.Println("UTC Start Time: ", StartTime)
	fmt.Println("Dev mode:       ", configuration.IsPostgresDeveloperModeEnabled())

	http.Handle("/api/", service.Mux)
	http.Handle("/", http.FileServer(assetFS()))
	http.Handle("/favicon.ico", http.NotFoundHandler())

	// Start http
	if err := http.ListenAndServe(configuration.GetHTTPAddress(), nil); err != nil {
		service.LogError("startup", "err", err)
	}

}

func printUserInfo() {
	u, err := user.Current()
	if err != nil {
		log.Printf("Failed to get current user: %s", err.Error())
	} else {
		log.Printf("Running as user name \"%s\" with UID %s.\n", u.Username, u.Uid)
		/*
			g, err := user.LookupGroupId(u.Gid)
			if err != nil {
				fmt.Printf("Failed to lookup group: %", err.Error())
			} else {
				fmt.Printf("Running with group \"%s\" with GID %s.\n", g.Name, g.Gid)
			}
		*/
	}
}
