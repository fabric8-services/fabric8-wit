package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/remotetracker"
	token "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware"
	"github.com/goadesign/goa/middleware/security/jwt"
)

var (
	// Commit current build commit set by build script
	Commit = "0"
	// BuildTime set by build script
	BuildTime = "0"
	// Development enables certain dev only features, like auto token generation
	Development = false
)

func main() {
	var dbHost string
	var scheduler *remotetracker.Scheduler

	flag.BoolVar(&Development, "dev", false, "Enable development related features, e.g. token generation endpoint")
	flag.StringVar(&dbHost, "dbhost", "", "The hostname of the db server")
	flag.Parse()

	if len(dbHost) == 0 {
		dbHost = os.Getenv("DBHOST")
	}

	if len(dbHost) == 0 {
		dbHost = "localhost"
	}

	db, err := gorm.Open("postgres", fmt.Sprintf("host=%s user=postgres password=mysecretpassword sslmode=disable", dbHost))
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
	defer db.Close()

	// Migrate the schema
	migration.Perform(db)

	// Scheduler to fetch and import remote tracker items
	scheduler = remotetracker.NewScheduler(db)
	defer scheduler.Stop()
	//scheduler.ScheduleAllQueries()

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
	c := NewLoginController(service)
	app.MountLoginController(service, c)
	// Mount "version" controller
	c2 := NewVersionController(service)
	app.MountVersionController(service, c2)

	// Mount "workitem" controller
	ts := models.NewGormTransactionSupport(db)
	repo := models.NewWorkItemRepository(ts)
	c3 := NewWorkitemController(service, repo, ts)
	app.MountWorkitemController(service, c3)

	// Mount "workitemtype" controller
	c4 := NewWorkitemtypeController(service)
	app.MountWorkitemtypeController(service, c4)

	// Mount "tracker" controller
	repo2 := models.NewTrackerRepository(ts)
	c5 := NewTrackerController(service, repo2, ts)
	app.MountTrackerController(service, c5)

	// Mount "trackerquery" controller
	repo3 := models.NewTrackerQueryRepository(ts)
	c6 := NewTrackerqueryController(service, repo3, ts, scheduler)
	app.MountTrackerqueryController(service, c6)

	fmt.Println("Git Commit SHA: ", Commit)
	fmt.Println("UTC Build Time: ", BuildTime)
	fmt.Println("Dev mode:       ", Development)

	http.Handle("/api/", service.Mux)
	http.Handle("/", http.FileServer(assetFS()))
	http.Handle("/favicon.ico", http.NotFoundHandler())

	// Start http
	if err := http.ListenAndServe(":8080", nil); err != nil {
		service.LogError("startup", "err", err)
	}

}
