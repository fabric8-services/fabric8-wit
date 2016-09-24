package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"time"

	"golang.org/x/net/context"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/almighty/almighty-core/transaction"
	token "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware"
	"github.com/goadesign/goa/middleware/security/jwt"
	"github.com/spf13/viper"
)

var (
	// Commit current build commit set by build script
	Commit = "0"
	// BuildTime set by build script
	BuildTime = "0"
)

func main() {
	// --------------------------------------------------------------------
	// Parse flags
	// --------------------------------------------------------------------
	var configFilePath string
	var printConfig bool
	var scheduler *remoteworkitem.Scheduler
	flag.StringVar(&configFilePath, "config", configuration.DefaultConfigFilePath, "Path to the config file to read")
	flag.BoolVar(&printConfig, "printConfig", false, "Prints the config (including merged environment variables) and exits")
	flag.Parse()

	// Override -config switch with environment variable if provided.
	if envConfigPath, ok := os.LookupEnv("ALMIGHTY_CONFIG_FILE_PATH"); ok {
		configFilePath = envConfigPath
	}

	var err error
	if err = configuration.SetupConfiguration(configFilePath); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	if printConfig {
		fmt.Printf("%s\n", configuration.GetConfiguration())
		os.Exit(0)
	}

	printUserInfo()

	var db *gorm.DB
	for i := 1; i <= viper.GetInt("postgres.connection.maxretries"); i++ {
		fmt.Printf("Opening DB connection attempt %d of %d\n", i, viper.GetInt("postgres.connection.maxretries"))
		db, err = gorm.Open("postgres",
			fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s",
				viper.GetString("postgres.host"),
				viper.GetInt64("postgres.port"),
				viper.GetString("postgres.user"),
				viper.GetString("postgres.password"),
				viper.GetString("postgres.sslmode"),
			))
		if err != nil {
			time.Sleep(viper.GetDuration("postgres.connection.retrysleep"))
		} else {
			defer db.Close()
			break
		}
	}
	if err != nil {
		panic("Could not open connection to database")
	}

	// Migrate the schema
	ts := models.NewGormTransactionSupport(db)
	witRepo := models.NewWorkItemTypeRepository(ts)
	wiRepo := models.NewWorkItemRepository(ts, witRepo)

	if err := transaction.Do(ts, func() error {
		return migration.Perform(context.Background(), ts.TX(), witRepo)
	}); err != nil {
		panic(err.Error())
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
	service.Use(middleware.ErrorHandler(service, true))
	service.Use(middleware.Recover())

	publicKey, err := token.ParseRSAPublicKeyFromPEM([]byte(RSAPublicKey))
	if err != nil {
		panic(err)
	}
	app.UseJWTMiddleware(service, jwt.New(publicKey, nil, app.NewJWTSecurity()))

	// Mount "login" controller
	loginCtrl := NewLoginController(service)
	app.MountLoginController(service, loginCtrl)
	// Mount "version" controller
	versionCtrl := NewVersionController(service)
	app.MountVersionController(service, versionCtrl)

	// Mount "workitem" controller
	workitemCtrl := NewWorkitemController(service, wiRepo, ts)
	app.MountWorkitemController(service, workitemCtrl)

	// Mount "workitemtype" controller
	workitemtypeCtrl := NewWorkitemtypeController(service, witRepo, ts)
	app.MountWorkitemtypeController(service, workitemtypeCtrl)

	ts2 := models.NewGormTransactionSupport(db)

	// Mount "tracker" controller
	repo2 := remoteworkitem.NewTrackerRepository(ts2)
	c5 := NewTrackerController(service, repo2, ts2, scheduler)
	app.MountTrackerController(service, c5)

	// Mount "trackerquery" controller
	repo3 := remoteworkitem.NewTrackerQueryRepository(ts2)
	c6 := NewTrackerqueryController(service, repo3, ts2, scheduler)
	app.MountTrackerqueryController(service, c6)

	fmt.Println("Git Commit SHA: ", Commit)
	fmt.Println("UTC Build Time: ", BuildTime)
	fmt.Println("Dev mode:       ", viper.GetBool("developer.mode.enabled"))

	http.Handle("/api/", service.Mux)
	http.Handle("/", http.FileServer(assetFS()))
	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		_, err := db.DB().Exec("select 1")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	})

	// Start http
	if err := http.ListenAndServe(viper.GetString("http.address"), nil); err != nil {
		service.LogError("startup", "err", err)
	}

}

func printUserInfo() {
	u, err := user.Current()
	if err != nil {
		fmt.Printf("Failed to get current user: %s", err.Error())
	} else {
		fmt.Printf("Running as user name \"%s\" with UID %s.\n", u.Username, u.Uid)
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
